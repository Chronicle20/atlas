package seeder

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce           sync.Once
	seederRunsTotal       *prometheus.CounterVec
	seederDurationSeconds *prometheus.HistogramVec
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		seederRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "atlas_seeder_runs_total",
			Help: "Count of atlas-seeder Seed() invocations by service, group, and outcome.",
		}, []string{"service", "group", "outcome"})
		seederDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "atlas_seeder_duration_seconds",
			Help:    "Wall-clock duration of atlas-seeder Seed() invocations.",
			Buckets: prometheus.DefBuckets,
		}, []string{"service", "group"})
		prometheus.MustRegister(seederRunsTotal, seederDurationSeconds)
	})
}

func ObserveSeederRun(service, group, outcome string, durationSeconds float64) {
	ensureMetrics()
	seederRunsTotal.WithLabelValues(service, group, outcome).Inc()
	seederDurationSeconds.WithLabelValues(service, group).Observe(durationSeconds)
}

func ResetMetricsForTest() {
	if seederRunsTotal != nil {
		prometheus.Unregister(seederRunsTotal)
	}
	if seederDurationSeconds != nil {
		prometheus.Unregister(seederDurationSeconds)
	}
	seederRunsTotal = nil
	seederDurationSeconds = nil
	metricsOnce = sync.Once{}
}
