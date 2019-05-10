/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package send

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/golang/groupcache/lru"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
)

// A UID of a node to be deleted, and the time at which it was deleted.
type DeleteNode struct {
	Time int64
	UID  string
}

// Represents the operations needed to bring the hub's version of this cluster's data into sync.
/* I'll put this back in once we add edges - Ethan
type Diff struct {
	nodes NodeDiff `json:"nodes"`
	// edges EdgeDiff
}
*/

// TODO import this type from the aggregator.
// This type is for marshaling json which we send to the aggregator - it has to match the aggregator's API
type Payload struct {
	DeletedResources []Deletion        `json:"deleteResources,omitempty"` // List of UIDs of nodes which need to be deleted
	AddResources     []transforms.Node `json:"addResources,omitempty"`    // List of Nodes which must be added
	UpdatedResources []transforms.Node `json:"updateResources,omitempty"` // List of Nodes that already existed which must be updated
	Hash             string            `json:"hash,omitempty"`            // Hash of the previous state, used by aggregator to determine whether it needs to ask for the complete data
	ClearAll         bool              `json:"clearAll,omitempty"`        // Whether or not the aggregator should clear all data it has for the cluster first
}

type Deletion struct {
	UID string `json:"uid,omitempty"`
}

func (p Payload) empty() bool {
	return len(p.DeletedResources) == 0 && len(p.AddResources) == 0 && len(p.UpdatedResources) == 0
}

// TODO import this type from the aggregator.
type GetResponse struct {
	Hash           string
	LastUpdated    string
	TotalResources int
	MaxQueueTime   int
}

// TODO import this type from the aggregator.
// SyncResponse - Response to a SyncEvent
type SyncResponse struct {
	Hash             string
	TotalAdded       int
	TotalUpdated     int
	TotalDeleted     int
	TotalResources   int
	UpdatedTimestamp time.Time
	DeleteErrors     []SyncError
	UpdateErrors     []SyncError
	AddErrors        []SyncError
}

// TODO import this type from the aggregator.
// SyncError is used to respond whith errors.
type SyncError struct {
	ResourceUID string
	Message     string
}

// Keeps the total data for this cluster as well as the data since the last send operation.
type Sender struct {
	aggregatorURL      string // URL of the aggregator, minus any path
	aggregatorSyncPath string // Path of the aggregator's POST route for syncing data, as of today /aggregator/clusters/{clustername}/sync
	httpClient         http.Client
	currentState       map[string]transforms.Node      // In the future this will be an object that has edges in it too
	previousState      map[string]transforms.Node      // In the future this will be an object that has edges in it too
	diffState          map[string]transforms.NodeEvent // In the future this will be an object that has edges in it too
	// lastHash           string                          // The hash that was sent with the last send - the first step of a send operation is to ask the aggregator for this hash, to determine whether we can send a diff or need to send the complete data.
	lastSentTime int64                     // Time at which we last successfully sent data to the hub. Gets reset to -1 if a send cycle fails.
	InputChannel chan transforms.NodeEvent // Put any nodes to be updated, added or deleted in here
	mutex        sync.Mutex                // Used to protect currentState and diffState as they are edited and read by multiple goroutines
	purgedNodes  *lru.Cache                // used to keep tracked to old purged nodes so the reconciler can throw away old events that predate deletion events
}

// Constructs a new Sender using the provided channels.
// Sends to the URL provided by aggregatorURL, listing itself as clusterName.
func NewSender(inputChan chan transforms.NodeEvent, aggregatorURL, clusterName string) *Sender {

	// Construct senders
	s := &Sender{
		aggregatorURL:      aggregatorURL,
		aggregatorSyncPath: strings.Join([]string{"/aggregator/clusters/", clusterName, "/sync"}, ""),
		httpClient:         getHTTPSClient(),
		previousState:      make(map[string]transforms.Node),
		currentState:       make(map[string]transforms.Node),
		diffState:          make(map[string]transforms.NodeEvent),
		// lastHash:           NEVER_SENT,
		lastSentTime: -1,
		InputChannel: inputChan,
		purgedNodes:  lru.New(500),
	}

	if !config.Cfg.DeployedInHub {
		s.aggregatorSyncPath = strings.Join([]string{"/", clusterName, "/aggregator/sync"}, "")
	}

	go s.Reconciler()

	return s
}

// Returns a payload and expected total resources, containing the add, update and delete operations since the last send.
// This function also RESETS THE DIFFS, so make sure you do something with the payload
func (s *Sender) diffPayload() (Payload, int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	payload := Payload{
		ClearAll: false,
	}

	for _, ndo := range s.diffState {
		if ndo.Operation == transforms.Create {
			payload.AddResources = append(payload.AddResources, ndo.Node)
		} else if ndo.Operation == transforms.Update {
			payload.UpdatedResources = append(payload.UpdatedResources, ndo.Node)
		} else if ndo.Operation == transforms.Delete {
			payload.DeletedResources = append(payload.DeletedResources, Deletion{UID: ndo.UID})
		}
	}

	expectedTotalResources := len(s.currentState)
	s.resetDiffs()
	return payload, expectedTotalResources
}

// Returns a payload and expected total resources, containing the complete set of resources as they currently exist in this cluster
// This function also RESETS THE DIFFS, so make sure you do something with the payload
func (s *Sender) completePayload() (Payload, int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	payload := Payload{
		ClearAll: true,
	}
	// Hash, Delete and Update aren't needed when we're sending all the data. We do need to fill out addResources, though.
	for _, n := range s.currentState {
		payload.AddResources = append(payload.AddResources, n)
	}
	expectedTotalResources := len(s.currentState)
	s.resetDiffs()
	return payload, expectedTotalResources
}

