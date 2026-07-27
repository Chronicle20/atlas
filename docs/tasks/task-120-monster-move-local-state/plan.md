# Monster Movement Local State (PS-3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove both REST calls from the steady-state monster move path in atlas-channel by projecting live monster state into an in-process mirror and fronting template attack-info lookups with an in-process TTL cache, closing the three missing `MP_CHANGED` emissions in atlas-monsters so the mirror stays accurate.

**Architecture:** A new `monster.LiveMirror` (singleton, tenant-scoped, `sync.RWMutex`) is seeded by the CREATED consumer handler's existing REST fetch and updated by `monster_status_event` events; `movement.Processor.ForMonster` reads it and falls back to REST only on miss (backfilling). `monster/information.GetById` gains a memory-backed positive/negative TTL cache with task-060 semantics. atlas-monsters additively emits `MP_CHANGED` on skill-cast deduct, basic-attack deduct, and recovery regen via the existing `mpChangedStatusEventProvider`.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via libs/atlas-kafka), Prometheus client_golang (new dep for atlas-channel), testing with miniredis (atlas-monsters only).

**Spec:** `docs/tasks/task-120-monster-move-local-state/design.md` (PRD: `prd.md` in same folder).

## Global Constraints

- No wire-level behavior change: ack packets, broadcast packets, and Kafka movement commands must be byte-identical to today for identical logical state (PRD §2, acceptance).
- All new state is tenant-keyed; one tenant's monsters never resolve from another tenant's entries (PRD NFR).
- Template cache env vars: `MONSTER_INFO_CACHE_ENABLED` (default `true`), `MONSTER_INFO_CACHE_TTL` (default `5m`, clamp `[1s, 24h]`), `MONSTER_INFO_CACHE_NEGATIVE_TTL` (default `30s`, clamp `[0s, 5m]`) — invalid values warn and fall back to defaults (design §5.4).
- Mirror sweep: constants, NOT env — interval `5m`, max entry age `30m` (design §3 OQ3).
- Metric names exactly (all labeled `tenant`): `atlas_channel_monster_mirror_hits_total`, `atlas_channel_monster_mirror_misses_total`, `atlas_channel_monster_mirror_fallback_total{outcome="success|failure"}`, `atlas_channel_monster_info_cache_hits_total{kind="positive|negative"}`, `atlas_channel_monster_info_cache_misses_total` (design §5.5).
- New MP-change reasons exactly: `SKILL_CAST`, `BASIC_ATTACK`, `RECOVERY`; `CharacterId=0` always; `SkillId` = mob skill id for `SKILL_CAST`, `0` otherwise (design §3 OQ2).
- Mirror events are update-only for MP/aggro: events must never create entries (design §5.1).
- Test setup uses the project Builder pattern; no `*_testhelpers.go` files (CLAUDE.md).
- This repo is a Go workspace (`go.work`); run `go test`/`go vet`/`go build` from the module directory (`services/atlas-channel/atlas.com/channel` or `services/atlas-monsters/atlas.com/monsters`).
- Verification gate before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` in both changed modules; `docker buildx bake atlas-channel atlas-monsters` from the worktree root; `tools/redis-key-guard.sh` from the worktree root (CLAUDE.md).
- Never write literal home/absolute paths into committed files.

---

### Task 1: LiveMirror + mirror metrics + main.go wiring (atlas-channel)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/metrics.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/builder.go` (add `controllerHasAggro` to the builder)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (evictor block ~line 287; REST server route initializers ~line 337)
- Modify: `services/atlas-channel/atlas.com/channel/go.mod` (+go.sum) via `go get`

**Interfaces:**
- Produces (used by Tasks 2 and 3):
  - `monster.LiveEntry` struct: `Field field.Model`, `MonsterId uint32`, `Mp uint32`, `MaxMp uint32`, `ControllerHasAggro bool`, `LastWrite time.Time`
  - `monster.GetLiveMirror() *LiveMirror`
  - `(*LiveMirror).Lookup(t tenant.Model, uniqueId uint32) (LiveEntry, bool)` — increments hit/miss counters internally
  - `(*LiveMirror).Put(t tenant.Model, uniqueId uint32, e LiveEntry)` — stamps `LastWrite`
  - `(*LiveMirror).UpdateMp(t tenant.Model, uniqueId uint32, mpAfter uint32)` — no-op when entry absent
  - `(*LiveMirror).UpdateAggro(t tenant.Model, uniqueId uint32, aggro bool)` — no-op when entry absent
  - `(*LiveMirror).Remove(t tenant.Model, uniqueId uint32)`
  - `(*LiveMirror).EvictTenant(tid uuid.UUID)`
  - `(*LiveMirror).SweepStale(now time.Time, maxAge time.Duration) int`
  - `monster.LiveEntryFromModel(mo Model) LiveEntry`
  - `monster.RecordMirrorFallback(t tenant.Model, success bool)`
  - builder: `(*modelBuilder).SetControllerHasAggro(aggro bool) *modelBuilder`

- [ ] **Step 1: Add the prometheus dependency**

Run (from `services/atlas-channel/atlas.com/channel`):
```bash
go get github.com/prometheus/client_golang@v1.23.2
go mod tidy
```
Expected: `go.mod` gains `github.com/prometheus/client_golang v1.23.2` (same version atlas-monsters uses). If `go mod tidy` complains about workspace modules, re-run with `GOWORK=off go mod tidy`.

- [ ] **Step 2: Add `controllerHasAggro` to the monster model builder**

In `services/atlas-channel/atlas.com/channel/monster/builder.go`:
- Add field `controllerHasAggro bool` to the `modelBuilder` struct.
- Add `controllerHasAggro: m.controllerHasAggro,` to `CloneModel` (the Model already carries the field — `model.go:34` — Clone currently drops it; `CloneModel` has no non-test callers in this package, verified by grep, so this is safe).
- Add the setter:

```go
// SetControllerHasAggro sets whether the controlling character currently has
// aggro on this monster.
func (b *modelBuilder) SetControllerHasAggro(aggro bool) *modelBuilder {
	b.controllerHasAggro = aggro
	return b
}
```
- Add `controllerHasAggro: b.controllerHasAggro,` to the `Model{...}` literal in `Build()`.

- [ ] **Step 3: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go`. NOTE: `newTestTenant(t)` already exists in this package (`inbox_test.go:11`) — reuse it, do NOT redefine it.

```go
package monster

import (
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
)

// newTestLiveMirror constructs an isolated mirror (bypasses the singleton +
// sweeper goroutine), mirroring the newTestStatusMirror pattern.
func newTestLiveMirror() *LiveMirror {
	return &LiveMirror{perTenant: map[uuid.UUID]map[uint32]LiveEntry{}}
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestLiveMirror_PutLookupRoundTrip(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	e := LiveEntry{Field: testField(), MonsterId: 100100, Mp: 50, MaxMp: 80, ControllerHasAggro: true}
	m.Put(tm, 7, e)

	got, ok := m.Lookup(tm, 7)
	if !ok {
		t.Fatalf("expected hit after Put")
	}
	if got.MonsterId != 100100 || got.Mp != 50 || got.MaxMp != 80 || !got.ControllerHasAggro {
		t.Fatalf("entry mismatch: %+v", got)
	}
	if got.LastWrite.IsZero() {
		t.Fatalf("Put must stamp LastWrite")
	}
}

func TestLiveMirror_LookupMiss(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	if _, ok := m.Lookup(tm, 999); ok {
		t.Fatalf("expected miss on empty mirror")
	}
}

func TestLiveMirror_UpdateMp_NoOpWhenAbsent(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.UpdateMp(tm, 7, 42)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("UpdateMp must never create an entry")
	}
}

func TestLiveMirror_UpdateAggro_NoOpWhenAbsent(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.UpdateAggro(tm, 7, true)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("UpdateAggro must never create an entry")
	}
}

