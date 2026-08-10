package coupon

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// useLimiterStore points the package-level limiter store at client for the
// duration of one test, restoring the previous store afterwards. InitLimiter
// itself is a sync.Once (it is called exactly once, from main.go), so the
// tests install the store directly rather than fighting the Once.
func useLimiterStore(t *testing.T, client *goredis.Client) {
	t.Helper()
	prev := limiterStore
	limiterStore = redis.NewTenantCounter(client, limiterNamespace)
	t.Cleanup(func() { limiterStore = prev })
}

func limiterTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 0, 83)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// limiterTestContext spins an in-memory redis and wires the limiter to it.
func limiterTestContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	useLimiterStore(t, client)
	return context.Background(), limiterTestTenant(t)
}

// limiterBrokenRedisContext wires the limiter to a genuinely unreachable
// Redis: a port that was bound just long enough to reserve it, then closed.
// Every command against it fails to dial.
func limiterBrokenRedisContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close reserved port: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:        addr,
		DialTimeout: 250 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() { _ = client.Close() })
	useLimiterStore(t, client)
	return context.Background(), limiterTestTenant(t)
}

func TestLimiterAllowsUntilTheThreshold(t *testing.T) {
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		ok, err := l.Allowed(ctx, tm, 42)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("attempt %d blocked, want allowed", i)
		}
		if err := l.RecordFailure(ctx, tm, 42); err != nil {
			t.Fatal(err)
		}
	}
	ok, err := l.Allowed(ctx, tm, 42)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("attempt 4 allowed, want blocked after 3 failures")
	}
}

func TestLimiterOnlyCountsFailures(t *testing.T) {
	// A successful redemption must not consume the account's budget.
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(2, time.Minute)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allowed(ctx, tm, 43); !ok {
			t.Fatalf("attempt %d blocked without any recorded failure", i)
		}
	}
}

func TestLimiterResetClearsTheCounter(t *testing.T) {
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(2, time.Minute)

	for i := 0; i < 2; i++ {
		if err := l.RecordFailure(ctx, tm, 47); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := l.Allowed(ctx, tm, 47); err != nil || ok {
		t.Fatalf("Allowed = (%v, %v) at the threshold, want (false, nil)", ok, err)
	}
	if err := l.Reset(ctx, tm, 47); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	ok, err := l.Allowed(ctx, tm, 47)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("Allowed = false after Reset, want the counter cleared")
	}
}

func TestLimiterIsPerAccount(t *testing.T) {
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(1, time.Minute)
	if err := l.RecordFailure(ctx, tm, 44); err != nil {
		t.Fatal(err)
	}
	if ok, _ := l.Allowed(ctx, tm, 45); !ok {
		t.Error("account 45 blocked by account 44's failure")
	}
}

func TestLimiterIsPerTenant(t *testing.T) {
	ctx, tm := limiterTestContext(t)
	other := limiterTestTenant(t)
	l := NewLimiter(1, time.Minute)
	if err := l.RecordFailure(ctx, tm, 48); err != nil {
		t.Fatal(err)
	}
	if ok, _ := l.Allowed(ctx, other, 48); !ok {
		t.Error("another tenant's account 48 blocked by this tenant's failure")
	}
}

func TestLimiterFailsOpenWhenRedisIsDown(t *testing.T) {
	// Redis being unreachable must not make every coupon un-redeemable. The
	// limiter is a brute-force brake, not an authorization gate: degrade to
	// allowing the attempt and let the ladder decide.
	ctx, tm := limiterBrokenRedisContext(t)
	l := NewLimiter(1, time.Minute)
	ok, err := l.Allowed(ctx, tm, 46)
	if !ok {
		t.Errorf("Allowed = false with redis down, want fail-open (err was %v)", err)
	}
	if err == nil {
		t.Error("Allowed returned a nil error with redis down; the caller should still be able to log the outage")
	}
}

func TestInitLimiterInstallsTheStore(t *testing.T) {
	prev := limiterStore
	t.Cleanup(func() { limiterStore = prev })
	limiterStore = nil

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	InitLimiter(client)
	if limiterStore == nil {
		t.Fatal("InitLimiter did not install a counter store")
	}
}
