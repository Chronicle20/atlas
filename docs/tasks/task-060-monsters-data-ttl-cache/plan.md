# atlas-monsters Data TTL Cache — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate >95% of repeat `GET /api/data/monsters/{id}` calls from `atlas-monsters` to `atlas-data` by introducing an in-process, tenant-scoped TTL cache fronting `monster/information.GetById`, plus a reusable generic cache library at `libs/atlas-cache`.

**Architecture:** Two-layer split. `libs/atlas-cache` ships a generic `Cache[K,V]` (positive + negative TTL, lazy expiration, single `RWMutex` over `map[K]entry`, no Prometheus dep, injectable clock). `services/atlas-monsters/.../monster/information/cache.go` adds a per-tenant registry of `Cache[uint32, Model]`, an env-var loader (`MONSTER_DATA_CACHE_{ENABLED,TTL,NEGATIVE_TTL}`), an error classifier keyed on `requests.ErrNotFound`, and Prometheus counters/gauges. `GetById` becomes a read-through wrapper preserving its curried signature so no caller changes.

**Tech Stack:** Go 1.24+ generics, std `sync.RWMutex` + `map`, `github.com/prometheus/client_golang/promauto`, existing `libs/atlas-tenant`, `libs/atlas-rest/requests`, `httptest` for service-side tests.

---

## File Structure (locked before tasks)

**New:**
- `libs/atlas-cache/go.mod` — module declaration, std-lib only.
- `libs/atlas-cache/go.sum` — empty.
- `libs/atlas-cache/cache.go` — `Cache` interface, `cache[K,V]` impl, `New[K,V]`.
- `libs/atlas-cache/config.go` — `Config` struct.
- `libs/atlas-cache/cache_test.go` — unit + race-clean concurrency tests.
- `libs/atlas-cache/bench_test.go` — `Get`/`Put` microbenchmarks.
- `libs/atlas-cache/README.md` — API + "lost on pod restart" note.
- `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go` — registry + loader + classifier.
- `services/atlas-monsters/atlas.com/monsters/monster/information/metrics.go` — promauto counter/gauge declarations.
- `services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go` — wrapper unit tests.

**Modified:**
- `services/atlas-monsters/atlas.com/monsters/monster/information/processor.go` — `GetById` becomes read-through.
- `services/atlas-monsters/atlas.com/monsters/go.mod` + `go.sum` — add `libs/atlas-cache` (with replace) and `prometheus/client_golang`.

**Untouched:** `model.go`, `requests.go`, `rest.go`, `rest_test.go`, `builder.go`, `main.go`, every caller of `GetById`.

---

## Task 1: Bootstrap `libs/atlas-cache` module

**Files:**
- Create: `libs/atlas-cache/go.mod`
- Create: `libs/atlas-cache/go.sum`
- Create: `libs/atlas-cache/README.md`

- [ ] **Step 1: Create the module file**

Write `libs/atlas-cache/go.mod`:

```
module github.com/Chronicle20/atlas/libs/atlas-cache

go 1.24
```

- [ ] **Step 2: Create empty go.sum**

```bash
touch libs/atlas-cache/go.sum
```

- [ ] **Step 3: Write the README**

Write `libs/atlas-cache/README.md`:

```markdown
# atlas-cache

Generic in-process TTL cache with distinct positive and negative entry
lifetimes. Concurrency-safe; lazy expiration on read; no background
goroutine; no I/O dependencies.

Intended for read-heavy reference data fronting an upstream HTTP service
(e.g. `data/monsters/{id}` lookups). Cache state is **per-process** and is
**lost on restart** — that is the supported invalidation mechanism for
upstream redeploys in v1. There is no admin endpoint, no event-driven
flush, and no cross-pod coherence.

## Usage

```go
import "github.com/Chronicle20/atlas/libs/atlas-cache"

c := cache.New[uint32, MyModel](cache.Config{
    TTL:         5 * time.Minute,
    NegativeTTL: 30 * time.Second,
})

if v, ok := c.Get(id); ok {
    return v // hit
}
if c.IsNegative(id) {
    return zero, ErrNotFound // negative hit
}
v, err := upstream(id)
if err == nil { c.Put(id, v) } else if isNotFound(err) { c.PutNegative(id) }
```

## Operability

- `Config.OnEviction(kind)` is called under the write lock when a lazy
  expiration purges an entry. Use it to bump a Prometheus counter.
- `Config.Now` is the clock function (defaults to `time.Now`). Inject a
  fake for deterministic tests.
- `NegativeTTL == 0` disables negative caching (`PutNegative` is a no-op,
  `IsNegative` always returns false).
- `Len() (positive, negative int)` is O(n); call it only for sampled
  metrics, not on hot paths.
```

- [ ] **Step 4: Verify the module builds (it has no code yet, but the empty module is valid)**

Run:
```bash
cd libs/atlas-cache && go build ./...
```
Expected: no output, exit 0. (No Go files yet; build is a no-op.)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-cache/go.mod libs/atlas-cache/go.sum libs/atlas-cache/README.md
git commit -m "feat(atlas-cache): bootstrap module skeleton"
```

---

## Task 2: Define `Config` and the `Cache` interface

**Files:**
- Create: `libs/atlas-cache/config.go`
- Create: `libs/atlas-cache/cache.go`

- [ ] **Step 1: Write `config.go`**

```go
package cache

