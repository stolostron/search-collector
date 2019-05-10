/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package send

import (
	"testing"
	"time"

	"github.com/golang/groupcache/lru"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

func TestReconcilerOutOfOrderDelete(t *testing.T) {
	s := &Sender{
		previousState: make(map[string]transforms.Node),
		currentState:  make(map[string]transforms.Node),
		diffState:     make(map[string]transforms.NodeEvent),
		lastSentTime:  -1,
		InputChannel:  make(chan transforms.NodeEvent),
		purgedNodes:   lru.New(10),
	}

	// need two calls to drain the queue
	go reconcileNode(s)
	go reconcileNode(s)

	s.InputChannel <- transforms.NodeEvent{
		Time:      time.Now().Unix(),
		Operation: transforms.Delete,
		Node: transforms.Node{
			UID: "test-event",
		},
	}

	s.InputChannel <- transforms.NodeEvent{
		Time:      time.Now().Unix() - 1000, // insert out of order based off of time
		Operation: transforms.Create,
		Node: transforms.Node{
			UID: "test-event",
		},
	}

	if _, found := s.currentState["test-event"]; found {
		t.Fatal("failed to ignore add event received out of order")
	}

	if _, found := s.purgedNodes.Get("test-event"); !found {
		t.Fatal("failed to added deleted NodeEvent to purgedNodes cache")
	}
}
