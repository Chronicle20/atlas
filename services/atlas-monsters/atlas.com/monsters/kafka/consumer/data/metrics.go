package data

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_monsters_data_events_processed_total",
		Help: "DATA_UPDATED events processed by the cache-invalidation consumer.",
	}, []string{"worker", "type", "action"})

	eventsConsumerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_monsters_data_events_consumer_errors_total",
		Help: "Errors encountered processing DATA_UPDATED events.",
	}, []string{"kind"})

	eventsSkippedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_monsters_data_events_consumer_skipped_total",
		Help: "DATA_UPDATED events skipped (unknown type or unrelated worker).",
	}, []string{"reason"})

	keysDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_monsters_data_events_keys_deleted_total",
		Help: "Redis keys deleted by cache-invalidation flushes, by tenant.",
	}, []string{"tenant"})
)