import "time"

// Config configures a Cache instance.
type Config struct {
	// TTL is the lifetime of a positive entry. Must be > 0.
	TTL time.Duration

	// NegativeTTL is the lifetime of a negative entry. Zero disables
	// negative caching (PutNegative is a no-op; IsNegative always
	// returns false).
	NegativeTTL time.Duration

	// Now is the clock function. nil falls back to time.Now.
	Now func() time.Time

	// OnEviction is called under the cache's write lock when a lazy
	// expiration removes an entry. nil disables the callback. kind is
	// "positive" or "negative".
	OnEviction func(kind string)
}
```

- [ ] **Step 2: Write the interface and constructor stub in `cache.go`**

```go
package cache

import (
	"sync"
	"time"
)

// Cache is a generic in-process TTL cache supporting distinct positive
// and negative entry lifetimes. All methods are safe for concurrent use.
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Put(key K, value V)
	PutNegative(key K)
	IsNegative(key K) bool
	Delete(key K)
	Len() (positive int, negative int)
}

type entry[V any] struct {
	value     V
	expiresAt time.Time
	negative  bool
}

type cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
	cfg     Config
	now     func() time.Time
}

// New constructs a Cache. Panics if cfg.TTL <= 0.
func New[K comparable, V any](cfg Config) Cache[K, V] {
	if cfg.TTL <= 0 {
		panic("atlas-cache: Config.TTL must be > 0")
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &cache[K, V]{
		entries: make(map[K]entry[V]),
		cfg:     cfg,
		now:     now,
	}
}

func (c *cache[K, V]) Get(key K) (V, bool)     { var z V; _ = key; return z, false }
func (c *cache[K, V]) Put(key K, value V)      { _, _ = key, value }
func (c *cache[K, V]) PutNegative(key K)       { _ = key }
func (c *cache[K, V]) IsNegative(key K) bool   { _ = key; return false }
func (c *cache[K, V]) Delete(key K)            { _ = key }
func (c *cache[K, V]) Len() (int, int)         { return 0, 0 }
```

The method bodies are deliberately empty — they will be implemented test-by-test in tasks 3–7.

- [ ] **Step 3: Verify it compiles**

Run:
```bash
cd libs/atlas-cache && go build ./...
```
Expected: exit 0, no output.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-cache/config.go libs/atlas-cache/cache.go
git commit -m "feat(atlas-cache): define Cache interface and Config"
```

---

## Task 3: Implement `Put` + `Get` (positive path) test-first

**Files:**
- Create: `libs/atlas-cache/cache_test.go`
- Modify: `libs/atlas-cache/cache.go`

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-cache/cache_test.go`:

```go
package cache

import (
	"testing"
	"time"
)

func newFakeClock() (*time.Time, func() time.Time) {
	t := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	now := &t
	return now, func() time.Time { return *now }
}

func advance(now *time.Time, d time.Duration) { *now = now.Add(d) }

func TestCache_Get_Miss(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute})
	if _, ok := c.Get(1); ok {
		t.Fatalf("Get(1) on empty cache: ok=true, want false")
	}
}

func TestCache_PutThenGet_Hit(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute})
	c.Put(7, "seven")
	v, ok := c.Get(7)
	if !ok || v != "seven" {
		t.Fatalf("Get(7) after Put: (%q, %v), want (\"seven\", true)", v, ok)
	}
}

func TestCache_Put_Overwrites(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute})
	c.Put(7, "first")
	c.Put(7, "second")
	v, _ := c.Get(7)
	if v != "second" {
		t.Fatalf("Get(7) = %q, want \"second\"", v)
	}
}
```

- [ ] **Step 2: Run tests, confirm failure**

Run:
```bash
cd libs/atlas-cache && go test -run TestCache_ -v
```
Expected: `TestCache_PutThenGet_Hit` fails (`ok=false`), `TestCache_Put_Overwrites` fails. `TestCache_Get_Miss` passes (stub returns false).

- [ ] **Step 3: Implement `Put` and `Get`**

Replace the stub methods in `libs/atlas-cache/cache.go`:

```go
func (c *cache[K, V]) Get(key K) (V, bool) {
	var zero V
	c.mu.RLock()
	e, ok := c.entries[key]
	if !ok || e.negative {
		c.mu.RUnlock()
		return zero, false
	}
	if !e.expiresAt.After(c.now()) {
		c.mu.RUnlock()
		c.evict(key, e, "positive")
		return zero, false
	}
	v := e.value
	c.mu.RUnlock()
	return v, true
}

func (c *cache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	c.entries[key] = entry[V]{
		value:     value,
		expiresAt: c.now().Add(c.cfg.TTL),
		negative:  false,
	}
	c.mu.Unlock()
}

