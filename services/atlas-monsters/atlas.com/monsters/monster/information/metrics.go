package information

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	hitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_hits_total",
			Help: "Cache hits for monster information lookups, by tenant and entry kind.",
		},
		[]string{"tenant", "kind"},
	)

	missesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_misses_total",
			Help: "Cache misses (upstream HTTP issued) for monster information lookups, by tenant.",
		},
		[]string{"tenant"},
	)

	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_errors_total",
			Help: "Upstream errors observed during monster information lookups, by tenant and classification.",
		},
		[]string{"tenant", "classification"},
	)

	redisErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_redis_errors_total",
			Help: "Redis-side errors during monster data cache operations. Each increment indicates a graceful fallthrough to upstream (or a discarded cache write).",
		},
		[]string{"tenant", "operation"},
	)
)
