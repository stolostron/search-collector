/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package send

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/groupcache/lru"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

func initTestSender() *Sender {
	return &Sender{
		previousState: make(map[string]transforms.Node),
		currentState:  make(map[string]transforms.Node),
		diffState:     make(map[string]transforms.NodeEvent),
		lastSentTime:  -1,
		InputChannel:  make(chan transforms.NodeEvent),
		purgedNodes:   lru.New(10),
	}
}

func TestReconcilerOutOfOrderDelete(t *testing.T) {
	s := initTestSender()
	ts := time.Now().Unix()

	go func() {
		s.InputChannel <- transforms.NodeEvent{
			Time:      ts,
			Operation: transforms.Delete,
			Node: transforms.Node{
				UID: "test-event",
			},
		}

		s.InputChannel <- transforms.NodeEvent{
			Time:      ts - 1000, // insert out of order based off of time
			Operation: transforms.Create,
			Node: transforms.Node{
				UID: "test-event",
			},
		}
	}()

	// need two calls to drain the queue
	reconcileNode(s)
	reconcileNode(s)

	if _, found := s.currentState["test-event"]; found {
		t.Fatal("failed to ignore add event received out of order")
	}

	if _, found := s.purgedNodes.Get("test-event"); !found {
		t.Fatal("failed to added deleted NodeEvent to purgedNodes cache")
	}
}

func TestReconcilerOutOfOrderAdd(t *testing.T) {
	s := initTestSender()
	ts := time.Now().Unix()

	go func() {
		s.InputChannel <- transforms.NodeEvent{
			Time:      ts,
			Operation: transforms.Create,
			Node: transforms.Node{
				UID: "test-event",
			},
		}

		s.InputChannel <- transforms.NodeEvent{
			Time:      ts - 1000, // insert out of order based off of time
			Operation: transforms.Create,
			Node: transforms.Node{
				UID: "test-event",
				Properties: map[string]interface{}{
					"staleData": true,
				},
			},
		}
	}()

	// need two calls to drain the queue
	reconcileNode(s)
	reconcileNode(s)

	testNode, ok := s.currentState["test-event"]
	if !ok {
		t.Fatal("failed to add test node to current state")
	}

	if _, ok := testNode.Properties["staleData"]; ok {
		t.Fatal("inserted nodes out of order: found stale data")
	}
}

func TestReconcilerAddDelete(t *testing.T) {
	s := initTestSender()

	go func() {
		s.InputChannel <- transforms.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: transforms.Create,
			Node: transforms.Node{
				UID: "test-event",
			},
		}
	}()

	reconcileNode(s)

	if _, ok := s.currentState["test-event"]; !ok {
		t.Fatal("failed to add test event to current state")
	}
	if _, ok := s.diffState["test-event"]; !ok {
		t.Fatal("failed to add test event to diff state")
	}

	go func() {
		s.InputChannel <- transforms.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: transforms.Delete,
			Node: transforms.Node{
				UID: "test-event",
			},
		}
	}()

	reconcileNode(s)

	if _, ok := s.currentState["test-event"]; ok {
		t.Fatal("failed to remove test event from current state")
	}
	if _, ok := s.diffState["test-event"]; ok {
		t.Fatal("failed to remove test event from diff state")
	}
}

func TestReconcilerRedundant(t *testing.T) {
	s := initTestSender()
	s.previousState["test-event"] = transforms.Node{
		UID: "test-event",
		Properties: map[string]interface{}{
			"very": "important",
		},
	}

	go func() {
		s.InputChannel <- transforms.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: transforms.Create,
			Node: transforms.Node{
				UID: "test-event",
				Properties: map[string]interface{}{
					"very": "important",
				},
			},
		}
	}()

	reconcileNode(s)

	if _, ok := s.diffState["test-event"]; ok {
		t.Fatal("failed to ignore redundant add event")
	}
}

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

	s := initTestSender()
	s.httpClient = *ts.Client()
	s.aggregatorURL = ts.URL

	payload := Payload{}

	err := s.send(payload, 5)
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

	s := initTestSender()
	s.httpClient = *ts.Client()
	s.aggregatorURL = ts.URL

	payload := Payload{}

	err := s.send(payload, 0)
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
			TotalResources:   n,
			TotalAdded:       n,
			UpdatedTimestamp: time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	s := initTestSender()
	s.httpClient = *ts.Client()
	s.aggregatorURL = ts.URL

	payload := Payload{
		ClearAll: false,
	}

	for i := 0; i < n; i++ {
		payload.AddResources = append(payload.AddResources, transforms.Node{
			UID: fmt.Sprintf("Node%d", i),
		})
	}

	err := s.send(payload, n)
	if err != nil {
		t.Fatal("send function reports error:", err)
	}
}