// evict removes key under the write lock if the live entry still matches
// the expired snapshot the caller observed. Idempotent across concurrent
// calls.
func (c *cache[K, V]) evict(key K, snapshot entry[V], kind string) {
	c.mu.Lock()
	cur, ok := c.entries[key]
	if ok && cur.expiresAt.Equal(snapshot.expiresAt) && cur.negative == snapshot.negative {
		delete(c.entries, key)
		if c.cfg.OnEviction != nil {
			c.cfg.OnEviction(kind)
		}
	}
	c.mu.Unlock()
}
```

- [ ] **Step 4: Run tests, confirm pass**

Run:
```bash
cd libs/atlas-cache && go test -run TestCache_ -v
```
Expected: all three tests PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-cache/cache.go libs/atlas-cache/cache_test.go
git commit -m "feat(atlas-cache): implement Put and Get for positive entries"
```

---

## Task 4: Implement positive-entry expiration via injected clock

**Files:**
- Modify: `libs/atlas-cache/cache_test.go`

- [ ] **Step 1: Add the failing tests**

Append to `libs/atlas-cache/cache_test.go`:

```go
func TestCache_Get_Expired(t *testing.T) {
	now, clock := newFakeClock()
	c := New[int, string](Config{TTL: time.Minute, Now: clock})
	c.Put(7, "seven")
	advance(now, time.Minute) // exactly TTL elapsed → expired (strict After)
	if _, ok := c.Get(7); ok {
		t.Fatalf("Get(7) at TTL boundary: ok=true, want false")
	}
}

func TestCache_Get_Expired_PurgesEntry(t *testing.T) {
	now, clock := newFakeClock()
	var evicted []string
	c := New[int, string](Config{
		TTL:        time.Minute,
		Now:        clock,
		OnEviction: func(kind string) { evicted = append(evicted, kind) },
	})
	c.Put(7, "seven")
	advance(now, 2*time.Minute)
	_, _ = c.Get(7) // triggers lazy eviction
	if pos, _ := c.Len(); pos != 0 {
		t.Fatalf("positive Len after expired Get = %d, want 0", pos)
	}
	if len(evicted) != 1 || evicted[0] != "positive" {
		t.Fatalf("evicted = %v, want [positive]", evicted)
	}
}
```

- [ ] **Step 2: Run, confirm failure**

Run:
```bash
cd libs/atlas-cache && go test -run TestCache_Get_Expired -v
```
Expected: `TestCache_Get_Expired_PurgesEntry` fails because `Len()` still returns `(0, 0)` from the stub.

- [ ] **Step 3: Implement `Len`**

Replace the `Len` stub in `cache.go`:

```go
func (c *cache[K, V]) Len() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := c.now()
	var pos, neg int
	for _, e := range c.entries {
		if !e.expiresAt.After(now) {
			continue
		}
		if e.negative {
			neg++
		} else {
			pos++
		}
	}
	return pos, neg
}
```

- [ ] **Step 4: Run, confirm pass**

Run:
```bash
cd libs/atlas-cache && go test -run TestCache_ -v
```
Expected: all five tests PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-cache/cache.go libs/atlas-cache/cache_test.go
git commit -m "feat(atlas-cache): lazy expiration + Len for positive entries"
```

---

## Task 5: Implement `PutNegative`, `IsNegative`, and negative expiration

**Files:**
- Modify: `libs/atlas-cache/cache_test.go`
- Modify: `libs/atlas-cache/cache.go`

- [ ] **Step 1: Add the failing tests**

Append to `cache_test.go`:

```go
func TestCache_PutNegative_IsNegative(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute, NegativeTTL: 10 * time.Second})
	c.PutNegative(42)
	if c.IsNegative(42) != true {
		t.Fatalf("IsNegative(42) = false, want true")
	}
	if _, ok := c.Get(42); ok {
		t.Fatalf("Get(42) on negative entry: ok=true, want false")
	}
}

func TestCache_NegativeExpires(t *testing.T) {
	now, clock := newFakeClock()
	var evicted []string
	c := New[int, string](Config{
		TTL: time.Minute, NegativeTTL: 10 * time.Second, Now: clock,
		OnEviction: func(kind string) { evicted = append(evicted, kind) },
	})
	c.PutNegative(42)
	advance(now, 11*time.Second)
	if c.IsNegative(42) {
		t.Fatalf("IsNegative(42) after expiry = true, want false")
	}
	if _, neg := c.Len(); neg != 0 {
		t.Fatalf("negative Len = %d, want 0", neg)
	}
	if len(evicted) != 1 || evicted[0] != "negative" {
		t.Fatalf("evicted = %v, want [negative]", evicted)
	}
}

func TestCache_NegativeTTLZero_Disabled(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute, NegativeTTL: 0})
	c.PutNegative(42)
	if c.IsNegative(42) {
		t.Fatalf("IsNegative(42) with NegativeTTL=0: true, want false")
	}
}

