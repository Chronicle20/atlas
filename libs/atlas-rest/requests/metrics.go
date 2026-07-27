package requests

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// reason ∈ {"503"}
var clientRetriesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_rest_client_retries_total",
		Help: "Number of REST client attempts retried after a retryable response, by reason.",
	},
	[]string{"reason"},
)
