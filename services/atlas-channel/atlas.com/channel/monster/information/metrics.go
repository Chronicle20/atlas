package information

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_info_cache_hits_total",
			Help: "Template-info cache hits, by tenant and entry kind.",
		},
		[]string{"tenant", "kind"},
	)

	cacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_info_cache_misses_total",
			Help: "Template-info cache misses (upstream HTTP issued), by tenant.",
		},
		[]string{"tenant"},
	)
)

func recordCacheHit(t tenant.Model, kind string) {
	cacheHitsTotal.WithLabelValues(t.Id().String(), kind).Inc()
}

func recordCacheMiss(t tenant.Model) {
	cacheMissesTotal.WithLabelValues(t.Id().String()).Inc()
}