func TestCache_PutOverridesNegative(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute, NegativeTTL: time.Minute})
	c.PutNegative(42)
	c.Put(42, "found")
	if c.IsNegative(42) {
		t.Fatalf("IsNegative(42) after Put: true, want false")
	}
	v, ok := c.Get(42)
	if !ok || v != "found" {
		t.Fatalf("Get(42) = (%q, %v), want (\"found\", true)", v, ok)
	}
}
```

- [ ] **Step 2: Run, confirm failure**

Run:
```bash
cd libs/atlas-cache && go test -run TestCache_PutNegative -v
```
Expected: failures — stubs return false / no-op.

- [ ] **Step 3: Implement negative methods**

Replace the `PutNegative` and `IsNegative` stubs:

```go
func (c *cache[K, V]) PutNegative(key K) {
	if c.cfg.NegativeTTL <= 0 {
		return
	}
	c.mu.Lock()
	c.entries[key] = entry[V]{
		expiresAt: c.now().Add(c.cfg.NegativeTTL),
		negative:  true,
	}
	c.mu.Unlock()
}

func (c *cache[K, V]) IsNegative(key K) bool {
	if c.cfg.NegativeTTL <= 0 {
		return false
	}
	c.mu.RLock()
	e, ok := c.entries[key]
	if !ok || !e.negative {
		c.mu.RUnlock()
		return false
	}
	if !e.expiresAt.After(c.now()) {
		c.mu.RUnlock()
		c.evict(key, e, "negative")
		return false
	}
	c.mu.RUnlock()
	return true
}
```

- [ ] **Step 4: Run, confirm pass**

Run:
```bash
cd libs/atlas-cache && go test -v
```
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-cache/cache.go libs/atlas-cache/cache_test.go
git commit -m "feat(atlas-cache): negative caching with independent TTL"
```

---

## Task 6: Implement `Delete`

**Files:**
- Modify: `libs/atlas-cache/cache_test.go`
- Modify: `libs/atlas-cache/cache.go`

- [ ] **Step 1: Add the failing test**

Append to `cache_test.go`:

```go
func TestCache_Delete(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute, NegativeTTL: time.Minute})
	c.Put(1, "a")
	c.PutNegative(2)
	c.Delete(1)
	c.Delete(2)
	if _, ok := c.Get(1); ok {
		t.Fatalf("Get(1) after Delete: ok=true, want false")
	}
	if c.IsNegative(2) {
		t.Fatalf("IsNegative(2) after Delete: true, want false")
	}
	if pos, neg := c.Len(); pos != 0 || neg != 0 {
		t.Fatalf("Len after Delete = (%d,%d), want (0,0)", pos, neg)
	}
}

func TestCache_Delete_Missing_NoOp(t *testing.T) {
	c := New[int, string](Config{TTL: time.Minute})
	c.Delete(99) // must not panic
}
```

- [ ] **Step 2: Run, confirm failure**

```bash
cd libs/atlas-cache && go test -run TestCache_Delete -v
```
Expected: `TestCache_Delete` fails.

- [ ] **Step 3: Implement `Delete`**

```go
func (c *cache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}
```

- [ ] **Step 4: Run, confirm pass**

```bash
cd libs/atlas-cache && go test -v
```
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-cache/cache.go libs/atlas-cache/cache_test.go
git commit -m "feat(atlas-cache): Delete"
```

---

## Task 7: Concurrency stress + race-detector test

**Files:**
- Modify: `libs/atlas-cache/cache_test.go`

- [ ] **Step 1: Add the test**

Add `"sync"` to the existing import block at the top of `cache_test.go` (so it reads `import ("sync"; "testing"; "time")`), then append to the file:

```go
func TestCache_Concurrent_GetPut(t *testing.T) {
	c := New[int, int](Config{TTL: time.Hour, NegativeTTL: time.Hour})
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 5_000; i++ {
				k := i % 64
				switch g % 4 {
				case 0:
					c.Put(k, i)
				case 1:
					c.Get(k)
				case 2:
					c.PutNegative(k + 1000)
				case 3:
					c.IsNegative(k + 1000)
				}
				if i%128 == 0 {
					c.Delete(k)
				}
			}
		}(g)
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run with the race detector**

```bash
cd libs/atlas-cache && go test -race -run TestCache_Concurrent -v
```
Expected: PASS, no DATA RACE warnings.

- [ ] **Step 3: Run the full suite under -race**

```bash
cd libs/atlas-cache && go test -race ./...
```
Expected: PASS, no warnings.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-cache/cache_test.go
git commit -m "test(atlas-cache): race-detector clean concurrency stress"
```

---

## Task 8: Microbenchmarks (perf gates from design §3.5)

**Files:**
- Create: `libs/atlas-cache/bench_test.go`

- [ ] **Step 1: Write the benchmarks**

```go
package cache

import (
	"testing"
	"time"
)

func BenchmarkCacheGet_Hit(b *testing.B) {
	c := New[int, int](Config{TTL: time.Hour})
	for i := 0; i < 1024; i++ {
		c.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i & 1023)
	}
}

func BenchmarkCacheGet_Miss(b *testing.B) {
	c := New[int, int](Config{TTL: time.Hour})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i)
	}
}

func BenchmarkCachePut(b *testing.B) {
	c := New[int, int](Config{TTL: time.Hour})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(i&1023, i)
	}
}
```

- [ ] **Step 2: Run the benchmarks**

```bash
cd libs/atlas-cache && go test -bench=. -benchmem -run=^$ ./...
```
Expected: PASS. Record the ns/op for `Get_Hit`. Targets per design §3.5: hit ≤ 200 ns/op, miss ≤ 100 ns/op, put ≤ 500 ns/op on commodity hardware.

If a target is missed by >2x, stop and revisit the locking model (`sync.Map` alternative discussed in design §3.2). Otherwise note the measured numbers in the commit message and continue.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-cache/bench_test.go
git commit -m "bench(atlas-cache): Get/Put microbenchmarks"
```

