package information

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// resetDataCache returns the wrapper to its pre-Init state so successive
// tests can call InitDataCache against a fresh miniredis with a fresh env
// snapshot. Call BEFORE each InitDataCache.
func resetDataCache(t *testing.T) {
	t.Helper()
	dataCachePtr = nil
	dataCacheOnce = sync.Once{}
}

// newRedis spins up a miniredis-backed *goredis.Client scoped to the test.
func newRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}

// ctxFor builds a tenant context. region must be at least 1 byte (used by
// TestGetById_TenantIsolation to encode tenant identity into responses).
func ctxFor(t *testing.T, region string) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// withFakeUpstream temporarily replaces upstreamFn for the duration of the
// test. Returns the call counter so tests can assert on issued requests.
func withFakeUpstream(t *testing.T, fn func(_ logrus.FieldLogger, _ context.Context, id uint32) (RestModel, error)) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	prev := upstreamFn
	upstreamFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) (RestModel, error) {
		calls.Add(1)
		return fn(l, ctx, id)
	}
	t.Cleanup(func() { upstreamFn = prev })
	return &calls
}

// --- Env-loader and classifier unit tests ----------------------------------

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv(envEnabled, "")
	t.Setenv(envTTL, "")
	t.Setenv(envNegativeTTL, "")

	cfg := loadConfig()
	if !cfg.enabled {
		t.Fatalf("enabled = false, want true (default)")
	}
	if cfg.ttl != defaultTTL {
		t.Fatalf("ttl = %s, want %s", cfg.ttl, defaultTTL)
	}
	if cfg.negativeTTL != defaultNegativeTTL {
		t.Fatalf("negativeTTL = %s, want %s", cfg.negativeTTL, defaultNegativeTTL)
	}
}

func TestLoadConfig_InvalidValues_FallBackToDefaults(t *testing.T) {
	t.Setenv(envEnabled, "maybe")
	t.Setenv(envTTL, "not-a-duration")
	t.Setenv(envNegativeTTL, "999h") // out of range > maxNegativeTTL

	cfg := loadConfig()
	if !cfg.enabled {
		t.Fatalf("enabled = false, want true (invalid bool falls back to default true)")
	}
	if cfg.ttl != defaultTTL {
		t.Fatalf("ttl = %s, want default %s", cfg.ttl, defaultTTL)
	}
	if cfg.negativeTTL != defaultNegativeTTL {
		t.Fatalf("negativeTTL = %s, want default %s", cfg.negativeTTL, defaultNegativeTTL)
	}
}

func TestLoadConfig_ExplicitFalse(t *testing.T) {
	t.Setenv(envEnabled, "false")
	t.Setenv(envTTL, "10m")
	t.Setenv(envNegativeTTL, "0s")

	cfg := loadConfig()
	if cfg.enabled {
		t.Fatalf("enabled = true, want false")
	}
	if cfg.negativeTTL != 0 {
		t.Fatalf("negativeTTL = %s, want 0s", cfg.negativeTTL)
	}
}

func TestClassifyError(t *testing.T) {
	if got := classifyError(requests.ErrNotFound); got != errKindNotFound {
		t.Fatalf("classifyError(ErrNotFound) = %v, want errKindNotFound", got)
	}
	wrapped := errors.New("boom: " + requests.ErrNotFound.Error())
	if got := classifyError(wrapped); got != errKindTransient {
		t.Fatalf("classifyError(string-wrapped) = %v, want errKindTransient (must be errors.Is, not string match)", got)
	}
	wrappedW := errors.Join(errors.New("upstream"), requests.ErrNotFound)
	if got := classifyError(wrappedW); got != errKindNotFound {
		t.Fatalf("classifyError(joined w/ ErrNotFound) = %v, want errKindNotFound", got)
	}
	if got := classifyError(requests.ErrBadRequest); got != errKindTransient {
		t.Fatalf("classifyError(ErrBadRequest) = %v, want errKindTransient", got)
	}
	if got := classifyError(errors.New("connection refused")); got != errKindTransient {
		t.Fatalf("classifyError(generic) = %v, want errKindTransient", got)
	}
}

func TestNotFoundError_IsNotFound(t *testing.T) {
	err := notFoundError(123)
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("notFoundError no longer wraps ErrNotFound: %v", err)
	}
}