func TestLiveMirror_UpdateMpAndAggro_MutatePresentEntry(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 7, LiveEntry{Field: testField(), MonsterId: 100100, Mp: 50, MaxMp: 80})
	before, _ := m.Lookup(tm, 7)

	time.Sleep(time.Millisecond)
	m.UpdateMp(tm, 7, 12)
	m.UpdateAggro(tm, 7, true)

	got, ok := m.Lookup(tm, 7)
	if !ok {
		t.Fatalf("expected hit")
	}
	if got.Mp != 12 || !got.ControllerHasAggro {
		t.Fatalf("updates not applied: %+v", got)
	}
	if got.MonsterId != 100100 || got.MaxMp != 80 {
		t.Fatalf("updates must not clobber other fields: %+v", got)
	}
	if !got.LastWrite.After(before.LastWrite) {
		t.Fatalf("every write must refresh LastWrite")
	}
}

func TestLiveMirror_Remove(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 7, LiveEntry{Field: testField(), MonsterId: 100100})
	m.Remove(tm, 7)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("expected miss after Remove")
	}
}

func TestLiveMirror_TenantIsolationAndEviction(t *testing.T) {
	m := newTestLiveMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)
	m.Put(t1, 7, LiveEntry{Field: testField(), MonsterId: 111})
	m.Put(t2, 7, LiveEntry{Field: testField(), MonsterId: 222})

	got, _ := m.Lookup(t1, 7)
	if got.MonsterId != 111 {
		t.Fatalf("cross-tenant bleed: %+v", got)
	}

	m.EvictTenant(t1.Id())
	if _, ok := m.Lookup(t1, 7); ok {
		t.Fatalf("expected t1 evicted")
	}
	if _, ok := m.Lookup(t2, 7); !ok {
		t.Fatalf("t2 must survive t1 eviction")
	}
}

func TestLiveMirror_SweepStale(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 1, LiveEntry{Field: testField(), MonsterId: 1})
	m.Put(tm, 2, LiveEntry{Field: testField(), MonsterId: 2})

	// Drive staleness with a synthetic "now" 31m ahead of both LastWrites.
	future := time.Now().Add(31 * time.Minute)
	evicted := m.SweepStale(future, 30*time.Minute)
	if evicted != 2 {
		t.Fatalf("expected both entries stale at now+31m, got %d", evicted)
	}

	m.Put(tm, 3, LiveEntry{Field: testField(), MonsterId: 3})
	evicted = m.SweepStale(time.Now(), 30*time.Minute)
	if evicted != 0 {
		t.Fatalf("fresh entry must survive, evicted %d", evicted)
	}
	if _, ok := m.Lookup(tm, 3); !ok {
		t.Fatalf("fresh entry must still be present")
	}
}

func TestLiveMirror_ConcurrentAccess(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				id := uint32(j % 10)
				m.Put(tm, id, LiveEntry{Field: testField(), MonsterId: id})
				m.UpdateMp(tm, id, uint32(j))
				m.UpdateAggro(tm, id, j%2 == 0)
				m.Lookup(tm, id)
				if j%50 == 0 {
					m.Remove(tm, id)
					m.SweepStale(time.Now(), time.Minute)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestLiveEntryFromModel_MapsAllFields(t *testing.T) {
	f := testField()
	mo, err := NewModelBuilder(7, f, 100100).
		SetMp(33).
		SetMaxMp(90).
		SetControllerHasAggro(true).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	e := LiveEntryFromModel(mo)
	if e.Field.WorldId() != f.WorldId() || e.Field.ChannelId() != f.ChannelId() || e.Field.MapId() != f.MapId() {
		t.Fatalf("field mismatch: %+v", e.Field)
	}
	if e.MonsterId != 100100 || e.Mp != 33 || e.MaxMp != 90 || !e.ControllerHasAggro {
		t.Fatalf("entry mismatch: %+v", e)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./monster/ -run TestLiveMirror -v`
Expected: FAIL to compile — `LiveMirror`, `LiveEntry`, `SetControllerHasAggro` undefined.

- [ ] **Step 5: Implement metrics.go**

Create `services/atlas-channel/atlas.com/channel/monster/metrics.go`:

```go
package monster

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	mirrorHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_hits_total",
			Help: "Live-monster mirror lookup hits, by tenant.",
		},
		[]string{"tenant"},
	)

	mirrorMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_misses_total",
			Help: "Live-monster mirror lookup misses, by tenant.",
		},
		[]string{"tenant"},
	)

	mirrorFallbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_mirror_fallback_total",
			Help: "REST fallbacks taken after a live-monster mirror miss, by tenant and outcome.",
		},
		[]string{"tenant", "outcome"},
	)
)

func recordMirrorHit(t tenant.Model) {
	mirrorHitsTotal.WithLabelValues(t.Id().String()).Inc()
}

func recordMirrorMiss(t tenant.Model) {
	mirrorMissesTotal.WithLabelValues(t.Id().String()).Inc()
}

// RecordMirrorFallback records the outcome of a REST fallback taken by a
// mirror consumer (movement path) after a Lookup miss.
func RecordMirrorFallback(t tenant.Model, success bool) {
	outcome := "failure"
	if success {
		outcome = "success"
	}
	mirrorFallbackTotal.WithLabelValues(t.Id().String(), outcome).Inc()
}
```

- [ ] **Step 6: Implement live_mirror.go**

Create `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`:

```go
package monster

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// Sweep policy is leak insurance, not a tuning surface — constants, not env
// (design §3 OQ3). Eviction of a still-live entry is harmless: the next move
// takes one REST fallback and re-backfills.
const (
	liveMirrorSweepInterval = 5 * time.Minute
	liveMirrorMaxEntryAge   = 30 * time.Minute
)

// LiveEntry is the projected live state of one monster — exactly the field
// set the movement path reads (movement/processor.go), plus MaxMp so the
// PS-1 attack path can adopt the mirror without redesign.
type LiveEntry struct {
	Field              field.Model
	MonsterId          uint32
	Mp                 uint32
	MaxMp              uint32
	ControllerHasAggro bool
	LastWrite          time.Time
}

// LiveMirror is a per-pod, in-memory, tenant-scoped projection of live
// monsters, keyed by monster object id. Populated by the CREATED consumer
// handler's REST fetch and the movement path's fallback backfill; updated by
// monster_status_event events; evicted on DESTROYED/KILLED, tenant drain,
// and a defensive staleness sweep.
type LiveMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]LiveEntry
}

var (
	liveMirrorOnce sync.Once
	liveMirror     *LiveMirror
)

// GetLiveMirror returns the process-wide singleton mirror, lazily
// initialising it (and starting its staleness sweeper) on first call.
func GetLiveMirror() *LiveMirror {
	liveMirrorOnce.Do(func() {
		liveMirror = &LiveMirror{perTenant: map[uuid.UUID]map[uint32]LiveEntry{}}
		go liveMirror.sweepLoop()
	})
	return liveMirror
}

func (m *LiveMirror) sweepLoop() {
	ticker := time.NewTicker(liveMirrorSweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		m.SweepStale(time.Now(), liveMirrorMaxEntryAge)
	}
}

// LiveEntryFromModel projects a full monster Model (REST shape) into a
// mirror entry. LastWrite is stamped by Put.
func LiveEntryFromModel(mo Model) LiveEntry {
	return LiveEntry{
		Field:              mo.Field(),
		MonsterId:          mo.MonsterId(),
		Mp:                 mo.Mp(),
		MaxMp:              mo.MaxMp(),
		ControllerHasAggro: mo.ControllerHasAggro(),
	}
}

// Lookup returns the entry for uniqueId, recording a hit/miss metric.
func (m *LiveMirror) Lookup(t tenant.Model, uniqueId uint32) (LiveEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		recordMirrorMiss(t)
		return LiveEntry{}, false
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		recordMirrorMiss(t)
		return LiveEntry{}, false
	}
	recordMirrorHit(t)
	return e, true
}