---

## Task 9: Wire `libs/atlas-cache` and `prometheus/client_golang` into atlas-monsters' go.mod

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/go.mod`
- Modify: `services/atlas-monsters/atlas.com/monsters/go.sum`

- [ ] **Step 1: Add the require + replace**

Edit `services/atlas-monsters/atlas.com/monsters/go.mod`. In the main `require ( ... )` block, add:

```
	github.com/Chronicle20/atlas/libs/atlas-cache v0.0.0
	github.com/prometheus/client_golang v1.23.2
```

In the `replace ( ... )` block (or append if not yet present), add:

```
	github.com/Chronicle20/atlas/libs/atlas-cache => ../../../../libs/atlas-cache
```

(The four `..` segments go from `services/atlas-monsters/atlas.com/monsters/` up to repo root, then down to `libs/atlas-cache`.)

- [ ] **Step 2: Run `go mod tidy`**

```bash
cd services/atlas-monsters/atlas.com/monsters && go mod tidy
```
Expected: indirect prometheus deps appear in `go.sum`. No errors.

- [ ] **Step 3: Verify the service still builds**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```
Expected: exit 0.

- [ ] **Step 4: Verify existing tests still pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```
Expected: all existing tests PASS unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/go.mod services/atlas-monsters/atlas.com/monsters/go.sum
git commit -m "build(atlas-monsters): add atlas-cache and prometheus deps"
```

---

## Task 10: Declare Prometheus metrics in `monster/information/metrics.go`

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/information/metrics.go`

- [ ] **Step 1: Write the metrics file**

```go
package information

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	hitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_hits_total",
			Help: "Cache hits for monster information lookups, by tenant and entry kind.",
		},
		[]string{"tenant", "kind"},
	)

	missesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_misses_total",
			Help: "Cache misses (upstream HTTP issued) for monster information lookups, by tenant.",
		},
		[]string{"tenant"},
	)

	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_errors_total",
			Help: "Upstream errors observed during monster information lookups, by tenant and classification.",
		},
		[]string{"tenant", "classification"},
	)

	cacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atlas_monsters_data_cache_size",
			Help: "Current count of cached monster information entries, by tenant and kind.",
		},
		[]string{"tenant", "kind"},
	)

	evictionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_monsters_data_cache_evictions_total",
			Help: "Lazy expirations of cached monster information entries, by tenant and reason.",
		},
		[]string{"tenant", "reason"},
	)
)
```

- [ ] **Step 2: Verify the package compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./monster/information/...
```
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/metrics.go
git commit -m "feat(monsters): declare data-cache prometheus metrics"
```

---

## Task 11: Add the env-var loader and per-tenant registry skeleton

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go`

- [ ] **Step 1: Write `cache.go`**

