// Copyright Contributors to the Open Cluster Management project

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PromRegistry = prometheus.NewRegistry()

	// EventsReceivedCount Counts of events informed by kubernetes
	EventsReceivedCount = promauto.With(PromRegistry).NewCounterVec(prometheus.CounterOpts{
		Name: "search_collector_events_received_count",
		Help: "Total events received by informers",
	}, []string{"resource_kind"})

	// ResourcesSentToIndexerCount Counts of informed resources after reconciliation
	ResourcesSentToIndexerCount = promauto.With(PromRegistry).NewCounterVec(prometheus.CounterOpts{
		Name: "search_collector_resources_sent_to_indexer_count",
		Help: "Total resources sent to indexer after reconciliation",
	}, []string{"resource_kind"})

	// SyncRequestTotal total number of HTTP requests sent to indexer
	SyncRequestTotal = promauto.With(PromRegistry).NewCounterVec(prometheus.CounterOpts{
		Name: "search_collector_sync_requests_total",
		Help: "Total number of HTTP requests sent",
	}, []string{"status_code", "sync_type"})
)
