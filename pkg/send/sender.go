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
	"context"
	"crypto/rand"
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/segmentio/kafka-go"
	"github.com/stolostron/search-collector/pkg/config"
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

func (s *Sender) reloadSender() {
	s.aggregatorURL = config.Cfg.AggregatorURL
	s.aggregatorSyncPath = strings.Join([]string{"/aggregator/clusters/", config.Cfg.ClusterName, "/sync"}, "")
	if !config.Cfg.DeployedInHub {
		s.aggregatorSyncPath = strings.Join([]string{"/", config.Cfg.ClusterName, "/aggregator/sync"}, "")
	}
	s.httpClient = getHTTPSClient()
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
//   - Aggregator busy
func (s *Sender) sendWithRetry(payload Payload, expectedTotalResources int, expectedTotalEdges int) error {
	retry := 0
	producer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"kafka-kafka-bootstrap.amq-streams.svc:9092"},
		Topic:    "resource-events",
		Balancer: &kafka.LeastBytes{},
	})
	for {
		sendError := s.send(payload, producer, expectedTotalResources, expectedTotalEdges)
		retry++
		nextRetryWait := sendInterval(retry)

		// If indexer was busy, wait and retry with the same payload.
		if sendError != nil && sendError.Error() == "Aggregator busy" {
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
		return sendError
	}
}

type KafkaEvent struct {
	Type    string      `json:"type"` // addResource, updateResource, deleteEdge, etc.
	Cluster string      `json:"cluster"`
	Payload interface{} `json:"payload"` // Node, Edge
}

// Sends data to the aggregator and returns an error if it didn't work.
// Pointer receiver because Sender contains a mutex - that freaked the linter out even though it
// doesn't use the mutex. Changed it so that if we do need to use the mutex we wont have any problems.
func (s *Sender) send(payload Payload, producer *kafka.Writer, expectedTotalResources int, expectedTotalEdges int) error {
	ctx := context.Background()
	klog.Infof("Writing resource messages to kafka")
	klog.Infof("Start of send FULL-CLUSTER-STATE")

	var messages []kafka.Message
	headers := make([]kafka.Header, 0)
	if payload.ClearAll {
		headers = append(headers, kafka.Header{Key: "ClearAll", Value: []byte("True")})
	}

	for c := 0; c < 100; c++ {
		if payload.ClearAll {

			ke := KafkaEvent{
				Type:    "CLEAR-ALL-START",
				Cluster: config.Cfg.ClusterName + strconv.Itoa(c),
				Payload: nil,
			}
			data, _ := json.Marshal(ke)
			messages = append(messages, kafka.Message{
				Key:     []byte(ke.Type),
				Headers: headers,
				Value:   data,
			})
		}
		for i := 0; i < 10; i++ { // The breaking combo of 100 clusters sending 10x the normal (~7000 resources) payload

			for _, r := range payload.AddResources {
				r.UID = strings.Split(r.UID, "/")[0] + strconv.Itoa(c) + "/" + strings.Split(r.UID, "/")[1] + strconv.Itoa(i) // local-cluster/abcd -> local-cluster0/abcd0
				ke := KafkaEvent{
					Type:    "addResource",
					Cluster: config.Cfg.ClusterName + strconv.Itoa(c),
					Payload: r,
				}
				data, _ := json.Marshal(ke)
				messages = append(messages, kafka.Message{
					Key:     []byte(ke.Type),
					Headers: headers,
					Value:   data,
				})
			}

			//for _, r := range payload.UpdatedResources {
			//	msg := KafkaEvent{
			//		Type:    "updateResource",
			//		Cluster: config.Cfg.ClusterName,
			//		Payload: r,
			//	}
			//	if err := sendEvent(ctx, producer, msg, headers); err != nil {
			//		return err
			//	}
			//}

			//for _, r := range payload.DeletedResources {
			//	msg := KafkaEvent{
			//		Type:    "deleteResource",
			//		Cluster: config.Cfg.ClusterName,
			//		Payload: r,
			//	}
			//	if err := sendEvent(ctx, producer, msg, headers); err != nil {
			//		return err
			//	}
			//}

			for _, e := range payload.AddEdges {
				ke := KafkaEvent{
					Type:    "addEdge",
					Cluster: config.Cfg.ClusterName + strconv.Itoa(c),
					Payload: e,
				}
				data, _ := json.Marshal(ke)
				messages = append(messages, kafka.Message{
					Key:     []byte(ke.Type),
					Headers: headers,
					Value:   data,
				})
			}

			//for _, e := range payload.DeleteEdges {
			//	msg := KafkaEvent{
			//		Type:    "deleteEdge",
			//		Cluster: config.Cfg.ClusterName,
			//		Payload: e,
			//	}
			//	if err := sendEvent(ctx, producer, msg, headers); err != nil {
			//		return err
			//	}
			//}

			if payload.ClearAll {

				ke := KafkaEvent{
					Type:    "CLEAR-ALL-END",
					Cluster: config.Cfg.ClusterName + strconv.Itoa(c),
					Payload: nil,
				}
				data, _ := json.Marshal(ke)
				messages = append(messages, kafka.Message{
					Key:     []byte(ke.Type),
					Headers: headers,
					Value:   data,
				})
			}
		}
	}
	klog.Infof("End of send FULL-CLUSTER-STATE")

	return producer.WriteMessages(ctx, messages...)
}

