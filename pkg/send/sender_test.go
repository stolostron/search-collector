/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
*/

package send

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/reconciler"
	"github.com/stolostron/search-collector/pkg/transforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPClient(statusCode int, body string) http.Client {
	return http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
}

func TestSenderWrongCount(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ValidationResponse{
			TotalResources: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	s := Sender{
		httpClient:               *ts.Client(),
		aggregatorURL:            ts.URL,
		aggregatorValidationPath: "/validation",
		lastSentTime:             1,
	}

	err := s.validate(5, 0)
	if err == nil {
		t.Fatal("validate function does not error when expected count differs")
	}

	message := "aggregator reported wrong number of total resources"
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected error to contain \"%s\": got \"%s\"", message, err.Error())
	}
	assert.Equal(t, int64(-1), s.lastSentTime)
}

func TestSenderUnavailable(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
	}))
	defer ts.Close()

	s := Sender{
		httpClient:    *ts.Client(),
		aggregatorURL: ts.URL,
	}

	payload := Payload{}

	_, err := s.send(payload)
	if err == nil {
		t.Fatal("send function does not error if server returns a 503")
	}

	message := "503 Service Unavailable"
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected error to contain \"%s\": got \"%s\"", message, err.Error())
	}
}

func TestSenderSuccessful(t *testing.T) {
	// number of nodes to add in this test
	n := 5

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SyncResponse{TotalAdded: n}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	s := Sender{
		httpClient:    *ts.Client(),
		aggregatorURL: ts.URL,
	}

	payload := Payload{
		ClearAll: false,
	}

	for i := 0; i < n; i++ {
		payload.AddResources = append(payload.AddResources, transforms.Node{
			UID: fmt.Sprintf("Node%d", i),
		})
	}

	_, err := s.send(payload)
	if err != nil {
		t.Fatal("send function reports error:", err)
	}
}

func TestValidationInterval(t *testing.T) {
	requestCount := 0
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_ = json.NewEncoder(w).Encode(ValidationResponse{TotalResources: 5, TotalEdges: 3})
	}))
	defer ts.Close()

	oldInterval := config.Cfg.ResourceCountIntervalMS
	config.Cfg.ResourceCountIntervalMS = 300000
	defer func() { config.Cfg.ResourceCountIntervalMS = oldInterval }()

	s := Sender{
		httpClient:               *ts.Client(),
		aggregatorURL:            ts.URL,
		aggregatorValidationPath: "/validation",
		lastValidationTime:       time.Now(),
	}
	assert.NoError(t, s.validate(5, 3))
	assert.Equal(t, 0, requestCount)

	s.lastValidationTime = time.Now().Add(-6 * time.Minute)
	assert.NoError(t, s.validate(5, 3))
	assert.Equal(t, 1, requestCount)
}

func TestValidationErrors(t *testing.T) {
	oldInterval := config.Cfg.ResourceCountIntervalMS
	config.Cfg.ResourceCountIntervalMS = 0
	defer func() { config.Cfg.ResourceCountIntervalMS = oldInterval }()

	tests := []struct {
		name        string
		client      http.Client
		url         string
		body        string
		resources   int
		edges       int
		errorText   string
		resetsState bool
	}{
		{
			name:      "invalid URL",
			client:    testHTTPClient(http.StatusOK, `{}`),
			url:       "https://invalid\n",
			errorText: "invalid control character",
		},
		{
			name: "transport error",
			client: http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("transport failed")
			})},
			errorText: "transport failed",
		},
		{
			name:      "non-OK response",
			client:    testHTTPClient(http.StatusTooManyRequests, "busy"),
			errorText: "429 Too Many Requests",
		},
		{
			name:      "invalid response",
			client:    testHTTPClient(http.StatusOK, "not-json"),
			errorText: "invalid character",
		},
		{
			name:        "edge count mismatch",
			client:      testHTTPClient(http.StatusOK, `{"TotalResources":5,"TotalEdges":2}`),
			resources:   5,
			edges:       3,
			errorText:   "wrong number of total intra edges",
			resetsState: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := Sender{
				httpClient:               test.client,
				aggregatorURL:            test.url,
				aggregatorValidationPath: "/validation",
				lastSentTime:             123,
			}
			err := s.validate(test.resources, test.edges)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorText)
			if test.resetsState {
				assert.Equal(t, int64(-1), s.lastSentTime)
			} else {
				assert.Equal(t, int64(123), s.lastSentTime)
			}
		})
	}
}

func TestValidationRequest(t *testing.T) {
	oldInterval := config.Cfg.ResourceCountIntervalMS
	config.Cfg.ResourceCountIntervalMS = 0
	defer func() { config.Cfg.ResourceCountIntervalMS = oldInterval }()

	client := http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://indexer/aggregator/clusters/test/validation", req.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"TotalResources":5,"TotalEdges":3}`)),
		}, nil
	})}
	s := Sender{
		httpClient:               client,
		aggregatorURL:            "https://indexer",
		aggregatorValidationPath: "/aggregator/clusters/test/validation",
	}

	require.NoError(t, s.validate(5, 3))
	assert.False(t, s.lastValidationTime.IsZero())
}

