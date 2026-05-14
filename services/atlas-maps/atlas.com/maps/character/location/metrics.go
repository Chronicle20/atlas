package location

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// locationResolutionsTotal counts location.Resolve outcomes by reason
// (e.g. "forced_return", "stay_put"). Uses promauto so the registration
// is idempotent against the default registerer.
var locationResolutionsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_maps_location_resolutions_total",
		Help: "Number of location.Resolve calls by outcome reason.",
	},
	[]string{"reason"},
)
