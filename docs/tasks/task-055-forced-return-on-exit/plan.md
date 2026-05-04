# Forced Return on Exit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move durable character location ownership from atlas-character to atlas-maps, introduce a single resolver that honors WZ `forcedReturn` on disconnect and channel-change, and retire three duplicate per-feature warp emits.

**Architecture:** atlas-maps gains a `character_locations` table, a `location.Processor` with `Resolve` / `Get` / `Set`, a REST GET endpoint, and consumers for `CHANGE_MAP` and a new `CHANGE_CHANNEL_REQUEST` topic. atlas-character drops the `map_id` and `instance` columns, deletes its `ChangeMap`/`ChangeChannel` paths, and queries atlas-maps for the values it still needs to keep on LOGIN/LOGOUT events as a backward-compat shim. atlas-channel pivots session bootstrap to atlas-maps and emits the new request topic from its channel-change handler. atlas-login pivots character-list to atlas-maps. atlas-transports drops its login-side transit-map warp; atlas-party-quests drops its disconnect-leave warp; atlas-maps timer drops its `CHANGE_MAP` emit.

**Tech Stack:** Go 1.22+, GORM, PostgreSQL, Redis (existing dep), Kafka via `libs/atlas-kafka`, JSON:API via `api2go/jsonapi`, OTel spans.

---

## How to consume this plan

- Phases are deploy-ordered. Phase 1 lands first (atlas-maps additive), Phases 5+ require atlas-maps in place.
- Within a phase, complete tasks top-to-bottom. Each task is TDD-shaped: failing test → minimal impl → passing test → commit.
- Verification commands are per-task. The phase-level gate (build + tests + smoke) lives at the end of each phase.
- All file paths are absolute or rooted at the worktree (`<home>/source/atlas-ms/atlas/.worktrees/task-055-forced-return-on-exit/`).
- Always read the file before editing it (project rule).
- Per `CLAUDE.md`: **never commit directly to `main`**. Commit on `task-055-forced-return-on-exit`.

---

## Phase 0 — Sentinel ergonomics (libs/atlas-constants)

### Task 0.1: Add `IsSentinel()` to `_map.Id`

**Files:**
- Modify: `libs/atlas-constants/map/model.go`
- Test: `libs/atlas-constants/map/model_test.go` (create if absent)

- [ ] **Step 1: Read the file**

Read `libs/atlas-constants/map/model.go` and `libs/atlas-constants/map/constants.go` to confirm `EmptyMapId = Id(999999999)` is at line 2267.

- [ ] **Step 2: Write failing test**

Create or append to `libs/atlas-constants/map/model_test.go`:

```go
package _map

import "testing"

func TestIdIsSentinel(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"sentinel", EmptyMapId, true},
		{"zero", Id(0), false},
		{"henesys", Id(100000000), false},
		{"kpq lobby", Id(103000890), false},
		{"one below sentinel", Id(999999998), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.id.IsSentinel(); got != c.want {
				t.Fatalf("IsSentinel() = %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run the test, expect failure**

Run: `cd libs/atlas-constants && go test ./map/... -run TestIdIsSentinel`
Expected: FAIL — `id.IsSentinel undefined`.

- [ ] **Step 4: Implement**

Append to `libs/atlas-constants/map/model.go` (after the existing `Model` block):

```go
// IsSentinel reports whether this map id is the WZ "no override" sentinel
// (999999999). Used by the forced-return resolver to decide whether to
// relocate the character on exit.
func (id Id) IsSentinel() bool {
	return id == EmptyMapId
}
```

- [ ] **Step 5: Run the test, expect pass**

Run: `cd libs/atlas-constants && go test ./map/... -run TestIdIsSentinel -v`
Expected: PASS for all five cases.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/map/model.go libs/atlas-constants/map/model_test.go
git commit -m "feat(constants): add Id.IsSentinel() for forced-return resolver"
```

---

## Phase 1 — atlas-maps location subsystem (additive)

This phase is **independently deployable**. Nothing else depends on it being live yet. After this phase, atlas-maps has a new table populated by mirroring the existing LOGIN / LOGOUT / MAP_CHANGED / CHANNEL_CHANGED events. The `Resolve` rule exists but is not yet invoked.

### Task 1.1: Define the `location` GORM entity

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/location/entity.go`

- [ ] **Step 1: Confirm placement**

Read `services/atlas-maps/atlas.com/maps/character/` to see the existing pattern (visit-style processor uses `processor.go`, `requests.go`, `rest.go` directly under `character/`). The new `location/` subpackage will mirror that pattern.

- [ ] **Step 2: Create the entity file**

Write `services/atlas-maps/atlas.com/maps/character/location/entity.go`:

```go
package location

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID  `gorm:"type:uuid;primaryKey;not null"`
	CharacterId uint32     `gorm:"primaryKey;not null"`
	WorldId     world.Id   `gorm:"not null"`
	ChannelId   channel.Id `gorm:"not null"`
	MapId       _map.Id    `gorm:"not null"`
	Instance    uuid.UUID  `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000000'"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (e entity) TableName() string {
	return "character_locations"
}
```

- [ ] **Step 3: Build to confirm it compiles**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./character/location/...`
Expected: build succeeds (no consumers yet).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/entity.go
git commit -m "feat(atlas-maps): define character_locations entity"
```

### Task 1.2: Define the immutable `Model` and Builder

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/location/model.go`

- [ ] **Step 1: Write the model**

Write `services/atlas-maps/atlas.com/maps/character/location/model.go`:

```go
package location

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
}

func (m Model) CharacterId() uint32   { return m.characterId }
func (m Model) WorldId() world.Id     { return m.worldId }
func (m Model) ChannelId() channel.Id { return m.channelId }
func (m Model) MapId() _map.Id        { return m.mapId }
func (m Model) Instance() uuid.UUID   { return m.instance }

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
}

type Builder struct{ m Model }

func NewBuilder(characterId uint32) *Builder {
	return &Builder{m: Model{characterId: characterId}}
}

func (b *Builder) SetWorldId(v world.Id) *Builder     { b.m.worldId = v; return b }
func (b *Builder) SetChannelId(v channel.Id) *Builder { b.m.channelId = v; return b }
func (b *Builder) SetMapId(v _map.Id) *Builder        { b.m.mapId = v; return b }
func (b *Builder) SetInstance(v uuid.UUID) *Builder   { b.m.instance = v; return b }
func (b *Builder) SetField(f field.Model) *Builder {
	b.m.worldId = f.WorldId()
	b.m.channelId = f.ChannelId()
	b.m.mapId = f.MapId()
	b.m.instance = f.Instance()
	return b
}
func (b *Builder) Build() Model { return b.m }
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./character/location/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/model.go
git commit -m "feat(atlas-maps): define location.Model and Builder"
```

### Task 1.3: Resolver — failing test

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/location/resolution.go` (placeholder, just so the package compiles)
- Create: `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`

- [ ] **Step 1: Add a `ResolutionReason` type stub**

Write `services/atlas-maps/atlas.com/maps/character/location/resolution.go`:

```go
package location

type ResolutionReason string

const (
	ReasonForcedReturn ResolutionReason = "forced_return"
	ReasonStayPut      ResolutionReason = "stay_put"
)
```

- [ ] **Step 2: Write the failing resolver test**

Read existing test patterns in `services/atlas-maps/atlas.com/maps/data/map/info/model_test.go` and `services/atlas-maps/atlas.com/maps/character/processor_test.go` for tenant-context conventions.

Write `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`:

```go
package location

import (
	"context"
	"errors"
	"testing"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// stubInfoProcessor lets us inject map data without atlas-data round-trips.
type stubInfoProcessor struct {
	out info.Model
	err error
}

func (s *stubInfoProcessor) GetById(_ _map.Id) (info.Model, error) {
	return s.out, s.err
}

func newCtxTenant(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func TestResolveForcedReturn(t *testing.T) {
	ctx := newCtxTenant(t)
	cur := field.NewBuilder(0, 0, _map.Id(103000800)).SetInstance(uuid.New()).Build()
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.Id(103000890)).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s, want %s", reason, ReasonForcedReturn)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance = %s, want Nil (relocation drops instance)", got.Instance())
	}
}

func TestResolveStayPut(t *testing.T) {
	ctx := newCtxTenant(t)
	inst := uuid.New()
	cur := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(inst).Build()
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.EmptyMapId).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason = %s, want %s", reason, ReasonStayPut)
	}
	if got.MapId() != _map.Id(100020000) {
		t.Fatalf("MapId = %d, want 100020000", got.MapId())
	}
	if got.Instance() != inst {
		t.Fatalf("Instance = %s, want %s (stay put preserves instance)", got.Instance(), inst)
	}
}

func TestResolveInfoError(t *testing.T) {
	ctx := newCtxTenant(t)
	inst := uuid.New()
	cur := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(inst).Build()
	stub := &stubInfoProcessor{err: errors.New("boom")}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve must not error on info failure (degrades to stay put): %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason on info-error = %s, want stay_put", reason)
	}
	if got.MapId() != cur.MapId() || got.Instance() != cur.Instance() {
		t.Fatalf("on info-error must return current field unchanged")
	}
}
```

