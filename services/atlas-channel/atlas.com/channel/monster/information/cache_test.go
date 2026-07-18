package information

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// resetInfoCache resets the singleton for test isolation (pattern:
// resetStatusMirror in the monster package).
func resetInfoCache() {
	infoCacheOnce = sync.Once{}
	infoCachePtr = nil
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testCtx(t *testing.T) (context.Context, tenant.Model) {
	tm := newTestTenant(t)
	return tenant.WithContext(context.Background(), tm), tm
}

func testModel(id uint32) Model {
	return NewModelBuilder().SetMonsterId(id).SetAttacks([]AttackInfo{{Pos: 1, ConMP: 5}}).Build()
}

func TestCache_PositiveHitAvoidsSecondFetch(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	ctx, _ := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testModel(id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(100100); err != nil {
		t.Fatalf("first: %v", err)
	}
	m, err := p.GetById(100100)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream called %d times, want 1", calls)
	}
	if len(m.Attacks()) != 1 || m.Attacks()[0].ConMP != 5 {
		t.Fatalf("cached model mismatch: %+v", m)
	}
}

func TestCache_ExpiredEntryRefetches(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	ctx, tm := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testModel(id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(100100); err != nil {
		t.Fatalf("first: %v", err)
	}

	// Force the entry past expiry (same-package test may reach internals).
	c := getInfoCache()
	c.mu.Lock()
	e := c.perTenant[tm.Id()][100100]
	e.expiresAt = time.Now().Add(-time.Second)
	c.perTenant[tm.Id()][100100] = e
	c.mu.Unlock()

	if _, err := p.GetById(100100); err != nil {
		t.Fatalf("refetch: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expired entry must refetch, upstream calls = %d", calls)
	}
}

func TestCache_NegativeCachesNotFound(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	ctx, _ := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return Model{}, fmt.Errorf("monster %d: %w", id, requests.ErrNotFound)
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(999); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("first must surface not-found, got %v", err)
	}
	if _, err := p.GetById(999); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("negative hit must synthesize not-found, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("negative hit must not refetch, upstream calls = %d", calls)
	}
}

func TestCache_TransientErrorsNotCached(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	ctx, _ := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
		calls++
		return Model{}, errors.New("connection refused")
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	_, _ = p.GetById(100100)
	_, _ = p.GetById(100100)
	if calls != 2 {
		t.Fatalf("transient errors must not be cached, upstream calls = %d", calls)
	}
}

func TestCache_DisabledPassesThrough(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	t.Setenv("MONSTER_INFO_CACHE_ENABLED", "false")
	ctx, _ := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testModel(id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	_, _ = p.GetById(100100)
	_, _ = p.GetById(100100)
	if calls != 2 {
		t.Fatalf("disabled cache must pass through, upstream calls = %d", calls)
	}
}

func TestCache_InvalidEnvFallsBackToDefaults(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	t.Setenv("MONSTER_INFO_CACHE_TTL", "banana")
	t.Setenv("MONSTER_INFO_CACHE_NEGATIVE_TTL", "48h") // out of clamp range

	cfg := getInfoCache().cfg
	if cfg.ttl != 5*time.Minute {
		t.Fatalf("invalid TTL must default to 5m, got %s", cfg.ttl)
	}
	if cfg.negativeTTL != 30*time.Second {
		t.Fatalf("out-of-range negative TTL must default to 30s, got %s", cfg.negativeTTL)
	}
	if !cfg.enabled {
		t.Fatalf("enabled must default to true")
	}
}

func TestCache_TenantIsolationAndEviction(t *testing.T) {
	resetInfoCache()
	t.Cleanup(resetInfoCache)
	ctx1, tm1 := testCtx(t)
	ctx2, _ := testCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testModel(id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	_, _ = NewProcessor(logrus.New(), ctx1).GetById(100100)
	_, _ = NewProcessor(logrus.New(), ctx2).GetById(100100)
	if calls != 2 {
		t.Fatalf("tenants must not share entries, upstream calls = %d", calls)
	}

	EvictTenant(tm1.Id())
	_, _ = NewProcessor(logrus.New(), ctx1).GetById(100100)
	if calls != 3 {
		t.Fatalf("evicted tenant must refetch, upstream calls = %d", calls)
	}
	_, _ = NewProcessor(logrus.New(), ctx2).GetById(100100)
	if calls != 3 {
		t.Fatalf("other tenant must survive eviction, upstream calls = %d", calls)
	}
}