func TestGetById_HitAvoidsUpstream(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, id uint32) (RestModel, error) {
		return RestModel{Hp: id * 10, Mp: 5}, nil
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	m1, err := get(100)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if m1.Hp() != 1000 {
		t.Fatalf("m1.Hp() = %d, want 1000 (upstream returned Hp=1000 via Extract)", m1.Hp())
	}
	m2, err := get(100)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if m2.Hp() != 1000 {
		t.Fatalf("m2.Hp() = %d, want 1000 (cache hit must round-trip Hp)", m2.Hp())
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1 (second call must hit cache)", got)
	}
}

func TestGetById_NegativeCache_AvoidsUpstream(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{}, requests.ErrNotFound
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	_, err1 := get(404)
	if !errors.Is(err1, requests.ErrNotFound) {
		t.Fatalf("first call err = %v, want errors.Is(_, ErrNotFound)", err1)
	}
	_, err2 := get(404)
	if !errors.Is(err2, requests.ErrNotFound) {
		t.Fatalf("second call err = %v, want errors.Is(_, ErrNotFound)", err2)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1 (negative cache must absorb second call)", got)
	}
}

func TestGetById_TransientErrorNotCached(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	transient := errors.New("connection refused")
	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{}, transient
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	if _, err := get(500); !errors.Is(err, transient) {
		t.Fatalf("first err = %v, want %v", err, transient)
	}
	if _, err := get(500); !errors.Is(err, transient) {
		t.Fatalf("second err = %v, want %v", err, transient)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2 (transient errors must not populate negative cache)", got)
	}
}

func TestGetById_BadRequestNotCached(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{}, requests.ErrBadRequest
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	for i := 0; i < 3; i++ {
		_, err := get(400)
		if !errors.Is(err, requests.ErrBadRequest) {
			t.Fatalf("err = %v, want ErrBadRequest", err)
		}
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3 (ErrBadRequest is transient, never cached)", got)
	}
}

func TestGetById_TenantIsolation(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	// Encode tenant region into Hp so each tenant produces a distinct
	// cacheable answer. Region must be >=1 byte; ctxFor enforces this.
	_ = withFakeUpstream(t, func(_ logrus.FieldLogger, ctx context.Context, id uint32) (RestModel, error) {
		tm := tenant.MustFromContext(ctx)
		return RestModel{Hp: uint32(tm.Region()[0]) + id}, nil
	})

	ctxA := ctxFor(t, "AMS")
	ctxB := ctxFor(t, "BMS")
	getA := GetById(logrus.New())(ctxA)
	getB := GetById(logrus.New())(ctxB)

	a1, err := getA(7)
	if err != nil {
		t.Fatalf("getA: %v", err)
	}
	b1, err := getB(7)
	if err != nil {
		t.Fatalf("getB: %v", err)
	}
	if a1.Hp() == b1.Hp() {
		t.Fatalf("tenants A and B saw the same Hp (%d) - isolation broken", a1.Hp())
	}
	a2, _ := getA(7)
	b2, _ := getB(7)
	if a2.Hp() != a1.Hp() || b2.Hp() != b1.Hp() {
		t.Fatalf("cached values diverged across calls; a=%d->%d b=%d->%d", a1.Hp(), a2.Hp(), b1.Hp(), b2.Hp())
	}
	if a2.Hp() == b2.Hp() {
		t.Fatalf("on second read, tenants A and B converged to the same Hp (%d) - cross-tenant key collision", a2.Hp())
	}
}

func TestGetById_KillSwitchBypassesCache(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "false")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{Hp: 1}, nil
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	for i := 0; i < 3; i++ {
		if _, err := get(1); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3 (kill-switch must bypass cache)", got)
	}
}

func TestGetById_NotInitialized_BypassesCache(t *testing.T) {
	resetDataCache(t) // no InitDataCache

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{Hp: 1}, nil
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	for i := 0; i < 2; i++ {
		if _, err := get(1); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2 (uninitialized cache must bypass)", got)
	}
}

func TestGetById_RedisDown_FallsThroughGracefully(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	rc, mr := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{Hp: 7}, nil
	})

	mr.Close()

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	for i := 0; i < 3; i++ {
		m, err := get(1)
		if err != nil {
			t.Fatalf("call %d returned error: %v (Redis-down must NOT surface to caller)", i, err)
		}
		if m.Hp() != 7 {
			t.Fatalf("call %d: m.Hp() = %d, want 7 (must serve upstream value)", i, m.Hp())
		}
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3 (Redis-down means every call must fall through)", got)
	}
}