This test requires:
- `info.NewBuilder().SetForcedReturnMapId(...).Build()` — a builder on `info.Model`. Currently the `info` package has no public builder; **add one in step 4 of this task**.
- `newProcessorWithInfo(...)` — a package-private constructor that injects an `info.Processor` for testability.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/...`
Expected: FAIL — undefined `info.NewBuilder` and `newProcessorWithInfo`.

- [ ] **Step 4: Add `info.Builder`**

Read `services/atlas-maps/atlas.com/maps/data/map/info/model.go`. Append:

```go
type Builder struct{ m Model }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetId(v Id) *Builder                    { b.m.id = v; return b }
func (b *Builder) SetTimeLimit(v int32) *Builder          { b.m.timeLimit = v; return b }
func (b *Builder) SetForcedReturnMapId(v Id) *Builder     { b.m.forcedReturnMapId = v; return b }
func (b *Builder) Build() Model                           { return b.m }
```

(`Id` is the package-local alias for `_map.Id` already imported by this file. If not, change the type to `_map.Id` and add the import.)

Run: `cd services/atlas-maps/atlas.com/maps && go build ./data/map/info/...`
Expected: build succeeds.

- [ ] **Step 5: Stub the processor**

Write `services/atlas-maps/atlas.com/maps/character/location/processor.go`:

```go
package location

import (
	"context"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Set(characterId uint32, f field.Model) (Model, error)
	Resolve(currentField field.Model) (field.Model, ResolutionReason, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	ip  info.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return newProcessorWithInfo(l, ctx, db, info.NewProcessor(l, ctx))
}

func newProcessorWithInfo(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, ip info.Processor) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, ip: ip}
}

func (p *ProcessorImpl) Resolve(cur field.Model) (field.Model, ResolutionReason, error) {
	md, err := p.ip.GetById(cur.MapId())
	if err != nil {
		p.l.WithError(err).Warnf("location.Resolve: map info unavailable for [%d]; staying put.", cur.MapId())
		return cur, ReasonStayPut, nil
	}
	if md.ForcedReturnMapId().IsSentinel() {
		return cur, ReasonStayPut, nil
	}
	resolved := field.NewBuilder(cur.WorldId(), cur.ChannelId(), md.ForcedReturnMapId()).SetInstance(uuid.Nil).Build()
	return resolved, ReasonForcedReturn, nil
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	panic("not yet implemented")
}

func (p *ProcessorImpl) Set(characterId uint32, f field.Model) (Model, error) {
	panic("not yet implemented")
}
```

- [ ] **Step 6: Run test, expect pass on resolver cases**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/... -run "TestResolve(ForcedReturn|StayPut|InfoError)" -v`
Expected: PASS for all three resolver tests.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/processor.go \
        services/atlas-maps/atlas.com/maps/character/location/resolution.go \
        services/atlas-maps/atlas.com/maps/character/location/processor_test.go \
        services/atlas-maps/atlas.com/maps/data/map/info/model.go
git commit -m "feat(atlas-maps): add location.Resolve forced-return decision"
```

### Task 1.4: `Set` / `GetById` against the DB

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/character/location/processor.go`
- Test: `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`

- [ ] **Step 1: Read existing GORM patterns**

Read `services/atlas-maps/atlas.com/maps/visit/processor.go` (or any peer that does `db.Save` + tenant filter) to confirm the project pattern for tenant-scoped reads/writes.

- [ ] **Step 2: Write failing test for `Set` then `GetById`**

Append to `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`:

```go
import (
	// add to the existing import block
	"github.com/glebarez/sqlite"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	return db
}

func TestSetThenGetById(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	p := NewProcessor(logrus.New(), ctx, db)

	f := field.NewBuilder(0, 1, _map.Id(103000890)).SetInstance(uuid.Nil).Build()
	if _, err := p.Set(uint32(42), f); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(42))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.ChannelId() != 1 {
		t.Fatalf("ChannelId = %d, want 1", got.ChannelId())
	}
}

func TestGetByIdMissing(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.GetById(uint32(999)); err == nil {
		t.Fatal("GetById on missing row should error (record not found)")
	}
}

func TestSetIsTenantScoped(t *testing.T) {
	db := newTestDB(t)
	tnA, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tnB, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), tnA)
	ctxB := tenant.WithContext(context.Background(), tnB)

	pA := NewProcessor(logrus.New(), ctxA, db)
	pB := NewProcessor(logrus.New(), ctxB, db)

	f := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(uuid.Nil).Build()
	if _, err := pA.Set(uint32(7), f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := pB.GetById(uint32(7)); err == nil {
		t.Fatal("tenant B must not see tenant A's row")
	}
}
```

If `glebarez/sqlite` is not yet in `go.mod`, add it: `cd services/atlas-maps/atlas.com/maps && go get github.com/glebarez/sqlite && go mod tidy`. (Several other Atlas services already use this for in-memory test DBs — check `services/atlas-maps` first; if a different sqlite driver is already vendored, use that.)

- [ ] **Step 3: Run, expect failure**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/... -run "TestSet|TestGetById" -v`
Expected: FAIL — `Set`/`GetById` panic with "not yet implemented".

- [ ] **Step 4: Implement `Set` and `GetById`**

Replace the panicking stubs in `services/atlas-maps/atlas.com/maps/character/location/processor.go`:

```go
func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	var e entity
	if err := p.db.WithContext(p.ctx).
		Where("tenant_id = ? AND character_id = ?", t.Id(), characterId).
		First(&e).Error; err != nil {
		return Model{}, err
	}
	return NewBuilder(e.CharacterId).
		SetWorldId(e.WorldId).
		SetChannelId(e.ChannelId).
		SetMapId(e.MapId).
		SetInstance(e.Instance).
		Build(), nil
}

func (p *ProcessorImpl) Set(characterId uint32, f field.Model) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	e := entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		UpdatedAt:   time.Now(),
	}
	if err := p.db.WithContext(p.ctx).Save(&e).Error; err != nil {
		return Model{}, err
	}
	return NewBuilder(e.CharacterId).SetField(f).Build(), nil
}
```

Add the imports `"time"` and `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` if not already present.

- [ ] **Step 5: Run, expect pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/... -v`
Expected: PASS for resolve, set, get, missing, tenant-scoped.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/processor.go \
        services/atlas-maps/atlas.com/maps/character/location/processor_test.go \
        services/atlas-maps/atlas.com/maps/go.mod services/atlas-maps/atlas.com/maps/go.sum
git commit -m "feat(atlas-maps): location processor Set/GetById with tenant scope"
```

### Task 1.5: REST resource (`GET /characters/{id}/location`)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/location/rest.go`
- Create: `services/atlas-maps/atlas.com/maps/character/location/resource.go`
- Modify: wherever atlas-maps registers REST routes (likely `services/atlas-maps/atlas.com/maps/main.go` or a `routes.go`)

- [ ] **Step 1: Read existing REST handler patterns**

Read `services/atlas-maps/atlas.com/maps/character/rest.go` and any `RegisterHandler(l)(si)` usages in `main.go` / route registration. Mirror the convention.

- [ ] **Step 2: Create the JSON:API REST model**

Write `services/atlas-maps/atlas.com/maps/character/location/rest.go`:

```go
package location

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string             { return "character-locations" }
func (r RestModel) GetID() string               { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}
func (r *RestModel) SetToOneReferenceID(_, _ string) error    { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.CharacterId(),
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Instance:  m.Instance(),
	}, nil
}
```

- [ ] **Step 3: Create the HTTP handler**

Write `services/atlas-maps/atlas.com/maps/character/location/resource.go`. Pattern-match the existing `services/atlas-maps/atlas.com/maps/character/rest.go` resource (or whichever GET-by-id handler atlas-maps already exposes). Pseudocode:

```go
package location

import (
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si server.HandlerDependency) func(db *gorm.DB) func(r *mux.Router) {
	return func(db *gorm.DB) func(r *mux.Router) {
		return func(r *mux.Router) {
			r.HandleFunc("/characters/{characterId}/location", server.RegisterHandler(si.Logger())(si)("get_character_location", handleGetLocation(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetLocation(db *gorm.DB) server.Handler {
	return func(d *server.HandlerDependency, c *server.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			cid64, err := strconv.ParseUint(vars["characterId"], 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			cid := uint32(cid64)
			m, err := NewProcessor(d.Logger(), c.Context(), db).GetById(cid)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Unable to load location for character [%d].", cid)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			rm, _ := Transform(m)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(rm)
		}
	}
}
```

The exact `server.HandlerDependency` / `server.RegisterHandler` shape **must match what other atlas-maps endpoints already use**. Read `services/atlas-maps/atlas.com/maps/main.go` and an existing resource like the visit one to confirm. Adjust the snippet above to match.

- [ ] **Step 4: Wire the route**

Modify `services/atlas-maps/atlas.com/maps/main.go` (or wherever existing routes register, e.g. `routes.go`). Add:

```go
location.InitResource(si)(db)(router)
```

next to whichever existing `InitResource` calls register character / map endpoints. Import path: `"atlas-maps/character/location"`.

- [ ] **Step 5: Build the service**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: build succeeds.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/rest.go \
        services/atlas-maps/atlas.com/maps/character/location/resource.go \
        services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): expose GET /characters/{id}/location"
```

### Task 1.6: Register the migration

**Files:**
- Modify: wherever atlas-maps invokes `AutoMigrate` for existing entities (search for `Migration(db)` or `db.AutoMigrate` calls in `services/atlas-maps`).

- [ ] **Step 1: Locate the migration site**

Run: `grep -rn "Migration(db)\|AutoMigrate" services/atlas-maps/atlas.com/maps/main.go services/atlas-maps/atlas.com/maps/database/ 2>/dev/null | head -10`

Identify the file where existing entity migrations are wired (probably `main.go` or a `database/init.go`).

- [ ] **Step 2: Add `location.Migration(db)`**

Append a call mirroring an existing entity's migration registration. Example shape:

```go
if err := location.Migration(db); err != nil {
	return err
}
```

Add the import `"atlas-maps/character/location"` if not already present.

- [ ] **Step 3: Build**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/main.go  # or wherever migrations register
git commit -m "feat(atlas-maps): auto-migrate character_locations table"
```

### Task 1.7: Mirror LOGIN/LOGOUT/MAP_CHANGED/CHANNEL_CHANGED into `character_locations`

> Goal: keep the new table in sync with current event flow without changing any other service yet. After this task, `character_locations` shadows the truth atlas-character holds in its `characters.map_id` column. Phase 2 swaps the dependency.

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`

- [ ] **Step 1: Read current handlers**

Re-read `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` lines 55-118 to confirm the current shapes of LOGIN, LOGOUT, MAP_CHANGED, CHANNEL_CHANGED handlers.

- [ ] **Step 2: Add `location.Set` calls**

In each handler, after the existing logic, call `location.NewProcessor(l, ctx, db).Set(event.CharacterId, f)`. For LOGIN, MAP_CHANGED, CHANNEL_CHANGED use the *new* field (where the character is now). For LOGOUT use the *current* field as it appears in the event (Phase 2 will swap this for the resolver-resolved field).

Example diff for `handleStatusEventLoginFunc`:

```go
func handleStatusEventLoginFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLoginBody]) {
		if event.Type == characterKafka.EventCharacterStatusTypeLogin {
			l.Debugf("Character [%d] has logged in. ...", ...)
			transactionId := uuid.New()
			f := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).SetInstance(event.Body.Instance).Build()
			p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
			_ = p.EnterAndEmit(transactionId, f, event.CharacterId)
			if _, err := location.NewProcessor(l, ctx, db).Set(event.CharacterId, f); err != nil {
				l.WithError(err).Warnf("location.Set on LOGIN failed for character [%d].", event.CharacterId)
			}
		}
	}
}
```

Apply analogous additions to:
- `handleStatusEventLogoutFunc` — `Set(event.CharacterId, f)` where `f` is built from the event body (not yet resolved).
- `handleStatusEventMapChangedFunc` — `Set(event.CharacterId, newField)` (the field built from `TargetMapId` / `TargetInstance`).
- `handleStatusEventChannelChangedFunc` — `Set(event.CharacterId, newField)` (post-channel-change field).

Add the import `"atlas-maps/character/location"`.

- [ ] **Step 3: Build**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS.

- [ ] **Step 4: Run consumer-package tests**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/character/... ./character/location/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go
git commit -m "feat(atlas-maps): mirror character status events into character_locations"
```

### Task 1.8: Phase 1 verification gate

- [ ] **Step 1: Build atlas-maps**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS.

- [ ] **Step 2: Run all atlas-maps tests**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./...`
Expected: PASS.

- [ ] **Step 3: Build the service Docker image**

Run: `cd services/atlas-maps && docker compose build` (if a `docker-compose.yml` exists) or `cd services/atlas-maps && docker build .` if a Dockerfile is at the service root.
Expected: image builds.

- [ ] **Step 4: Stop here for review**

This is a natural review checkpoint. Phase 2 changes runtime behavior (Resolve fires on LOGOUT). Get a checkpoint review before proceeding.

---

## Phase 2 — Resolver fires on LOGOUT (atlas-maps)

After this phase, disconnects on a forced-return map result in the `character_locations` row pointing to the WZ target. atlas-character still owns its own `characters.map_id` column (unchanged). The two go out of sync on disconnect for forced-return maps — but atlas-character is the source of truth for client login *only* until Phase 5. The system continues to log players in at `characters.map_id` until then; this is intentional staging.

### Task 2.1: Wire `Resolve` into the LOGOUT handler

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` (`handleStatusEventLogoutFunc`)
- Test: integration test or augment unit tests

- [ ] **Step 1: Update the LOGOUT handler**

Replace the LOGOUT handler body added in Task 1.7's mirror. After resolving, write the resolved field to `character_locations`:

```go
func handleStatusEventLogoutFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
		if event.Type != characterKafka.EventCharacterStatusTypeLogout {
			return
		}
		l.Debugf("Character [%d] has logged out. ...", ...)
		transactionId := uuid.New()
		current := field.NewBuilder(event.WorldId, event.Body.ChannelId, event.Body.MapId).SetInstance(event.Body.Instance).Build()

		lp := location.NewProcessor(l, ctx, db)
		resolved, reason, err := lp.Resolve(current)
		if err != nil {
			l.WithError(err).Warnf("location.Resolve on LOGOUT failed for [%d]; staying put.", event.CharacterId)
			resolved = current
			reason = location.ReasonStayPut
		}
		if reason != location.ReasonStayPut {
			l.WithFields(logrus.Fields{
				"character_id":     event.CharacterId,
				"current_map_id":   current.MapId(),
				"resolved_map_id":  resolved.MapId(),
				"resolution_reason": string(reason),
			}).Info("forced-return resolution on LOGOUT")
		}
		if _, err := lp.Set(event.CharacterId, resolved); err != nil {
			l.WithError(err).Warnf("location.Set on LOGOUT failed for character [%d].", event.CharacterId)
		}

		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
		_ = p.ExitAndEmit(transactionId, current, event.CharacterId)
	}
}
```

Note we still pass `current` to `ExitAndEmit` — domain bookkeeping in the existing exit consumers (timer cancel etc.) needs the *original* map to function. Resolution is a write-side concern.

- [ ] **Step 2: Build**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS.

- [ ] **Step 3: Add a unit test for the LOGOUT handler's resolution path**

If consumer-handler-level tests don't exist yet for this file, the resolver itself is already covered. Add a focused integration-style test under `character/location/` instead:

Append to `services/atlas-maps/atlas.com/maps/character/location/processor_test.go`:

```go
func TestResolveAndSetForcedReturnPersists(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.Id(103000890)).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	cur := field.NewBuilder(0, 0, _map.Id(103000800)).SetInstance(uuid.New()).Build()
	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatal(err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s", reason)
	}
	if _, err := p.Set(uint32(7), resolved); err != nil {
		t.Fatal(err)
	}

	got, err := p.GetById(uint32(7))
	if err != nil {
		t.Fatal(err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance must be Nil after relocation")
	}
}
```

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/... -run TestResolveAndSetForcedReturnPersists -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go \
        services/atlas-maps/atlas.com/maps/character/location/processor_test.go
