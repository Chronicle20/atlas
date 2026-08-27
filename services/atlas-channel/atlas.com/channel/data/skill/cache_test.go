package skill

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

// resetSkillCache resets the singleton for test isolation (pattern:
// resetInfoCache in monster/information/cache_test.go).
func resetSkillCache() {
	skillCacheOnce = sync.Once{}
	skillCachePtr = nil
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testSkillCtx(t *testing.T) (context.Context, tenant.Model) {
	tm := newTestTenant(t)
	return tenant.WithContext(context.Background(), tm), tm
}

func testSkillModel(t *testing.T, id uint32) Model {
	t.Helper()
	m, err := Extract(RestModel{Id: id})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

func TestSkillCache_PositiveHitAvoidsSecondFetch(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx, _ := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testSkillModel(t, id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(2301002); err != nil {
		t.Fatalf("first: %v", err)
	}
	m, err := p.GetById(2301002)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream called %d times, want 1", calls)
	}
	if m.Id() != 2301002 {
		t.Fatalf("cached model mismatch: %+v", m)
	}
	if m.Effects() == nil {
		t.Fatalf("cached model Effects() must be intact, got nil")
	}
}

func TestSkillCache_ExpiredEntryRefetches(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx, tm := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testSkillModel(t, id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(2301002); err != nil {
		t.Fatalf("first: %v", err)
	}

	// Force the entry past expiry (same-package test may reach internals).
	c := getSkillCache()
	c.mu.Lock()
	e := c.perTenant[tm.Id()][2301002]
	e.expiresAt = time.Now().Add(-time.Second)
	c.perTenant[tm.Id()][2301002] = e
	c.mu.Unlock()

	if _, err := p.GetById(2301002); err != nil {
		t.Fatalf("refetch: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expired entry must refetch, upstream calls = %d", calls)
	}
}

func TestSkillCache_NegativeCachesNotFound(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx, _ := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return Model{}, fmt.Errorf("skill %d: %w", id, requests.ErrNotFound)
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	if _, err := p.GetById(999999); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("first must surface not-found, got %v", err)
	}
	if _, err := p.GetById(999999); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("negative hit must synthesize not-found, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("negative hit must not refetch, upstream calls = %d", calls)
	}
}

func TestSkillCache_TransientErrorNotCached(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx, _ := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
		calls++
		return Model{}, errors.New("connection refused")
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	_, _ = p.GetById(2301002)
	_, _ = p.GetById(2301002)
	if calls != 2 {
		t.Fatalf("transient errors must not be cached, upstream calls = %d", calls)
	}
}

func TestSkillCache_DisabledBypasses(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	t.Setenv("SKILL_DATA_CACHE_ENABLED", "false")
	ctx, _ := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testSkillModel(t, id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	_, _ = p.GetById(2301002)
	_, _ = p.GetById(2301002)
	if calls != 2 {
		t.Fatalf("disabled cache must pass through, upstream calls = %d", calls)
	}
}

func TestSkillCache_TenantIsolation(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx1, tm1 := testSkillCtx(t)
	ctx2, _ := testSkillCtx(t)

	calls := 0
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls++
		return testSkillModel(t, id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	_, _ = NewProcessor(logrus.New(), ctx1).GetById(2301002)
	_, _ = NewProcessor(logrus.New(), ctx2).GetById(2301002)
	if calls != 2 {
		t.Fatalf("tenants must not share entries, upstream calls = %d", calls)
	}

	EvictTenant(tm1.Id())
	_, _ = NewProcessor(logrus.New(), ctx1).GetById(2301002)
	if calls != 3 {
		t.Fatalf("evicted tenant must refetch, upstream calls = %d", calls)
	}
	_, _ = NewProcessor(logrus.New(), ctx2).GetById(2301002)
	if calls != 3 {
		t.Fatalf("other tenant must survive eviction, upstream calls = %d", calls)
	}
}

func TestSkillCache_ConcurrentAccess(t *testing.T) {
	resetSkillCache()
	t.Cleanup(resetSkillCache)
	ctx, _ := testSkillCtx(t)

	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		return testSkillModel(t, id), nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	p := NewProcessor(logrus.New(), ctx)
	skillIds := []uint32{2301002, 2301003, 2301004, 2301005}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				id := skillIds[(g+i)%len(skillIds)]
				if _, err := p.GetById(id); err != nil {
					t.Errorf("concurrent GetById(%d): %v", id, err)
				}
			}
		}(g)
	}
	wg.Wait()
}