const numWorkers = 5

// appears not faster in testing over a single goroutine utilizing batching
func (s *Sender) sendConcurrent(payload Payload, producer *kafka.Writer) error {
	ctx := context.Background()
	klog.Infof("Writing resources concurrently to kafka")
	klog.Infof("%d addresources and %d addedges", len(payload.AddResources), len(payload.AddEdges))

	headers := make([]kafka.Header, 0)
	if payload.ClearAll {
		headers = append(headers, kafka.Header{Key: "ClearAll", Value: []byte("True")})
	}

	// Write CLEAR-ALL-START
	if payload.ClearAll {
		klog.Infof("Start of send FULL-CLUSTER-STATE")
		startMsg := kafka.Message{
			Key:     []byte("CLEAR-ALL-START"),
			Headers: headers,
			Value:   mustJSON(KafkaEvent{Type: "CLEAR-ALL-START", Cluster: config.Cfg.ClusterName}),
		}
		if err := producer.WriteMessages(ctx, startMsg); err != nil {
			return err
		}
	}

	// Set up a shared channel for messages
	msgCh := make(chan kafka.Message, 1000)

	// Start workers
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			var batch []kafka.Message
			const batchSize = 200 // Tune
			for msg := range msgCh {
				batch = append(batch, msg)
				if len(batch) >= batchSize {
					if err := producer.WriteMessages(ctx, batch...); err != nil {
						klog.Errorf("Failed to write Kafka batch: %v", err)
					}
					batch = batch[:0]
				}
			}
			// flush any remaining
			if len(batch) > 0 {
				if err := producer.WriteMessages(ctx, batch...); err != nil {
					klog.Errorf("Failed to write final Kafka batch: %v", err)
				}
			}
		}()
	}

	// Feed resources into the channel
	for _, r := range payload.AddResources {
		event := KafkaEvent{Type: "addResource", Cluster: config.Cfg.ClusterName, Payload: r}
		msgCh <- kafka.Message{
			Key:     []byte("addResource"),
			Headers: headers,
			Value:   mustJSON(event),
		}
	}

	for _, e := range payload.AddEdges {
		event := KafkaEvent{Type: "addEdge", Cluster: config.Cfg.ClusterName, Payload: e}
		msgCh <- kafka.Message{
			Key:     []byte("addEdge"),
			Headers: headers,
			Value:   mustJSON(event),
		}
	}

	close(msgCh)
	wg.Wait()

	// Write CLEAR-ALL-END
	if payload.ClearAll {
		klog.Infof("End of send FULL-CLUSTER-STATE")
		endMsg := kafka.Message{
			Key:     []byte("CLEAR-ALL-END"),
			Headers: headers,
			Value:   mustJSON(KafkaEvent{Type: "CLEAR-ALL-END", Cluster: config.Cfg.ClusterName}),
		}
		if err := producer.WriteMessages(ctx, endMsg); err != nil {
			return err
		}
	}

	return nil
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

//func sendEvent(ctx context.Context, producer *kafka.Writer, messages []kafka.Message, headers []kafka.Header) error {
//	return producer.WriteMessages(ctx, messages...)
//}

// Sends data to the aggregator.
// Attempts to send a diff, then just sends the complete if the aggregator appears to need that.
func (s *Sender) Sync() error {
	if s.lastSentTime == -1 { // If we have never sent before, we just send the complete.
		klog.Info("First time sending or last Sync cycle failed, sending complete payload")
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
		if err != nil {
			klog.Error("Sync sender error. ", err)
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
			klog.V(3).Info("Nothing to send, skipping send cycle.")
			return nil
		}
		klog.V(2).Info("Sending empty payload for heartbeat.")
	}
	err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
	if err != nil {
		// If something went wrong here, form a new complete payload (only necessary because
		// currentState may have changed since we got it, and we have to keep our diffs synced)
		klog.Warning("Error on diff payload sending: ", err)
		payload, expectedTotalResources, expectedTotalEdges := s.completePayload()
		klog.Warning("Retrying with complete payload")
		err := s.sendWithRetry(payload, expectedTotalResources, expectedTotalEdges)
		if err != nil {
			klog.Error("Error resending complete payload.")
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
func (s *Sender) StartSendLoop(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		klog.V(3).Info("Beginning Send Cycle")
		_ = s.Sync()
		//if err != nil {
		//	klog.Error("SEND ERROR: ", err)
		//	// Increase the backoffFactor, doubling the wait time. Stops increasing after it passes the max
		//	// wait time so that we don't overflow int. Can be changed with env:MAX_BACKOFF_MS
		//	if sendInterval(backoffFactor) < time.Duration(config.Cfg.MaxBackoffMS)*time.Millisecond {
		//		backoffFactor++
		//	}
		//} else {
		//	klog.V(2).Info("Send Cycle Completed Successfully")
		//	backoffFactor = 1 // Reset backoff to 1 because we had a sucessful send.
		//}
		//
		//nextSendWait := sendInterval(backoffFactor)
		//if backoffFactor > 1 {
		//	klog.Warningf("Error during last sync. Resending in %s.", nextSendWait)
		//}
		//// Sleep either for the current backed off interval, or the maximum time defined in the config
		//time.Sleep(nextSendWait)
	}
}

// Returns the smaller of two ints
func min(a, b int) int {
	if a > b {
		return b
	}
	return a
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
