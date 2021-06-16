/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package send

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/open-cluster-management/search-collector/pkg/config"
	"github.com/open-cluster-management/search-collector/pkg/reconciler"
	tr "github.com/open-cluster-management/search-collector/pkg/transforms"
)

// A UID of a node to be deleted, and the time at which it was deleted.
type DeleteNode struct {
	Time int64
	UID  string
}

// This type is for marshaling json which we send to the aggregator - it has to match the aggregator's API
type Payload struct {
	DeletedResources []tr.Deletion `json:"deleteResources,omitempty"` // List of UIDs of nodes which need to be deleted
	AddResources     []tr.Node     `json:"addResources,omitempty"`    // List of Nodes which must be added
	UpdatedResources []tr.Node     `json:"updateResources,omitempty"` // List of Nodes that exist and must be updated

	AddEdges    []tr.Edge `json:"addEdges,omitempty"`    // List of Edges which must be added
	DeleteEdges []tr.Edge `json:"deleteEdges,omitempty"` // List of Edges which must be deleted
	ClearAll    bool      `json:"clearAll,omitempty"`    // Tells the aggregator to clear existing data first.
	RequestId   int       `json:"requestId,omitempty"`   // Unique ID to track each request for debug.
	Version     string    `json:"version,omitempty"`     // Version of this collector
}

func (p Payload) empty() bool {
	return len(p.DeletedResources) == 0 && len(p.AddResources) == 0 && len(p.UpdatedResources) == 0 &&
		len(p.AddEdges) == 0 && len(p.DeleteEdges) == 0
}

// SyncResponse - Response to a SyncEvent
type SyncResponse struct {
	TotalAdded        int
	TotalUpdated      int
	TotalDeleted      int
	TotalResources    int
	TotalEdgesAdded   int
	TotalEdgesDeleted int
	TotalEdges        int
	AddErrors         []SyncError
	UpdateErrors      []SyncError
	DeleteErrors      []SyncError
	AddEdgeErrors     []SyncError
	DeleteEdgeErrors  []SyncError
	Version           string
}

// SyncError is used to respond with errors.
type SyncError struct {
	ResourceUID string
	Message     string
}

// Keeps the total data for this cluster as well as the data since the last send operation.
type Sender struct {
	aggregatorURL      string // URL of the aggregator, minus any path
	aggregatorSyncPath string // Path of the aggregator's POST route [ /aggregator/clusters/{clustername}/sync ]
	httpClient         http.Client
	lastSentTime       int64 // Time we last successfully sent data to the hub. Gets reset to -1 if a send cycle fails.
	rec                *reconciler.Reconciler
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

// Generates a random ID for the request.
func generateRequestId() int {
	max := big.NewInt(999099)
	valBig, err := rand.Int(rand.Reader, max)
	if err != nil {
		glog.Warning("Error generating RequestID.")
		return 0
	}
	return int(valBig.Int64())
}

// Returns a payload and expected total resources, add, update, and delete operations since the last send.
func (s *Sender) diffPayload() (Payload, int, int) {

	diff := s.rec.Diff()

	payload := Payload{
		ClearAll:  false,
		RequestId: generateRequestId(),
		Version:   config.COLLECTOR_API_VERSION,

		AddResources:     diff.AddNodes,
		UpdatedResources: diff.UpdateNodes,
		DeletedResources: diff.DeleteNodes,

		AddEdges:    diff.AddEdges,
		DeleteEdges: diff.DeleteEdges,
	}

	return payload, diff.TotalNodes, diff.TotalEdges
}

// Fetches complete state from the reconciler and transforms into payload struct
func (s *Sender) completePayload() (Payload, int, int) {

	complete := s.rec.Complete()

	// Delete and Update aren't needed when we're sending all the data. Just fill out the adds.
	payload := Payload{
		ClearAll:     true,
		RequestId:    generateRequestId(),
		AddResources: complete.Nodes,

		AddEdges: complete.Edges,
	}
	return payload, complete.TotalNodes, complete.TotalEdges
}

// Send will retry after recoverable errors.
//  - Aggregator busy
func (s *Sender) sendWithRetry(payload Payload, expectedTotalResources int, expectedTotalEdges int) error {
	retry := 0
	for {
		sendError := s.send(payload, expectedTotalResources, expectedTotalEdges)

		if sendError != nil && sendError.Error() == "Aggregator busy" {
			retry++
			waitMS := int(math.Min(float64(retry*15*1000), float64(config.Cfg.MaxBackoffMS)))
			glog.Warningf("Received busy response from Aggregator. Resending in %d ms.", waitMS)
			time.Sleep(time.Duration(waitMS) * time.Millisecond)
			continue
		} else if sendError != nil && sendError.Error() == "401 Unauthorized" {
			retry++
			waitMS := int(math.Min(float64(retry*15*1000), float64(config.Cfg.MaxBackoffMS)))
			glog.Warningf("Received Unauthorized response from Aggregator. Resending in %d ms after resetting config.",
				waitMS)
			config.InitConfig()
			s.httpClient = getHTTPSClient()
			time.Sleep(time.Duration(waitMS) * time.Millisecond)
			continue
		}
		//TODO: Handle unauthorized here with retry and reset config
		return sendError
	}
}

// Sends data to the aggregator and returns an error if it didn't work.
// Pointer receiver because Sender contains a mutex - that freaked the linter out even though it
// doesn't use the mutex. Changed it so that if we do need to use the mutex we wont have any problems.
func (s *Sender) send(payload Payload, expectedTotalResources int, expectedTotalEdges int) error {
	glog.Infof("Sending Resources { request: %d, add: %d, update: %d, delete: %d edge add: %d edge delete: %d }",
		payload.RequestId, len(payload.AddResources), len(payload.UpdatedResources), len(payload.DeletedResources),
		len(payload.AddEdges), len(payload.DeleteEdges))

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	payloadBuffer := bytes.NewBuffer(payloadBytes)
	resp, err := s.httpClient.Post(s.aggregatorURL+s.aggregatorSyncPath, "application/json", payloadBuffer)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		glog.Error("httpClient error: ", err)
		return err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.New("Aggregator busy")
	} else if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("POST to: %s responded with error. StatusCode: %d  Message: %s",
			s.aggregatorURL+s.aggregatorSyncPath, resp.StatusCode, resp.Status)
		if resp.StatusCode == http.StatusUnauthorized {
			msg = "401 Unauthorized"
		}
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
		msg := fmt.Sprintf("Aggregator reported wrong number of total resources. Expected %d, got %d",
			expectedTotalResources, r.TotalResources)
		return errors.New(msg)
	}

	if r.TotalEdges != (expectedTotalEdges + len(r.DeleteEdgeErrors) - len(r.AddEdgeErrors)) {
		msg := fmt.Sprintf("Aggregator reported wrong number of total intra edges. Expected %d, got %d",
			expectedTotalEdges, r.TotalEdges)
		return errors.New(msg)
	}

	// Check the total
	return nil
}

