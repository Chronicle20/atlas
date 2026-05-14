package lock

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	acquiredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_acquired_total",
			Help: "Number of times this pod transitioned from non-leader to leader for a given lease name.",
		},
		[]string{"name"},
	)

	// reason ∈ {renew_failed, context_cancelled, released, panic}
	lostTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_lost_total",
			Help: "Number of times this pod transitioned from leader to non-leader.",
		},
		[]string{"name", "reason"},
	)

	renewFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_renew_failed_total",
			Help: "Number of single renewal attempts that failed (does not always cause leader loss).",
		},
		[]string{"name"},
	)

	// reason ∈ {held_by_other, redis_error}
	acquireFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_acquire_failed_total",
			Help: "Number of failed acquire attempts.",
		},
		[]string{"name", "reason"},
	)
)