// Put stores a full, authoritative entry (CREATED seed or fallback backfill).
func (m *LiveMirror) Put(t tenant.Model, uniqueId uint32, e LiveEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		tenantMap = map[uint32]LiveEntry{}
		m.perTenant[t.Id()] = tenantMap
	}
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// UpdateMp sets the entry's MP to the absolute post-mutation value. Update
// only: events must never create entries, because the event envelope cannot
// supply ControllerHasAggro/MaxMp — a partial entry with a defaulted-false
// aggro flag would make the client render the mob idle. An absent entry is
// created authoritatively by the movement fallback instead (design §5.1).
func (m *LiveMirror) UpdateMp(t tenant.Model, uniqueId uint32, mpAfter uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	e.Mp = mpAfter
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// UpdateAggro sets the entry's controller-aggro flag. Update only — see
// UpdateMp for why events never create entries.
func (m *LiveMirror) UpdateAggro(t tenant.Model, uniqueId uint32, aggro bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	e.ControllerHasAggro = aggro
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}

// Remove evicts one monster (DESTROYED/KILLED).
func (m *LiveMirror) Remove(t tenant.Model, uniqueId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	delete(tenantMap, uniqueId)
}

// EvictTenant drops every entry for the tenant. Invoked by
// listener.RegisterEvictor when the last listener for the tenant drains.
func (m *LiveMirror) EvictTenant(tid uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.perTenant, tid)
}

// SweepStale evicts entries whose LastWrite is older than maxAge relative
// to now, returning the number evicted. Exposed for tests; production runs
// it from the sweeper ticker.
func (m *LiveMirror) SweepStale(now time.Time, maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	evicted := 0
	for tid, tenantMap := range m.perTenant {
		for id, e := range tenantMap {
			if now.Sub(e.LastWrite) > maxAge {
				delete(tenantMap, id)
				evicted++
			}
		}
		if len(tenantMap) == 0 {
			delete(m.perTenant, tid)
		}
	}
	return evicted
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./monster/ -v`
Expected: PASS (all new `TestLiveMirror_*`/`TestLiveEntryFromModel_*` plus all pre-existing monster package tests).

- [ ] **Step 8: Wire main.go — evictor + /metrics mount**

In `services/atlas-channel/atlas.com/channel/main.go`:

1. In the `listener.RegisterEvictor` block (~line 287), add one line next to `monsterDomain.GetStatusMirror().EvictTenant(tid)`:

```go
		monsterDomain.GetLiveMirror().EvictTenant(tid)
```

2. In the REST server chain (~line 337), add the metrics mount above the existing `/debug/consumers` initializer:

```go
		AddRouteInitializer(restserver.MountHandler("/metrics", promhttp.Handler())).
```

3. Add the import:

```go
	"github.com/prometheus/client_golang/prometheus/promhttp"
```

NOTE: `MountHandler` mounts under `SetBasePath("/api/")`, so the endpoint is **`/api/metrics`** — same known-gotcha family as the `/api/readyz` probe path. This is intentional; scrape config must use `/api/metrics` (design §5.5).

- [ ] **Step 9: Build and vet**

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/live_mirror.go \
        services/atlas-channel/atlas.com/channel/monster/metrics.go \
        services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go \
        services/atlas-channel/atlas.com/channel/monster/builder.go \
        services/atlas-channel/atlas.com/channel/main.go \
        services/atlas-channel/atlas.com/channel/go.mod \
        services/atlas-channel/atlas.com/channel/go.sum
git commit -m "feat(task-120): live-monster mirror with metrics and sweep in atlas-channel"
```

---

### Task 2: Mirror write paths in the monster_status_event consumer (atlas-channel)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go`

**Interfaces:**
- Consumes (Task 1): `monster.GetLiveMirror()`, `LiveEntryFromModel`, `Put`, `UpdateMp`, `UpdateAggro`, `Remove`; builder `SetControllerHasAggro`.
- Produces: package-level seam `monsterGetByIdFn` (test-swappable REST fetch used by `handleStatusEventCreated`). All existing packet-emitting behavior in every handler is unchanged (PRD FR-1.3).

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go` (reuse the file's existing `newTestTenant`/`newTestServer` helpers; add imports as needed — `atlas-channel/monster` is already imported as `monster` there? check the file header: it imports `"atlas-channel/monster"` — reuse the alias used by the existing tests). Each test uses a fresh tenant, which isolates it inside the singleton mirror.

```go
func TestHandleStatusEventCreated_SeedsLiveMirror(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		return monster.NewModelBuilder(uniqueId, f, 100100).
			SetMp(60).
			SetMaxMp(90).
			SetControllerHasAggro(true).
			Build()
	}
	defer func() { monsterGetByIdFn = prev }()

	e := monster2.StatusEvent[monster2.StatusEventCreatedBody]{
		WorldId:   0,
		ChannelId: 1,
		MapId:     100000000,
		UniqueId:  7001,
		MonsterId: 100100,
		Type:      monster2.EventStatusCreated,
		Body:      monster2.StatusEventCreatedBody{ActorId: 1},
	}
	handleStatusEventCreated(sc, nil)(logrus.New(), ctx, e)

	got, ok := monster.GetLiveMirror().Lookup(tm, 7001)
	if !ok {
		t.Fatalf("CREATED must seed the mirror")
	}
	if got.MonsterId != 100100 || got.Mp != 60 || got.MaxMp != 90 || !got.ControllerHasAggro {
		t.Fatalf("seed mismatch: %+v", got)
	}
}

func TestHandleStatusEventCreated_FetchError_NoSeed(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("boom")
	}
	defer func() { monsterGetByIdFn = prev }()

	e := monster2.StatusEvent[monster2.StatusEventCreatedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7002,
		MonsterId: 100100, Type: monster2.EventStatusCreated,
	}
	handleStatusEventCreated(sc, nil)(logrus.New(), ctx, e)

	if _, ok := monster.GetLiveMirror().Lookup(tm, 7002); ok {
		t.Fatalf("fetch failure must not seed the mirror")
	}
}

func TestHandleStatusEventMpChanged_UpdatesMirrorForUnknownReasonWithoutSession(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 7003, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 60, MaxMp: 90, ControllerHasAggro: true})

	e := monster2.StatusEvent[monster2.StatusEventMpChangedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7003,
		MonsterId: 100100, Type: monster2.EventStatusMpChanged,
		Body: monster2.StatusEventMpChangedBody{Reason: "SKILL_CAST", Amount: 23, MonsterMpAfter: 37},
	}
	// No session exists for CharacterId 0 — the mirror update must land anyway.
	handleStatusEventMpChanged(sc, nil)(logrus.New(), ctx, e)

	got, ok := monster.GetLiveMirror().Lookup(tm, 7003)
	if !ok || got.Mp != 37 {
		t.Fatalf("MP_CHANGED must update mirror before session gating / reason dispatch, got %+v ok=%v", got, ok)
	}
	if !got.ControllerHasAggro {
		t.Fatalf("MP update must not clobber aggro")
	}
}

func TestHandleStatusEventStartStopAggro_UpdateMirrorAggro(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 7004, monster.LiveEntry{Field: f, MonsterId: 100100})

	sce := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 1, ControllerHasAggro: true},
	}
	handleStatusEventStartControl(sc, nil)(logrus.New(), ctx, sce)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); !got.ControllerHasAggro {
		t.Fatalf("START_CONTROL must set aggro from body")
	}

	ste := monster2.StatusEvent[monster2.StatusEventStopControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusStopControl,
		Body: monster2.StatusEventStopControlBody{ActorId: 1},
	}
	handleStatusEventStopControl(sc, nil)(logrus.New(), ctx, ste)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); got.ControllerHasAggro {
		t.Fatalf("STOP_CONTROL must clear aggro (no controller => no aggro)")
	}

	ace := monster2.StatusEvent[monster2.StatusEventAggroChangedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusAggroChanged,
		Body: monster2.StatusEventAggroChangedBody{ControllerCharacterId: 1, ControllerHasAggro: true},
	}
	handleStatusEventAggroChanged(sc, nil)(logrus.New(), ctx, ace)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); !got.ControllerHasAggro {
		t.Fatalf("AGGRO_CHANGED must set aggro from body")
	}
}