// Clears out diffState and copies currentState into previousState.
// NOT THREADSAFE with anything that edits structures in s, locking left up to the caller.
func (s *Sender) resetDiffs() {
	s.diffState = make(map[string]transforms.NodeEvent) // We have to reset the diff every time we try to prepare something to send, so that it doesn't get out of sync with the complete/old.
	s.previousState = make(map[string]transforms.Node, len(s.currentState))

	for uid, node := range s.currentState {
		s.previousState[uid] = node
	}
}

// Sends data to the aggregator and returns an error if it didn't work.
// Pointer receiver because Sender contains a mutex - that freaked the linter out even though it doesn't use the mutex. Changed it so that if we do need to use the mutex we wont have any problems.
func (s *Sender) send(payload Payload, expectedTotalResources int) error {
	glog.Infof("Sending { add: %d, update: %d, delete: %d }", len(payload.AddResources), len(payload.UpdatedResources), len(payload.DeletedResources))

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// glog.Warning(string(payloadBytes))
	payloadBuffer := bytes.NewBuffer(payloadBytes)
	resp, err := s.httpClient.Post(s.aggregatorURL+s.aggregatorSyncPath, "application/json", payloadBuffer)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("POST to: %s responsed with error. StatusCode: %d  Message: %s", s.aggregatorURL+s.aggregatorSyncPath, resp.StatusCode, resp.Status)
		return errors.New(msg)
	}

	r := SyncResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		glog.Error("Error decoding JSON response.")
		return err
	}

	// Compare size that comes back in r to size that we track, accounting for the errors reported by the aggregator.
	if r.TotalResources != (expectedTotalResources + len(r.DeleteErrors) - len(r.AddErrors)) {
		msg := fmt.Sprintf("Aggregator reported wrong number of total resources. Expected %d, got %d", expectedTotalResources, r.TotalResources)
		return errors.New(msg) //TODO This maybe should be declared at the package level and then just returned
	}
	return nil
}

// Sends data to the aggregator. Attempts to send a diff, then just sends the complete if the aggregator appears to need that.
func (s *Sender) Sync() error {
	if s.lastSentTime == -1 { // If we have never sent before, we just send the complete.
		glog.Info("First time sending or last Sync cycle failed, sending complete payload")
		payload, expectedTotalResources := s.completePayload()
		err := s.send(payload, expectedTotalResources)
		if err != nil {
			return err
		}

		s.lastSentTime = time.Now().Unix()
		return nil
	}

	// If this isn't the first time we've sent, we can now attempt to send a diff.
	payload, expectedTotalResources := s.diffPayload()
	if payload.empty() {
		// check if a ping is necessary
		if time.Now().Unix()-s.lastSentTime < int64(config.Cfg.HeartbeatMS/1000) {
			glog.Info("Nothing to send, skipping send cycle")
			return nil
		}
		glog.Info("Sending empty payload for heartbeat")
	} else {
		glog.Info("Sending diff payload")
	}
	err := s.send(payload, expectedTotalResources)
	if err != nil { // If something went wrong here, form a new complete payload (only necessary because currentState may have changed since we got it, and we have to keep our diffs synced)
		glog.Warning("Error on diff payload sending: ", err)
		payload, expectedTotalResources := s.completePayload()
		glog.Warning("Retrying with complete payload")
		err := s.send(payload, expectedTotalResources)
		if err != nil {
			s.lastSentTime = -1 // If this retry fails, we want to start over with a complete payload next time, so we reset as if we've not sent anything before.
			return err
		}
		s.lastSentTime = time.Now().Unix()
		return nil
	}

	s.lastSentTime = time.Now().Unix()
	return nil
}

// This method takes a channel and constantly receives from it, reconciling the input with whatever is currently stored in the sender.
func (s *Sender) Reconciler() {
	glog.Info("Reconciler Routine Started")
	for {
		reconcileNode(s)
	}
}

// This is a separate funcition so we can defer the mutex unlock and guarantee the lock is lifted every iteration
func reconcileNode(s *Sender) {
	ne := <-s.InputChannel

	// Take care of diffState and currentState
	s.mutex.Lock() // Have to lock before the if statements, little awkward but if we made the decision to go ahead and edit and then blocked, we could end up getting out of order
	defer s.mutex.Unlock()

	// Check whether we already have this node in our current/purged state with a more up to date time. If so, we ignore the version of it we're currently processing.
	otherNode, inCurrent := s.diffState[ne.Node.UID]
	nodeInterface, inPurged := s.purgedNodes.Get(ne.Node.UID)

	if inCurrent && otherNode.Time > ne.Time {
		return
	}
	if inPurged {
		puregedNode, ok := nodeInterface.(transforms.NodeEvent)
		if ok && puregedNode.Time > ne.Time {
			return
		}
	}

	previousNode, inPrevious := s.previousState[ne.Node.UID]

	if ne.Operation == transforms.Delete {
		delete(s.currentState, ne.UID) // Get rid of it from our currentState, if it was ever there.
		s.purgedNodes.Add(ne.UID, ne)  // Add this to the list of node purged resources

		if inPrevious {
			s.diffState[ne.UID] = ne // Since it was in the previous, we need to have a deletion diff.
		} else {
			delete(s.diffState, ne.UID) // Otherwise no need to send a payload, just remove from local memory
		}
	} else { // This is either an update or create, which look very similar. TODO actually combine the two.
		op := transforms.Create
		if inPrevious { // If this was in the previous, our operation for diffs is update, not create
			op = transforms.Update

			// skip updates if new event is redundant to our previous state (a property that we don't care about triggered an update)
			if reflect.DeepEqual(ne.Node, previousNode) {
				return
			}
		}
		s.currentState[ne.UID] = ne.Node
		ne.Operation = op
		s.diffState[ne.UID] = ne
	}
}
