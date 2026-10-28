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
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/metrics"
	"github.com/stolostron/search-collector/pkg/reconciler"
	tr "github.com/stolostron/search-collector/pkg/transforms"
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
	TotalEdgesAdded   int
	TotalEdgesDeleted int
	AddErrors         []SyncError
	UpdateErrors      []SyncError
	DeleteErrors      []SyncError
	AddEdgeErrors     []SyncError
	DeleteEdgeErrors  []SyncError
	Version           string
}

// ValidationResponse contains the resource and edge totals returned from the Indexer.
type ValidationResponse struct {
	TotalResources int
	TotalEdges     int
}

// SyncError is used to respond with errors.
type SyncError struct {
	ResourceUID string
	Message     string
}

// Keeps the total data for this cluster as well as the data since the last send operation.
type Sender struct {
	aggregatorURL            string // URL of the aggregator, minus any path
	aggregatorSyncPath       string // Path of the aggregator's POST route [ /aggregator/clusters/{clustername}/sync ]
	aggregatorValidationPath string // Path of the aggregator's validation GET route [ /aggregator/clusters/{clustername}/validation ]
	httpClient               http.Client
	lastSentTime             int64 // Time we last successfully sent data to the hub. Gets reset to -1 if a send cycle fails.
	lastValidationTime       time.Time
	rec                      *reconciler.Reconciler
	tlsReload                <-chan struct{} // Signals that the TLS profile ConfigMap has changed.
}

func (s *Sender) reloadSender() {
	s.aggregatorURL = config.Cfg.AggregatorURL
	s.aggregatorSyncPath = strings.Join([]string{"/aggregator/clusters/", config.Cfg.ClusterName, "/sync"}, "")
	s.aggregatorValidationPath = strings.Join([]string{"/aggregator/clusters/", config.Cfg.ClusterName, "/validation"}, "")
	if !config.Cfg.DeployedInHub {
		s.aggregatorSyncPath = strings.Join([]string{"/", config.Cfg.ClusterName, "/aggregator/sync"}, "")
		s.aggregatorValidationPath = strings.Join([]string{"/", config.Cfg.ClusterName, "/aggregator/validation"}, "")
	}
	s.httpClient = getHTTPSClient()
}

// Constructs a new Sender using the provided channels.
// Sends to the URL provided by aggregatorURL, listing itself as clusterName.
// tlsReload is an optional channel that signals the TLS profile has changed and the HTTP client
// should be recreated. Pass nil to disable TLS hot-reload.
func NewSender(rec *reconciler.Reconciler, aggregatorURL, clusterName string,
	tlsReload <-chan struct{},
) *Sender {
	// Construct senders
	s := &Sender{
		aggregatorURL:            aggregatorURL,
		aggregatorSyncPath:       strings.Join([]string{"/aggregator/clusters/", clusterName, "/sync"}, ""),
		aggregatorValidationPath: strings.Join([]string{"/aggregator/clusters/", clusterName, "/validation"}, ""),
		httpClient:               getHTTPSClient(),
		lastSentTime:             -1,
		lastValidationTime:       time.Time{},
		rec:                      rec,
		tlsReload:                tlsReload,
	}

	if !config.Cfg.DeployedInHub {
		s.aggregatorSyncPath = strings.Join([]string{"/", clusterName, "/aggregator/sync"}, "")
		s.aggregatorValidationPath = strings.Join([]string{"/", clusterName, "/aggregator/validation"}, "")
	}

	return s
}

// Returns a payload and expected total resources, add, update, and delete operations since the last send.
func (s *Sender) diffPayload() (Payload, int, int) {
	diff := s.rec.Diff()

	payload := Payload{
		ClearAll: false,
		Version:  config.COLLECTOR_API_VERSION,

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
		AddResources: complete.Nodes,

		AddEdges: complete.Edges,
	}
	return payload, complete.TotalNodes, complete.TotalEdges
}

