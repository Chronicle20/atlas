package trade

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The design §12 counters, named with this repo's atlas_<service>_ prefix and
// the Prometheus _total suffix. Every one is tenant-labelled, matching the
// existing counters in atlas-channel.
//
//	design §12 name              this file
//	trade_rooms_opened        -> atlas_trades_rooms_opened_total
//	trade_settled             -> atlas_trades_settled_total
//	trade_cancelled           -> atlas_trades_cancelled_total{reason}
//	trade_settlement_failed   -> atlas_trades_settlement_failed_total{reason}
//	trade_meso_taxed_total    -> atlas_trades_meso_taxed_total
//	trade_reservation_expired -> atlas_trades_reservation_expired_total
var (
	roomsOpenedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_rooms_opened_total",
			Help: "Trade rooms opened, by tenant.",
		},
		[]string{"tenant"},
	)

	settledTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_settled_total",
			Help: "Trades that settled successfully and were written to the ledger, by tenant.",
		},
		[]string{"tenant"},
	)

	cancelledTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_cancelled_total",
			Help: "Trade rooms torn down without settling, by tenant and leaveReason key.",
		},
		[]string{"tenant", "reason"},
	)

	settlementFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_settlement_failed_total",
			Help: "Settlements refused or failed, by tenant and reason. A settlement pre-check refusal carries its leaveReason key; a saga failure carries SAGA_FAILED.",
		},
		[]string{"tenant", "reason"},
	)

	mesoTaxedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_meso_taxed_total",
			Help: "Mesos destroyed as trade tax, by tenant. Counted at the ledger write, so it reflects mesos actually moved.",
		},
		[]string{"tenant"},
	)

	reservationExpiredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_trades_reservation_expired_total",
			Help: "Staged items whose reservation no longer covered the asset at settlement time, by tenant.",
		},
		[]string{"tenant"},
	)
)

func recordRoomOpened(t tenant.Model) {
	roomsOpenedTotal.WithLabelValues(t.Id().String()).Inc()
}

func recordSettled(t tenant.Model, mesoTaxed uint32) {
	settledTotal.WithLabelValues(t.Id().String()).Inc()
	if mesoTaxed > 0 {
		mesoTaxedTotal.WithLabelValues(t.Id().String()).Add(float64(mesoTaxed))
	}
}

func recordCancelled(t tenant.Model, reason string) {
	cancelledTotal.WithLabelValues(t.Id().String(), reason).Inc()
}

func recordSettlementFailed(t tenant.Model, reason string) {
	settlementFailedTotal.WithLabelValues(t.Id().String(), reason).Inc()
}

func recordReservationExpired(t tenant.Model) {
	reservationExpiredTotal.WithLabelValues(t.Id().String()).Inc()
}
