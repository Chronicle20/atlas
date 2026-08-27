package snapshot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	componentCore      = "core"
	componentSkills    = "skills"
	componentInventory = "inventory"
	componentBuffs     = "buffs"
	componentPosition  = "position"

	outcomeHit             = "hit"
	outcomeMiss            = "miss"
	outcomeFallbackSuccess = "fallback_success"
	outcomeFallbackFailure = "fallback_failure"

	kindEventUpdate       = "event_update"
	kindInvalidation      = "invalidation"
	kindBackfill          = "backfill"
	kindBackfillDiscarded = "backfill_discarded"
)

var (
	snapshotReadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_char_snapshot_reads_total",
			Help: "Character snapshot reads by tenant, component, and outcome.",
		},
		[]string{"tenant", "component", "outcome"},
	)

	snapshotUpdatesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_char_snapshot_updates_total",
			Help: "Character snapshot state transitions by tenant, component, and kind.",
		},
		[]string{"tenant", "component", "kind"},
	)
)

func recordRead(t tenant.Model, component, outcome string) {
	snapshotReadsTotal.WithLabelValues(t.Id().String(), component, outcome).Inc()
}

func recordUpdate(t tenant.Model, component, kind string) {
	snapshotUpdatesTotal.WithLabelValues(t.Id().String(), component, kind).Inc()
}