// Send will retry after recoverable errors.
//   - indexer busy
func (s *Sender) sendWithRetry(payload Payload) (SyncResponse, error) {
	retry := 0
	for {
		response, sendError := s.send(payload)
		retry++
		nextRetryWait := sendInterval(retry)

		// If indexer was busy, wait and retry with the same payload.
		if sendError != nil && sendError.Error() == "indexer busy" {
			klog.Warningf("Received busy response from Indexer. Resending in %s.", nextRetryWait)
			time.Sleep(nextRetryWait)
			continue
		}
		// For other errors, wait, reload the config, and re-send the full state payload.
		if sendError != nil {
			klog.Warningf("Received error response [%s] from Indexer. Resetting config and resending in %s.",
				sendError.Error(), nextRetryWait)
			time.Sleep(nextRetryWait)
			config.InitConfig() // re-initialize config to get the latest certificate.
			s.reloadSender()    // reload sender variables - Aggregator URL, path and client
		}
		return response, sendError
	}
}

// Sends data to the aggregator and returns an error if it didn't work.
// Pointer receiver because Sender contains a mutex - that freaked the linter out even though it
// doesn't use the mutex. Changed it so that if we do need to use the mutex we wont have any problems.
func (s *Sender) send(payload Payload) (SyncResponse, error) {
	r := SyncResponse{}
	klog.Infof("Sending Resources { add: %2d, update: %2d, delete: %2d, edge add: %2d, edge delete: %2d }",
		len(payload.AddResources), len(payload.UpdatedResources), len(payload.DeletedResources),
		len(payload.AddEdges), len(payload.DeleteEdges))

	syncType := "sync"
	if payload.ClearAll {
		syncType = "resync"
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return r, err
	}
	payloadBuffer := bytes.NewBuffer(payloadBytes)

	req, err := http.NewRequest("POST", s.aggregatorURL+s.aggregatorSyncPath, payloadBuffer)
	if err != nil {
		return r, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Overwrite-State", strconv.FormatBool(payload.ClearAll))

	resp, err := s.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		// #nosec G307
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
	}
	if err != nil {
		klog.Error("httpClient error: ", err)
		return r, err
	}

	metrics.SyncRequestTotal.WithLabelValues(strconv.Itoa(resp.StatusCode), syncType).Inc()
	if resp.StatusCode == http.StatusTooManyRequests {
		return r, errors.New("indexer busy")
	} else if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("POST to: %s responded with error. StatusCode: %d  Message: %s",
			s.aggregatorURL+s.aggregatorSyncPath, resp.StatusCode, resp.Status)
		if resp.StatusCode == http.StatusUnauthorized {
			msg = "401 Unauthorized"
		}
		return r, errors.New(msg)
	}

	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		klog.Error("Error decoding JSON response.")
		return r, err
	}

	return r, nil
}

// validate checks the indexer's persisted totals after the configured interval has elapsed.
func (s *Sender) validate(expectedTotalResources int, expectedTotalEdges int) error {
	interval := time.Duration(config.Cfg.DataValidationIntervalMS) * time.Millisecond
	if !s.lastValidationTime.IsZero() && time.Since(s.lastValidationTime) < interval {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, s.aggregatorURL+s.aggregatorValidationPath, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		// #nosec G307
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET from: %s responded with error. StatusCode: %d Message: %s",
			s.aggregatorURL+s.aggregatorValidationPath, resp.StatusCode, resp.Status)
	}

	validationResponse := ValidationResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&validationResponse); err != nil {
		return err
	}

	s.lastValidationTime = time.Now()
	if validationResponse.TotalResources != expectedTotalResources {
		s.lastSentTime = -1
		return fmt.Errorf("indexer reported wrong number of total resources. Expected %d, got %d",
			expectedTotalResources, validationResponse.TotalResources)
	}
	if validationResponse.TotalEdges != expectedTotalEdges {
		s.lastSentTime = -1
		return fmt.Errorf("indexer reported wrong number of total intra edges. Expected %d, got %d",
			expectedTotalEdges, validationResponse.TotalEdges)
	}

	return nil
}

