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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/reconciler"
	tr "github.ibm.com/IBMPrivateCloud/search-collector/pkg/transforms"
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
	DeletedResources []Deletion `json:"deleteResources,omitempty"` // List of UIDs of nodes which need to be deleted
	AddResources     []tr.Node  `json:"addResources,omitempty"`    // List of Nodes which must be added
	UpdatedResources []tr.Node  `json:"updateResources,omitempty"` // List of Nodes that already existed which must be updated

	AddEdges    []tr.Edge `json:"addEdges,omitempty"`    // List of Edges which must be added
	DeleteEdges []tr.Edge `json:"deleteEdges,omitempty"` // List of Edges which must be deleted
	Hash        string    `json:"hash,omitempty"`        // Hash of the previous state, used by aggregator to determine whether it needs to ask for the complete data
	ClearAll    bool      `json:"clearAll,omitempty"`    // Whether or not the aggregator should clear all data it has for the cluster first
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
	// lastHash           string                          // The hash that was sent with the last send - the first step of a send operation is to ask the aggregator for this hash, to determine whether we can send a diff or need to send the complete data.
	lastSentTime int64 // Time at which we last successfully sent data to the hub. Gets reset to -1 if a send cycle fails.
	rec          *reconciler.Reconciler
}

// Constructs a new Sender using the provided channels.
// Sends to the URL provided by aggregatorURL, listing itself as clusterName.
func NewSender(rec *reconciler.Reconciler, aggregatorURL, clusterName string) *Sender {

	// Construct senders
	s := &Sender{
		aggregatorURL:      aggregatorURL,
		aggregatorSyncPath: strings.Join([]string{"/aggregator/clusters/", clusterName, "/sync"}, ""),
		httpClient:         getHTTPSClient(),
		lastSentTime:       -1,
		rec:                rec,
	}

	if !config.Cfg.DeployedInHub {
		s.aggregatorSyncPath = strings.Join([]string{"/", clusterName, "/aggregator/sync"}, "")
	}

	return s
}

// Returns a payload and expected total resources, containing the add, update and delete operations since the last send.
func (s *Sender) diffPayload() (Payload, int) {

	diff := s.rec.Diff()
	// glog.Warningf("%+v", diff) //RM

	payload := Payload{
		ClearAll: false,

		AddResources:     diff.AddNodes,
		UpdatedResources: diff.UpdateNodes,
		DeletedResources: make([]Deletion, len(diff.DeleteNodes)),

		AddEdges:    diff.AddEdges,
		DeleteEdges: diff.DeleteEdges,
	}

	for _, uid := range diff.DeleteNodes {
		payload.DeletedResources = append(payload.DeletedResources, Deletion{uid})
	}

	return payload, s.rec.ResourceCount()
}

// Returns a payload and expected total resources, containing the complete set of resources as they currently exist in this cluster
// This function also RESETS THE DIFFS, so make sure you do something with the payload
func (s *Sender) completePayload() (Payload, int) {

	complete := s.rec.Complete()
	// glog.Warningf("%+v", complete) //RM

	// Hash, Delete and Update aren't needed when we're sending all the data. Just fill out the adds.
	payload := Payload{
		ClearAll: true,

		AddResources: complete.Nodes,

		AddEdges: complete.Edges,
	}
	return payload, s.rec.ResourceCount()
}

// Sends data to the aggregator and returns an error if it didn't work.
// Pointer receiver because Sender contains a mutex - that freaked the linter out even though it doesn't use the mutex. Changed it so that if we do need to use the mutex we wont have any problems.
func (s *Sender) send(payload Payload, expectedTotalResources int) error {
	glog.Infof("Sending Resources { add: %d, update: %d, delete: %d edge add: %d edge delete: %d }", len(payload.AddResources), len(payload.UpdatedResources), len(payload.DeletedResources), len(payload.AddEdges), len(payload.DeleteEdges))

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
		msg := fmt.Sprintf("POST to: %s responded with error. StatusCode: %d  Message: %s", s.aggregatorURL+s.aggregatorSyncPath, resp.StatusCode, resp.Status)
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
			glog.V(2).Info("Nothing to send, skipping send cycle")
			return nil
		}
		glog.Info("Sending empty payload for heartbeat")
	} else {
		glog.V(2).Info("Sending diff payload")
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
