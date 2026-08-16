package requests

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// reason ∈ {"503"}. environment is sourced from env.Self() — this
// process's own environment — not from any request/response header, so
// cardinality stays bounded by the small number of concurrently deployed
// environments (FR-10.3, design §14).
var clientRetriesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_rest_client_retries_total",
		Help: "Number of REST client attempts retried after a retryable response, by reason.",
	},
	[]string{"reason", "environment"},
)

// selfEnvironment is the process-local environment label value for outbound
// REST metrics (FR-10).
func selfEnvironment() string {
	return string(env.Self())
}
