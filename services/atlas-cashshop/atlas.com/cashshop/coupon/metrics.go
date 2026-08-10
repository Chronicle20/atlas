package coupon

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// attemptsTotal counts every redemption attempt that got past the handler,
	// labelled by the client-facing outcome key ("SUCCESS" or one of the
	// coupon.ErrorKey* values). The outcome label is a CLOSED set of at most
	// eight values, so it cannot explode cardinality.
	//
	// NEVER label by the coupon code: it is a secret AND unbounded.
	attemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_cashshop_coupon_attempts_total",
			Help: "Coupon redemption attempts, by tenant and outcome.",
		},
		[]string{"tenant", "outcome"},
	)

	// rateLimitedTotal counts attempts short-circuited by the limiter. These
	// are reported to the player as INVALID_COUPON_CODE, so they are
	// indistinguishable from a genuine miss in attemptsTotal — this counter is
	// the only way to see brute-force pressure. A blocked attempt increments
	// BOTH, which is what makes the two series comparable.
	rateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_cashshop_coupon_rate_limited_total",
			Help: "Coupon attempts blocked by the per-account rate limiter, by tenant.",
		},
		[]string{"tenant"},
	)
)

// outcomeSuccess is the attemptsTotal label for a committed redemption. There
// is deliberately NO "compensated redemptions" counter: design §2 replaced
// compensation with a transaction rollback, so PRD §8's third counter has no
// event to count. Rollbacks appear as the non-SUCCESS outcomes already here.
const outcomeSuccess = "SUCCESS"
