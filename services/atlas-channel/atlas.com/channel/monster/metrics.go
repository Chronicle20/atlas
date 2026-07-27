package monster

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	mirrorHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_hits_total",
			Help: "Live-monster mirror lookup hits, by tenant.",
		},
		[]string{"tenant"},
	)

	mirrorMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_misses_total",
			Help: "Live-monster mirror lookup misses, by tenant.",
		},
		[]string{"tenant"},
	)

	mirrorFallbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_fallback_total",
			Help: "REST fallbacks taken after a live-monster mirror miss, by tenant and outcome.",
		},
		[]string{"tenant", "outcome"},
	)
)

func recordMirrorHit(t tenant.Model) {
	mirrorHitsTotal.WithLabelValues(t.Id().String()).Inc()
}

func recordMirrorMiss(t tenant.Model) {
	mirrorMissesTotal.WithLabelValues(t.Id().String()).Inc()
}

// RecordMirrorFallback records the outcome of a REST fallback taken by a
// mirror consumer (movement path) after a Lookup miss.
func RecordMirrorFallback(t tenant.Model, success bool) {
	outcome := "failure"
	if success {
		outcome = "success"
	}
	mirrorFallbackTotal.WithLabelValues(t.Id().String(), outcome).Inc()
}
