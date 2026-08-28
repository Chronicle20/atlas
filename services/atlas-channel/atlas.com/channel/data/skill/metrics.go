package skill

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var skillDataCacheTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_channel_skill_data_cache_total",
		Help: "Skill-data cache lookups by tenant and outcome.",
	},
	[]string{"tenant", "outcome"},
)

func recordCache(t tenant.Model, outcome string) {
	skillDataCacheTotal.WithLabelValues(t.Id().String(), outcome).Inc()
}