git commit -m "feat(atlas-maps): apply forced-return resolver on LOGOUT"
```

### Task 2.2: Drop `CHANGE_MAP` emit from `timer.ForceReturnIfTracked`

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/processor.go`

- [ ] **Step 1: Read & confirm**

Read `services/atlas-maps/atlas.com/maps/map/timer/processor.go` lines 140-170. The emit lives on line 159 (`p.emitChangeMap(entry)`).

- [ ] **Step 2: Remove the emit and supporting code**

Replace `ForceReturnIfTracked` body (keep span, keep timer-stop, keep log line):

```go
func (p *ProcessorImpl) ForceReturnIfTracked(characterId uint32) bool {
	entry, ok := p.r.ClaimAny(p.t, characterId)
	if !ok {
		return false
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Disconnect")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(entry.Field().WorldId())),
		attribute.Int("map.id", int(entry.Field().MapId())),
	)
	defer span.End()
	if entry.Timer() != nil {
		entry.Timer().Stop()
	}
	p.l.Warnf("MapTimer.Disconnect: tenant=[%s] character=[%d] map=[%d] (forced-return persistence handled by location.Resolve).",
		p.t.Id(), characterId, entry.Field().MapId())
	return true
}
```

Delete `emitChangeMap` and the `changeMapProvider` it called (search for `changeMapProvider` in this file's `producer.go` — if it's the only caller, delete the provider too).

- [ ] **Step 3: Update existing tests**

Run `grep -n "emitChangeMap\|changeMapProvider\|ForceReturnIfTracked" services/atlas-maps/atlas.com/maps/map/timer/`. Adjust any test that asserts on the emitted command — change to assert no emit, or just drop the assertion.

- [ ] **Step 4: Build & test**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./map/timer/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/timer/
git commit -m "refactor(atlas-maps): retire timer CHANGE_MAP emit (location resolver covers it)"
```

### Task 2.3: Phase 2 verification gate

- [ ] Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...` — Expect PASS.
- [ ] If practical, run a smoke scenario: deploy atlas-maps in dev, log a character into KPQ map (`103000800`), disconnect, query `GET /characters/{id}/location` — expect `mapId: 103000890`. (Skip if dev infra is unavailable; cover instead in Phase 11 integration tests.)

---

## Phase 3 — `CHANGE_CHANNEL_REQUEST` topic

> A new Kafka topic. atlas-channel will produce; atlas-maps will consume. Phase 6 wires the producer; this phase wires the consumer side and the topic constant.

### Task 3.1: Define the topic + message shape

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/character/channel_change.go` (or append to the existing character message file)

- [ ] **Step 1: Read existing topic definitions**

Read `services/atlas-channel/atlas.com/channel/kafka/message/character/` to find where character-related command topics live in atlas-channel (or whether atlas-channel imports atlas-character's). atlas-character's `kafka/message/character/kafka.go:13` defines `EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"`. The new topic is distinct (it carries channel-change *requests*, not arbitrary character commands), so define it new.

- [ ] **Step 2: Add the constant + body**

If the file exists, append. Otherwise create `services/atlas-channel/atlas.com/channel/kafka/message/character/channel_change.go`:

```go
package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/google/uuid"
)

const (
	EnvCommandTopicChannelChangeRequest = "COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST"

	CommandChannelChangeRequest = "CHANNEL_CHANGE_REQUEST"
)

type ChannelChangeRequestCommand struct {
	TransactionId   uuid.UUID  `json:"transactionId"`
	CharacterId     uint32     `json:"characterId"`
	WorldId         byte       `json:"worldId"`
	OldChannelId    channel.Id `json:"oldChannelId"`
	TargetChannelId channel.Id `json:"targetChannelId"`
}
```

(`world.Id` is `byte`. Use `byte` directly here to avoid the import; pattern-match adjacent files.)

- [ ] **Step 3: Build atlas-channel**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./kafka/message/character/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/character/channel_change.go
git commit -m "feat(atlas-channel): define CHANGE_CHANNEL_REQUEST kafka topic"
```

### Task 3.2: Add the topic env var to atlas-maps' env loader / docker-compose

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/main.go` or wherever the topic envs are surfaced.
- Modify: `services/atlas-maps/docker-compose.yml` (if topic env vars are listed there).

- [ ] **Step 1: Inspect how other topic envs flow**

Run: `grep -rn "EnvEventTopicCharacterStatus\|EnvCommandTopic\b" services/atlas-maps/atlas.com/maps/main.go services/atlas-maps/docker-compose.yml services/atlas-maps/.env* 2>/dev/null | head -20`

The pattern in atlas-maps is `topic.EnvProvider(l)(envName)()` to resolve topic name from env var. No code wiring is needed in `main.go` for topic env vars themselves — they're read at consumer-init time.

- [ ] **Step 2: Add the env var to docker-compose / .env**

If `services/atlas-maps/docker-compose.yml` lists topic env vars, add:

```yaml
- COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST=character_channel_change_request
```

(Naming aligned with existing topic naming conventions in the repo. Inspect `docker-compose.yml` for the precise topic-name format.)

If a `.env` template is used (`services/atlas-maps/.env.example` etc.), add a matching line.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-maps/docker-compose.yml services/atlas-maps/.env.example
git commit -m "chore(atlas-maps): expose CHANGE_CHANNEL_REQUEST topic env var"
```

### Task 3.3: atlas-maps consumer for `CHANGE_CHANNEL_REQUEST`

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/channel_change_request.go`
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` (register the new consumer + handler)

- [ ] **Step 1: Write the consumer body**

Create `services/atlas-maps/atlas.com/maps/kafka/consumer/character/channel_change_request.go`:

```go
package character

import (
	"context"

	"atlas-maps/character/location"
	channelChMsg "atlas-channel/kafka/message/character" // import the constant + struct from atlas-channel
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/kafka/producer"
	_map "atlas-maps/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func handleChannelChangeRequestFunc(db *gorm.DB) message.Handler[channelChMsg.ChannelChangeRequestCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c channelChMsg.ChannelChangeRequestCommand) {
		l.Debugf("CHANGE_CHANNEL_REQUEST received for character [%d] target channel [%d].", c.CharacterId, c.TargetChannelId)
		lp := location.NewProcessor(l, ctx, db)
		cur, err := lp.GetById(c.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("CHANGE_CHANNEL_REQUEST: cannot load location for [%d]; ignoring.", c.CharacterId)
			return
		}
		// Build a hypothetical post-handoff field on the target channel, then resolve.
		target := field.NewBuilder(cur.WorldId(), c.TargetChannelId, cur.MapId()).SetInstance(cur.Instance()).Build()
		resolved, reason, err := lp.Resolve(target)
		if err != nil {
			l.WithError(err).Warnf("CHANGE_CHANNEL_REQUEST: resolve failed for [%d]; staying put on target channel.", c.CharacterId)
			resolved = target
			reason = location.ReasonStayPut
		}
		if _, err := lp.Set(c.CharacterId, resolved); err != nil {
			l.WithError(err).Errorf("CHANGE_CHANNEL_REQUEST: location.Set failed for [%d].", c.CharacterId)
			return
		}
		if reason != location.ReasonStayPut {
			l.WithFields(logrus.Fields{
				"character_id":      c.CharacterId,
				"target_channel":    c.TargetChannelId,
				"resolved_map_id":   resolved.MapId(),
				"resolution_reason": string(reason),
			}).Info("forced-return resolution on CHANNEL_CHANGE_REQUEST")
		}

		oldField := cur.Field()
		newField := resolved
		// Emit CHANNEL_CHANGED on the existing character status topic so other
		// services react as today.
		_ = message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
			return buf.Put(characterKafka.EnvEventTopicCharacterStatus,
				channelChangedStatusProvider(uuid.New(), c.CharacterId, c.WorldId, c.OldChannelId, c.TargetChannelId, oldField, newField))
		})

		// Update map-side registries and Redis presence (existing helper).
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
		_ = p.TransitionChannelAndEmit(uuid.New(), newField, c.OldChannelId, c.CharacterId)
	}
}
```

The `channelChangedStatusProvider` referenced does not yet exist in atlas-maps' character producer file. atlas-character has a `changeChannelEventProvider` you can mirror. Read `services/atlas-character/atlas.com/character/character/producer.go` lines 79-95 for the wire shape, and write a matching provider in `services/atlas-maps/atlas.com/maps/kafka/producer/character.go` (or wherever atlas-maps' character-status producers live; check `services/atlas-maps/atlas.com/maps/kafka/message/character/` and `services/atlas-maps/atlas.com/maps/kafka/producer/`).

- [ ] **Step 2: Add the matching producer**

Read `services/atlas-character/atlas.com/character/character/producer.go:79-100` to see the full provider implementation and the body type `ChangeChannelEventLoginBody`. Read `services/atlas-maps/atlas.com/maps/kafka/message/character/` to confirm what types atlas-maps already uses for the same topic (atlas-maps currently *consumes* `StatusEvent[ChangeChannelEventLoginBody]` from atlas-character — so the body type already lives in atlas-maps' kafka/message/character package).

Add a producer in `services/atlas-maps/atlas.com/maps/kafka/producer/` (path-match an existing producer like `services/atlas-maps/atlas.com/maps/kafka/producer/character.go` if it exists, otherwise create `character.go`):

```go
package producer

