package data

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	eventsEmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_data_events_emitted_total",
		Help: "Successful Kafka emits of data lifecycle events, by worker and type.",
	}, []string{"worker", "type"})

	eventsEmitFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "atlas_data_events_emit_failures_total",
		Help: "Failed Kafka emits of data lifecycle events, by worker and type.",
	}, []string{"worker", "type"})
)