// Sends data to the aggregator.
// Attempts to send a diff, then just sends the complete if the aggregator appears to need that.
func (s *Sender) Sync() error {
	if s.lastSentTime == -1 { // If we have never sent before, we just send the complete.
		glog.Info("First time sending or last Sync cycle failed, sending complete payload")
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
		if err != nil {
			glog.Error("Sync sender error. ", err)
			return err
		}

		s.lastSentTime = time.Now().Unix()
		return nil
	}

	// If this isn't the first time we've sent, we can now attempt to send a diff.
	payload, expectedTotalResources, expectedTotalEdges := s.diffPayload()
	if payload.empty() {
		// check if a ping is necessary
		if time.Now().Unix()-s.lastSentTime < int64(config.Cfg.HeartbeatMS/1000) {
			glog.V(3).Info("Nothing to send, skipping send cycle.")
			return nil
		}
		glog.V(2).Info("Sending empty payload for heartbeat.")
	}
	err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
	if err != nil {
		// If something went wrong here, form a new complete payload (only necessary because
		// currentState may have changed since we got it, and we have to keep our diffs synced)
		glog.Warning("Error on diff payload sending: ", err)
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		glog.Warning("Retrying with complete payload")
		err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
		if err != nil {
			glog.Error("Error resending complete payload.")
			// If this retry fails, we want to start over with a complete payload next time,
			// so we reset as if we've not sent anything before.
			s.lastSentTime = -1
			return err
		}
		s.lastSentTime = time.Now().Unix()
		return nil
	}

	s.lastSentTime = time.Now().Unix()
	return nil
}

// Starts the send loop to send data on an interval.
// In case of error it backoffs and retries.
func (s *Sender) StartSendLoop() {

	// Used for exponential backoff, increased each interval. Has to be a float64 since I use it with math.Exp2()
	backoffFactor := float64(0)

	for {
		glog.V(3).Info("Beginning Send Cycle")
		err := s.Sync()
		if err != nil {
			glog.Error("SEND ERROR: ", err)
			// Increase the backoffFactor, doubling the wait time. Stops doubling it after it passes the max
			// wait time so that we don't overflow int. Can be changed with env:MAX_BACKOFF_MS
			if time.Duration(config.Cfg.ReportRateMS)*time.Duration(math.Exp2(backoffFactor))*time.Millisecond <
				time.Duration(config.Cfg.MaxBackoffMS)*time.Millisecond {
				backoffFactor++
			}
		} else {
			glog.V(2).Info("Send Cycle Completed Successfully")
			backoffFactor = float64(0) // Reset backoff to 0 because we had a sucessful send.
		}
		nextSleepInterval := config.Cfg.ReportRateMS * int(math.Exp2(backoffFactor))
		timeToSleep := time.Duration(min(nextSleepInterval, config.Cfg.MaxBackoffMS)) * time.Millisecond
		if backoffFactor > 0 {
			glog.Warning("Backing off send interval because of error response from aggregator. Sleeping for ", timeToSleep)
		}
		// Sleep either for the current backed off interval, or the maximum time defined in the config
		time.Sleep(timeToSleep)
	}
}

// Returns the smaller of two ints
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
