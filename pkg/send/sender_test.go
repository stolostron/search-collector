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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stolostron/search-collector/pkg/transforms"
	"github.com/stretchr/testify/assert"
)

func TestSenderWrongCount(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SyncResponse{
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
		httpClient:    *ts.Client(),
		aggregatorURL: ts.URL,
	}

	payload := Payload{}

	err := s.send(payload, 5, 0)
	if err == nil {
		t.Fatal("send function does not error when expected count differs")
	}

	message := "Aggregator reported wrong number of total resources"
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected error to contain \"%s\": got \"%s\"", message, err.Error())
	}
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

	err := s.send(payload, 0, 0)
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
		response := SyncResponse{
			TotalResources: n,
			TotalAdded:     n,
			TotalEdges:     n,
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

	err := s.send(payload, n, n)
	if err != nil {
		t.Fatal("send function reports error:", err)
	}
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
	assert.LessOrEqual(t, wait.Milliseconds(), int64(5000))
}
