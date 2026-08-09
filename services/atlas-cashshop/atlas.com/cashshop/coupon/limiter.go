package coupon

import (
	"context"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const limiterNamespace = "coupon-attempts"

var (
	limiterOnce  sync.Once
	limiterStore *redis.TenantCounter

	limiterUnwiredOnce sync.Once
)

// warnIfLimiterUnwired logs ONCE when no store was ever installed.
//
// Without it an unwired limiter is invisible: Allowed returns (true, nil) and
// RecordFailure/Reset return nil, so brute-force braking would be silently off
// in production with every test still passing. The fail-open behaviour itself
// is deliberate and unchanged — this only makes the condition observable.
func warnIfLimiterUnwired(l logrus.FieldLogger) {
	if limiterStore != nil {
		return
	}
	limiterUnwiredOnce.Do(func() {
		l.Warnf("Coupon rate limiter has no counter store; brute-force braking is DISABLED. InitLimiter was never called.")
	})
}

// InitLimiter wires the shared Redis client. Called once from main.go, like
// the other registry initializers in this project.
func InitLimiter(client *goredis.Client) {
	limiterOnce.Do(func() {
		limiterStore = redis.NewTenantCounter(client, limiterNamespace)
	})
}

// Limiter brakes coupon brute-forcing by counting an account's FAILED attempts
// inside a fixed window. Callers that exceed the budget are answered with the
// ordinary INVALID_COUPON_CODE result rather than a distinct "rate limited"
// one: a distinct result would tell an attacker they had found a REAL code and
// merely run out of attempts.
//
// It FAILS OPEN: when Redis is unreachable, Allowed returns true. This is a
// brute-force brake, not an authorization gate — a Redis outage must not make
// every coupon un-redeemable for every player.
type Limiter struct {
	attempts uint32
	window   time.Duration
}

func NewLimiter(attempts uint32, window time.Duration) Limiter {
	return Limiter{attempts: attempts, window: window}
}

func limiterKey(accountId uint32) string {
	return strconv.FormatUint(uint64(accountId), 10)
}

// Allowed reports whether this account may make another attempt. It only reads
// the account's failure count and compares it against the budget — it never
// increments. Only RecordFailure increments.
//
// A read error (Redis down) is returned alongside allowed=true so the caller
// can log the outage without denying the redemption.
func (l Limiter) Allowed(ctx context.Context, t tenant.Model, accountId uint32) (bool, error) {
	if limiterStore == nil {
		return true, nil
	}
	n, err := limiterStore.Get(ctx, t, limiterKey(accountId))
	if err != nil {
		return true, err
	}
	return n < int64(l.attempts), nil
}

// RecordFailure counts one failed attempt against the account's window. The
// window is fixed: it starts at the first failure and does not slide forward
// as further failures arrive.
func (l Limiter) RecordFailure(ctx context.Context, t tenant.Model, accountId uint32) error {
	if limiterStore == nil {
		return nil
	}
	_, err := limiterStore.IncrWithTTL(ctx, t, limiterKey(accountId), l.window)
	return err
}

// Reset clears the account's counter after a successful redemption, so a
// player who mistyped a few times before getting it right is not left one
// failure away from a block.
func (l Limiter) Reset(ctx context.Context, t tenant.Model, accountId uint32) error {
	if limiterStore == nil {
		return nil
	}
	return limiterStore.Remove(ctx, t, limiterKey(accountId))
}