func TestHandleStatusEventDestroyedAndKilled_RemoveMirrorEntry(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	f := field.NewBuilder(0, 1, 100000000).Build()

	monster.GetLiveMirror().Put(tm, 7005, monster.LiveEntry{Field: f, MonsterId: 100100})
	de := monster2.StatusEvent[monster2.StatusEventDestroyedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7005,
		MonsterId: 100100, Type: monster2.EventStatusDestroyed,
	}
	handleStatusEventDestroyed(sc, nil)(logrus.New(), ctx, de)
	if _, ok := monster.GetLiveMirror().Lookup(tm, 7005); ok {
		t.Fatalf("DESTROYED must evict the mirror entry")
	}

	monster.GetLiveMirror().Put(tm, 7006, monster.LiveEntry{Field: f, MonsterId: 100100})
	ke := monster2.StatusEvent[monster2.StatusEventKilledBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7006,
		MonsterId: 100100, Type: monster2.EventStatusKilled,
	}
	handleStatusEventKilled(sc, nil)(logrus.New(), ctx, ke)
	if _, ok := monster.GetLiveMirror().Lookup(tm, 7006); ok {
		t.Fatalf("KILLED must evict the mirror entry")
	}
}
```

Check the actual field names on `StatusEventAggroChangedBody`/`StatusEventKilledBody` in `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` and adjust struct literals to compile (the bodies exist; only unversed literal fields like `ControllerCharacterId` need verifying against the struct). Some handlers make REST calls to atlas-maps that fail fast in tests and only log — that is expected and does not affect the mirror assertions, because every mirror mutation is placed before or independent of those calls.

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/consumer/monster/ -run 'TestHandleStatusEvent' -v`
Expected: FAIL to compile — `monsterGetByIdFn` undefined (and mirror asserts fail once it compiles).

- [ ] **Step 3: Implement the consumer changes**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`:

1. Add the seam near the existing `monsterStatSetBroadcaster` vars (~line 362):

```go
// monsterGetByIdFn is the REST-fetch seam for handleStatusEventCreated so
// tests can seed the live mirror without an HTTP fake (same pattern as the
// broadcaster spy vars above).
var monsterGetByIdFn = func(l logrus.FieldLogger, ctx context.Context, uniqueId uint32) (monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetById(uniqueId)
}
```

2. `handleStatusEventCreated`: replace the direct call at line 130 and seed the mirror right after the fetch, before packet emission:

```go
		m, err := monsterGetByIdFn(l, ctx, e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", e.UniqueId)
			return
		}

		// Seed the live mirror from the model already fetched for the spawn
		// packet (design §3 OQ1). An event landing between the REST read and
		// this Put has a millisecond window at spawn time (MP at max) and
		// self-corrects on the next MP/aggro event.
		monster.GetLiveMirror().Put(tenant.MustFromContext(ctx), e.UniqueId, monster.LiveEntryFromModel(m))
```

3. `handleStatusEventDestroyed`: next to the existing `monster.GetStatusMirror().OnMonsterGone(t, e.UniqueId)` (line ~184), add:

```go
		monster.GetLiveMirror().Remove(t, e.UniqueId)
```

4. `handleStatusEventKilled`: next to its `OnMonsterGone` call (line ~268), add:

```go
		monster.GetLiveMirror().Remove(tenant.MustFromContext(ctx), e.UniqueId)
```

5. `handleStatusEventStartControl`: immediately after the `sc.Is` gate, add:

```go
		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ControllerHasAggro)
```

6. `handleStatusEventStopControl`: immediately after the `sc.Is` gate, add:

```go
		// No controller => no aggro (design §5.2).
		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, false)
```

7. `handleStatusEventAggroChanged`: immediately after the `sc.Is` gate (BEFORE the REST `GetById`, which can fail and return early), add:

```go
		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ControllerHasAggro)
```

8. `handleStatusEventMpChanged`: immediately after the `sc.Is` gate and BEFORE the session lookup early-return (line ~569) and before the Reason switch, add:

```go
		// Mirror MP before any session gating or Reason dispatch so the live
		// mirror tracks every MP mutation — including Reasons this handler
		// doesn't otherwise act on and events whose character has no local
		// session (design §5.2).
		monster.GetLiveMirror().UpdateMp(tenant.MustFromContext(ctx), e.UniqueId, e.Body.MonsterMpAfter)
```

No other lines in any handler change.

- [ ] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./kafka/consumer/monster/ -v`
Expected: PASS (new tests plus all pre-existing consumer tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go
git commit -m "feat(task-120): project monster_status_event stream into live mirror"
```

---

### Task 3: Movement path consumes the mirror (atlas-channel)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor.go:111-160`
- Test: `services/atlas-channel/atlas.com/channel/movement/processor_test.go`

**Interfaces:**
- Consumes (Task 1): `monster.GetLiveMirror().Lookup/Put`, `monster.LiveEntryFromModel`, `monster.RecordMirrorFallback`, `monster.LiveEntry`.
- Produces: `(*Processor).resolveLiveMonster(objectId uint32) (monster.LiveEntry, error)` and package-level seam `monsterByIdFn`.

**Behavioral invariants (PRD FR-2.2/FR-2.3/FR-2.4, acceptance):**
- Fallback failure keeps today's exact log line `"Unable to locate monster [%d] moving."` and returns the error.
- The field-consistency rejection keeps its log line and its RETURN VALUE: the old code's `return err` at line 120 always returned **nil** there (err was nil after a successful GetById). The new code must `return nil` with a comment — do NOT "fix" this into a non-nil error; that would be behavior drift.
- `ackMp` seeds from `uint16(entry.Mp)`; `useSkills` from `entry.ControllerHasAggro`; the inbox take-and-clear, packet writers, snap logic, and Kafka command emission are untouched.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/movement/processor_test.go` (this file currently holds pure-function tests only; add imports: `context`, `errors`, `atlas-channel/monster`, `github.com/Chronicle20/atlas/libs/atlas-constants/field`, `github.com/Chronicle20/atlas/libs/atlas-tenant`, `github.com/google/uuid`, `github.com/sirupsen/logrus`):

```go
func newMovementTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newMovementTestProcessor(t *testing.T) (*Processor, tenant.Model) {
	t.Helper()
	tm := newMovementTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	return NewProcessor(logrus.New(), ctx, nil), tm
}

func movementTestField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestResolveLiveMonster_WarmPath_ZeroRest(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	monster.GetLiveMirror().Put(tm, 8001, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 44, ControllerHasAggro: true})

	calls := 0
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		calls++
		return monster.Model{}, errors.New("REST must not be called on the warm path")
	}
	defer func() { monsterByIdFn = prev }()

	entry, err := p.resolveLiveMonster(8001)
	if err != nil {
		t.Fatalf("warm path errored: %v", err)
	}
	if calls != 0 {
		t.Fatalf("warm path made %d REST calls, want 0", calls)
	}
	if entry.Mp != 44 || !entry.ControllerHasAggro || entry.MonsterId != 100100 {
		t.Fatalf("entry mismatch: %+v", entry)
	}
}

func TestResolveLiveMonster_MissFallsBackOnceAndBackfills(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	calls := 0
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, objectId uint32) (monster.Model, error) {
		calls++
		return monster.NewModelBuilder(objectId, f, 100100).
			SetMp(70).
			SetMaxMp(90).
			SetControllerHasAggro(true).
			Build()
	}
	defer func() { monsterByIdFn = prev }()

	entry, err := p.resolveLiveMonster(8002)
	if err != nil {
		t.Fatalf("fallback errored: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first resolve made %d REST calls, want exactly 1", calls)
	}
	if entry.Mp != 70 || !entry.ControllerHasAggro {
		t.Fatalf("fallback entry mismatch: %+v", entry)
	}

	// Second resolve must be served from the backfilled mirror.
	if _, err := p.resolveLiveMonster(8002); err != nil {
		t.Fatalf("second resolve errored: %v", err)
	}
	if calls != 1 {
		t.Fatalf("second resolve made a REST call (total %d), want mirror hit", calls)
	}
	if got, ok := monster.GetLiveMirror().Lookup(tm, 8002); !ok || got.Mp != 70 {
		t.Fatalf("fallback must backfill the mirror, got %+v ok=%v", got, ok)
	}
}