```go
package information

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	cache "github.com/Chronicle20/atlas/libs/atlas-cache"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// --- Configuration ---------------------------------------------------------

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

const (
	envEnabled     = "MONSTER_DATA_CACHE_ENABLED"
	envTTL         = "MONSTER_DATA_CACHE_TTL"
	envNegativeTTL = "MONSTER_DATA_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute
)

var (
	cfgOnce sync.Once
	cfg     cacheConfig

	tenantCachesMu sync.RWMutex
	tenantCaches   = make(map[uuid.UUID]cache.Cache[uint32, Model])

	// configLogger is the logger used for one-time configuration warnings.
	// Tests may override it; in production the first call site swaps in
	// the service logger.
	configLogger logrus.FieldLogger = logrus.StandardLogger()
)

func loadConfig() {
	cfg.enabled = parseBoolEnv(envEnabled, true)
	cfg.ttl = parseDurationEnv(envTTL, defaultTTL, minTTL, maxTTL)
	cfg.negativeTTL = parseDurationEnv(envNegativeTTL, defaultNegativeTTL, minNegativeTTL, maxNegativeTTL)
}

func parseBoolEnv(name string, def bool) bool {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	switch v {
	case "true", "TRUE", "True", "1", "yes", "y":
		return true
	case "false", "FALSE", "False", "0", "no", "n":
		return false
	default:
		configLogger.Warnf("invalid bool for %s=%q; using default %v", name, v, def)
		return def
	}
}

func parseDurationEnv(name string, def, min, max time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		configLogger.Warnf("invalid duration for %s=%q; using default %s", name, v, def)
		return def
	}
	if d < min || d > max {
		configLogger.Warnf("%s=%s out of range [%s, %s]; using default %s", name, d, min, max, def)
		return def
	}
	return d
}

// --- Per-tenant registry ---------------------------------------------------

// cacheFor returns the per-tenant cache, creating it on first use. Returns
// nil when the kill-switch is set, signaling the caller to bypass the
// cache entirely.
func cacheFor(t tenant.Model) cache.Cache[uint32, Model] {
	cfgOnce.Do(loadConfig)
	if !cfg.enabled {
		return nil
	}
	id := t.Id()

	tenantCachesMu.RLock()
	c, ok := tenantCaches[id]
	tenantCachesMu.RUnlock()
	if ok {
		return c
	}

	tenantCachesMu.Lock()
	defer tenantCachesMu.Unlock()
	if c, ok = tenantCaches[id]; ok {
		return c
	}
	tenantStr := id.String()
	c = cache.New[uint32, Model](cache.Config{
		TTL:         cfg.ttl,
		NegativeTTL: cfg.negativeTTL,
		OnEviction: func(kind string) {
			reason := "expired_" + kind
			evictionsTotal.WithLabelValues(tenantStr, reason).Inc()
		},
	})
	tenantCaches[id] = c
	return c
}

// --- Error classification --------------------------------------------------

type errKind int

const (
	errKindTransient errKind = iota
	errKindNotFound
)

// classifyError decides whether a failed upstream lookup should be cached
// as a negative entry. The underlying transport at libs/atlas-rest/requests
// returns the sentinel requests.ErrNotFound on HTTP 404; all other
// failures (network, 5xx, parse, retry exhaustion, requests.ErrBadRequest)
// are treated as transient and not cached.
func classifyError(err error) errKind {
	if errors.Is(err, requests.ErrNotFound) {
		return errKindNotFound
	}
	return errKindTransient
}

// notFoundError synthesizes a not-found error for negative-cache hits so
// callers see the same errors.Is(err, requests.ErrNotFound) shape they
// would see from a live miss.
func notFoundError(monsterId uint32) error {
	return fmt.Errorf("monster %d not found: %w", monsterId, requests.ErrNotFound)
}

// --- Upstream fetch hook (test-overridable) -------------------------------

// upstreamFn is the indirection that lets cache_test.go inject a fake
// upstream without standing up a full httptest.Server. Production code
// uses upstreamFetch.
var upstreamFn = upstreamFetch

func upstreamFetch(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(monsterId), Extract)()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./monster/information/...
```
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/cache.go
git commit -m "feat(monsters): per-tenant data cache registry and config loader"
```

---

## Task 12: Wire `GetById` into the read-through cache

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/processor.go`

- [ ] **Step 1: Replace `processor.go`**

```go
package information

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// GetById returns the monster information for monsterId, served from a
// per-tenant in-process TTL cache when enabled. Signature is preserved
// for all existing call sites.
func GetById(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) (Model, error) {
	return func(ctx context.Context) func(monsterId uint32) (Model, error) {
		return func(monsterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c := cacheFor(t)
			if c == nil {
				// Kill-switch path: behave exactly like the pre-cache implementation.
				return upstreamFn(l, ctx, monsterId)
			}
			tenantStr := t.Id().String()

			if v, ok := c.Get(monsterId); ok {
				hitsTotal.WithLabelValues(tenantStr, "positive").Inc()
				return v, nil
			}
			if c.IsNegative(monsterId) {
				hitsTotal.WithLabelValues(tenantStr, "negative").Inc()
				return Model{}, notFoundError(monsterId)
			}

			missesTotal.WithLabelValues(tenantStr).Inc()
			v, err := upstreamFn(l, ctx, monsterId)
			if err == nil {
				c.Put(monsterId, v)
				return v, nil
			}
			switch classifyError(err) {
			case errKindNotFound:
				errorsTotal.WithLabelValues(tenantStr, "not_found").Inc()
				c.PutNegative(monsterId)
			default:
				errorsTotal.WithLabelValues(tenantStr, "transient").Inc()
			}
			return Model{}, err
		}
	}
}
```

