/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package reconciler

import (
	"testing"
	"time"

	lru "github.com/golang/groupcache/lru"
	tr "github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

func initTestReconciler() *Reconciler {
	return &Reconciler{
		currentNodes:  make(map[string]tr.Node),
		previousNodes: make(map[string]tr.Node),
		diffNodes:     make(map[string]tr.NodeEvent),

		edgeFuncs: make(map[string]func(ns tr.NodeStore) []tr.Edge),

		Input:       make(chan tr.NodeEvent),
		purgedNodes: lru.New(CACHE_SIZE),
	}
}

func TestReconcilerOutOfOrderDelete(t *testing.T) {
	s := initTestReconciler()
	ts := time.Now().Unix()

	go func() {
		s.Input <- tr.NodeEvent{
			Time:      ts,
			Operation: tr.Delete,
			Node: tr.Node{
				UID: "test-event",
			},
		}

		s.Input <- tr.NodeEvent{
			Time:      ts - 1000, // insert out of order based off of time
			Operation: tr.Create,
			Node: tr.Node{
				UID: "test-event",
			},
		}
	}()

	// need two calls to drain the queue
	s.reconcileNode()
	s.reconcileNode()

	if _, found := s.currentNodes["test-event"]; found {
		t.Fatal("failed to ignore add event received out of order")
	}

	if _, found := s.purgedNodes.Get("test-event"); !found {
		t.Fatal("failed to added deleted NodeEvent to purgedNodes cache")
	}
}

func TestReconcilerOutOfOrderAdd(t *testing.T) {
	s := initTestReconciler()
	ts := time.Now().Unix()

	go func() {
		s.Input <- tr.NodeEvent{
			Time:      ts,
			Operation: tr.Create,
			Node: tr.Node{
				UID: "test-event",
			},
		}

		s.Input <- tr.NodeEvent{
			Time:      ts - 1000, // insert out of order based off of time
			Operation: tr.Create,
			Node: tr.Node{
				UID: "test-event",
				Properties: map[string]interface{}{
					"staleData": true,
				},
			},
		}
	}()

	// need two calls to drain the queue
	s.reconcileNode()
	s.reconcileNode()

	testNode, ok := s.currentNodes["test-event"]
	if !ok {
		t.Fatal("failed to add test node to current state")
	}

	if _, ok := testNode.Properties["staleData"]; ok {
		t.Fatal("inserted nodes out of order: found stale data")
	}
}

func TestReconcilerAddDelete(t *testing.T) {
	s := initTestReconciler()

	go func() {
		s.Input <- tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Create,
			Node: tr.Node{
				UID: "test-event",
			},
		}
	}()

	s.reconcileNode()

	if _, ok := s.currentNodes["test-event"]; !ok {
		t.Fatal("failed to add test event to current state")
	}
	if _, ok := s.diffNodes["test-event"]; !ok {
		t.Fatal("failed to add test event to diff state")
	}

	go func() {
		s.Input <- tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Delete,
			Node: tr.Node{
				UID: "test-event",
			},
		}
	}()

	s.reconcileNode()

	if _, ok := s.currentNodes["test-event"]; ok {
		t.Fatal("failed to remove test event from current state")
	}
	if _, ok := s.diffNodes["test-event"]; ok {
		t.Fatal("failed to remove test event from diff state")
	}
}

func TestReconcilerRedundant(t *testing.T) {
	s := initTestReconciler()
	s.previousNodes["test-event"] = tr.Node{
		UID: "test-event",
		Properties: map[string]interface{}{
			"very": "important",
		},
	}

	go func() {
		s.Input <- tr.NodeEvent{
			Time:      time.Now().Unix(),
			Operation: tr.Create,
			Node: tr.Node{
				UID: "test-event",
				Properties: map[string]interface{}{
					"very": "important",
				},
			},
		}
	}()

	s.reconcileNode()

	if _, ok := s.diffNodes["test-event"]; ok {
		t.Fatal("failed to ignore redundant add event")
	}
}