func TestResolveLiveMonster_FallbackError_Propagates(t *testing.T) {
	p, tm := newMovementTestProcessor(t)

	wantErr := errors.New("monsters unavailable")
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, wantErr
	}
	defer func() { monsterByIdFn = prev }()

	if _, err := p.resolveLiveMonster(8003); !errors.Is(err, wantErr) {
		t.Fatalf("fallback error must propagate unchanged, got %v", err)
	}
	if _, ok := monster.GetLiveMirror().Lookup(tm, 8003); ok {
		t.Fatalf("failed fallback must not backfill the mirror")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./movement/ -run TestResolveLiveMonster -v`
Expected: FAIL to compile — `resolveLiveMonster` and `monsterByIdFn` undefined.

- [ ] **Step 3: Implement the movement changes**

In `services/atlas-channel/atlas.com/channel/movement/processor.go`:

1. Add the seam and resolver (above `ForMonster`):

```go
// monsterByIdFn is the REST fallback seam for resolveLiveMonster. Package-
// level var (precedent: the broadcaster spy vars in the monster consumer) so
// tests can prove the warm path performs zero REST calls.
var monsterByIdFn = func(l logrus.FieldLogger, ctx context.Context, objectId uint32) (monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetById(objectId)
}

// resolveLiveMonster resolves the monster's live state from the in-process
// mirror, falling back to REST on a miss and backfilling the mirror so
// subsequent moves for this monster are local (FR-2.1/FR-2.2).
func (p *Processor) resolveLiveMonster(objectId uint32) (monster.LiveEntry, error) {
	entry, ok := monster.GetLiveMirror().Lookup(p.t, objectId)
	if ok {
		return entry, nil
	}
	p.l.Debugf("Live mirror miss for monster [%d]; falling back to REST.", objectId)
	mo, err := monsterByIdFn(p.l, p.ctx, objectId)
	if err != nil {
		monster.RecordMirrorFallback(p.t, false)
		p.l.WithError(err).Errorf("Unable to locate monster [%d] moving.", objectId)
		return monster.LiveEntry{}, err
	}
	monster.RecordMirrorFallback(p.t, true)
	entry = monster.LiveEntryFromModel(mo)
	monster.GetLiveMirror().Put(p.t, objectId, entry)
	return entry, nil
}
```

2. Replace the head of `ForMonster` (current lines 112-134) with:

```go
	entry, err := p.resolveLiveMonster(objectId)
	if err != nil {
		return err
	}

	if f.WorldId() != entry.Field.WorldId() || f.ChannelId() != entry.Field.ChannelId() || f.MapId() != entry.Field.MapId() {
		p.l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", objectId, characterId)
		// Preserves pre-mirror behavior: the old code returned `err` here,
		// which was always nil after a successful GetById.
		return nil
	}
	// Forecast the post-decrement MP for basic attacks (Cosmic compat — the
	// v83 client gates on the ack carrying decremented MP). For melee /
	// non-basic-attack actions, ackMp passes through unchanged.
	ackMp := uint16(entry.Mp)
	pos0, isBasicAttack := basicAttackPos(skill)
	if isBasicAttack {
		info, ierr := monsterinfo.NewProcessor(p.l, p.ctx).GetById(entry.MonsterId)
		if ierr != nil {
			p.l.WithError(ierr).Debugf("Unable to fetch attack info for monster template [%d]; ack uses unchanged MP.", entry.MonsterId)
		} else {
			ackMp = computeAckMp(ackMp, pos0, info.Attacks())
		}
	}
```

3. In the ack goroutine (current line 144), replace `useSkills := mo.ControllerHasAggro()` with:

```go
		useSkills := entry.ControllerHasAggro
```

Nothing else in `ForMonster` changes (the goroutines, inbox logic, snap logic, and command emission stay byte-for-byte).

- [ ] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./movement/ -v`
Expected: PASS — new `TestResolveLiveMonster_*` plus the pre-existing `TestComputeAckMp_*`/`TestNarrowSkill_*` regression tests (which pin the `ackMp` math unchanged).

- [ ] **Step 5: Build the whole service and commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/movement/processor.go \
        services/atlas-channel/atlas.com/channel/movement/processor_test.go
git commit -m "feat(task-120): serve monster movement from live mirror with REST miss-fallback"
```

---

### Task 4: Template-info TTL cache (atlas-channel)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/monster/information/cache.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/metrics.go`
- Create: `services/atlas-channel/atlas.com/channel/monster/information/cache_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/information/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (evictor block)

**Interfaces:**
- Consumes: existing `information.RestModel`, `Extract`, `requestById`, `requests.ErrNotFound`.
- Produces: `information.EvictTenant(tid uuid.UUID)` (for main.go); `Processor.GetById(monsterId uint32) (Model, error)` — signature unchanged, now cached (FR-3.2). Test seam `upstreamFn` (task-060 precedent).

**Semantics (design §5.4, ported from `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go` but memory-backed):**
- Env: `MONSTER_INFO_CACHE_ENABLED` (default true), `MONSTER_INFO_CACHE_TTL` (default 5m, clamp [1s, 24h]), `MONSTER_INFO_CACHE_NEGATIVE_TTL` (default 30s, clamp [0s, 5m]); read once via `sync.Once`; invalid values warn and use defaults.
- Negative caching only for `errors.Is(err, requests.ErrNotFound)`; transient errors never cached; negative hits synthesize an error wrapping `requests.ErrNotFound`.
- Lazy expiry on read (expired ⇒ miss ⇒ refetch-and-overwrite); no sweeper. Concurrent same-key misses may duplicate the fetch (no singleflight) — accepted, matches task-060.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/monster/information/cache_test.go`:

```go
package information

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	return Model{monsterId: id, attacks: []AttackInfo{{Pos: 1, ConMP: 5}}}
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./monster/information/ -v`
Expected: FAIL to compile — `upstreamFn`, `getInfoCache`, `infoCacheOnce`, `EvictTenant` undefined.

- [ ] **Step 3: Implement cache.go**

Create `services/atlas-channel/atlas.com/channel/monster/information/cache.go`:

```go
package information

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// In-process, tenant-scoped TTL cache fronting GetById. Semantics ported
// from the task-060 Redis-backed cache in atlas-monsters
// (services/atlas-monsters/.../monster/information/cache.go) but
// memory-backed per the task-120 PRD user decision (no Redis hop on the
// movement hot path). Concurrent same-key misses may duplicate the upstream
// fetch (no singleflight) — bounded by template count, matches task-060.

const (
	envEnabled     = "MONSTER_INFO_CACHE_ENABLED"
	envTTL         = "MONSTER_INFO_CACHE_TTL"
	envNegativeTTL = "MONSTER_INFO_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute
)

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

// configLogger is the logger used for one-time configuration warnings.
var configLogger logrus.FieldLogger = logrus.StandardLogger()

func loadConfig() cacheConfig {
	return cacheConfig{
		enabled:     parseBoolEnv(envEnabled, true),
		ttl:         parseDurationEnv(envTTL, defaultTTL, minTTL, maxTTL),
		negativeTTL: parseDurationEnv(envNegativeTTL, defaultNegativeTTL, minNegativeTTL, maxNegativeTTL),
	}
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

func parseDurationEnv(name string, def, lo, hi time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		configLogger.Warnf("invalid duration for %s=%q; using default %s", name, v, def)
		return def
	}
	if d < lo || d > hi {
		configLogger.Warnf("%s=%s out of range [%s, %s]; using default %s", name, d, lo, hi, def)
		return def
	}
	return d
}

type cacheEntry struct {
	model     Model
	negative  bool
	expiresAt time.Time
}

type infoCache struct {
	cfg       cacheConfig
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]cacheEntry
}

var (
	infoCacheOnce sync.Once
	infoCachePtr  *infoCache
)

func getInfoCache() *infoCache {
	infoCacheOnce.Do(func() {
		infoCachePtr = &infoCache{
			cfg:       loadConfig(),
			perTenant: map[uuid.UUID]map[uint32]cacheEntry{},
		}
	})
	return infoCachePtr
}

// lookup returns a non-expired entry. Expired entries are treated as misses
// and overwritten in place by the subsequent refetch (lazy expiry — no
// sweeper; population is O(distinct templates)).
func (c *infoCache) lookup(tid uuid.UUID, monsterId uint32, now time.Time) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		return cacheEntry{}, false
	}
	e, ok := tenantMap[monsterId]
	if !ok || now.After(e.expiresAt) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *infoCache) put(tid uuid.UUID, monsterId uint32, e cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tenantMap, ok := c.perTenant[tid]
	if !ok {
		tenantMap = map[uint32]cacheEntry{}
		c.perTenant[tid] = tenantMap
	}
	tenantMap[monsterId] = e
}

// EvictTenant drops every cached template entry for the tenant. Invoked by
// listener.RegisterEvictor in main.go.
func EvictTenant(tid uuid.UUID) {
	c := getInfoCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.perTenant, tid)
}

// upstreamFn is the test-overridable upstream fetch (task-060 precedent).
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(monsterId), Extract)()
}

// notFoundError synthesizes a not-found error for negative-cache hits so
// callers see the same errors.Is(err, requests.ErrNotFound) shape they
// would see from a live 404.
func notFoundError(monsterId uint32) error {
	return fmt.Errorf("monster %d not found: %w", monsterId, requests.ErrNotFound)
}
```

- [ ] **Step 4: Implement metrics.go**

Create `services/atlas-channel/atlas.com/channel/monster/information/metrics.go`:

```go
package information

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	cacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_info_cache_hits_total",
			Help: "Template-info cache hits, by tenant and entry kind.",
		},
		[]string{"tenant", "kind"},
	)

	cacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_monster_info_cache_misses_total",
			Help: "Template-info cache misses (upstream HTTP issued), by tenant.",
		},
		[]string{"tenant"},
	)
)

func recordCacheHit(t tenant.Model, kind string) {
	cacheHitsTotal.WithLabelValues(t.Id().String(), kind).Inc()
}

func recordCacheMiss(t tenant.Model) {
	cacheMissesTotal.WithLabelValues(t.Id().String()).Inc()
}
```

- [ ] **Step 5: Rewrite processor.go GetById as a read-through cache**

Replace the body of `GetById` in `services/atlas-channel/atlas.com/channel/monster/information/processor.go` (signature unchanged — FR-3.2):

```go
package information

import (
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// GetById returns the parsed template attack info for monsterId, served
// from a tenant-scoped in-process read-through TTL cache when enabled.
func (p *Processor) GetById(monsterId uint32) (Model, error) {
	c := getInfoCache()
	if !c.cfg.enabled {
		return upstreamFn(p.l, p.ctx, monsterId)
	}

	t := tenant.MustFromContext(p.ctx)
	now := time.Now()

	if e, ok := c.lookup(t.Id(), monsterId, now); ok {
		if e.negative {
			recordCacheHit(t, "negative")
			return Model{}, notFoundError(monsterId)
		}
		recordCacheHit(t, "positive")
		return e.model, nil
	}

	recordCacheMiss(t)
	m, err := upstreamFn(p.l, p.ctx, monsterId)
	if err == nil {
		c.put(t.Id(), monsterId, cacheEntry{model: m, expiresAt: now.Add(c.cfg.ttl)})
		return m, nil
	}
	// Negative caching only for the not-found sentinel; transient errors
	// (network, 5xx, parse) are never cached (task-060 classification).
	if errors.Is(err, requests.ErrNotFound) && c.cfg.negativeTTL > 0 {
		c.put(t.Id(), monsterId, cacheEntry{negative: true, expiresAt: now.Add(c.cfg.negativeTTL)})
	}
	return Model{}, err
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./monster/information/ -v`
Expected: PASS — all `TestCache_*` plus the pre-existing `rest_test.go` tests.

- [ ] **Step 7: Wire the tenant evictor**

In `services/atlas-channel/atlas.com/channel/main.go`, add to the `listener.RegisterEvictor` block (next to the Task 1 line):

```go
		monsterinfo.EvictTenant(tid)
```

with import:

```go
	monsterinfo "atlas-channel/monster/information"
```

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/information/cache.go \
        services/atlas-channel/atlas.com/channel/monster/information/metrics.go \
        services/atlas-channel/atlas.com/channel/monster/information/cache_test.go \
        services/atlas-channel/atlas.com/channel/monster/information/processor.go \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-120): in-process TTL cache for monster template info"
