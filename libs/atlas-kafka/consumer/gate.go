package consumer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// gateVerdict is the outcome of the ownership gate for one message.
type gateVerdict int

const (
	gateProcess gateVerdict = iota
	gateSkipNotOwner
	gateDropUnresolvable
)

var (
	gateProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_kafka_gate_processed_total",
			Help: "Messages that passed the ownership gate and were handed to domain handlers.",
		},
		[]string{"service", "environment"},
	)

	gateSkippedNotOwner = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_kafka_gate_skipped_not_owner_total",
			Help: "Messages acknowledged without domain processing because this deployment is not the message's environment owner (FR-4.4).",
		},
		[]string{"service", "environment"},
	)

	gateDroppedUnresolvable = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_kafka_gate_dropped_unresolvable_total",
			Help: "Messages acknowledged and dropped because their environment could not be resolved to any owning deployment (FR-4.7, D4).",
		},
		[]string{"service", "environment"},
	)
)

// decide is a pure function of the registry state and the message's
// environment — no I/O, no clock beyond the registry's own staleness.
//
// mismatched is true when the header parser recorded a disagreement between
// the ENVIRONMENT header and the tenant it names (FR-7.7); the message is
// unresolvable and must be dropped rather than executed under either
// candidate environment.
func decide(r env.Registry, self env.Id, service string, msgEnv env.Id, mismatched bool) gateVerdict {
	if mismatched {
		return gateDropUnresolvable // FR-7.7
	}
	if msgEnv == "" {
		return gateProcess // FR-1.8
	}
	if r.Stale() && msgEnv != self {
		// A pod's own environment comes from an env var and cannot go
		// stale; every other environment fails closed (design §4.3).
		return gateDropUnresolvable
	}
	if !r.IsActive(msgEnv) {
		return gateDropUnresolvable // FR-4.7 / D4
	}
	if !r.IsOwner(msgEnv, service) {
		return gateSkipNotOwner // FR-4.4
	}
	return gateProcess
}