- [ ] **Step 2: Verify the service compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```
Expected: exit 0.

- [ ] **Step 3: Run the existing test suite to confirm we did not break anything**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```
Expected: PASS. (Existing callers of `GetById` either use function-typed lookups in tests or never run the real `GetById` path, so the wrapper is invisible to them.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/processor.go
git commit -m "feat(monsters): GetById is now a tenant-scoped read-through cache"
```

---

## Task 13: Wrapper unit tests — cache hit, miss, and tenant isolation

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go`

- [ ] **Step 1: Write the tests**

Create `services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go` with these contents:

```go
package information

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// resetCacheState wipes per-tenant caches and resets the config Once so
// each test sees a clean slate under per-test env vars. Tests using this
// helper must NOT run in parallel — the singletons are package-scoped.
func resetCacheState(t *testing.T) {
	t.Helper()
	tenantCachesMu.Lock()
	for k := range tenantCaches {
		delete(tenantCaches, k)
	}
	tenantCachesMu.Unlock()
	cfgOnce = sync.Once{}
}

// _ = uuid.Nil keeps the uuid import live; resetCacheState uses uuid.UUID
// implicitly through tenantCaches' map type.
var _ = uuid.Nil

func ctxFor(t *testing.T, region string) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func TestGetById_HitAvoidsUpstream(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "true")
	t.Setenv("MONSTER_DATA_CACHE_TTL", "1m")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "30s")

	var calls atomic.Int32
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		calls.Add(1)
		return Model{hp: uint32(id) * 10}, nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	if _, err := get(100); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := get(100); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

func TestGetById_NegativeCache_AvoidsUpstream(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "true")
	t.Setenv("MONSTER_DATA_CACHE_TTL", "1m")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "30s")

	var calls atomic.Int32
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
		calls.Add(1)
		return Model{}, requests.ErrNotFound
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	_, err1 := get(404)
	if !errors.Is(err1, requests.ErrNotFound) {
		t.Fatalf("first call err = %v, want ErrNotFound", err1)
	}
	_, err2 := get(404)
	if !errors.Is(err2, requests.ErrNotFound) {
		t.Fatalf("second call err = %v, want ErrNotFound", err2)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

func TestGetById_TransientErrorNotCached(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "true")
	t.Setenv("MONSTER_DATA_CACHE_TTL", "1m")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "30s")

	var calls atomic.Int32
	transient := errors.New("connection refused")
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
		calls.Add(1)
		return Model{}, transient
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)

	if _, err := get(500); !errors.Is(err, transient) {
		t.Fatalf("err = %v, want transient", err)
	}
	if _, err := get(500); !errors.Is(err, transient) {
		t.Fatalf("second err = %v, want transient", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2 (transient errors must not cache)", got)
	}
}

func TestGetById_TenantIsolation(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "true")
	t.Setenv("MONSTER_DATA_CACHE_TTL", "1m")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "30s")

	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, ctx context.Context, id uint32) (Model, error) {
		tm := tenant.MustFromContext(ctx)
		// Encode tenant identity into the response so the test can check
		// each tenant sees its own value.
		hash := uint32(tm.Region()[0])
		return Model{hp: hash + id}, nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctxA := ctxFor(t, "A-region")
	ctxB := ctxFor(t, "B-region")
	getA := GetById(logrus.New())(ctxA)
	getB := GetById(logrus.New())(ctxB)

	a, err := getA(7)
	if err != nil {
		t.Fatalf("getA: %v", err)
	}
	b, err := getB(7)
	if err != nil {
		t.Fatalf("getB: %v", err)
	}
	if a.Hp() == b.Hp() {
		t.Fatalf("tenant A and B saw the same value (%d) — isolation broken", a.Hp())
	}
	// Re-read to confirm cache hit returns each tenant's own value.
	a2, _ := getA(7)
	b2, _ := getB(7)
	if a2.Hp() != a.Hp() || b2.Hp() != b.Hp() {
		t.Fatalf("cached values diverged across calls; a=%d->%d b=%d->%d", a.Hp(), a2.Hp(), b.Hp(), b2.Hp())
	}
}

func TestGetById_KillSwitchBypassesCache(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "false")

	var calls atomic.Int32
	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (Model, error) {
		calls.Add(1)
		return Model{hp: 1}, nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	_, _ = get(1)
	_, _ = get(1)
	_, _ = get(1)
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3 (kill-switch must bypass cache)", got)
	}
}

func TestLoadConfig_InvalidValues_FallBackToDefaults(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_TTL", "not-a-duration")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "999h")
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "")

	loadConfig()

	if cfg.ttl != defaultTTL {
		t.Fatalf("cfg.ttl = %s, want default %s", cfg.ttl, defaultTTL)
	}
	if cfg.negativeTTL != defaultNegativeTTL {
		t.Fatalf("cfg.negativeTTL = %s, want default %s", cfg.negativeTTL, defaultNegativeTTL)
	}
	if !cfg.enabled {
		t.Fatalf("cfg.enabled = false, want true (empty env should default to true)")
	}
}

```

- [ ] **Step 2: Run the tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -run TestGetById -v ./monster/information/...
```
Expected: all five `TestGetById_*` and `TestLoadConfig_*` PASS.

- [ ] **Step 3: Run the full service test suite under -race**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```
Expected: PASS, no DATA RACE warnings, no regressions in other packages.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go
git commit -m "test(monsters): wrapper-level cache tests (hit, negative, isolation, kill-switch)"
```

---

## Task 14: Cache-size sampling goroutine

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go`

- [ ] **Step 1: Add the sampler**

Append to `cache.go`:

```go
// Sample interval for the cache_size gauge. Matches the existing
// RegistryAudit cadence in monster/.
const cacheSizeSampleInterval = 30 * time.Second

var samplerOnce sync.Once

// startSampler launches a single goroutine that periodically polls Len()
// on every per-tenant cache and sets the cache_size gauge. Idempotent.
func startSampler() {
	samplerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(cacheSizeSampleInterval)
			defer ticker.Stop()
			for range ticker.C {
				sampleCacheSizes()
			}
		}()
	})
}

func sampleCacheSizes() {
	tenantCachesMu.RLock()
	snapshot := make(map[uuid.UUID]cache.Cache[uint32, Model], len(tenantCaches))
	for k, v := range tenantCaches {
		snapshot[k] = v
	}
	tenantCachesMu.RUnlock()
	for id, c := range snapshot {
		pos, neg := c.Len()
		tenantStr := id.String()
		cacheSize.WithLabelValues(tenantStr, "positive").Set(float64(pos))
		cacheSize.WithLabelValues(tenantStr, "negative").Set(float64(neg))
	}
}
```

- [ ] **Step 2: Kick off the sampler from `cacheFor` on first tenant**

In `cacheFor`, just after the line `tenantCaches[id] = c`, insert:

```go
	startSampler()
```

So the section becomes:

```go
	tenantCaches[id] = c
	startSampler()
	return c
```

- [ ] **Step 3: Add a unit test for `sampleCacheSizes`**

Append to `cache_test.go`:

```go
func TestSampleCacheSizes_PopulatesGauge(t *testing.T) {
	resetCacheState(t)
	t.Setenv("MONSTER_DATA_CACHE_ENABLED", "true")
	t.Setenv("MONSTER_DATA_CACHE_TTL", "1m")
	t.Setenv("MONSTER_DATA_CACHE_NEGATIVE_TTL", "30s")

	prev := upstreamFn
	upstreamFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (Model, error) {
		return Model{hp: id}, nil
	}
	t.Cleanup(func() { upstreamFn = prev })

	ctx := ctxFor(t, "GMS")
	get := GetById(logrus.New())(ctx)
	for i := uint32(1); i <= 3; i++ {
		if _, err := get(i); err != nil {
			t.Fatalf("get(%d): %v", i, err)
		}
	}

	// Drive the sampler synchronously rather than waiting on the goroutine.
	sampleCacheSizes()

	// We cannot easily read promauto gauge values without registering a
	// custom registry; instead assert via Cache.Len directly.
	tenantCachesMu.RLock()
	defer tenantCachesMu.RUnlock()
	if len(tenantCaches) != 1 {
		t.Fatalf("tenantCaches len = %d, want 1", len(tenantCaches))
	}
	for _, c := range tenantCaches {
		pos, _ := c.Len()
		if pos != 3 {
			t.Fatalf("positive Len = %d, want 3", pos)
		}
	}
}
```

- [ ] **Step 4: Run, verify pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/information/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/cache.go services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go
git commit -m "feat(monsters): periodic cache_size gauge sampling"
```

---

## Task 15: Verify acceptance criteria — full local verification pass

**Files:** none modified.

- [ ] **Step 1: Build everything**

```bash
cd libs/atlas-cache && go build ./...
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```
Expected: both exit 0.

- [ ] **Step 2: Run all tests with the race detector**

```bash
cd libs/atlas-cache && go test -race ./...
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```
Expected: both PASS, no DATA RACE warnings.

- [ ] **Step 3: Run benchmarks one more time and record numbers**

```bash
cd libs/atlas-cache && go test -bench=. -benchmem -run=^$ ./...
```
Expected: `Get_Hit` ≤ 200 ns/op, `Get_Miss` ≤ 100 ns/op, `Put` ≤ 500 ns/op (or ≤2x of those targets, per design §3.5).

- [ ] **Step 4: Cross-check the PRD §10 acceptance list**

Walk PRD §10 manually:

- New `libs/atlas-cache` module with positive/negative TTL, injectable `Now`, lazy expiration, race-clean tests — covered by Tasks 1–8.
- `GetById` is read-through, scoped per tenant — Task 12.
- Cache hit avoids HTTP — `TestGetById_HitAvoidsUpstream` (Task 13).
- 404 records negative entry, subsequent calls hit it — `TestGetById_NegativeCache_AvoidsUpstream` (Task 13).
- Transient errors not cached — `TestGetById_TransientErrorNotCached` (Task 13).
- Two tenants isolated — `TestGetById_TenantIsolation` (Task 13).
- Kill-switch bypasses cache — `TestGetById_KillSwitchBypassesCache` (Task 13).
- All five metrics declared and labeled — Task 10 + wired in Tasks 12, 14.
- `go build ./...` clean in both modules — verified in this Task.
- `go test -race ./...` clean in both modules — verified in this Task.
- ≥95% reduction in `GET /api/data/monsters/{id}` traffic — **manual** verification against running stack; not part of the automated plan. Document the result in the PR description.
- PRD `Status` and `Date implemented` updates — done at PR-merge time, outside this plan.

If any automated criterion above fails, return to the relevant task. If only the manual ≥95% check is outstanding, proceed.

- [ ] **Step 5: Final commit if nothing changed**

If steps 1–3 ran clean with no edits, there is nothing to commit. Proceed to plan-complete sign-off.

If a fix was required, commit it under the relevant prior task style and re-run steps 1–3.

---

## Self-Review Notes (run before handing off)

- **Spec coverage:** every PRD §10 line item is mapped to a task above (see Task 15 step 4).
- **Placeholder scan:** no "TBD", no "implement appropriate handling" without code, no "see Task N" cross-references that omit the code.
- **Type consistency:** the cache value type is `Model` everywhere (Tasks 11, 12, 14); the cache key is `uint32`; tenant ids are `uuid.UUID`; the `Cache` interface signatures in Tasks 2/3/4/5/6 match how Task 11 calls them; `requests.ErrNotFound` is used identically in Tasks 11 and 13.
- **Design correction noted:** design.md §4.4 referenced a `requests.HTTPError` typed error that does not exist. Task 11 uses `errors.Is(err, requests.ErrNotFound)` instead (recorded in `context.md` §3.2).