```

---

### Task 5: MP_CHANGED emission on skill-cast and basic-attack deducts (atlas-monsters + channel constants)

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` (~line 36)
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (`UseSkill` mobskill fetch ~line 602 and deduct ~lines 626-633; `UseBasicAttack` deduct ~lines 822-828)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` (~line 107, constants only)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

**Interfaces:**
- Consumes: `mpChangedStatusEventProvider(m Model, characterId uint32, skillId uint32, reason string, amount uint32)` (`producer.go:130`), `GetMonsterRegistry().DeductMp(t, uniqueId, amount) (Model, error)` (`registry.go:604`), `p.emit` seam (`processor.go:85`), `testInformationLookup` (`processor.go:68`).
- Produces: constants `MpChangeReasonSkillCast = "SKILL_CAST"`, `MpChangeReasonBasicAttack = "BASIC_ATTACK"`, `MpChangeReasonRecovery = "RECOVERY"` in BOTH services' kafka files (Task 6 uses `MpChangeReasonRecovery`); test seam `testMobSkillLookup`.
- Mixed-version safety: atlas-channel's `handleStatusEventMpChanged` routes unknown Reasons to its `default:` debug branch (and, after Task 2, still updates the mirror) — no deploy-order constraint.

- [ ] **Step 1: Add the reason constants (both services)**

In `services/atlas-monsters/atlas.com/monsters/monster/kafka.go`, extend the const block containing `MpChangeReasonMpEater` (line 36):

```go
	MpChangeReasonMpEater     = "MP_EATER"
	MpChangeReasonSkillCast   = "SKILL_CAST"
	MpChangeReasonBasicAttack = "BASIC_ATTACK"
	MpChangeReasonRecovery    = "RECOVERY"
```

In `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`, extend the const block containing `MpChangeReasonMpEater` (line 107) with the same three constants (consumer-side documentation; no handler behavior change).

- [ ] **Step 2: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`. Follow the wiring of `TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown` (line 1481) exactly — miniredis + registry + `testInformationLookup` — and add an emit recorder:

```go
// mpChangedRecorder returns an emitter that collects MP_CHANGED envelopes.
func mpChangedRecorder(t *testing.T, out *[]statusEvent[statusEventMpChangedBody]) emitter {
	t.Helper()
	return func(topic string, provider model.Provider[[]kafka.Message]) error {
		msgs, err := provider()
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			var env statusEvent[statusEventMpChangedBody]
			if err := json.Unmarshal(msg.Value, &env); err != nil {
				continue
			}
			if env.Type == EventMonsterStatusMpChanged {
				*out = append(*out, env)
			}
		}
		return nil
	}
}

func TestUseBasicAttack_Deduct_EmitsMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{reg: atlasredis.NewRegistry[string, int64](rc, "monster-attack-cooldown", func(s string) string { return s })}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100)

	var emitted []statusEvent[statusEventMpChangedBody]
	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten, emit: mpChangedRecorder(t, &emitted)}

	p.UseBasicAttack(m.UniqueId(), uint8(1)) // 0-indexed; matches Pos=2

	if len(emitted) != 1 {
		t.Fatalf("expected exactly 1 MP_CHANGED, got %d", len(emitted))
	}
	e := emitted[0]
	if e.Body.Reason != MpChangeReasonBasicAttack {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, MpChangeReasonBasicAttack)
	}
	if e.Body.CharacterId != 0 || e.Body.SkillId != 0 {
		t.Errorf("CharacterId/SkillId = %d/%d, want 0/0", e.Body.CharacterId, e.Body.SkillId)
	}
	if e.Body.Amount != 5 || e.Body.MonsterMpAfter != 95 {
		t.Errorf("Amount/MonsterMpAfter = %d/%d, want 5/95", e.Body.Amount, e.Body.MonsterMpAfter)
	}
	if e.UniqueId != m.UniqueId() {
		t.Errorf("UniqueId = %d, want %d", e.UniqueId, m.UniqueId())
	}
}

func TestUseBasicAttack_NoDeduct_NoMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{reg: atlasredis.NewRegistry[string, int64](rc, "monster-attack-cooldown", func(s string) string { return s })}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewModelBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 0, AttackAfter: 0}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100)

	var emitted []statusEvent[statusEventMpChangedBody]
	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten, emit: mpChangedRecorder(t, &emitted)}

	p.UseBasicAttack(m.UniqueId(), uint8(1))

	if len(emitted) != 0 {
		t.Fatalf("ConMP=0 must not emit MP_CHANGED, got %d", len(emitted))
	}
}

func TestUseSkill_Deduct_EmitsMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	defer func() { cooldownReg = prevCooldown }()

	// Skill 126 (Slow) maps to SkillCategoryDebuff; with inFieldFn returning
	// no targets the executor is a no-op, isolating the deduct+emit.
	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		return mobskill.NewModelBuilder().
			SetSkillId(skillId).
			SetLevel(level).
			SetMpCon(10).
			Build(), nil
	}
	defer func() { testMobSkillLookup = prevSkill }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100)

	var emitted []statusEvent[statusEventMpChangedBody]
	p := &ProcessorImpl{
		l:         logrus.New(),
		ctx:       tenant.WithContext(ctx, ten),
		t:         ten,
		emit:      mpChangedRecorder(t, &emitted),
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	p.UseSkill(m.UniqueId(), 1, 126, 1)

	if len(emitted) != 1 {
		t.Fatalf("expected exactly 1 MP_CHANGED, got %d", len(emitted))
	}
	e := emitted[0]
	if e.Body.Reason != MpChangeReasonSkillCast {
		t.Errorf("Reason = %q, want %q", e.Body.Reason, MpChangeReasonSkillCast)
	}
	if e.Body.CharacterId != 0 {
		t.Errorf("CharacterId = %d, want 0", e.Body.CharacterId)
	}
	if e.Body.SkillId != 126 {
		t.Errorf("SkillId = %d, want the mob skill id 126", e.Body.SkillId)
	}
	if e.Body.Amount != 10 || e.Body.MonsterMpAfter != 90 {
		t.Errorf("Amount/MonsterMpAfter = %d/%d, want 10/90", e.Body.Amount, e.Body.MonsterMpAfter)
	}

	got, _ := r.GetMonster(ten, m.UniqueId())
	if got.Mp() != 90 {
		t.Errorf("registry Mp = %d, want 90", got.Mp())
	}
}

func TestUseSkill_ZeroMpCon_NoMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	defer func() { cooldownReg = prevCooldown }()

	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		return mobskill.NewModelBuilder().SetSkillId(skillId).SetLevel(level).Build(), nil
	}
	defer func() { testMobSkillLookup = prevSkill }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100)

	var emitted []statusEvent[statusEventMpChangedBody]
	p := &ProcessorImpl{
		l:         logrus.New(),
		ctx:       tenant.WithContext(ctx, ten),
		t:         ten,
		emit:      mpChangedRecorder(t, &emitted),
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	p.UseSkill(m.UniqueId(), 1, 126, 1)

	if len(emitted) != 0 {
		t.Fatalf("MpCon=0 must not emit MP_CHANGED, got %d", len(emitted))
	}
}
```

Add any missing imports the file doesn't already have (`encoding/json`, `atlas-monsters/monster/mobskill`). Note: `UseSkill` still performs one live `information.GetById` HTTP attempt for the animation delay (line ~668) which fails fast in tests and leaves `animDelay=0` (synchronous execution) — this matches how existing `UseSkill`-exercising tests behave.

- [ ] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-monsters/atlas.com/monsters`): `go test ./monster/ -run 'TestUseSkill_|TestUseBasicAttack_Deduct|TestUseBasicAttack_NoDeduct' -v`
Expected: FAIL to compile — `testMobSkillLookup`, `MpChangeReasonSkillCast`, `MpChangeReasonBasicAttack` undefined.

- [ ] **Step 4: Implement the processor changes**

In `services/atlas-monsters/atlas.com/monsters/monster/processor.go`:

1. Add the mobskill seam next to `testInformationLookup` (line ~68):

```go
// testMobSkillLookup is a test-only override for mobskill.GetByIdAndLevel.
// When nil (production), UseSkill calls mobskill.GetByIdAndLevel normally.
var testMobSkillLookup func(skillId uint16, level uint16) (mobskill.Model, error)
```

2. In `UseSkill` (line ~602), route the fetch through the seam (same shape as the `testInformationLookup` branch at line ~785):

```go
	// Fetch skill definition from data service
	var sd mobskill.Model
	if testMobSkillLookup != nil {
		sd, err = testMobSkillLookup(uint16(skillId), uint16(skillLevel))
	} else {
		sd, err = mobskill.GetByIdAndLevel(p.l)(p.ctx)(uint16(skillId), uint16(skillLevel))
	}
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve mob skill [%d] level [%d].", skillId, skillLevel)
		return
	}
```

(`err` is already declared earlier in the function by the `GetMonster` call — keep `=`, not `:=`, to avoid shadowing.)

3. In `UseSkill`, replace the deduct block (lines ~626-633):

```go
	// Deduct MP
	if sd.MpCon() > 0 {
		post, derr := GetMonsterRegistry().DeductMp(p.t, uniqueId, sd.MpCon())
		if derr != nil {
			p.l.WithError(derr).Errorf("Unable to deduct MP from monster [%d].", uniqueId)
			return
		}
		// The MP-sufficiency gate above guarantees no clamp, so the
		// requested MpCon is the exact amount deducted.
		if eerr := p.emit(EnvEventTopicMonsterStatus, mpChangedStatusEventProvider(post, 0, uint32(skillId), MpChangeReasonSkillCast, sd.MpCon())); eerr != nil {
			p.l.WithError(eerr).Errorf("Unable to emit MP_CHANGED for monster [%d] skill cast.", uniqueId)
		}
	}
```

4. In `UseBasicAttack`, replace the deduct block (lines ~822-828):

```go
	if atk.ConMP > 0 {
		post, derr := GetMonsterRegistry().DeductMp(p.t, uniqueId, uint32(atk.ConMP))
		if derr != nil {
			p.l.WithError(derr).Errorf("UseBasicAttack: DeductMp failed for monster [%d].", uniqueId)
			return
		}
		// The MP-sufficiency gate above guarantees no clamp, so ConMP is
		// the exact amount deducted.
		if eerr := p.emit(EnvEventTopicMonsterStatus, mpChangedStatusEventProvider(post, 0, 0, MpChangeReasonBasicAttack, uint32(atk.ConMP))); eerr != nil {
			p.l.WithError(eerr).Errorf("UseBasicAttack: unable to emit MP_CHANGED for monster [%d].", uniqueId)
		}
	}
```

5. Update `TestUseBasicAttack_HappyPath_DeductsMpAndRegistersCooldown` (processor_test.go line ~1481): its `ProcessorImpl` literal has no `emit`, which is now dereferenced on the deduct path — add `emit: func(string, model.Provider[[]kafka.Message]) error { return nil },` to that literal. Scan the other `ProcessorImpl` literals in `UseBasicAttack`/`UseSkill` tests: literals that never reach a successful deduct (on-cooldown, insufficient-MP, no-attack-info, ConMP=0) need no change.

- [ ] **Step 5: Run tests to verify they pass**

Run (from `services/atlas-monsters/atlas.com/monsters`): `go test -race ./monster/ -v`
Expected: PASS — new tests plus all pre-existing monster tests (especially the `TestUseBasicAttack_*` suite and `drain_mp_test.go`).

- [ ] **Step 6: Build both services and commit**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./...` and `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/kafka.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor_test.go \
        services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go
git commit -m "feat(task-120): emit MP_CHANGED on monster skill-cast and basic-attack MP deducts"
```

---

### Task 6: MP_CHANGED emission on recovery regen (atlas-monsters)

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/recovery_task_test.go`

**Interfaces:**
- Consumes: `mpChangedStatusEventProvider`, `MpChangeReasonRecovery` (Task 5), `ApplyRecovery`'s existing `(Model, hpApplied bool, mpApplied bool, error)` return (`registry.go:497`).
- Produces: new seam type `recoveryMpEmitFn func(t tenant.Model, m Model, amount uint32) error` and field `mpEmitFn` on `MonsterRecoveryTask`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-monsters/atlas.com/monsters/monster/recovery_task_test.go`, following the file's existing `MonsterRecoveryTask` literal pattern (the first test in the file seeds a registry monster, drops MP via `r.DeductMp`, and wires `applyFn: r.ApplyRecovery` — copy its exact setup shape, including tenant/field/CreateMonster args):