func TestSendErrorResponses(t *testing.T) {
	tests := []struct {
		name      string
		sender    Sender
		payload   Payload
		errorText string
	}{
		{
			name:      "marshal error",
			sender:    Sender{},
			payload:   Payload{AddResources: []transforms.Node{{Properties: map[string]interface{}{"bad": func() {}}}}},
			errorText: "unsupported type",
		},
		{
			name:      "invalid URL",
			sender:    Sender{aggregatorURL: "https://invalid\n"},
			errorText: "invalid control character",
		},
		{
			name: "transport error",
			sender: Sender{httpClient: http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("transport failed")
			})}},
			errorText: "transport failed",
		},
		{
			name:      "busy",
			sender:    Sender{httpClient: testHTTPClient(http.StatusTooManyRequests, "busy")},
			errorText: "indexer busy",
		},
		{
			name:      "unauthorized",
			sender:    Sender{httpClient: testHTTPClient(http.StatusUnauthorized, "unauthorized")},
			errorText: "401 Unauthorized",
		},
		{
			name:      "invalid response",
			sender:    Sender{httpClient: testHTTPClient(http.StatusOK, "not-json")},
			errorText: "invalid character",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.sender.send(test.payload)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorText)
		})
	}
}

func TestSendWithRetryReturnsResponse(t *testing.T) {
	s := Sender{httpClient: testHTTPClient(http.StatusOK, `{"TotalAdded":4}`)}

	response, err := s.sendWithRetry(Payload{})

	require.NoError(t, err)
	assert.Equal(t, 4, response.TotalAdded)
}

func TestSenderPathsInHub(t *testing.T) {
	oldCfg := config.Cfg
	config.Cfg.DeployedInHub = true
	defer func() { config.Cfg = oldCfg }()

	s := NewSender(nil, "https://indexer", "test-cluster", nil)
	assert.Equal(t, "/aggregator/clusters/test-cluster/sync", s.aggregatorSyncPath)
	assert.Equal(t, "/aggregator/clusters/test-cluster/validation", s.aggregatorValidationPath)
	assert.False(t, s.lastValidationTime.IsZero())

	config.Cfg.AggregatorURL = "https://reloaded"
	config.Cfg.ClusterName = "other-cluster"
	s.reloadSender()
	assert.Equal(t, "https://reloaded", s.aggregatorURL)
	assert.Equal(t, "/aggregator/clusters/other-cluster/sync", s.aggregatorSyncPath)
	assert.Equal(t, "/aggregator/clusters/other-cluster/validation", s.aggregatorValidationPath)
}

func TestSyncSerializesSendAndValidation(t *testing.T) {
	oldInterval := config.Cfg.ResourceCountIntervalMS
	config.Cfg.ResourceCountIntervalMS = 0
	defer func() { config.Cfg.ResourceCountIntervalMS = oldInterval }()

	postStarted := make(chan struct{})
	releasePost := make(chan struct{})
	validationStarted := make(chan struct{}, 1)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/sync":
			close(postStarted)
			<-releasePost
			_ = json.NewEncoder(w).Encode(SyncResponse{})
		case "/validation":
			validationStarted <- struct{}{}
			_ = json.NewEncoder(w).Encode(ValidationResponse{})
		default:
			http.NotFound(w, req)
		}
	}))
	defer ts.Close()

	s := Sender{
		aggregatorURL:            ts.URL,
		aggregatorSyncPath:       "/sync",
		aggregatorValidationPath: "/validation",
		httpClient:               *ts.Client(),
		lastSentTime:             -1,
		rec:                      reconciler.NewReconciler(),
	}
	syncDone := make(chan error, 1)
	go func() { syncDone <- s.Sync() }()

	<-postStarted
	select {
	case <-validationStarted:
		t.Fatal("validation started before the sync response completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(releasePost)
	require.NoError(t, <-syncDone)
	select {
	case <-validationStarted:
	default:
		t.Fatal("validation was not requested after sync completed")
	}
}

func TestSyncWithEmptyDiffOnlyValidates(t *testing.T) {
	oldInterval := config.Cfg.ResourceCountIntervalMS
	config.Cfg.ResourceCountIntervalMS = 0
	defer func() { config.Cfg.ResourceCountIntervalMS = oldInterval }()

	postCount := 0
	getCount := 0
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			postCount++
			_ = json.NewEncoder(w).Encode(SyncResponse{})
			return
		}
		getCount++
		_ = json.NewEncoder(w).Encode(ValidationResponse{})
	}))
	defer ts.Close()

	s := Sender{
		aggregatorURL:            ts.URL,
		aggregatorSyncPath:       "/sync",
		aggregatorValidationPath: "/validation",
		httpClient:               *ts.Client(),
		lastSentTime:             time.Now().Unix(),
		rec:                      reconciler.NewReconciler(),
	}

	require.NoError(t, s.Sync())
	assert.Equal(t, 0, postCount)
	assert.Equal(t, 1, getCount)
}

func Test_minIsB(t *testing.T) {
	min := min(11, 99)
	assert.Equal(t, 11, min)
}
func Test_minIsA(t *testing.T) {
	min := min(99, 11)
	assert.Equal(t, 11, min)
}

func Test_sendInterval(t *testing.T) {
	wait := sendInterval(1)
	assert.GreaterOrEqual(t, wait.Milliseconds(), int64(1000))
	assert.LessOrEqual(t, wait.Milliseconds(), int64(7000))
}

func Test_sendInterval_maxBackoff(t *testing.T) {
	wait := sendInterval(50)
	assert.Equal(t, int64(600000), wait.Milliseconds())
}

func Test_sendInterval_minBackoff(t *testing.T) {
	wait := sendInterval(0)
	assert.GreaterOrEqual(t, wait.Milliseconds(), int64(0))
	assert.LessOrEqual(t, wait.Milliseconds(), int64(6000))
}