import (
	characterKafka "atlas-maps/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ChannelChangedStatusProvider(transactionId uuid.UUID, characterId uint32, worldId byte, oldChannelId, newChannelId byte, oldField, newField field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    newChannelId,
			OldChannelId: oldChannelId,
			MapId:        newField.MapId(),
			Instance:     newField.Instance(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

The exact field names on `ChangeChannelEventLoginBody` must match what atlas-maps' message package already declares. Confirm by reading `services/atlas-maps/atlas.com/maps/kafka/message/character/`.

- [ ] **Step 3: Register the consumer + handler**

Modify `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`:

In `InitConsumers`, register the new topic:

```go
rf(consumer2.NewConfig(l)("channel_change_request")(channelChMsg.EnvCommandTopicChannelChangeRequest)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
```

In `InitHandlers`, register the handler:

```go
t, _ = topic.EnvProvider(l)(channelChMsg.EnvCommandTopicChannelChangeRequest)()
if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChannelChangeRequestFunc(db)))); err != nil {
	return err
}
```

Add the import. atlas-maps importing atlas-channel might create a module-level dependency cycle. Verify with `go build`. If a cycle exists, the cleanest fix is to move the topic constant + struct to a third location (e.g., `libs/atlas-kafka-messages/character/` if that exists, or a small new shared lib). **Document any new shared lib path here in this task.**

- [ ] **Step 4: Build atlas-maps**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS. If it fails on import cycle, follow the fallback in Step 3.

- [ ] **Step 5: Build atlas-channel**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/ \
        services/atlas-maps/atlas.com/maps/kafka/producer/
git commit -m "feat(atlas-maps): consume CHANGE_CHANNEL_REQUEST and emit CHANNEL_CHANGED"
```

### Task 3.4: Phase 3 verification gate

- [ ] Run `cd services/atlas-maps/atlas.com/maps && go test ./... && go build ./...`. Expect PASS.
- [ ] Run `cd services/atlas-channel/atlas.com/channel && go test ./... && go build ./...`. Expect PASS.

---

## Phase 4 — atlas-maps takes over `CHANGE_MAP` command + emits `MAP_CHANGED`

> Currently atlas-character consumes `COMMAND_TOPIC_CHARACTER` `CHANGE_MAP` and emits `MAP_CHANGED` status. We migrate that to atlas-maps. To avoid a window where both services emit MAP_CHANGED (different consumer groups), Phases 4 and 5 are bundled into the same deploy.

### Task 4.1: Add a `handleChangeMap` consumer in atlas-maps

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`
- (Possibly) Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/change_map.go`

- [ ] **Step 1: Read the existing atlas-character handler**

Re-read `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` lines 117-129 (the `handleChangeMap`) and the `ChangeMapBody` struct.

- [ ] **Step 2: Add the consumer registration in atlas-maps**

In `InitConsumers` (services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go), add a registration for the character command topic:

```go
rf(consumer2.NewConfig(l)("character_command")(characterKafka.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
```

The constant name in atlas-maps' message package may differ (it currently consumes status, not command). Add `EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"` to `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go` if it's not already there, plus the `ChangeMapBody` struct + `CommandChangeMap` constant — pattern-match what atlas-character defines. Or import directly from atlas-character if no cycle results.

- [ ] **Step 3: Implement the handler**

Add to atlas-maps:

```go
func handleChangeMapFunc(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c characterKafka.Command[characterKafka.ChangeMapBody]) {
		if c.Type != characterKafka.CommandChangeMap {
			return
		}
		newField := field.NewBuilder(c.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()

		lp := location.NewProcessor(l, ctx, db)
		old, err := lp.GetById(c.CharacterId)
		oldField := newField // fall back if no prior row
		if err == nil {
			oldField = old.Field()
		}
		if _, err := lp.Set(c.CharacterId, newField); err != nil {
			l.WithError(err).Errorf("CHANGE_MAP: location.Set failed for [%d].", c.CharacterId)
			return
		}

		// Emit MAP_CHANGED status event.
		_ = message.Emit(producer.ProviderImpl(l)(ctx))(func(buf *message.Buffer) error {
			return buf.Put(characterKafka.EnvEventTopicCharacterStatus,
				mapChangedStatusProvider(c.TransactionId, c.CharacterId, c.WorldId, oldField, newField, c.Body.PortalId))
		})

		// Existing TransitionMapAndEmit covers the in-process registry + timer hooks.
		p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)
		_ = p.TransitionMapAndEmit(c.TransactionId, newField, c.CharacterId, oldField)
	}
}
```

`mapChangedStatusProvider` is the producer mirror — implement in `services/atlas-maps/atlas.com/maps/kafka/producer/character.go` mirroring atlas-character's `mapChangedEventProvider` (read `services/atlas-character/atlas.com/character/character/producer.go` for the exact body shape).

- [ ] **Step 4: Register the handler**

In `InitHandlers`:

```go
t, _ = topic.EnvProvider(l)(characterKafka.EnvCommandTopic)()
if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeMapFunc(db)))); err != nil {
	return err
}
```

- [ ] **Step 5: Build**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/ \
        services/atlas-maps/atlas.com/maps/kafka/producer/ \
        services/atlas-maps/atlas.com/maps/kafka/message/character/
git commit -m "feat(atlas-maps): take over CHANGE_MAP command and emit MAP_CHANGED"
```

---

## Phase 5 — atlas-character drops map ownership

> This phase removes location code from atlas-character. Bundle deploy with Phase 4 to avoid double-emit overlap.

### Task 5.1: Remove `handleChangeMap` from atlas-character consumer

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go`

- [ ] **Step 1: Edit consumer.go**

Read the file. Delete:
- Line 38 (`handleChangeMap` registration in `InitHandlers`).
- Lines 117-129 (the `handleChangeMap` function).

- [ ] **Step 2: Build**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go
git commit -m "refactor(atlas-character): drop CHANGE_MAP consumer (atlas-maps now owns)"
```

### Task 5.2: Remove `ChangeChannel` call from session consumer

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/consumer/session/consumer.go`

- [ ] **Step 1: Edit**

Delete lines 85-88 (the `ChangeChannelAndEmit` call). Keep the registry/history bookkeeping above it. The CHANNEL_CHANGED emission moves to atlas-maps via the request topic.

- [ ] **Step 2: Build & test**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./kafka/consumer/session/...`
Expected: PASS. (Some tests might assert the call was made — adjust those tests to assert it is *not* made.)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/session/consumer.go
git commit -m "refactor(atlas-character): drop ChangeChannelAndEmit from session consumer"
```

### Task 5.3: Delete `ChangeMap`, `ChangeChannel`, supporting helpers from processor

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go`

- [ ] **Step 1: Read & confirm**

Read lines 70-90 (interface declaration) and lines 410-468 (implementations). Targets to remove:
- `ChangeChannelAndEmit` (line 410), `ChangeChannel` (line 416).
- `ChangeMapAndEmit` (line 426), `ChangeMap` (line 432).
- `positionAtPortal` (line 441) — only called by ChangeMap.
- `announceMapChangedWithBuffer` (line 452), `announceMapChanged` (line 461).

Also remove:
- The matching method declarations on the `Processor` interface (lines 75-78 and any siblings).

- [ ] **Step 2: Edit**

Use `Edit` with full surrounding context to remove each block. Run incrementally and verify build between deletions.

- [ ] **Step 3: Build**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: PASS. If any caller still references the deleted methods (mocks, tests), fix them in this same task.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-character/atlas.com/character && go test ./...`
Expected: PASS. Some tests may assert on the deleted methods — delete those test cases.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/processor.go \
        services/atlas-character/atlas.com/character/character/processor_test.go  # if touched
git commit -m "refactor(atlas-character): delete ChangeMap and ChangeChannel paths"
```

### Task 5.4: Drop `MapId`/`Instance` getters from `Model` and Builder setters

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/model.go`

- [ ] **Step 1: Read**

Read lines 99-105 (getters) and lines 401-410 (builder).

- [ ] **Step 2: Edit**

Remove `MapId()`, `Instance()` from `Model`. Remove `mapId`, `instance` private fields. Remove `SetMapId`, `SetInstance` from the builder. Remove any reference to those fields in `CloneModel`/`Clone()`.

- [ ] **Step 3: Build**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: PASS for the model package; a wave of compile errors elsewhere is expected because callers across the package still reference these methods. Fix them as part of Tasks 5.5 / 5.6 (Logout / Login pivot, REST pivot). Iterate: build, fix, build, fix.

- [ ] **Step 4: Note transient broken state**

Don't commit yet. Tasks 5.5–5.7 must complete before the build is green again.

### Task 5.5: Pivot `Login` and `Logout` to query atlas-maps for event payload

**Files:**
- Create: `services/atlas-character/atlas.com/character/location/requests.go` — atlas-maps client stub
- Modify: `services/atlas-character/atlas.com/character/character/processor.go`

- [ ] **Step 1: Write the atlas-maps client**

Read an existing atlas-maps REST client used by another service (or `libs/atlas-rest/requests` patterns). Create `services/atlas-character/atlas.com/character/location/requests.go`:

```go
package location

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "character-locations" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// GetField returns the durable field stored in atlas-maps for the given character.
// Caller must pass a logger and a context with tenant.
func GetField(l logrus.FieldLogger, ctx context.Context, characterId uint32) (field.Model, error) {
	url := atlasMapsURL(characterId)
	provider := requests.Provider[RestModel, field.Model](l, ctx)(
		requests.MakeGetRequest[RestModel](url),
		func(rm RestModel) (field.Model, error) {
			return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
		},
	)
	return provider()
}

func atlasMapsURL(characterId uint32) string {
	host := envHost("MAPS_SERVICE_HOST", "atlas-maps")
	port := envPort("MAPS_SERVICE_PORT", "8080")
	return fmt.Sprintf("http://%s:%s/characters/%d/location", host, port, characterId)
}
```

The exact `requests.Provider` and request-builder shape varies repo-wide. Read `services/atlas-character/atlas.com/character/skill/requests.go` (or equivalent existing atlas-character client) to confirm the form. Adjust the snippet above to compile.

The `envHost`/`envPort` helper conventions in atlas-character live somewhere central — re-use them. (If unsure, run: `grep -rn "envHost\|MAPS_SERVICE_HOST\|http.*atlas-maps" services/atlas-character/atlas.com/character/ | head -10`.)

- [ ] **Step 2: Modify `Login`**

Original (line 391):

```go
return mb.Put(character2.EnvEventTopicCharacterStatus,
    loginEventProvider(transactionId, c.Id(),
        field.NewBuilder(channel.WorldId(), channel.Id(), c.MapId()).SetInstance(c.Instance()).Build()))
```

Replacement:

```go
f, err := location.GetField(p.l, p.ctx, c.Id())
if err != nil {
    p.l.WithError(err).Warnf("Login: atlas-maps location lookup failed for [%d]; emitting with zero map.", c.Id())
    f = field.NewBuilder(channel.WorldId(), channel.Id(), 0).SetInstance(uuid.Nil).Build()
}
return mb.Put(character2.EnvEventTopicCharacterStatus,
    loginEventProvider(transactionId, c.Id(), f))
```

Apply analogous change to `Logout` at line 405.

Add the import `"atlas-character/location"`.

- [ ] **Step 3: Build & test**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./character/...`
Expected: PASS for character package. (Other packages may still error if they reference dropped methods — they're handled in Task 5.6.)

- [ ] **Step 4: Commit**

Hold the commit until Task 5.6 lands; tasks 5.4–5.6 land as a single commit unless intermediate states build cleanly.

### Task 5.6: Pivot `RestModel` Transform/Extract to inject location from atlas-maps

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/rest.go`
- Modify: `services/atlas-character/atlas.com/character/character/transformer.go` (or wherever `Transform` lives)

- [ ] **Step 1: Read**

Read both files. Today (`rest.go:40-41,105-106` and `transformer.go:143-144`):

```go
MapId    _map.Id
Instance uuid.UUID
```

The `RestModel` keeps the JSON fields (D11). The `Transform` function needs to populate them via an in-flight atlas-maps lookup.

- [ ] **Step 2: Modify `Transform`**

Pseudocode (adjust to match the actual `Transform` signature):

```go
func Transform(l logrus.FieldLogger, ctx context.Context) func(m Model) (RestModel, error) {
	return func(m Model) (RestModel, error) {
		f, err := location.GetField(l, ctx, m.Id())
		if err != nil {
			l.WithError(err).Warnf("Transform: atlas-maps location lookup failed for [%d]; using zero values.", m.Id())
			f = field.NewBuilder(0, 0, 0).SetInstance(uuid.Nil).Build()
		}
		return RestModel{
			// ...existing fields...
			MapId:    f.MapId(),
			Instance: f.Instance(),
		}, nil
	}
}
```

If the existing `Transform` is parameter-less, this requires changing every caller. **Confirm caller surface first** with `grep -rn "character.Transform\|\.Transform(" services/atlas-character/atlas.com/character/ | head -20`. Adjust the curry shape to match what's least intrusive.

If the caller surface is too wide, alternative: keep `Transform` parameter-less but consult a package-private context-bound lookup helper. The cleanest path is whatever minimizes touchpoints.

- [ ] **Step 3: Modify `Extract`**

`Extract` (REST → Model, used when atlas-character receives a character via REST) does the reverse. Today it calls `SetMapId(m.MapId).SetInstance(m.Instance)`. Since `Model` no longer has those setters, drop the calls.

- [ ] **Step 4: Build**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: PASS. If still erroring, hunt down the next caller and fix.

- [ ] **Step 5: Combined commit for 5.4 + 5.5 + 5.6**

```bash
git add services/atlas-character/atlas.com/character/character/ \
        services/atlas-character/atlas.com/character/location/
git commit -m "refactor(atlas-character): remove map ownership; query atlas-maps for shim"
```

### Task 5.7: Drop `map_id`, `instance` columns from the entity (migration)

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/entity.go`

- [ ] **Step 1: Edit**

Remove fields `MapId` and `Instance` (lines 41-42) and the `SetMapId`/`SetInstance` references in the `transformer.go` `Build()` chain (around line 143-144 — already addressed in 5.6 if you covered there).

- [ ] **Step 2: Add explicit drop to migration**

GORM's `AutoMigrate` does **not** drop columns. Add an explicit drop in the migration setup wherever atlas-character runs migrations:

```go
if db.Migrator().HasColumn(&entity{}, "MapId") {
    if err := db.Migrator().DropColumn(&entity{}, "MapId"); err != nil {
        return err
    }
}
if db.Migrator().HasColumn(&entity{}, "Instance") {
    if err := db.Migrator().DropColumn(&entity{}, "Instance"); err != nil {
        return err
    }
}
```

(Read existing atlas-character migration patterns first — `grep -n "Migrator()\|AutoMigrate" services/atlas-character/atlas.com/character/...`.)

- [ ] **Step 3: Add a backfill script**

Create `services/atlas-character/atlas.com/character/scripts/backfill_character_locations.sql`:

```sql
-- One-shot operator script. Runs after atlas-maps deploy with the new
-- character_locations table, before atlas-character drops map_id/instance.
-- Re-run is idempotent (PK collision = upsert via ON CONFLICT).
INSERT INTO character_locations
  (tenant_id, character_id, world_id, channel_id, map_id, instance, updated_at)
SELECT
  tenant_id,
  id,
  world,
  0,
  map_id,
  instance,
  NOW()
FROM characters
ON CONFLICT (tenant_id, character_id) DO UPDATE
  SET map_id = EXCLUDED.map_id,
      instance = EXCLUDED.instance,
      updated_at = NOW();
```

- [ ] **Step 4: Build**

Run: `cd services/atlas-character/atlas.com/character && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/entity.go \
        services/atlas-character/atlas.com/character/character/migration*.go \
        services/atlas-character/atlas.com/character/scripts/
git commit -m "refactor(atlas-character): drop map_id/instance columns + backfill script"
```

### Task 5.8: Phase 5 verification gate

- [ ] Run `cd services/atlas-character/atlas.com/character && go test ./...`. Expect PASS.
- [ ] Run `cd services/atlas-character/atlas.com/character && go build ./...`. Expect PASS.
- [ ] Re-run atlas-maps tests: `cd services/atlas-maps/atlas.com/maps && go test ./...`. Expect PASS.

---

## Phase 6 — atlas-channel pivot

### Task 6.1: Pivot session bootstrap to atlas-maps

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go` (function `processStateReturn`, line ~169)
- Create: `services/atlas-channel/atlas.com/channel/maps/location/requests.go` (atlas-maps client for atlas-channel — pattern-match Task 5.5)

- [ ] **Step 1: Create the client**

Mirror the client added in Task 5.5 (`services/atlas-character/atlas.com/character/location/requests.go`) under `services/atlas-channel/atlas.com/channel/maps/location/requests.go`. Identical body except the package name. (If a shared lib exists for cross-service REST clients, prefer that — but the project pattern is per-service requests files.)

- [ ] **Step 2: Pivot the bootstrap**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go` line 169:

Replace:

```go
s = sp.SetMapId(s.SessionId(), c.MapId())
```

With:

```go
f, lerr := location.GetField(l, ctx, c.Id())
if lerr != nil {
    l.WithError(lerr).Errorf("Session bootstrap: atlas-maps unreachable for [%d]; aborting.", c.Id())
    return sp.Destroy(s)  // D12 — fail closed
}
s = sp.SetMapId(s.SessionId(), f.MapId())
```

Add the import `"atlas-channel/maps/location"`.

If `Destroy` is the wrong escape hatch (e.g., the code wants to surface an error packet), follow the existing error path used elsewhere in this file.

- [ ] **Step 3: Build & test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/maps/location/ \
        services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go
git commit -m "refactor(atlas-channel): pivot session bootstrap to atlas-maps location"
```

### Task 6.2: Emit `CHANGE_CHANNEL_REQUEST` from `channel_change.go`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/channel_change.go`

- [ ] **Step 1: Read**

Re-read the existing handler. The HP > 0 gate stays. Today the code calls `as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelChange{...})`. That call stays — atlas-account still progresses session state for IP/port handoff. The new emit is *additional*.

- [ ] **Step 2: Add the producer**

Create `services/atlas-channel/atlas.com/channel/character/producer.go` (or extend an existing file in this directory) with a provider:

```go
func ChannelChangeRequestProvider(transactionId uuid.UUID, characterId uint32, worldId byte, oldChannelId, targetChannelId channel.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := characterMsg.ChannelChangeRequestCommand{
		TransactionId:   transactionId,
		CharacterId:     characterId,
		WorldId:         worldId,
		OldChannelId:    oldChannelId,
		TargetChannelId: targetChannelId,
	}
	return producer.SingleMessageProvider(key, value)
}
```

Pattern-match an adjacent producer file in atlas-channel for the exact shape of `producer.SingleMessageProvider` and `model.Provider`.

- [ ] **Step 3: Emit from the handler**

In `channel_change.go` after the HP gate, before/after `as.UpdateState`, emit:

```go
prod := producer2.ProviderImpl(l)(ctx)
if err := prod(characterMsg.EnvCommandTopicChannelChangeRequest)(ChannelChangeRequestProvider(uuid.New(), s.CharacterId(), byte(s.WorldId()), s.ChannelId(), p.ChannelId())); err != nil {
    l.WithError(err).Errorf("Failed to emit CHANGE_CHANNEL_REQUEST for [%d].", s.CharacterId())
}
```

(Adjust to actual session API — `s.ChannelId()` etc.)

- [ ] **Step 4: Build & test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/channel_change.go \
        services/atlas-channel/atlas.com/channel/character/producer.go
git commit -m "feat(atlas-channel): emit CHANGE_CHANNEL_REQUEST on channel-change packet"
```

### Task 6.3: Phase 6 verification gate

- [ ] Run `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`. Expect PASS.

---

## Phase 7 — atlas-login pivot

### Task 7.1: Replace `c.MapId()` in character_list.go with atlas-maps lookup

**Files:**
- Modify: `services/atlas-login/atlas.com/login/socket/writer/character_list.go`
- Create: `services/atlas-login/atlas.com/login/maps/location/requests.go` (mirror Task 5.5/6.1)

- [ ] **Step 1: Create the atlas-maps client**

Mirror Task 5.5 under atlas-login.

- [ ] **Step 2: Read character_list.go around line 41**

```go
uint32(c.MapId()), c.SpawnPoint(),
```

- [ ] **Step 3: Pivot**

```go
mapId := _map.Id(0)
f, err := location.GetField(l, ctx, c.Id())
if err != nil {
    l.WithError(err).Warnf("character_list: atlas-maps location unreachable for [%d]; rendering map=0.", c.Id())
} else {
    mapId = f.MapId()
}
// ... in the writer call:
uint32(mapId), c.SpawnPoint(),
```

The function signature may not pass `l` and `ctx` directly — check and thread them through if needed.

- [ ] **Step 4: Build & test**

Run: `cd services/atlas-login/atlas.com/login && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/socket/writer/character_list.go \
        services/atlas-login/atlas.com/login/maps/location/
git commit -m "refactor(atlas-login): pivot character list mapId to atlas-maps"
```

---

## Phase 8 — atlas-transports cleanup

### Task 8.1: Remove `HandleLogin` transit-map detection branch

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/processor.go`

- [ ] **Step 1: Read**

Re-read lines 283-299 (`HandleLogin` and the branch).

- [ ] **Step 2: Edit — replace function with a no-op**

```go
func (p *ProcessorImpl) HandleLogin(mb *message.Buffer) func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
	return func(characterId uint32, mapId _map.Id, worldId world.Id, channelId channel.Id) error {
		// Forced-return on disconnect (atlas-maps location.Resolve) ensures the
		// player is never persisted on a transit map. The crash-recovery branch
		// that used to re-warp from a transit map back to route.StartMapId is
		// no longer necessary.
		return nil
	}
}
```

If `HandleLoginAndEmit` is the only caller and now does nothing, consider removing both. Caller search:

```bash
grep -rn "HandleLogin\|HandleLoginAndEmit" services/atlas-transports/atlas.com/transports/ | head
```

If safely removable, delete and update callers. Otherwise leave the no-op for clarity.

Also: `route.StartMapId()` was only used by this branch and `WarpToRouteStartMapOnLogoutAndEmit` (which doesn't exist per design D3 footnote — confirm). If unused, delete it from `route.Model`. If still used elsewhere, leave it. Document either way.

- [ ] **Step 3: Build & test**

Run: `cd services/atlas-transports/atlas.com/transports && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/processor.go \
        services/atlas-transports/atlas.com/transports/route/  # if StartMapId removed
git commit -m "refactor(atlas-transports): remove transit-map login detection (resolver covers it)"
```

---

## Phase 9 — atlas-party-quests cleanup

### Task 9.1: Skip warp emit on disconnect leave

**Files:**
- Modify: `services/atlas-party-quests/atlas.com/party-quests/instance/processor.go`

- [ ] **Step 1: Read**

Re-read lines 917-963 (`Leave` function with `reason string` parameter, line 953 has the `mb.Put(character2.EnvCommandTopic, warpCharacterProvider(...))`).

- [ ] **Step 2: Edit — guard the warp emit**

Replace:

```go
err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(ce.WorldId(), ce.ChannelId(), characterId, exitMap, uuid.Nil))
if err != nil {
    p.l.WithError(err).Errorf("Failed to warp character [%d] to exit map.", characterId)
}
```

With:

```go
if reason != "disconnect" {
    err = mb.Put(character2.EnvCommandTopic, warpCharacterProvider(ce.WorldId(), ce.ChannelId(), characterId, exitMap, uuid.Nil))
    if err != nil {
        p.l.WithError(err).Errorf("Failed to warp character [%d] to exit map.", characterId)
    }
}
```

- [ ] **Step 3: Add a unit test**

Find or create `services/atlas-party-quests/atlas.com/party-quests/instance/processor_test.go`. Add a test that calls `Leave(mb)("disconnect")` and asserts `mb` does NOT contain a `warpCharacterProvider` message on `character2.EnvCommandTopic`. Mirror the existing test scaffolding in this package.

- [ ] **Step 4: Build & test**

Run: `cd services/atlas-party-quests/atlas.com/party-quests && go build ./... && go test ./instance/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-party-quests/atlas.com/party-quests/instance/
git commit -m "refactor(atlas-party-quests): skip exit warp on disconnect leave"
```

---

## Phase 10 — Redis presence migration (atlas-maps)

> Per design D10. The in-memory `getCharacterRegistry` singleton becomes Redis-backed. Independent of forced-return correctness, but design lists it under task-055 scope.

### Task 10.1: Read existing Redis usage in atlas-maps

- [ ] **Step 1: Survey**

Run: `grep -rn "redis\|Redis" services/atlas-maps/atlas.com/maps/ | head -20`. Identify the existing Redis client wiring (it is used for spawn cache per project memory `reference_atlas_maps_spawn_cache.md`).

- [ ] **Step 2: Document plan or punt**

If Redis client is already injected and ergonomic, proceed with Task 10.2. If wiring it requires significant scaffolding **and** the test surface is large, **defer to a follow-up task** (note in `audit.md` and `docs/TODO.md`). The design accepts in-memory operation in the short term as long as the data is single-pod-correct; multi-pod-safe is the long-term goal.

### Task 10.2 (conditional): Migrate `getCharacterRegistry`

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/character/registry.go`

- [ ] **Step 1: Replace in-memory map with Redis ops**

Use Redis keys:
- `atlas:maps:online:{tenantId}:{characterId}` — hash with `{world_id, channel_id, map_id, instance}`.
- `atlas:maps:presence:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` — set of `characterId`.

Populate on LOGIN/MAP_CHANGED/CHANNEL_CHANGED. Clean on LOGOUT.

Concrete implementation depends on the existing Redis client surface. Inspect the spawn-cache code path for the canonical Atlas Redis client.

- [ ] **Step 2: Tests**

Add unit tests covering SADD/SREM on map transitions. If a Redis test container is in CI, integration tests cover this; otherwise mock the client interface.

- [ ] **Step 3: Build & test**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/character/registry.go
git commit -m "refactor(atlas-maps): move character presence registry to Redis"
```

> If this phase is deferred per Task 10.1, skip and document in `audit.md` under "deferred to follow-up".

---

## Phase 11 — Verification, observability, integration

### Task 11.1: Add metric counters for resolution outcomes

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/character/location/processor.go`

- [ ] **Step 1: Read existing metric patterns**

Run: `grep -rn "prometheus\|otel\|metric" services/atlas-maps/atlas.com/maps/ | head -10`. Find how existing counters are registered.

- [ ] **Step 2: Add the counter**

Inside `Resolve`, increment `atlas_maps_location_resolutions_total` with label `reason`:

```go
locationResolutionsTotal.WithLabelValues(string(reason)).Inc()
```

Register the counter in init() or wherever atlas-maps' metrics setup lives.

- [ ] **Step 3: Build & test**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/
git commit -m "feat(atlas-maps): record resolution-reason counter"
```

### Task 11.2: Update OTel spans for resolver

- [ ] **Step 1: Add span attributes**

Inside `Resolve`, start a span (mirror `MapTimer.Disconnect` pattern in `services/atlas-maps/atlas.com/maps/map/timer/processor.go:148`):

```go
_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "Location.Resolve")
span.SetAttributes(
    attribute.Int("current.map.id", int(cur.MapId())),
    attribute.Int("forced.return.map.id", int(md.ForcedReturnMapId())),
    attribute.String("resolution.reason", string(reason)),
    attribute.String("tenant.id", tenant.MustFromContext(p.ctx).Id().String()),
)
defer span.End()
```

- [ ] **Step 2: Build & test, commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/processor.go
git commit -m "feat(atlas-maps): add Location.Resolve OTel span"
```

### Task 11.3: Integration test (or manual verification scenarios)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/character/location/integration_test.go` (build tag-gated if heavyweight)

- [ ] **Step 1: Decide test strategy**

Atlas tests are predominantly unit tests with in-memory deps. A full multi-service integration test requires a docker-compose stack. Two options:

  a. **Lightweight integration test inside atlas-maps**: spin up a tenant context, an in-memory DB, and a stub `info.Processor` that returns hand-crafted maps. Drive `Resolve` + `Set` end-to-end for the eight scenarios in design §8 table I1-I8.

  b. **Manual verification checklist** in `audit.md`: after deploy to dev, run through the eight scenarios with a live client.

Option (a) is mandatory; option (b) is a stretch goal for the first end-to-end deploy.

- [ ] **Step 2: Implement option (a)**

Build the test cases I1, I3, I4, I5, I6 against the in-memory infra. I2 (transit map) and I7 (concurrent disconnect-during-channel-change) and I8 (atlas-maps unreachable) require multi-service setup — gate those with `//go:build integration` and rely on the manual scenarios.

```go
//go:build integration

package location_test

// scenarios I2, I7, I8 — run with -tags=integration and a live stack.
```

- [ ] **Step 3: Run unit-flavored integration tests**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/location/... -v`
Expected: PASS for I1, I3, I4, I5, I6.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character/location/integration_test.go
git commit -m "test(atlas-maps): cover forced-return scenarios I1/I3/I4/I5/I6"
```

### Task 11.4: Final verification gate

- [ ] **Step 1: Build all affected services**

Run in parallel (separate terminals or sequentially):

```
cd services/atlas-maps/atlas.com/maps && go build ./...
cd services/atlas-character/atlas.com/character && go build ./...
cd services/atlas-channel/atlas.com/channel && go build ./...
cd services/atlas-login/atlas.com/login && go build ./...
cd services/atlas-transports/atlas.com/transports && go build ./...
cd services/atlas-party-quests/atlas.com/party-quests && go build ./...
cd libs/atlas-constants && go build ./...
```

Expected: all PASS.

- [ ] **Step 2: Run all affected services' tests**

Same six services + libs/atlas-constants, run `go test ./...`. Expected: PASS.

- [ ] **Step 3: Verify Docker builds**

Run `docker compose build` in any service whose Dockerfile depends on a libs/atlas-constants version pin (`grep -l "atlas-constants" services/*/go.mod` to identify them). Per project guideline: shared-library changes must build cleanly in Docker.

- [ ] **Step 4: Manual smoke (optional, requires dev stack)**

- Log into KPQ entry, walk into KPQ room (`103000800`), DC client, log back in → arrive at KPQ lobby (`103000890`).
- Log onto Henesys Hunting Ground 1 (`100020000`), DC client, log back in → arrive at same map (sentinel).
- On any forced-return map, change channels → arrive on new channel at the forced-return target.

- [ ] **Step 5: Update audit.md placeholder**

If `docs/tasks/task-055-forced-return-on-exit/audit.md` does not yet exist, leave its creation to the code-review phase (`superpowers:requesting-code-review`).

---

## Self-review checklist (executor: skim before opening PR)

- [ ] **Spec coverage**: PRD §10 acceptance criteria each map to a test or documented parity delta. Specifically:
  - PRD criterion "disconnect at HP=0 with returnMap=Y" → documented as parity delta from D7 (no automated test).
  - PRD criterion "channel-change on forcedReturn map" → covered by `Resolve` unit + Task 11.3 I5.
  - PRD criterion "PQ Leave on disconnect drops warp emit" → Task 9.1 unit test.
  - PRD criterion "Timer ForceReturnIfTracked retires CHANGE_MAP emit" → no emit assertion in Task 2.2 + atlas-maps full-test run.
- [ ] **Type consistency**: `field.Model`, `_map.Id`, `channel.Id`, `world.Id` used consistently across all new code. `IsSentinel()` is the only sentinel comparison.
- [ ] **No placeholders**: all "TODO in design" items map to a follow-up TODO in `docs/TODO.md`, not stubs in code.
- [ ] **Docker build**: any service consuming a changed libs/atlas-constants version was Docker-built.
- [ ] **CLAUDE.md rules respected**: no commits to `main`, all commits on `task-055-forced-return-on-exit`, no destructive git ops without confirmation.

---

## Rollback notes

Per design §5, rollback = revert in reverse phase order. Backfill is reversible by reading `character_locations` back into `atlas-character.characters`. Two rows scale makes any rollback path trivial.