// Sends data to the aggregator.
// Attempts to send a diff, then just sends the complete if the aggregator appears to need that.
func (s *Sender) Sync() error {
	if s.lastSentTime == -1 { // If we have never sent before, we just send the complete.
		klog.Info("First time sending or last Sync cycle failed, sending complete payload")
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		response, err := s.sendWithRetry(payload)
		if err != nil {
			klog.Error("Sync sender error. ", err)
			return err
		}

		s.lastSentTime = time.Now().Unix()
		return s.validate(expectedTotalResources+len(response.DeleteErrors)-len(response.AddErrors),
			expectedTotalEdges+len(response.DeleteEdgeErrors)-len(response.AddEdgeErrors))
	}

	// If this isn't the first time we've sent, we can now attempt to send a diff.
	payload, expectedTotalResources, expectedTotalEdges := s.diffPayload()
	if payload.empty() {
		klog.V(3).Info("Nothing to send, skipping send cycle.")
		return s.validate(expectedTotalResources, expectedTotalEdges)
	}
	response, err := s.sendWithRetry(payload)
	if err != nil {
		// If something went wrong here, form a new complete payload (only necessary because
		// currentState may have changed since we got it, and we have to keep our diffs synced)
		klog.Warning("Error on diff payload sending: ", err)
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		klog.Warning("Retrying with complete payload")
		response, err = s.sendWithRetry(payload)
		if err != nil {
			klog.Error("Error resending complete payload.")
			// If this retry fails, we want to start over with a complete payload next time,
			// so we reset as if we've not sent anything before.
			s.lastSentTime = -1
			return err
		}
		s.lastSentTime = time.Now().Unix()
		return s.validate(expectedTotalResources+len(response.DeleteErrors)-len(response.AddErrors),
			expectedTotalEdges+len(response.DeleteEdgeErrors)-len(response.AddEdgeErrors))
	}

	s.lastSentTime = time.Now().Unix()
	return s.validate(expectedTotalResources+len(response.DeleteErrors)-len(response.AddErrors),
		expectedTotalEdges+len(response.DeleteEdgeErrors)-len(response.AddEdgeErrors))
}

// Starts the send loop to send data on an interval.
// In case of error it backoffs and retries.
func (s *Sender) StartSendLoop(ctx context.Context) {
	// Used for exponential backoff, increased each interval. Has to be a float64 since I use it with math.Exp2()
	backoffFactor := 1 // Note: must be 1. Using 0 will send the next payload immediately.

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check for TLS profile changes before each send cycle.
		s.checkTLSReload()

		klog.V(3).Info("Beginning Send Cycle")
		err := s.Sync()
		if err != nil {
			klog.Error("SEND ERROR: ", err)
			// Increase the backoffFactor, doubling the wait time. Stops increasing after it passes the max
			// wait time so that we don't overflow int. Can be changed with env:MAX_BACKOFF_MS
			if sendInterval(backoffFactor) < time.Duration(config.Cfg.MaxBackoffMS)*time.Millisecond {
				backoffFactor++
			}
		} else {
			klog.V(2).Info("Send Cycle Completed Successfully")
			backoffFactor = 1 // Reset backoff to 1 because we had a sucessful send.
		}

		nextSendWait := sendInterval(backoffFactor)
		if backoffFactor > 1 {
			klog.Warningf("Error during last sync. Resending in %s.", nextSendWait)
		}
		// Sleep either for the current backed off interval, or the maximum time defined in the config
		time.Sleep(nextSendWait)
	}
}

// checkTLSReload drains the TLS reload channel and rebuilds the HTTP client if signaled.
func (s *Sender) checkTLSReload() {
	if s.tlsReload == nil {
		return
	}
	select {
	case <-s.tlsReload:
		klog.Info("TLS profile changed, reloading HTTP client")
		s.httpClient = getHTTPSClient()
	default:
	}
}

// Compute the time interval to wait before next send or retry (backoff).
func sendInterval(retry int) time.Duration {
	nextInterval := int(1000*math.Exp2(float64(retry))) + addJitter()
	return time.Duration(min(nextInterval, config.Cfg.MaxBackoffMS)) * time.Millisecond
}

// Generate a random jitter to add to the backoff retry to prevent clients from retrying at the same interval.
func addJitter() int {
	max := big.NewInt(int64(config.Cfg.RetryJitterMS))
	j, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return int(j.Int64())
}