// TestGetById_HTTPRoundTrip_Integration exercises the real upstream path:
// requests.GetRequest -> retry loop -> jsonapi.Unmarshal of the body, plus
// the real 404 -> requests.ErrNotFound mapping. It does NOT replace
// upstreamFn; it stands up an httptest server and points DATA_SERVICE_URL
// at it (libs/atlas-rest/requests/url.go RootUrl("DATA")).
func TestGetById_HTTPRoundTrip_Integration(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "30s")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		switch r.URL.Path {
		case "/api/data/monsters/100":
			body := map[string]any{
				"data": map[string]any{
					"type": "monsters",
					"id":   "100",
					"attributes": map[string]any{
						"name":          "Pig",
						"hp":            uint32(1000),
						"mp":            uint32(50),
						"experience":    uint32(0),
						"level":         uint32(1),
						"weapon_attack": uint32(0),
					},
				},
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_ = json.NewEncoder(w).Encode(body)
		case "/api/data/monsters/404":
			http.NotFound(w, r)
		default:
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	// Positive path through real HTTP transport + real jsonapi decode.
	m1, err := get(100)
	if err != nil {
		t.Fatalf("first call (200 path): %v", err)
	}
	if m1.Hp() != 1000 {
		t.Fatalf("m1.Hp() = %d, want 1000 (real JSON:API decode)", m1.Hp())
	}
	// Cache hit - no second HTTP request.
	m2, err := get(100)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if m2.Hp() != 1000 {
		t.Fatalf("m2.Hp() = %d, want 1000 (cache hit)", m2.Hp())
	}

	// Negative path: real 404 -> real requests.ErrNotFound.
	if _, err := get(404); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("first 404 err = %v, want errors.Is(_, requests.ErrNotFound)", err)
	}
	// Negative cache must absorb the second call.
	if _, err := get(404); !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("second 404 err = %v, want errors.Is(_, requests.ErrNotFound)", err)
	}

	// Two HTTP calls total: one for 100, one for 404. Cache absorbs the rest.
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("upstream HTTP hits = %d, want 2 (1x id=100, 1x id=404)", got)
	}
}

func TestGetById_NegativeTTLZero_DisablesNegativeCache(t *testing.T) {
	resetDataCache(t)
	t.Setenv(envEnabled, "true")
	t.Setenv(envTTL, "1m")
	t.Setenv(envNegativeTTL, "0s")

	rc, _ := newRedis(t)
	InitDataCache(rc)

	calls := withFakeUpstream(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) (RestModel, error) {
		return RestModel{}, requests.ErrNotFound
	})

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	for i := 0; i < 3; i++ {
		_, err := get(404)
		if !errors.Is(err, requests.ErrNotFound) {
			t.Fatalf("call %d err = %v, want ErrNotFound", i, err)
		}
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3 (NegativeTTL=0 must disable negative caching)", got)
	}
}

func TestFlushTenant_ClearsBothNamespaces(t *testing.T) {
	client, _ := newRedis(t)
	resetDataCache(t)
	InitDataCache(client)

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := context.Background()

	for _, id := range []uint32{1, 2, 3} {
		if err := dataCachePtr.posReg.Put(ctx, tm, id, RestModel{}); err != nil {
			t.Fatalf("posReg.Put: %v", err)
		}
	}
	for _, id := range []uint32{99, 100} {
		if err := dataCachePtr.negReg.Put(ctx, tm, id, struct{}{}); err != nil {
			t.Fatalf("negReg.Put: %v", err)
		}
	}

	deleted, err := FlushTenant(ctx, tm)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 5 {
		t.Fatalf("deleted = %d, want 5", deleted)
	}
}

func TestFlushTenant_TenantIsolation(t *testing.T) {
	client, _ := newRedis(t)
	resetDataCache(t)
	InitDataCache(client)

	tA, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tB, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()

	_ = dataCachePtr.posReg.Put(ctx, tA, uint32(1), RestModel{})
	_ = dataCachePtr.posReg.Put(ctx, tB, uint32(1), RestModel{})

	if _, err := FlushTenant(ctx, tA); err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if ok, _ := dataCachePtr.posReg.Exists(ctx, tA, uint32(1)); ok {
		t.Fatal("tA key still exists")
	}
	if ok, _ := dataCachePtr.posReg.Exists(ctx, tB, uint32(1)); !ok {
		t.Fatal("tB key should still exist")
	}
}

func TestFlushTenant_KillSwitchNoOp(t *testing.T) {
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "false")
	client, _ := newRedis(t)
	resetDataCache(t)
	InitDataCache(client)

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	deleted, err := FlushTenant(context.Background(), tm)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestFlushTenant_NilCacheNoOp(t *testing.T) {
	resetDataCache(t)
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	deleted, err := FlushTenant(context.Background(), tm)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}