```go
func TestRecovery_MpApplied_EmitsMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, tm, f, 100100, 0, 0, 0, 5, 0, 3000, 100)
	// Below max MP so Run() processes the monster (maxMp seeds from mp=100).
	if _, err := r.DeductMp(tm, m.UniqueId(), 60); err != nil {
		t.Fatalf("DeductMp: %v", err)
	}

	var amounts []uint32
	var afters []uint32
	tk := &MonsterRecoveryTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			return information.NewModelBuilder().SetMpRecovery(10).Build(), nil
		},
		applyFn: r.ApplyRecovery,
		emitFn:  func(_ tenant.Model, _ Model) error { return nil },
		mpEmitFn: func(_ tenant.Model, post Model, amount uint32) error {
			amounts = append(amounts, amount)
			afters = append(afters, post.Mp())
			return nil
		},
	}
	tk.Run()

	if len(amounts) != 1 {
		t.Fatalf("expected exactly 1 MP_CHANGED emit, got %d", len(amounts))
	}
	if amounts[0] != 10 {
		t.Errorf("amount = %d, want 10 (applied regen)", amounts[0])
	}
	if afters[0] != 50 {
		t.Errorf("post MP = %d, want 50 (40+10)", afters[0])
	}
}

func TestRecovery_MpNotApplied_NoMpChangedEmit(t *testing.T) {
	r := GetMonsterRegistry()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, tm, f, 100100, 0, 0, 0, 5, 0, 3000, 100)
	if _, err := r.DeductMp(tm, m.UniqueId(), 60); err != nil {
		t.Fatalf("DeductMp: %v", err)
	}

	called := false
	tk := &MonsterRecoveryTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			// HP-only recovery: mpRecovery=0 => real ApplyRecovery returns
			// mpApplied=false.
			return information.NewModelBuilder().SetHpRecovery(10).Build(), nil
		},
		applyFn:  r.ApplyRecovery,
		emitFn:   func(_ tenant.Model, _ Model) error { return nil },
		mpEmitFn: func(_ tenant.Model, _ Model, _ uint32) error { called = true; return nil },
	}
	tk.Run()

	if called {
		t.Fatalf("mpApplied=false must not emit MP_CHANGED")
	}
}
```

(`information.NewModelBuilder().SetMpRecovery/SetHpRecovery/SetAttacks` all exist — `monster/information/builder.go:25-40`. If the exact deduct amount interacts with the HP damage-idle window differently than expected, mirror the first test in this file, which already proves the `DeductMp`-then-`ApplyRecovery` regen path.)

- [ ] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-monsters/atlas.com/monsters`): `go test ./monster/ -run TestRecovery_Mp -v`
Expected: FAIL to compile — `mpEmitFn` field undefined.

- [ ] **Step 3: Implement the recovery changes**

In `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go`:

1. Add the seam type next to `recoveryEmitFn`:

```go
// recoveryMpEmitFn publishes the MP_CHANGED event for applied MP regen.
// Production wraps producer.ProviderImpl(...); tests intercept.
type recoveryMpEmitFn func(t tenant.Model, m Model, amount uint32) error
```

2. Add field `mpEmitFn recoveryMpEmitFn` to the `MonsterRecoveryTask` struct.

3. Wire it in `NewMonsterRecoveryTask` next to the existing `tk.emitFn` wiring:

```go
	tk.mpEmitFn = func(t tenant.Model, m Model, amount uint32) error {
		tctx := tenant.WithContext(tk.ctx, t)
		return producer.ProviderImpl(tk.l)(tctx)(EnvEventTopicMonsterStatus)(
			mpChangedStatusEventProvider(m, 0, 0, MpChangeReasonRecovery, amount),
		)
	}
```

4. In `Run()`, capture the currently-discarded `mpApplied` (line ~113) and emit after the existing `hpApplied` block:

```go
			updated, hpApplied, mpApplied, err := tk.applyFn(ten, m.UniqueId(), hpR, mpR, nowMs)
			if err != nil {
				tk.l.WithError(err).Debugf(
					"Recovery: apply failed for monster [%d]; skipping.", m.UniqueId())
				continue
			}
			if hpApplied {
				if err := tk.emitFn(ten, updated); err != nil {
					tk.l.WithError(err).Debugf(
						"Recovery: HP-bar emit failed for monster [%d].", updated.UniqueId())
				}
			}
			if mpApplied {
				// Best-effort applied amount from the pre/post snapshots;
				// the mirror consumer only reads MonsterMpAfter, which the
				// post model carries authoritatively.
				var amount uint32
				if updated.Mp() > m.Mp() {
					amount = updated.Mp() - m.Mp()
				}
				if err := tk.mpEmitFn(ten, updated, amount); err != nil {
					tk.l.WithError(err).Debugf(
						"Recovery: MP_CHANGED emit failed for monster [%d].", updated.UniqueId())
				}
			}
```

5. Update existing `MonsterRecoveryTask` literals in `recovery_task_test.go`: any literal whose `applyFn` can yield `mpApplied=true` must set `mpEmitFn: func(_ tenant.Model, _ Model, _ uint32) error { return nil },` (a nil `mpEmitFn` panics). This INCLUDES the first test in the file (~line 33), which wires the REAL `applyFn: r.ApplyRecovery` with `SetMpRecovery(5)` and a below-max monster — that returns `mpApplied=true`. Verify every literal in the file.

- [ ] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-monsters/atlas.com/monsters`): `go test -race ./monster/ -v`
Expected: PASS — new tests plus all pre-existing recovery-task tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go \
        services/atlas-monsters/atlas.com/monsters/monster/recovery_task_test.go
git commit -m "feat(task-120): emit MP_CHANGED when recovery tick applies MP regen"
```

---

### Task 7: Full verification gate

**Files:** none (verification only; fix-and-rebuild cycles amend the relevant task's files).

- [ ] **Step 1: Module gates**

```bash
cd services/atlas-channel/atlas.com/channel && go vet ./... && go test -race ./... && go build ./...
cd ../../../atlas-monsters/atlas.com/monsters && go vet ./... && go test -race ./... && go build ./...
```
Expected: all clean/PASS.

- [ ] **Step 2: Docker bake (mandatory — go build will NOT catch Dockerfile COPY gaps)**

From the worktree root:

```bash
docker buildx bake atlas-channel atlas-monsters
```
Expected: both images build successfully. No new shared lib was added, so no Dockerfile/go.work edits should be needed — if bake fails on a missing `COPY libs/...`, fix the root `Dockerfile` per CLAUDE.md.

- [ ] **Step 3: Redis key guard**

From the worktree root:

```bash
tools/redis-key-guard.sh
```
Expected: clean (this task adds no raw go-redis usage; the new caches are in-process).

- [ ] **Step 4: Acceptance sweep against the PRD**

Confirm each PRD acceptance criterion maps to a passing artifact:
- Zero REST on warm path → `TestResolveLiveMonster_WarmPath_ZeroRest` (counting fake, not inspection).
- Miss ⇒ one fallback ⇒ backfill ⇒ next move REST-free → `TestResolveLiveMonster_MissFallsBackOnceAndBackfills`.
- DESTROYED/KILLED evict + post-death move falls back with today's error behavior → `TestHandleStatusEventDestroyedAndKilled_RemoveMirrorEntry` + `TestResolveLiveMonster_FallbackError_Propagates`.
- Template cache positive/negative/TTL/env → `TestCache_*`.
- ackMp/useSkills/field-rejection/packet bytes unchanged → `TestComputeAckMp_*` (pre-existing, untouched) + `TestLiveEntryFromModel_MapsAllFields` + the unchanged goroutine bodies in `ForMonster` (diff review).
- Tenant isolation → `TestLiveMirror_TenantIsolationAndEviction` + `TestCache_TenantIsolationAndEviction`.
- Metrics exposed → `/api/metrics` mount (Task 1) + counters in `monster/metrics.go`, `monster/information/metrics.go`.

- [ ] **Step 5: Commit any verification fixes and stop**

Do NOT open a PR yet — per CLAUDE.md, run `superpowers:requesting-code-review` first (plan-adherence + backend-guidelines reviewers), then `superpowers:finishing-a-development-branch`.
