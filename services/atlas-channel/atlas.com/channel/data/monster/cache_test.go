package monster

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

// resetMonsterCache resets the singleton for test isolation (pattern:
// resetSkillCache in data/skill/cache_test.go).
func resetMonsterCache() {
	monsterCacheOnce = sync.Once{}
	monsterCachePtr = nil
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testMonsterCtx(t *testing.T) (context.Context, tenant.Model) {
	tm := newTestTenant(t)
	return tenant.WithContext(context.Background(), tm), tm
}

func testMonsterModel(t *testing.T, id uint32) Model {
	t.Helper()
	m, err := Extract(RestModel{Id: id, Boss: true, TagColor: 6})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// TestMonsterDataCache is table-driven over the monster-template TTL
// cache's distinct behaviours, ported from TestSkillCache in
// data/skill/cache_test.go.
func TestMonsterDataCache(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "PositiveHitAvoidsSecondFetch",
			run: func(t *testing.T) {
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, _ := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					calls++
					return testMonsterModel(t, id), nil
				}
				t.Cleanup(func() { upstreamFn = prev })

				m1, err := getByIdCached(logrus.New(), ctx, 8800002)
				if err != nil {
					t.Fatalf("first: %v", err)
				}
				m2, err := getByIdCached(logrus.New(), ctx, 8800002)
				if err != nil {
					t.Fatalf("second: %v", err)
				}
				if calls != 1 {
					t.Fatalf("upstream called %d times, want 1", calls)
				}
				want := testMonsterModel(t, 8800002)
				if m1 != want || m2 != want {
					t.Fatalf("cached model mismatch: m1=%+v m2=%+v want=%+v", m1, m2, want)
				}
			},
		},
		{
			name: "ExpiredEntryRefetches",
			run: func(t *testing.T) {
				t.Setenv("MONSTER_DATA_CACHE_TTL", "1s")
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, tm := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					calls++
					return testMonsterModel(t, id), nil
				}
				t.Cleanup(func() { upstreamFn = prev })

				c := getMonsterCache()
				c.put(tm, 8800002, cacheEntry{m: testMonsterModel(t, 8800002), expiresAt: time.Now().Add(-time.Second)})

				if _, err := getByIdCached(logrus.New(), ctx, 8800002); err != nil {
					t.Fatalf("refetch: %v", err)
				}
				if calls != 1 {
					t.Fatalf("expired entry must refetch, upstream calls = %d", calls)
				}
			},
		},
		{
			name: "NegativeCachesNotFound",
			run: func(t *testing.T) {
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, _ := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					calls++
					return Model{}, fmt.Errorf("monster %d: %w", id, requests.ErrNotFound)
				}
				t.Cleanup(func() { upstreamFn = prev })

				if _, err := getByIdCached(logrus.New(), ctx, 999999); !errors.Is(err, requests.ErrNotFound) {
					t.Fatalf("first must surface not-found, got %v", err)
				}
				if _, err := getByIdCached(logrus.New(), ctx, 999999); !errors.Is(err, requests.ErrNotFound) {
					t.Fatalf("negative hit must synthesize not-found, got %v", err)
				}
				if calls != 1 {
					t.Fatalf("negative hit must not refetch, upstream calls = %d", calls)
				}
			},
		},
		{
			name: "TransientErrorNotCached",
			run: func(t *testing.T) {
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, _ := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
					calls++
					return Model{}, errors.New("boom")
				}
				t.Cleanup(func() { upstreamFn = prev })

				_, _ = getByIdCached(logrus.New(), ctx, 8800002)
				_, _ = getByIdCached(logrus.New(), ctx, 8800002)
				if calls != 2 {
					t.Fatalf("transient errors must not be cached, upstream calls = %d", calls)
				}
			},
		},
		{
			name: "DisabledBypasses",
			run: func(t *testing.T) {
				t.Setenv("MONSTER_DATA_CACHE_ENABLED", "false")
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, _ := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					calls++
					return testMonsterModel(t, id), nil
				}
				t.Cleanup(func() { upstreamFn = prev })

				_, _ = getByIdCached(logrus.New(), ctx, 8800002)
				_, _ = getByIdCached(logrus.New(), ctx, 8800002)
				if calls != 2 {
					t.Fatalf("disabled cache must pass through, upstream calls = %d", calls)
				}
			},
		},
		{
			name: "TenantIsolation",
			run: func(t *testing.T) {
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx1, _ := testMonsterCtx(t)
				ctx2, _ := testMonsterCtx(t)

				calls := 0
				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					calls++
					return testMonsterModel(t, id), nil
				}
				t.Cleanup(func() { upstreamFn = prev })

				_, _ = getByIdCached(logrus.New(), ctx1, 8800002)
				_, _ = getByIdCached(logrus.New(), ctx2, 8800002)
				if calls != 2 {
					t.Fatalf("tenants must not share entries, upstream calls = %d", calls)
				}
			},
		},
		{
			name: "ConcurrentAccess",
			run: func(t *testing.T) {
				resetMonsterCache()
				t.Cleanup(resetMonsterCache)
				ctx, tm := testMonsterCtx(t)

				seeded := testMonsterModel(t, 8800002)
				c := getMonsterCache()
				c.put(tm, 8800002, cacheEntry{m: seeded, expiresAt: time.Now().Add(defaultTTL)})

				prev := upstreamFn
				upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
					return testMonsterModel(t, id), nil
				}
				t.Cleanup(func() { upstreamFn = prev })

				var wg sync.WaitGroup
				for g := 0; g < 50; g++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						m, err := getByIdCached(logrus.New(), ctx, 8800002)
						if err != nil {
							t.Errorf("concurrent getByIdCached: %v", err)
							return
						}
						if m != seeded {
							t.Errorf("concurrent result mismatch: got %+v want %+v", m, seeded)
						}
					}()
				}
				wg.Wait()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
