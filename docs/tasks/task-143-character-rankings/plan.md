# Login-Screen Character Rankings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** New atlas-rankings service that periodically computes per-world overall/job character rankings per tenant, exposes them over REST, and atlas-login populates the four rank fields on the character-select screen (replacing hardcoded zeros), failing open to zeros.

**Architecture:** atlas-rankings is a standard GORM + JSON:API DDD service modeled on atlas-fame (no Kafka). A leader-gated (libs/atlas-lock) 60s ticker scans atlas-character over REST per tenant, computes rankings in pure functions, and batch-upserts + prunes a `character_rankings` table; a `ranking_cycles` table drives per-tenant cadence read from a new `rankings` configuration resource in atlas-tenants. atlas-login bulk-fetches rankings once per character-list build and merges them onto its immutable models.

**Tech Stack:** Go, GORM (Postgres prod / sqlite tests), api2go JSON:API, gorilla/mux, libs/atlas-{database,rest,model,tenant,constants,lock,redis,service,tracing}.

## Global Constraints

- Verification gate per changed module: `go test -race ./...`, `go vet ./...`, `go build ./...` clean; `docker buildx bake atlas-rankings` from repo root (new module); `tools/redis-key-guard.sh` clean from repo root (no GOWORK=off prefix).
- Types from `libs/atlas-constants` only: `world.Id`, `job.Id` (DOM-21). No locally invented job constants; job category is derived arithmetic `uint16(jobId / 100)`.
- Immutable models: private fields + getters + Builder. Processor iface + Impl, `NewProcessor(l, ctx, db)`.
- Providers never take `tenantId` — tenant filtering is automatic via `db.WithContext(ctx)` GORM callbacks registered by `database.Connect`. Only creates set `TenantId`. Entity field name is `TenantId` (not `TenantID`).
- Test files opening sqlite directly MUST call `database.RegisterTenantCallbacks(logrus.New(), db)` after `gorm.Open`. Builder pattern for test setup; NO `*_testhelpers.go` files.
- JSON:API client target structs MUST implement `SetToOneReferenceID`, `SetToManyReferenceIDs`, `SetReferencedStructs` stubs (libs/atlas-rest CLAUDE.md; atlas-character responses have `relationships`).
- Readiness probe path is `/api/readyz` (mounted under base path `/api/`), never bare `/readyz`.
- Never hardcode `RANKINGS_SERVICE_URL` (or any `*_SERVICE_URL`) in `deploy/k8s/base/env-configmap.yaml` — `requests.RootUrl` falls back to `BASE_SERVICE_URL`.
- Default recompute interval: **60 minutes** when tenant config is absent/erroring. Base scheduler tick: **60 seconds**.
- Rank ordering: `level DESC, experience DESC, characterId ASC`, 1-based. Move = `previousRank − newRank`; no previous row → 0. GM exclusion rule: `gm > 0`.
- No `// TODO`, stubs, or 501s in landed commits.
- `go mod tidy` only AFTER the imports exist in source. Never `go work sync`.
- Commit messages end with: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: atlas-rankings module scaffold (go.mod, logger, rest package, entities, main.go)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/go.mod`
- Create: `services/atlas-rankings/atlas.com/rankings/logger/init.go`
- Create: `services/atlas-rankings/atlas.com/rankings/rest/handler.go`
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/entity.go`
- Create: `services/atlas-rankings/atlas.com/rankings/main.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/entity_test.go`
- Modify: `go.work` (add module to `use` block)

**Interfaces:**
- Produces: package `ranking` with `Migration(db *gorm.DB) error`, `Entity` (table `character_rankings`), `CycleEntity` (table `ranking_cycles`). Module name `atlas-rankings`.
- Produces: package `rest` with `HandlerDependency` (`Logger()`, `DB()`, `Context()`), `HandlerContext` (`ServerInformation()`), `GetHandler`, `RegisterHandler(l)(db)(si)(name, handler)`.

- [x] **Step 1: Create the module skeleton**

Create `services/atlas-rankings/atlas.com/rankings/go.mod`. Copy `services/atlas-fame/atlas.com/fame/go.mod` verbatim, then: change line 1 to `module atlas-rankings`; delete the `github.com/Chronicle20/atlas/libs/atlas-kafka` and `github.com/segmentio/kafka-go` require lines; add to the replace block:

```
replace github.com/Chronicle20/atlas/libs/atlas-lock => ../../../../libs/atlas-lock
```

(The fame go.mod already has replaces for constants, database, model, rest, service, tenant, tracing, redis, retry, etc. — keep them all; `go mod tidy` in Step 6 trims unused requires.)

Copy `services/atlas-fame/atlas.com/fame/logger/init.go` verbatim to `services/atlas-rankings/atlas.com/rankings/logger/init.go` (package `logger`; it is service-agnostic).

Copy `services/atlas-character/atlas.com/character/rest/handler.go` to `services/atlas-rankings/atlas.com/rankings/rest/handler.go` (package `rest`; it only imports libs and stdlib). Delete the character-inventory-specific `InventoryTypeHandler`/`ParseInventoryType` block at the end of the file; keep everything else (`HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler`, `ParseInput`, `RegisterHandler`, `RegisterInputHandler`, `CharacterIdHandler`, `ParseCharacterId`).

Add to `go.work` `use (…)` block, alphabetically after `./services/atlas-quest/atlas.com/quest`:

```
	./services/atlas-rankings/atlas.com/rankings
```

- [x] **Step 2: Write the failing entity/migration test**

Create `services/atlas-rankings/atlas.com/rankings/ranking/entity_test.go`:

```go
package ranking

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	database.RegisterTenantCallbacks(logrus.New(), db)
	return db
}

func TestMigrationCreatesTables(t *testing.T) {
	db := testDatabase(t)
	if !db.Migrator().HasTable("character_rankings") {
		t.Fatal("character_rankings table not created")
	}
	if !db.Migrator().HasTable("ranking_cycles") {
		t.Fatal("ranking_cycles table not created")
	}
}
```

- [x] **Step 3: Run test to verify it fails**

Run (from `services/atlas-rankings/atlas.com/rankings`): `go test ./ranking/ -run TestMigrationCreatesTables -v`
Expected: FAIL — compile error, `Migration` undefined (entity.go does not exist yet).

- [x] **Step 4: Write entity.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/entity.go`:

```go
package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &CycleEntity{})
}

// Entity is one ranked character. 0 is never stored as a rank — unranked
// characters simply have no row.
type Entity struct {
	TenantId        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_rankings_tenant_character;index:idx_rankings_tenant_world"`
	Id              uuid.UUID `gorm:"type:uuid;primaryKey"`
	CharacterId     uint32    `gorm:"not null;uniqueIndex:idx_rankings_tenant_character"`
	WorldId         world.Id  `gorm:"not null;index:idx_rankings_tenant_world"`
	JobCategory     uint16    `gorm:"not null"`
	OverallRank     uint32    `gorm:"not null"`
	OverallRankMove int32     `gorm:"not null"`
	JobRank         uint32    `gorm:"not null"`
	JobRankMove     int32     `gorm:"not null"`
	ComputedAt      time.Time `gorm:"not null"`
}

func (e *Entity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func (e Entity) TableName() string {
	return "character_rankings"
}

// CycleEntity tracks recompute cadence and observability per tenant. It
// exists (rather than MAX(computed_at)) so a tenant with zero eligible
// characters still records cycle progress and does not busy-loop.
type CycleEntity struct {
	TenantId         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Id               uuid.UUID `gorm:"type:uuid;primaryKey"`
	LastStartedAt    time.Time `gorm:"not null"`
	LastCompletedAt  *time.Time
	CharactersRanked uint32
	DurationMs       uint32
}

func (e *CycleEntity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func (e CycleEntity) TableName() string {
	return "ranking_cycles"
}
```

- [x] **Step 5: Write main.go**

Create `services/atlas-rankings/atlas.com/rankings/main.go` (route initializer and ticker are wired in Tasks 7–8; `db` is used by `database.SetMigrations` at connect time):

```go
package main

import (
	"atlas-rankings/logger"
	"atlas-rankings/ranking"
	"os"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-rankings"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{baseUrl: "", prefix: "/api/"}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(ranking.Migration))
	_ = db

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountReadiness("/readyz", func() bool { return true })).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

(`_ = db` is removed in Task 7 when `ranking.InitResource` consumes it.)

- [x] **Step 6: Tidy and run tests**

Run from `services/atlas-rankings/atlas.com/rankings`:

```bash
go mod tidy
go build ./...
go test -race ./ranking/ -v
go vet ./...
```

Expected: build clean, `TestMigrationCreatesTables` PASS, vet clean.

- [x] **Step 7: Commit**

```bash
git add services/atlas-rankings go.work go.work.sum
git commit -m "feat(task-143): scaffold atlas-rankings service module with ranking entities"
```

---

### Task 2: ranking domain Model + Builder

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/model.go`
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/builder.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/model_test.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/entity.go` (add `Make`/`MakeCycle`)

**Interfaces:**
- Produces: `Model` with getters `CharacterId() uint32`, `WorldId() world.Id`, `JobCategory() uint16`, `OverallRank() uint32`, `OverallRankMove() int32`, `JobRank() uint32`, `JobRankMove() int32`, `ComputedAt() time.Time`; `NewBuilder() *Builder` with `SetCharacterId/SetWorldId/SetJobCategory/SetOverallRank/SetOverallRankMove/SetJobRank/SetJobRankMove/SetComputedAt` and `Build() Model`; `Make(e Entity) (Model, error)`; `CycleModel` with `LastStartedAt() time.Time`, `LastCompletedAt() *time.Time`, `CharactersRanked() uint32`, `DurationMs() uint32`; `MakeCycle(e CycleEntity) (CycleModel, error)`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-rankings/atlas.com/rankings/ranking/model_test.go`:

```go
package ranking

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestBuilderRoundTrip(t *testing.T) {
	now := time.Now()
	m := NewBuilder().
		SetCharacterId(42).
		SetWorldId(world.Id(1)).
		SetJobCategory(2).
		SetOverallRank(17).
		SetOverallRankMove(2).
		SetJobRank(4).
		SetJobRankMove(-1).
		SetComputedAt(now).
		Build()

	if m.CharacterId() != 42 || m.WorldId() != world.Id(1) || m.JobCategory() != 2 {
		t.Fatalf("identity fields lost: %+v", m)
	}
	if m.OverallRank() != 17 || m.OverallRankMove() != 2 || m.JobRank() != 4 || m.JobRankMove() != -1 {
		t.Fatalf("rank fields lost: %+v", m)
	}
	if !m.ComputedAt().Equal(now) {
		t.Fatalf("computedAt lost")
	}
}

func TestMakeFromEntity(t *testing.T) {
	now := time.Now()
	e := Entity{CharacterId: 7, WorldId: world.Id(0), JobCategory: 21, OverallRank: 1, OverallRankMove: 0, JobRank: 1, JobRankMove: 3, ComputedAt: now}
	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}
	if m.CharacterId() != 7 || m.JobCategory() != 21 || m.JobRankMove() != 3 {
		t.Fatalf("Make lost fields: %+v", m)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./ranking/ -run 'TestBuilderRoundTrip|TestMakeFromEntity' -v`
Expected: FAIL — `NewBuilder`, `Make` undefined.

- [x] **Step 3: Write model.go and builder.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/model.go`:

```go
package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
	characterId     uint32
	worldId         world.Id
	jobCategory     uint16
	overallRank     uint32
	overallRankMove int32
	jobRank         uint32
	jobRankMove     int32
	computedAt      time.Time
}

func (m Model) CharacterId() uint32     { return m.characterId }
func (m Model) WorldId() world.Id       { return m.worldId }
func (m Model) JobCategory() uint16     { return m.jobCategory }
func (m Model) OverallRank() uint32     { return m.overallRank }
func (m Model) OverallRankMove() int32  { return m.overallRankMove }
func (m Model) JobRank() uint32         { return m.jobRank }
func (m Model) JobRankMove() int32      { return m.jobRankMove }
func (m Model) ComputedAt() time.Time   { return m.computedAt }

type CycleModel struct {
	lastStartedAt    time.Time
	lastCompletedAt  *time.Time
	charactersRanked uint32
	durationMs       uint32
}

func (m CycleModel) LastStartedAt() time.Time    { return m.lastStartedAt }
func (m CycleModel) LastCompletedAt() *time.Time { return m.lastCompletedAt }
func (m CycleModel) CharactersRanked() uint32    { return m.charactersRanked }
func (m CycleModel) DurationMs() uint32          { return m.durationMs }
```

Create `services/atlas-rankings/atlas.com/rankings/ranking/builder.go`:

```go
package ranking

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	characterId     uint32
	worldId         world.Id
	jobCategory     uint16
	overallRank     uint32
	overallRankMove int32
	jobRank         uint32
	jobRankMove     int32
	computedAt      time.Time
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetCharacterId(v uint32) *Builder    { b.characterId = v; return b }
func (b *Builder) SetWorldId(v world.Id) *Builder      { b.worldId = v; return b }
func (b *Builder) SetJobCategory(v uint16) *Builder    { b.jobCategory = v; return b }
func (b *Builder) SetOverallRank(v uint32) *Builder    { b.overallRank = v; return b }
func (b *Builder) SetOverallRankMove(v int32) *Builder { b.overallRankMove = v; return b }
func (b *Builder) SetJobRank(v uint32) *Builder        { b.jobRank = v; return b }
func (b *Builder) SetJobRankMove(v int32) *Builder     { b.jobRankMove = v; return b }
func (b *Builder) SetComputedAt(v time.Time) *Builder  { b.computedAt = v; return b }

func (b *Builder) Build() Model {
	return Model{
		characterId:     b.characterId,
		worldId:         b.worldId,
		jobCategory:     b.jobCategory,
		overallRank:     b.overallRank,
		overallRankMove: b.overallRankMove,
		jobRank:         b.jobRank,
		jobRankMove:     b.jobRankMove,
		computedAt:      b.computedAt,
	}
}
```

Append to `services/atlas-rankings/atlas.com/rankings/ranking/entity.go`:

```go
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetCharacterId(e.CharacterId).
		SetWorldId(e.WorldId).
		SetJobCategory(e.JobCategory).
		SetOverallRank(e.OverallRank).
		SetOverallRankMove(e.OverallRankMove).
		SetJobRank(e.JobRank).
		SetJobRankMove(e.JobRankMove).
		SetComputedAt(e.ComputedAt).
		Build(), nil
}

func MakeCycle(e CycleEntity) (CycleModel, error) {
	return CycleModel{
		lastStartedAt:    e.LastStartedAt,
		lastCompletedAt:  e.LastCompletedAt,
		charactersRanked: e.CharactersRanked,
		durationMs:       e.DurationMs,
	}, nil
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test -race ./ranking/ -v`
Expected: PASS (all three tests).

- [x] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking
git commit -m "feat(task-143): ranking domain model and builder"
```

---

### Task 3: pure ranking computation (compute.go)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/compute.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/compute_test.go`

**Interfaces:**
- Produces: `Input{CharacterId uint32; WorldId world.Id; JobId job.Id; Level byte; Experience uint32}`, `Ranked{CharacterId uint32; WorldId world.Id; JobCategory uint16; OverallRank uint32; JobRank uint32}`, `JobCategory(jobId job.Id) uint16`, `Rank(inputs []Input) []Ranked`, `Move(prev uint32, next uint32) int32`. Pure functions, no side effects. Callers (Task 6) filter GM characters BEFORE building `Input`s.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-rankings/atlas.com/rankings/ranking/compute_test.go`:

```go
package ranking

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func rankedById(rs []Ranked) map[uint32]Ranked {
	m := make(map[uint32]Ranked, len(rs))
	for _, r := range rs {
		m[r.CharacterId] = r
	}
	return m
}

func TestJobCategory(t *testing.T) {
	cases := []struct {
		jobId job.Id
		want  uint16
	}{
		{job.Id(0), 0},     // beginner
		{job.Id(100), 1},   // warrior
		{job.Id(112), 1},   // hero
		{job.Id(200), 2},   // magician
		{job.Id(312), 3},   // bowman 4th
		{job.Id(412), 4},   // thief 4th
		{job.Id(522), 5},   // pirate 4th
		{job.Id(1000), 10}, // noblesse
		{job.Id(1112), 11}, // dawn warrior 3rd
		{job.Id(2000), 20}, // aran beginner
		{job.Id(2112), 21}, // aran 4th
	}
	for _, c := range cases {
		if got := JobCategory(c.jobId); got != c.want {
			t.Errorf("JobCategory(%d) = %d, want %d", c.jobId, got, c.want)
		}
	}
}

func TestRankOrderingAndTiebreaks(t *testing.T) {
	// level DESC, experience DESC, characterId ASC
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 100, Level: 50, Experience: 100},
		{CharacterId: 2, WorldId: 0, JobId: 100, Level: 70, Experience: 5},
		{CharacterId: 3, WorldId: 0, JobId: 100, Level: 50, Experience: 200},
		{CharacterId: 4, WorldId: 0, JobId: 100, Level: 50, Experience: 100}, // ties char 1 on level+exp; id 4 > 1
	}
	got := rankedById(Rank(inputs))
	if got[2].OverallRank != 1 {
		t.Errorf("char 2 (highest level) rank = %d, want 1", got[2].OverallRank)
	}
	if got[3].OverallRank != 2 {
		t.Errorf("char 3 (level tie, more exp) rank = %d, want 2", got[3].OverallRank)
	}
	if got[1].OverallRank != 3 || got[4].OverallRank != 4 {
		t.Errorf("characterId ASC tiebreak violated: char1=%d char4=%d, want 3 and 4", got[1].OverallRank, got[4].OverallRank)
	}
}

func TestRankUniquePerWorld(t *testing.T) {
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 0, Level: 10, Experience: 0},
		{CharacterId: 2, WorldId: 0, JobId: 0, Level: 10, Experience: 0},
		{CharacterId: 3, WorldId: 1, JobId: 0, Level: 5, Experience: 0},
	}
	got := rankedById(Rank(inputs))
	if got[1].OverallRank == got[2].OverallRank {
		t.Errorf("ranks must be unique within a world (strict total order)")
	}
	if got[3].OverallRank != 1 {
		t.Errorf("worlds must rank independently: char 3 rank = %d, want 1", got[3].OverallRank)
	}
}

func TestJobRankRestrictedToCategory(t *testing.T) {
	inputs := []Input{
		{CharacterId: 1, WorldId: 0, JobId: 100, Level: 90, Experience: 0}, // warrior, overall 1
		{CharacterId: 2, WorldId: 0, JobId: 200, Level: 80, Experience: 0}, // magician, overall 2
		{CharacterId: 3, WorldId: 0, JobId: 110, Level: 70, Experience: 0}, // warrior, overall 3
	}
	got := rankedById(Rank(inputs))
	if got[1].JobRank != 1 || got[3].JobRank != 2 {
		t.Errorf("warrior job ranks = %d,%d, want 1,2", got[1].JobRank, got[3].JobRank)
	}
	if got[2].JobRank != 1 {
		t.Errorf("magician job rank = %d, want 1 (own category)", got[2].JobRank)
	}
	if got[1].JobCategory != 1 || got[2].JobCategory != 2 {
		t.Errorf("job categories wrong: %+v", got)
	}
}

func TestMove(t *testing.T) {
	cases := []struct {
		prev, next uint32
		want       int32
	}{
		{0, 5, 0},   // first-seen → 0
		{5, 3, 2},   // moved up
		{3, 5, -2},  // moved down
		{4, 4, 0},   // unchanged
	}
	for _, c := range cases {
		if got := Move(c.prev, c.next); got != c.want {
			t.Errorf("Move(%d,%d) = %d, want %d", c.prev, c.next, got, c.want)
		}
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./ranking/ -run 'TestJobCategory|TestRank|TestMove' -v`
Expected: FAIL — `Input`, `Rank`, `Move`, `JobCategory` undefined.

- [x] **Step 3: Write compute.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/compute.go`:

```go
package ranking

import (
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Input is one eligible (non-GM) character snapshot. Eligibility filtering
// (gm > 0 excluded entirely) happens before Inputs are built.
type Input struct {
	CharacterId uint32
	WorldId     world.Id
	JobId       job.Id
	Level       byte
	Experience  uint32
}

// Ranked is the computed placement for one character. Ranks are 1-based and
// unique within their scope: the characterId tiebreak makes the order a
// strict total order, so dense and ordinal ranking coincide.
type Ranked struct {
	CharacterId uint32
	WorldId     world.Id
	JobCategory uint16
	OverallRank uint32
	JobRank     uint32
}

// JobCategory buckets a job id, Cosmic parity: jobId / 100.
// 0=beginner, 1=warrior, 2=magician, 3=bowman, 4=thief, 5=pirate;
// Cygnus (10-15) and Aran (20-21) fall out of the same division.
func JobCategory(jobId job.Id) uint16 {
	return uint16(jobId / 100)
}

func less(a Input, b Input) bool {
	if a.Level != b.Level {
		return a.Level > b.Level
	}
	if a.Experience != b.Experience {
		return a.Experience > b.Experience
	}
	return a.CharacterId < b.CharacterId
}

// Rank computes per-world overall and job-category placements ordered by
// level DESC, experience DESC, characterId ASC. Job ranks reuse the same
// sorted order restricted to each category.
func Rank(inputs []Input) []Ranked {
	byWorld := make(map[world.Id][]Input)
	for _, i := range inputs {
		byWorld[i.WorldId] = append(byWorld[i.WorldId], i)
	}

	results := make([]Ranked, 0, len(inputs))
	for wid, ws := range byWorld {
		sort.Slice(ws, func(i, j int) bool { return less(ws[i], ws[j]) })

		jobPos := make(map[uint16]uint32)
		for idx, c := range ws {
			cat := JobCategory(c.JobId)
			jobPos[cat]++
			results = append(results, Ranked{
				CharacterId: c.CharacterId,
				WorldId:     wid,
				JobCategory: cat,
				OverallRank: uint32(idx + 1),
				JobRank:     jobPos[cat],
			})
		}
	}
	return results
}

// Move is previousRank − newRank (positive = moved up). A character with no
// previous entry (prev == 0; 0 is never a stored rank) moves 0.
func Move(prev uint32, next uint32) int32 {
	if prev == 0 {
		return 0
	}
	return int32(prev) - int32(next)
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test -race ./ranking/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking
git commit -m "feat(task-143): pure ranking computation with ordering, job categories, move math"
```

---

### Task 4: DB providers and administrator (upsert, prune, cycles)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/provider.go`
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/administrator.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/administrator_test.go`

**Interfaces:**
- Produces (package-private, consumed by Task 6 processor):
  - `byCharacterIdEntityProvider(characterId uint32) database.EntityProvider[Entity]`
  - `byCharacterIdsEntityProvider(characterIds []uint32) database.EntityProvider[[]Entity]`
  - `allEntityProvider() database.EntityProvider[[]Entity]`
  - `cycleEntityProvider() database.EntityProvider[CycleEntity]`
  - `upsertBatch(db *gorm.DB, tenantId uuid.UUID, entities []Entity) error` — `ON CONFLICT (tenant_id, character_id) DO UPDATE`, 500/batch
  - `pruneBefore(db *gorm.DB, cycleTime time.Time) error` — tenant-scoped via callbacks
  - `startCycle(db *gorm.DB, tenantId uuid.UUID, startedAt time.Time) error` — upsert on `tenant_id`
  - `completeCycle(db *gorm.DB, tenantId uuid.UUID, completedAt time.Time, charactersRanked uint32, durationMs uint32) error`

- [x] **Step 1: Write the failing tests**

Create `services/atlas-rankings/atlas.com/rankings/ranking/administrator_test.go`:

```go
package ranking

import (
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func testTenantContext(t *testing.T) (tenant.Model, context.Context) {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to register tenant: %v", err)
	}
	return tm, tenant.WithContext(context.Background(), tm)
}

func entityFor(characterId uint32, rank uint32, computedAt time.Time) Entity {
	return Entity{
		CharacterId: characterId,
		WorldId:     world.Id(0),
		JobCategory: 1,
		OverallRank: rank,
		JobRank:     rank,
		ComputedAt:  computedAt,
	}
}

func TestUpsertBatchInsertsAndUpdates(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	t1 := time.Now()
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(1, 1, t1), entityFor(2, 2, t1)}); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	t2 := t1.Add(time.Hour)
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(1, 2, t2), entityFor(2, 1, t2)}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	var count int64
	if err := tdb.Model(&Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows after upsert, got %d", count)
	}

	e, err := byCharacterIdEntityProvider(1)(tdb)()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if e.OverallRank != 2 {
		t.Fatalf("upsert did not update rank: got %d, want 2", e.OverallRank)
	}
}

func TestPruneBeforeRemovesStaleRows(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	t1 := time.Now()
	t2 := t1.Add(time.Hour)
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(1, 1, t1)}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := upsertBatch(tdb, tm.Id(), []Entity{entityFor(2, 1, t2)}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	if err := pruneBefore(tdb, t2); err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	if _, err := byCharacterIdEntityProvider(1)(tdb)(); err == nil {
		t.Fatal("stale row for character 1 should be pruned")
	}
	if _, err := byCharacterIdEntityProvider(2)(tdb)(); err != nil {
		t.Fatalf("current row for character 2 should survive: %v", err)
	}
}

func TestTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	tmA, ctxA := testTenantContext(t)
	_, ctxB := testTenantContext(t)
	dbA := db.WithContext(ctxA)
	dbB := db.WithContext(ctxB)

	now := time.Now()
	if err := upsertBatch(dbA, tmA.Id(), []Entity{entityFor(1, 1, now)}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	if _, err := byCharacterIdEntityProvider(1)(dbB)(); err == nil {
		t.Fatal("tenant B must not read tenant A rows")
	}

	// Prune under B must not delete A's rows.
	if err := pruneBefore(dbB, now.Add(time.Hour)); err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if _, err := byCharacterIdEntityProvider(1)(dbA)(); err != nil {
		t.Fatalf("tenant A row must survive tenant B prune: %v", err)
	}
}

func TestCycleRows(t *testing.T) {
	db := testDatabase(t)
	tm, ctx := testTenantContext(t)
	tdb := db.WithContext(ctx)

	if _, err := cycleEntityProvider()(tdb)(); err == nil {
		t.Fatal("expected no cycle row initially")
	}

	start := time.Now()
	if err := startCycle(tdb, tm.Id(), start); err != nil {
		t.Fatalf("startCycle failed: %v", err)
	}
	if err := completeCycle(tdb, tm.Id(), start.Add(time.Second), 10, 1000); err != nil {
		t.Fatalf("completeCycle failed: %v", err)
	}

	// Second cycle upserts the same row.
	if err := startCycle(tdb, tm.Id(), start.Add(time.Hour)); err != nil {
		t.Fatalf("second startCycle failed: %v", err)
	}
	c, err := cycleEntityProvider()(tdb)()
	if err != nil {
		t.Fatalf("cycle read failed: %v", err)
	}
	if !c.LastStartedAt.After(start) {
		t.Fatalf("second start not recorded: %v", c.LastStartedAt)
	}
	if c.CharactersRanked != 10 {
		t.Fatalf("completion fields lost: %+v", c)
	}

	var count int64
	if err := tdb.Model(&CycleEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 cycle row, got %d", count)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./ranking/ -run 'TestUpsert|TestPrune|TestTenantIsolation|TestCycleRows' -v`
Expected: FAIL — `upsertBatch`, `pruneBefore`, `byCharacterIdEntityProvider`, `cycleEntityProvider`, `startCycle`, `completeCycle` undefined.

- [x] **Step 3: Write provider.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/provider.go`:

```go
package ranking

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(characterId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ?", characterId).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

func byCharacterIdsEntityProvider(characterIds []uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Where("character_id IN ?", characterIds).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(result)
	}
}

func allEntityProvider() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(result)
	}
}

func cycleEntityProvider() database.EntityProvider[CycleEntity] {
	return func(db *gorm.DB) model.Provider[CycleEntity] {
		var result CycleEntity
		err := db.First(&result).Error
		if err != nil {
			return model.ErrorProvider[CycleEntity](err)
		}
		return model.FixedProvider(result)
	}
}
```

- [x] **Step 4: Write administrator.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/administrator.go`:

```go
package ranking

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const upsertBatchSize = 500

func upsertBatch(db *gorm.DB, tenantId uuid.UUID, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}
	for i := range entities {
		entities[i].TenantId = tenantId
		if entities[i].Id == uuid.Nil {
			entities[i].Id = uuid.New()
		}
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"world_id", "job_category",
			"overall_rank", "overall_rank_move",
			"job_rank", "job_rank_move",
			"computed_at",
		}),
	}).CreateInBatches(&entities, upsertBatchSize).Error
}

// pruneBefore removes rows not restamped by the current cycle — deleted
// characters and characters that became GM. Tenant scoping comes from the
// GORM delete callback on the context-bearing db handle.
func pruneBefore(db *gorm.DB, cycleTime time.Time) error {
	return db.Where("computed_at < ?", cycleTime).Delete(&Entity{}).Error
}

func startCycle(db *gorm.DB, tenantId uuid.UUID, startedAt time.Time) error {
	e := CycleEntity{
		TenantId:      tenantId,
		Id:            uuid.New(),
		LastStartedAt: startedAt,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"last_started_at": startedAt}),
	}).Create(&e).Error
}

func completeCycle(db *gorm.DB, tenantId uuid.UUID, completedAt time.Time, charactersRanked uint32, durationMs uint32) error {
	return db.Model(&CycleEntity{}).
		Where("tenant_id = ?", tenantId).
		Updates(map[string]interface{}{
			"last_completed_at": completedAt,
			"characters_ranked": charactersRanked,
			"duration_ms":       durationMs,
		}).Error
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `go test -race ./ranking/ -v`
Expected: PASS (all tests so far).

- [x] **Step 6: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking
git commit -m "feat(task-143): ranking DB providers, batch upsert, prune, cycle rows"
```

---

### Task 5: foreign REST clients (character, tenant, configuration)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/character/rest.go`
- Create: `services/atlas-rankings/atlas.com/rankings/character/model.go`
- Create: `services/atlas-rankings/atlas.com/rankings/character/requests.go`
- Create: `services/atlas-rankings/atlas.com/rankings/character/processor.go`
- Create: `services/atlas-rankings/atlas.com/rankings/tenant/rest.go`
- Create: `services/atlas-rankings/atlas.com/rankings/tenant/requests.go`
- Create: `services/atlas-rankings/atlas.com/rankings/tenant/processor.go`
- Create: `services/atlas-rankings/atlas.com/rankings/configuration/rest.go`
- Create: `services/atlas-rankings/atlas.com/rankings/configuration/requests.go`
- Test: `services/atlas-rankings/atlas.com/rankings/character/processor_test.go`
- Test: `services/atlas-rankings/atlas.com/rankings/configuration/requests_test.go`

**Interfaces:**
- Produces: `character.Model` with `Id() uint32`, `WorldId() world.Id`, `JobId() job.Id`, `Level() byte`, `Experience() uint32`, `Gm() int`; `character.Processor` with `GetAll() ([]Model, error)`; `character.NewProcessor(l, ctx) Processor`.
- Produces: `tenant.Processor` (package `tenant`, local) with `GetAll() ([]tenant.Model, error)` returning lib `atlas-tenant` models; `NewProcessor(l, ctx)`.
- Produces: `configuration.GetRecomputeInterval(l, ctx) func(tenantId uuid.UUID) time.Duration` and `configuration.DefaultRecomputeInterval = 60 * time.Minute`.
- Env: `CHARACTERS_SERVICE_URL` / `TENANTS_SERVICE_URL` with `BASE_SERVICE_URL` fallback via `requests.RootUrl`.

- [x] **Step 1: Write the failing character-client test (httptest, realistic fixture WITH relationships)**

The libs/atlas-rest CLAUDE.md requires an httptest-backed test with a `relationships` block — the jsonapi stubs are load-bearing, and FakeClient-style mocks would not catch their absence.

Create `services/atlas-rankings/atlas.com/rankings/character/processor_test.go`:

```go
package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const charactersFixture = `{
  "data": [
    {
      "type": "characters",
      "id": "1",
      "attributes": {
        "accountId": 1000,
        "worldId": 0,
        "name": "Alpha",
        "level": 50,
        "experience": 1234,
        "jobId": 112,
        "gm": 0
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    },
    {
      "type": "characters",
      "id": "2",
      "attributes": {
        "accountId": 1001,
        "worldId": 1,
        "name": "Beta",
        "level": 30,
        "experience": 55,
        "jobId": 0,
        "gm": 1
      },
      "relationships": {
        "equipment": {"data": []},
        "inventories": {"data": []}
      }
    }
  ]
}`

func TestGetAllDecodesTrimmedModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(charactersFixture))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	cs, err := NewProcessor(logrus.New(), ctx).GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(cs))
	}
	if cs[0].Id() != 1 || cs[0].Level() != 50 || cs[0].Experience() != 1234 || cs[0].JobId() != 112 || cs[0].Gm() != 0 {
		t.Fatalf("character 1 decoded wrong: %+v", cs[0])
	}
	if cs[1].WorldId() != 1 || cs[1].Gm() != 1 {
		t.Fatalf("character 2 decoded wrong: %+v", cs[1])
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./character/ -v`
Expected: FAIL — package does not exist / `NewProcessor` undefined.

- [x] **Step 3: Write the character client**

Create `services/atlas-rankings/atlas.com/rankings/character/rest.go`:

```go
package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a trimmed read model of atlas-character's characters
// resource — only the attributes the ranking computation needs.
type RestModel struct {
	Id         uint32   `json:"-"`
	AccountId  uint32   `json:"accountId"`
	WorldId    world.Id `json:"worldId"`
	Level      byte     `json:"level"`
	Experience uint32   `json:"experience"`
	JobId      job.Id   `json:"jobId"`
	Gm         int      `json:"gm"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// Relationship stubs — required because atlas-character responses carry a
// relationships block (equipment/inventories) and api2go errors without them.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:         r.Id,
		worldId:    r.WorldId,
		jobId:      r.JobId,
		level:      r.Level,
		experience: r.Experience,
		gm:         r.Gm,
	}, nil
}
```

Create `services/atlas-rankings/atlas.com/rankings/character/model.go`:

```go
package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
	id         uint32
	worldId    world.Id
	jobId      job.Id
	level      byte
	experience uint32
	gm         int
}

func (m Model) Id() uint32         { return m.id }
func (m Model) WorldId() world.Id  { return m.worldId }
func (m Model) JobId() job.Id      { return m.jobId }
func (m Model) Level() byte        { return m.level }
func (m Model) Experience() uint32 { return m.experience }
func (m Model) Gm() int            { return m.gm }
```

Create `services/atlas-rankings/atlas.com/rankings/character/requests.go`:

```go
package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestAll() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + Resource)
}
```

Create `services/atlas-rankings/atlas.com/rankings/character/processor.go`:

```go
package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// AllProvider returns a provider for all characters of the tenant in context.
	AllProvider() model.Provider[[]Model]
	// GetAll returns all characters of the tenant in context.
	GetAll() ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAll(), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return p.AllProvider()()
}
```

- [x] **Step 4: Run character test to verify it passes**

Run: `go mod tidy && go test -race ./character/ -v`
Expected: PASS.

- [x] **Step 5: Write the tenant client (copy atlas-transports precedent)**

Copy these three files verbatim from `services/atlas-transports/atlas.com/transports/tenant/` to `services/atlas-rankings/atlas.com/rankings/tenant/`: `rest.go`, `requests.go`, `processor.go`. They only import libs (`atlas-rest/requests`, `atlas-tenant`, `atlas-model/model`) — no service-local imports, so they are portable unchanged. The package exposes `NewProcessor(l, ctx).GetAll() ([]tenant.Model, error)` against `GET {TENANTS_SERVICE_URL|BASE_SERVICE_URL}tenants`.

- [x] **Step 6: Write the failing configuration-client test**

Create `services/atlas-rankings/atlas.com/rankings/configuration/requests_test.go`:

```go
package configuration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func TestGetRecomputeIntervalConfigured(t *testing.T) {
	tenantId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"rankings","id":"` + tenantId.String() + `","attributes":{"recomputeIntervalMinutes":15}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(tenantId)
	if got != 15*time.Minute {
		t.Fatalf("interval = %v, want 15m", got)
	}
}

func TestGetRecomputeIntervalDefaultsOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(uuid.New())
	if got != DefaultRecomputeInterval {
		t.Fatalf("interval = %v, want default %v", got, DefaultRecomputeInterval)
	}
}

func TestGetRecomputeIntervalDefaultsOnZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"rankings","id":"x","attributes":{"recomputeIntervalMinutes":0}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(uuid.New())
	if got != DefaultRecomputeInterval {
		t.Fatalf("interval = %v, want default %v", got, DefaultRecomputeInterval)
	}
}
```

- [x] **Step 7: Run test to verify it fails**

Run: `go test ./configuration/ -v`
Expected: FAIL — package does not exist.

- [x] **Step 8: Write the configuration client**

Create `services/atlas-rankings/atlas.com/rankings/configuration/rest.go`:

```go
package configuration

// RestModel is the rankings configuration resource served by atlas-tenants
// at /tenants/{tenantId}/configurations/rankings.
type RestModel struct {
	Id                       string `json:"-"`
	RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
```

Create `services/atlas-rankings/atlas.com/rankings/configuration/requests.go`:

```go
package configuration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultRecomputeInterval applies when a tenant has no rankings
// configuration (or the read fails) — FR-4.
const DefaultRecomputeInterval = 60 * time.Minute

const byTenant = "tenants/%s/configurations/rankings"

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

func requestByTenantId(tenantId uuid.UUID) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+byTenant, tenantId))
}

// GetRecomputeInterval resolves the tenant's recompute cadence. Missing
// config (404) is the expected unconfigured state; any other error is
// logged. Both fall back to the default so one tenant's config problem
// never stalls its recompute entirely.
func GetRecomputeInterval(l logrus.FieldLogger, ctx context.Context) func(tenantId uuid.UUID) time.Duration {
	return func(tenantId uuid.UUID) time.Duration {
		rm, err := requestByTenantId(tenantId)(l, ctx)
		if err != nil {
			if !errors.Is(err, requests.ErrNotFound) {
				l.WithError(err).Warnf("Unable to read rankings configuration for tenant [%s]; using default interval.", tenantId)
			}
			return DefaultRecomputeInterval
		}
		if rm.RecomputeIntervalMinutes == 0 {
			return DefaultRecomputeInterval
		}
		return time.Duration(rm.RecomputeIntervalMinutes) * time.Minute
	}
}
```

- [x] **Step 9: Run all module tests**

Run from module root: `go mod tidy && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS/clean.

- [x] **Step 10: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings
git commit -m "feat(task-143): character, tenant, and configuration REST clients"
```

---

### Task 6: ranking Processor — reads, IsDue, Recompute

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/processor.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/processor_test.go`

**Interfaces:**
- Consumes: Task 3 `Rank`/`Move`/`Input`, Task 4 providers/administrator, Task 5 `character.Processor`.
- Produces:

```go
type CharacterSupplier func() ([]character.Model, error)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	ByCharacterIdsProvider(characterIds []uint32) model.Provider[[]Model]
	GetByCharacterIds(characterIds []uint32) ([]Model, error)
	IsDue(interval time.Duration, now time.Time) (bool, error)
	Recompute(now time.Time) error
	WithCharacterSupplier(s CharacterSupplier) Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor
```

`GetByCharacterId` returns `gorm.ErrRecordNotFound`-wrapped error when absent (Task 7 maps to 404).

- [x] **Step 1: Write the failing tests**

Create `services/atlas-rankings/atlas.com/rankings/ranking/processor_test.go`:

```go
package ranking

import (
	"errors"
	"testing"
	"time"

	"atlas-rankings/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// characterFixture builds a character.Model via its JSON:API extract path —
// the package exposes no test constructor, and Extract is the production
// decode path anyway.
func characterFixture(t *testing.T, id uint32, worldId byte, jobId uint16, level byte, exp uint32, gm int) character.Model {
	t.Helper()
	rm := character.RestModel{
		AccountId:  1,
		WorldId:    world.Id(worldId),
		Level:      level,
		Experience: exp,
		JobId:      job.Id(jobId),
		Gm:         gm,
	}
	rm.Id = id
	m, err := character.Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

func supplierOf(cs ...character.Model) CharacterSupplier {
	return func() ([]character.Model, error) { return cs, nil }
}

func TestRecomputeRanksAndExcludesGms(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 90, 0, 0), // warrior lvl 90 → overall 1
		characterFixture(t, 2, 0, 200, 80, 0, 0), // magician lvl 80 → overall 2
		characterFixture(t, 3, 0, 100, 70, 0, 0), // warrior lvl 70 → overall 3
		characterFixture(t, 4, 0, 900, 99, 0, 1), // GM — excluded entirely, not counted
	))

	if err := p.Recompute(time.Now()); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	m1, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if m1.OverallRank() != 1 || m1.JobRank() != 1 || m1.OverallRankMove() != 0 {
		t.Fatalf("char 1: %+v", m1)
	}
	m2, _ := p.GetByCharacterId(2)
	if m2.OverallRank() != 2 || m2.JobRank() != 1 {
		t.Fatalf("char 2 (GM must not shift ranks): %+v", m2)
	}
	m3, _ := p.GetByCharacterId(3)
	if m3.OverallRank() != 3 || m3.JobRank() != 2 {
		t.Fatalf("char 3: %+v", m3)
	}
	if _, err := p.GetByCharacterId(4); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GM must have no row, got err=%v", err)
	}
}

func TestRecomputeMovesAcrossTwoCycles(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	first := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
	))
	if err := first.Recompute(time.Now()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Character 1 levels past character 2; character 3 appears.
	second := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 70, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
		characterFixture(t, 3, 0, 200, 10, 0, 0),
	))
	if err := second.Recompute(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	m1, _ := second.GetByCharacterId(1)
	if m1.OverallRank() != 1 || m1.OverallRankMove() != 1 || m1.JobRankMove() != 1 {
		t.Fatalf("char 1 move: %+v", m1)
	}
	m2, _ := second.GetByCharacterId(2)
	if m2.OverallRank() != 2 || m2.OverallRankMove() != -1 {
		t.Fatalf("char 2 move: %+v", m2)
	}
	m3, _ := second.GetByCharacterId(3)
	if m3.OverallRankMove() != 0 || m3.JobRankMove() != 0 {
		t.Fatalf("first-seen char 3 must move 0: %+v", m3)
	}
}

func TestRecomputePrunesDepartedCharacters(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	first := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
		characterFixture(t, 2, 0, 100, 60, 0, 0),
	))
	if err := first.Recompute(time.Now()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Character 2 deleted, character 1 became GM.
	second := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 1),
	))
	if err := second.Recompute(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	if _, err := second.GetByCharacterId(1); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("became-GM character must be pruned")
	}
	if _, err := second.GetByCharacterId(2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("deleted character must be pruned")
	}
}

func TestGetByCharacterIdsOmitsUnknown(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf(
		characterFixture(t, 1, 0, 100, 50, 0, 0),
	))
	if err := p.Recompute(time.Now()); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	ms, err := p.GetByCharacterIds([]uint32{1, 999})
	if err != nil {
		t.Fatalf("bulk read: %v", err)
	}
	if len(ms) != 1 || ms[0].CharacterId() != 1 {
		t.Fatalf("unknown ids must be omitted: %+v", ms)
	}
}

func TestIsDue(t *testing.T) {
	db := testDatabase(t)
	_, ctx := testTenantContext(t)
	l := logrus.New()

	p := NewProcessor(l, ctx, db).WithCharacterSupplier(supplierOf())
	now := time.Now()

	due, err := p.IsDue(time.Hour, now)
	if err != nil || !due {
		t.Fatalf("no cycle row must be due: due=%v err=%v", due, err)
	}

	if err := p.Recompute(now); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	due, err = p.IsDue(time.Hour, now.Add(30*time.Minute))
	if err != nil || due {
		t.Fatalf("30m into a 60m interval must not be due: due=%v err=%v", due, err)
	}

	due, err = p.IsDue(time.Hour, now.Add(61*time.Minute))
	if err != nil || !due {
		t.Fatalf("61m into a 60m interval must be due: due=%v err=%v", due, err)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./ranking/ -run 'TestRecompute|TestGetByCharacterIds|TestIsDue' -v`
Expected: FAIL — `NewProcessor`, `CharacterSupplier` undefined.

- [x] **Step 3: Write processor.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/processor.go`:

```go
package ranking

import (
	"context"
	"errors"
	"time"

	"atlas-rankings/character"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CharacterSupplier abstracts the atlas-character scan so tests can inject
// fixtures without an HTTP server.
type CharacterSupplier func() ([]character.Model, error)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	ByCharacterIdsProvider(characterIds []uint32) model.Provider[[]Model]
	GetByCharacterIds(characterIds []uint32) ([]Model, error)
	// IsDue reports whether the tenant's recompute interval has elapsed
	// since the last cycle start (true when no cycle has ever run).
	IsDue(interval time.Duration, now time.Time) (bool, error)
	// Recompute scans characters, ranks them, upserts rows stamped with
	// now, prunes rows older than now, and records the cycle. Idempotent
	// and convergent — a crashed run is fully repaired by the next one.
	Recompute(now time.Time) error
	WithCharacterSupplier(s CharacterSupplier) Processor
}

type ProcessorImpl struct {
	l          logrus.FieldLogger
	ctx        context.Context
	db         *gorm.DB
	t          tenant.Model
	characters CharacterSupplier
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	cp := character.NewProcessor(l, ctx)
	return &ProcessorImpl{
		l:          l,
		ctx:        ctx,
		db:         db,
		t:          tenant.MustFromContext(ctx),
		characters: cp.GetAll,
	}
}

func (p *ProcessorImpl) WithCharacterSupplier(s CharacterSupplier) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: p.db, t: p.t, characters: s}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return model.Map(Make)(byCharacterIdEntityProvider(characterId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ByCharacterIdsProvider(characterIds []uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdsEntityProvider(characterIds)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterIds(characterIds []uint32) ([]Model, error) {
	return p.ByCharacterIdsProvider(characterIds)()
}

func (p *ProcessorImpl) IsDue(interval time.Duration, now time.Time) (bool, error) {
	c, err := cycleEntityProvider()(p.db.WithContext(p.ctx))()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return now.Sub(c.LastStartedAt) >= interval, nil
}

func (p *ProcessorImpl) Recompute(now time.Time) error {
	tdb := p.db.WithContext(p.ctx)
	wallStart := time.Now()

	if err := startCycle(tdb, p.t.Id(), now); err != nil {
		return err
	}

	cs, err := p.characters()
	if err != nil {
		return err
	}

	inputs := make([]Input, 0, len(cs))
	for _, c := range cs {
		if c.Gm() > 0 {
			continue
		}
		inputs = append(inputs, Input{
			CharacterId: c.Id(),
			WorldId:     c.WorldId(),
			JobId:       c.JobId(),
			Level:       c.Level(),
			Experience:  c.Experience(),
		})
	}

	ranked := Rank(inputs)

	prev, err := allEntityProvider()(tdb)()
	if err != nil {
		return err
	}
	prevById := make(map[uint32]Entity, len(prev))
	for _, e := range prev {
		prevById[e.CharacterId] = e
	}

	entities := make([]Entity, 0, len(ranked))
	worldCounts := make(map[byte]int)
	for _, r := range ranked {
		var prevOverall, prevJob uint32
		if pe, ok := prevById[r.CharacterId]; ok {
			prevOverall = pe.OverallRank
			prevJob = pe.JobRank
		}
		entities = append(entities, Entity{
			CharacterId:     r.CharacterId,
			WorldId:         r.WorldId,
			JobCategory:     r.JobCategory,
			OverallRank:     r.OverallRank,
			OverallRankMove: Move(prevOverall, r.OverallRank),
			JobRank:         r.JobRank,
			JobRankMove:     Move(prevJob, r.JobRank),
			ComputedAt:      now,
		})
		worldCounts[byte(r.WorldId)]++
	}

	if err := upsertBatch(tdb, p.t.Id(), entities); err != nil {
		return err
	}
	if err := pruneBefore(tdb, now); err != nil {
		return err
	}

	duration := time.Since(wallStart)
	if err := completeCycle(tdb, p.t.Id(), time.Now(), uint32(len(entities)), uint32(duration.Milliseconds())); err != nil {
		return err
	}

	p.l.WithFields(logrus.Fields{
		"tenant":      p.t.Id().String(),
		"ranked":      len(entities),
		"worlds":      len(worldCounts),
		"world_sizes": worldCounts,
		"duration":    duration.String(),
	}).Infof("Rankings recompute cycle completed.")
	return nil
}
```

Note: the job move is computed against the previous job rank regardless of a category change (design §3.4 — "did my displayed number improve").

- [x] **Step 4: Run tests to verify they pass**

Run: `go test -race ./ranking/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking
git commit -m "feat(task-143): ranking processor with recompute cycle, due-check, bulk reads"
```

---

### Task 7: rankings REST resource (rest.go, resource.go, main.go wiring)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/rest.go`
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/resource.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/main.go` (wire `InitResource`, drop `_ = db`)
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/resource_test.go`

**Interfaces:**
- Consumes: Task 6 `Processor.GetByCharacterIds` / `GetByCharacterId`, Task 1 `rest.RegisterHandler`.
- Produces: `InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`; JSON:API resource type `rankings`, id = characterId; routes `GET /rankings/characters?ids=…` and `GET /rankings/characters/{characterId}` under base path `/api/`.

- [x] **Step 1: Write the failing handler tests**

Create `services/atlas-rankings/atlas.com/rankings/ranking/resource_test.go`:

```go
package ranking

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInfo struct{}

func (s testServerInfo) GetBaseURL() string { return "" }
func (s testServerInfo) GetPrefix() string  { return "/api/" }

func testRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter().PathPrefix("/api/").Subrouter()
	InitResource(testServerInfo{})(db)(router, logrus.New())
	return router
}

func tenantHeaders(r *http.Request, tm tenant.Model) {
	r.Header.Set("TENANT_ID", tm.Id().String())
	r.Header.Set("REGION", tm.Region())
	r.Header.Set("MAJOR_VERSION", strconv.Itoa(int(tm.MajorVersion())))
	r.Header.Set("MINOR_VERSION", strconv.Itoa(int(tm.MinorVersion())))
}

func seedRanking(t *testing.T, db *gorm.DB, tm tenant.Model, characterId uint32, rank uint32) {
	t.Helper()
	e := Entity{
		TenantId:    tm.Id(),
		CharacterId: characterId,
		WorldId:     0,
		JobCategory: 1,
		OverallRank: rank,
		JobRank:     rank,
		ComputedAt:  time.Now(),
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestBulkEndpoint(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	seedRanking(t, db, tm, 1, 17)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters?ids=1,999", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []struct {
			Id         string          `json:"id"`
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Id != "1" {
		t.Fatalf("unknown ids must be omitted: %s", rec.Body.String())
	}
}

func TestBulkEndpointBadIds(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	for _, ids := range []string{"", "abc", "1,abc", ","} {
		req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters?ids="+ids, nil)
		tenantHeaders(req, tm)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("ids=%q status = %d, want 400", ids, rec.Code)
		}
	}
}

func TestSingleEndpoint(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	seedRanking(t, db, tm, 7, 3)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters/7", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/rankings/characters/999", nil)
	tenantHeaders(req, tm)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing ranking status = %d, want 404", rec.Code)
	}
}
```

Note: `seedRanking` uses `db.Create` on a context-free handle with an explicit `TenantId` — the create callback only injects the tenant when the field is zero, so this seeds tenant-scoped rows without a context.

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./ranking/ -run 'TestBulkEndpoint|TestSingleEndpoint' -v`
Expected: FAIL — `InitResource` undefined.

- [x] **Step 3: Write rest.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/rest.go`:

```go
package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id          uint32    `json:"-"`
	WorldId     world.Id  `json:"worldId"`
	Rank        uint32    `json:"rank"`
	RankMove    int32     `json:"rankMove"`
	JobRank     uint32    `json:"jobRank"`
	JobRankMove int32     `json:"jobRankMove"`
	ComputedAt  time.Time `json:"computedAt"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.CharacterId(),
		WorldId:     m.WorldId(),
		Rank:        m.OverallRank(),
		RankMove:    m.OverallRankMove(),
		JobRank:     m.JobRank(),
		JobRankMove: m.JobRankMove(),
		ComputedAt:  m.ComputedAt(),
	}, nil
}
```

- [x] **Step 4: Write resource.go**

Create `services/atlas-rankings/atlas.com/rankings/ranking/resource.go`:

```go
package ranking

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"atlas-rankings/rest"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/rankings").Subrouter()
			r.HandleFunc("/characters", registerGet("get_rankings_for_characters", handleGetRankingsForCharacters)).Methods(http.MethodGet).Queries("ids", "{ids}")
			// Bare /characters (no ids query) is a caller error, not a missing route.
			r.HandleFunc("/characters", registerGet("get_rankings_missing_ids", handleMissingIds)).Methods(http.MethodGet)
			r.HandleFunc("/characters/{characterId}", registerGet("get_ranking_for_character", handleGetRankingForCharacter)).Methods(http.MethodGet)
		}
	}
}

func handleMissingIds(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func parseIds(raw string) ([]uint32, bool) {
	parts := strings.Split(raw, ",")
	ids := make([]uint32, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, false
		}
		ids = append(ids, uint32(id))
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}

func handleGetRankingsForCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := parseIds(mux.Vars(r)["ids"])
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterIds(ids)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get rankings for characters.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetRankingForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterId, err := strconv.ParseUint(mux.Vars(r)["characterId"], 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterId(uint32(characterId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			d.Logger().WithError(err).Errorf("Unable to get ranking for character %d.", characterId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := Transform(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
```

- [x] **Step 5: Wire into main.go**

In `services/atlas-rankings/atlas.com/rankings/main.go`, delete the `_ = db` line and add the route initializer before `MountReadiness`:

```go
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(ranking.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", func() bool { return true })).
		Run()
```

- [x] **Step 6: Run tests to verify they pass**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: PASS/clean.

- [x] **Step 7: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings
git commit -m "feat(task-143): rankings JSON:API resource with bulk and single character lookup"
```

---

### Task 8: scheduler task + leader election (tasks/, leaderconfig.go, main.go)

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/tasks/task.go`
- Create: `services/atlas-rankings/atlas.com/rankings/tasks/recompute.go`
- Create: `services/atlas-rankings/atlas.com/rankings/leaderconfig.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/main.go`
- Test: `services/atlas-rankings/atlas.com/rankings/tasks/recompute_test.go`

**Interfaces:**
- Consumes: Task 5 `tenant.NewProcessor(l, ctx).GetAll()`, `configuration.GetRecomputeInterval`, Task 6 `ranking.Processor`.
- Produces: `tasks.Register(l, ctx)(t Task)` (Task iface: `Run()`, `SleepTime() time.Duration`); `tasks.NewRecomputeTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *RecomputeTask`.
- Env (parsed by leaderconfig.go): `RANKINGS_LEADER_ELECTION_ENABLED` (default true), `RANKINGS_LEADER_TTL`, `RANKINGS_LEADER_REFRESH`, `RANKINGS_LEADER_BACKOFF`.

- [x] **Step 1: Copy the ticker registry**

Copy `services/atlas-monsters/atlas.com/monsters/tasks/task.go` verbatim to `services/atlas-rankings/atlas.com/rankings/tasks/task.go` (package `tasks`; `Task` interface + `Register` — service-agnostic).

- [x] **Step 2: Write the failing scheduler tests**

Create `services/atlas-rankings/atlas.com/rankings/tasks/recompute_test.go`:

```go
package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"atlas-rankings/ranking"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type fakeProcessor struct {
	due          bool
	dueErr       error
	recomputeErr error
	recomputed   *int
}

func (f fakeProcessor) ByCharacterIdProvider(uint32) model.Provider[ranking.Model] {
	return func() (ranking.Model, error) { return ranking.Model{}, nil }
}
func (f fakeProcessor) GetByCharacterId(uint32) (ranking.Model, error) { return ranking.Model{}, nil }
func (f fakeProcessor) ByCharacterIdsProvider([]uint32) model.Provider[[]ranking.Model] {
	return func() ([]ranking.Model, error) { return nil, nil }
}
func (f fakeProcessor) GetByCharacterIds([]uint32) ([]ranking.Model, error) { return nil, nil }
func (f fakeProcessor) IsDue(time.Duration, time.Time) (bool, error)        { return f.due, f.dueErr }
func (f fakeProcessor) Recompute(time.Time) error {
	if f.recomputeErr != nil {
		return f.recomputeErr
	}
	*f.recomputed++
	return nil
}
func (f fakeProcessor) WithCharacterSupplier(ranking.CharacterSupplier) ranking.Processor {
	return f
}

func testTenants(t *testing.T, n int) []tenant.Model {
	t.Helper()
	ts := make([]tenant.Model, 0, n)
	for i := 0; i < n; i++ {
		tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
		if err != nil {
			t.Fatalf("tenant: %v", err)
		}
		ts = append(ts, tm)
	}
	return ts
}

func TestRunSkipsFailingTenantAndContinues(t *testing.T) {
	ts := testTenants(t, 3)
	countA, countC := 0, 0

	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(ctx context.Context) ranking.Processor {
			tm := tenant.MustFromContext(ctx)
			switch tm.Id() {
			case ts[0].Id():
				return fakeProcessor{due: true, recomputed: &countA}
			case ts[1].Id():
				return fakeProcessor{due: true, recomputeErr: errors.New("boom"), recomputed: new(int)}
			default:
				return fakeProcessor{due: true, recomputed: &countC}
			}
		},
	}

	task.Run()

	if countA != 1 || countC != 1 {
		t.Fatalf("tenant B failure must not stop others: A=%d C=%d", countA, countC)
	}
}

func TestRunSkipsNotDueTenants(t *testing.T) {
	ts := testTenants(t, 1)
	count := 0
	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return ts, nil },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(context.Context) ranking.Processor {
			return fakeProcessor{due: false, recomputed: &count}
		},
	}
	task.Run()
	if count != 0 {
		t.Fatalf("not-due tenant must not recompute, got %d", count)
	}
}

func TestRunToleratesTenantEnumerationFailure(t *testing.T) {
	task := &RecomputeTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: time.Minute,
		tenants:  func() ([]tenant.Model, error) { return nil, errors.New("tenants down") },
		intervalFor: func(context.Context, uuid.UUID) time.Duration {
			return time.Hour
		},
		processorFor: func(context.Context) ranking.Processor {
			t.Fatal("must not construct a processor when enumeration fails")
			return nil
		},
	}
	task.Run() // must not panic
	if task.SleepTime() != time.Minute {
		t.Fatalf("SleepTime = %v, want 1m", task.SleepTime())
	}
}
```

- [x] **Step 3: Run tests to verify they fail**

Run: `go test ./tasks/ -v`
Expected: FAIL — `RecomputeTask` undefined.

- [x] **Step 4: Write recompute.go**

Create `services/atlas-rankings/atlas.com/rankings/tasks/recompute.go`:

```go
package tasks

import (
	"context"
	"time"

	"atlas-rankings/configuration"
	"atlas-rankings/ranking"
	tenantclient "atlas-rankings/tenant"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RecomputeTask ticks every interval (60s base tick), re-enumerates tenants
// and re-reads each tenant's configured cadence on EVERY tick — never a
// boot-time snapshot — so new tenants and config changes take effect without
// a redeploy, with staleness bounded by one tick.
type RecomputeTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	tenants      func() ([]tenant.Model, error)
	intervalFor  func(ctx context.Context, tenantId uuid.UUID) time.Duration
	processorFor func(ctx context.Context) ranking.Processor
}

func NewRecomputeTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *RecomputeTask {
	return &RecomputeTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		tenants: func() ([]tenant.Model, error) {
			return tenantclient.NewProcessor(l, ctx).GetAll()
		},
		intervalFor: func(tctx context.Context, tenantId uuid.UUID) time.Duration {
			return configuration.GetRecomputeInterval(l, tctx)(tenantId)
		},
		processorFor: func(tctx context.Context) ranking.Processor {
			return ranking.NewProcessor(l, tctx, db)
		},
	}
}

func (t *RecomputeTask) SleepTime() time.Duration {
	return t.interval
}

func (t *RecomputeTask) Run() {
	ts, err := t.tenants()
	if err != nil {
		t.l.WithError(err).Warnf("Unable to enumerate tenants; skipping rankings recompute tick.")
		return
	}

	for _, ten := range ts {
		tctx := tenant.WithContext(t.ctx, ten)
		interval := t.intervalFor(tctx, ten.Id())
		p := t.processorFor(tctx)

		now := time.Now()
		due, err := p.IsDue(interval, now)
		if err != nil {
			t.l.WithError(err).WithField("tenant", ten.Id().String()).Warnf("Unable to determine rankings cycle due-ness; skipping tenant.")
			continue
		}
		if !due {
			continue
		}
		if err := p.Recompute(now); err != nil {
			t.l.WithError(err).WithField("tenant", ten.Id().String()).Errorf("Rankings recompute failed; continuing with remaining tenants.")
			continue
		}
	}
}
```

- [x] **Step 5: Run tests to verify they pass**

Run: `go test -race ./tasks/ -v`
Expected: PASS.

- [x] **Step 6: Add leaderconfig.go and wire main.go**

Copy `services/atlas-monsters/atlas.com/monsters/leaderconfig.go` verbatim to `services/atlas-rankings/atlas.com/rankings/leaderconfig.go`, then rename the four env constants:

```go
const (
	envLeaderEnabled = "RANKINGS_LEADER_ELECTION_ENABLED"
	envLeaderTTL     = "RANKINGS_LEADER_TTL"
	envLeaderRefresh = "RANKINGS_LEADER_REFRESH"
	envLeaderBackoff = "RANKINGS_LEADER_BACKOFF"
	...
)
```

(Keep the default constants and all four helper functions — `leaderEnabled`, `leaderTTL`, `leaderRefresh`, `leaderBackoff`, `parseDurationInRange` — unchanged.)

Modify `services/atlas-rankings/atlas.com/rankings/main.go`. Final content:

```go
package main

import (
	"atlas-rankings/logger"
	"atlas-rankings/ranking"
	"atlas-rankings/tasks"
	"context"
	"os"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"github.com/sirupsen/logrus"
)

const serviceName = "atlas-rankings"

const baseTick = time.Minute

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{baseUrl: "", prefix: "/api/"}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(ranking.Migration))
	rc := atlas.Connect(l)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(ranking.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountReadiness("/readyz", func() bool { return true })).
		Run()

	registerRecompute := func(l logrus.FieldLogger, ctx context.Context) {
		tasks.Register(l, ctx)(tasks.NewRecomputeTask(l, ctx, db, baseTick))
	}

	if leaderEnabled(l) {
		ttl := leaderTTL(l)
		le, err := lock.New(rc, "rankings-recompute",
			lock.WithTTL(ttl),
			lock.WithRefreshInterval(leaderRefresh(l, ttl)),
			lock.WithBackoff(leaderBackoff(l)),
			lock.WithLogger(l),
		)
		if err != nil {
			l.WithError(err).Fatal("Unable to construct LeaderElection.")
		}
		go func() {
			err := le.Run(tdm.Context(), func(leaderCtx context.Context) {
				registerRecompute(l, leaderCtx)
				<-leaderCtx.Done()
			})
			if err != nil {
				l.WithError(err).Errorf("LeaderElection.Run exited with error.")
			}
		}()
	} else {
		l.Warnf("RANKINGS_LEADER_ELECTION_ENABLED=false — recompute runs unconditionally on this pod.")
		registerRecompute(l, tdm.Context())
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

- [x] **Step 7: Full module verification**

Run from module root:

```bash
go mod tidy
go test -race ./...
go vet ./...
go build ./...
```

Expected: all clean. (`go mod tidy` now pulls atlas-lock/atlas-redis into requires.)

- [x] **Step 8: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings
git commit -m "feat(task-143): leader-gated per-tenant recompute scheduler"
```

---

### Task 9: deployment scaffolding (services.json, bake, k8s, ingress, Bruno, README)

**Files:**
- Modify: `.github/config/services.json` (insert entry after `atlas-quest`)
- Modify: `docker-bake.hcl` (insert into `go_services` after `"atlas-quest"`)
- Create: `deploy/k8s/base/atlas-rankings.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml` (insert `- atlas-rankings.yaml` between `atlas-quest.yaml` and `atlas-rates.yaml`)
- Modify: `deploy/k8s/overlays/main/patches/db-name-suffix.yaml` (add block, alphabetical position)
- Modify: `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml` (add block, alphabetical position)
- Modify: `deploy/shared/routes.conf` (add location block, alphabetical among the `/api/*` blocks)
- Create: `services/atlas-rankings/.bruno/bruno.json`, `collection.bru`, `environments/*.bru`, `Get Rankings For Characters.bru`, `Get Ranking For Character.bru`
- Create: `services/atlas-rankings/README.md`

**Interfaces:**
- Consumes: the complete Task 1–8 module.
- Produces: bakeable image target `atlas-rankings`, routable `/api/rankings/*`, deployable manifests.

- [x] **Step 1: Register the service in services.json and docker-bake.hcl**

In `.github/config/services.json`, insert after the `atlas-quest` entry (alphabetical: rankings < rates):

```json
{"name": "atlas-rankings", "type": "go-service", "path": "services/atlas-rankings", "module_path": "services/atlas-rankings/atlas.com/rankings", "docker_image": "ghcr.io/chronicle20/atlas-rankings/atlas-rankings", "docker_context": "."},
```

(Match the surrounding entries' JSON formatting exactly — one object per array element, same key order.)

In `docker-bake.hcl`, inside `go_services = [...]`, insert after `"atlas-quest",`:

```hcl
  "atlas-rankings",
```

(Both lists are hand-synced — HCL cannot read services.json.)

- [x] **Step 2: Write the k8s base manifest**

Create `deploy/k8s/base/atlas-rankings.yaml`:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-rankings
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atlas-rankings
  template:
    metadata:
      labels:
        app: atlas-rankings
    spec:
      containers:
      - name: rankings
        image: ghcr.io/chronicle20/atlas-rankings/atlas-rankings:latest
        ports:
        - containerPort: 8080
        readinessProbe:
          httpGet:
            path: /api/readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: DB_NAME
          value: "atlas-rankings"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: DB_USER
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: DB_PASSWORD
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-rankings
spec:
  selector:
    app: atlas-rankings
  ports:
  - protocol: TCP
    port: 8080
```

(`REDIS_URL` comes from the `atlas-env` configmap; no per-service Redis env needed. Probe path is `/api/readyz` — the base-path bug pattern.)

Register it in `deploy/k8s/base/kustomization.yaml`: insert `  - atlas-rankings.yaml` between `  - atlas-quest.yaml` and `  - atlas-rates.yaml`.

- [x] **Step 3: Add DB_NAME suffix overlay patches**

In `deploy/k8s/overlays/main/patches/db-name-suffix.yaml`, insert at the alphabetical position (after the `atlas-quest` block, before `atlas-rates`):

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-rankings
spec:
  template:
    spec:
      containers:
        - name: rankings
          env:
            - name: DB_NAME
              value: "atlas-rankings-main"
```

In `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`, same position:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-rankings
spec:
  template:
    spec:
      containers:
        - name: rankings
          env:
            - name: DB_NAME
              value: "atlas-rankings-PLACEHOLDER_ATLAS_ENV"
```

- [x] **Step 4: Add the ingress route and sync**

In `deploy/shared/routes.conf`, insert at the alphabetical position among the simple `/api/*` blocks:

```nginx
location ~ ^/api/rankings(/.*)?$ {
  set $u "atlas-rankings:8080";
  proxy_pass http://$u$request_uri;
}
```

Then run:

```bash
./deploy/scripts/sync-k8s-ingress-routes.sh
```

and stage whatever files it regenerates.

- [x] **Step 5: Bruno collection**

Create `services/atlas-rankings/.bruno/bruno.json`:

```json
{
  "version": "1",
  "name": "atlas-rankings",
  "type": "collection",
  "ignore": [
    "node_modules",
    ".git"
  ]
}
```

Create `services/atlas-rankings/.bruno/collection.bru`:

```
headers {
  TENANT_ID: {{TENANT_ID}}
  REGION: {{REGION}}
  MAJOR_VERSION: {{MAJOR_VERSION}}
  MINOR_VERSION: {{MINOR_VERSION}}
}
```

Copy the three environment files verbatim from `services/atlas-gachapons/.bruno/environments/` (`Atlas - K3S.bru`, `Local.bru`, `Local Debug.bru`) into `services/atlas-rankings/.bruno/environments/`.

Create `services/atlas-rankings/.bruno/Get Rankings For Characters.bru`:

```
meta {
  name: Get Rankings For Characters
  type: http
  seq: 1
}

get {
  url: {{scheme}}://{{host}}:{{port}}/api/rankings/characters?ids=1,2,3
  body: none
  auth: inherit
}
```

Create `services/atlas-rankings/.bruno/Get Ranking For Character.bru`:

```
meta {
  name: Get Ranking For Character
  type: http
  seq: 2
}

get {
  url: {{scheme}}://{{host}}:{{port}}/api/rankings/characters/1
  body: none
  auth: inherit
}
```

- [x] **Step 6: Service README**

Create `services/atlas-rankings/README.md`:

```markdown
# atlas-rankings

Computes per-world character rankings (overall and per job category) for
each tenant on a configurable cadence and serves them over REST. Consumed
by atlas-login to populate the character-select info board (rank, job rank,
movement arrows).

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/rankings/characters?ids={id},{id},…` | Bulk fetch. One `rankings` resource per requested character id that has an entry; unknown ids are omitted (callers default to zeros). Empty/unparseable ids → 400. |
| GET | `/api/rankings/characters/{characterId}` | Single fetch; 404 when no entry exists. |

Resource attributes: `worldId`, `rank`, `rankMove`, `jobRank`, `jobRankMove`
(moves are signed: positive = moved up), `computedAt`. Tenant headers
required. No write endpoints — rankings are computed, never client-mutated.

## Recompute

A 60s base ticker (leader-gated via libs/atlas-lock lease
`rankings-recompute`, so the standard 2 replicas never double-compute)
re-enumerates tenants from atlas-tenants each tick and runs a recompute for
every tenant whose configured interval has elapsed. Each cycle:

1. `GET /characters` from atlas-character (full tenant scan).
2. Exclude `gm > 0` characters entirely (not ranked, not counted).
3. Per world: order by `level DESC, experience DESC, characterId ASC`
   (1-based, unique); job rank is the same order restricted to
   `jobId / 100` categories.
4. Moves are `previousRank − newRank` against the prior cycle; first-seen
   characters move 0.
5. Batch upsert on `(tenant_id, character_id)`, then prune rows not
   restamped by this cycle (deleted/became-GM characters drop out).

The cycle is idempotent and convergent; a crash mid-cycle is repaired by the
next run (moves may read 0 for one cycle). One tenant's failure is logged
and skipped, never fatal.

## Configuration

Per-tenant cadence lives in atlas-tenants:
`GET/POST/PATCH/DELETE /api/tenants/{tenantId}/configurations/rankings` with
attribute `recomputeIntervalMinutes`. Absent/zero → default 60 minutes. The
config is re-read every tick — changes apply without a redeploy.

Environment: standard DB_* and REST_PORT; `REDIS_URL` (leader lease);
`CHARACTERS_SERVICE_URL` / `TENANTS_SERVICE_URL` with `BASE_SERVICE_URL`
fallback; `RANKINGS_LEADER_ELECTION_ENABLED|TTL|REFRESH|BACKOFF` (defaults
true/30s/TTL÷3/5s).

## Scaling note

Recompute cost scales with total tenant character count: one full
`GET /characters` read per tenant per cycle and an O(n log n) in-memory
sort. Acceptable at tens of thousands of characters; adopt list-endpoint
pagination (task-117) as a drop-in improvement if populations outgrow it.
```

- [x] **Step 7: Verify bake and manifests**

Run from the repo root:

```bash
docker buildx bake atlas-rankings
kubectl kustomize deploy/k8s/overlays/main > /dev/null
kubectl kustomize deploy/k8s/overlays/pr > /dev/null
```

Expected: image builds cleanly; both overlays render without errors. If bake fails on a missing lib COPY, the shared Dockerfile is missing nothing for this task (no new libs) — re-check services.json/docker-bake entries instead.

- [x] **Step 8: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy services/atlas-rankings/.bruno services/atlas-rankings/README.md
git commit -m "feat(task-143): atlas-rankings deployment scaffolding (bake, k8s, ingress, bruno)"
```

---

### Task 10: atlas-tenants `rankings` configuration resource

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (append RankingsRestModel + helpers)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/kafka.go` (event types + provider)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/provider.go` (GetRankingsProvider)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/processor.go` (interface + impl)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/mock/processor.go` (matching funcs — MANDATORY)
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/resource.go` (handlers + routes)
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/rankings_test.go`

**Interfaces:**
- Produces REST routes on atlas-tenants: `GET/POST/PATCH/DELETE /tenants/{tenantId}/configurations/rankings`, resource type `rankings`, attribute `recomputeIntervalMinutes` (uint32). GET 404s when unconfigured (atlas-rankings' client then defaults to 60m).
- Produces `Processor` additions: `GetRankings(tenantId) (map[string]interface{}, error)`, `RankingsProvider(tenantId) model.Provider[map[string]interface{}]`, `CreateRankings(mb)(tenantId)(rankings) (Model, error)` + `CreateRankingsAndEmit`, `UpdateRankings(mb)(tenantId)(rankings) (Model, error)` + `UpdateRankingsAndEmit`, `DeleteRankings(mb)(tenantId) error` + `DeleteRankingsAndEmit`.
- Storage: one `configurations` row per tenant with `resource_name = "rankings"`, `resource_data = {"data": {single object}}` (single-object variant of the routes/vessels pattern).

- [x] **Step 1: Write the failing processor tests**

Create `services/atlas-tenants/atlas.com/tenants/configuration/rankings_test.go`:

```go
package configuration

import (
	"context"
	"errors"
	"testing"

	"atlas-tenants/kafka/message"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func rankingsTestDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := MigrateEntities(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func rankingsAttrs(minutes float64) map[string]interface{} {
	return map[string]interface{}{
		"type": "rankings",
		"id":   uuid.New().String(),
		"attributes": map[string]interface{}{
			"recomputeIntervalMinutes": minutes,
		},
	}
}

func TestRankingsCreateGetRoundTrip(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := p.GetRankings(tenantId)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	attrs, _ := got["attributes"].(map[string]interface{})
	if v, _ := attrs["recomputeIntervalMinutes"].(float64); v != 15 {
		t.Fatalf("interval = %v, want 15", attrs["recomputeIntervalMinutes"])
	}
}

func TestRankingsGetAbsentIsNotFound(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	_, err := p.GetRankings(uuid.New())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRankingsUpdateReplaces(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.UpdateRankings(mb)(tenantId)(rankingsAttrs(45)); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := p.GetRankings(tenantId)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	attrs, _ := got["attributes"].(map[string]interface{})
	if v, _ := attrs["recomputeIntervalMinutes"].(float64); v != 45 {
		t.Fatalf("interval = %v, want 45", attrs["recomputeIntervalMinutes"])
	}
}

func TestRankingsUpdateAbsentFails(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)

	_, err := p.UpdateRankings(message.NewBuffer())(uuid.New())(rankingsAttrs(45))
	if err == nil {
		t.Fatal("update of absent config must fail")
	}
}

func TestRankingsDelete(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)
	tenantId := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantId)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.DeleteRankings(mb)(tenantId); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := p.GetRankings(tenantId); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not-found after delete, got %v", err)
	}
}

func TestRankingsTenantsIsolated(t *testing.T) {
	db := rankingsTestDb(t)
	p := NewProcessor(logrus.New(), context.Background(), db)
	tenantA := uuid.New()
	tenantB := uuid.New()
	mb := message.NewBuffer()

	if _, err := p.CreateRankings(mb)(tenantA)(rankingsAttrs(15)); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, err := p.CreateRankings(mb)(tenantB)(rankingsAttrs(30)); err != nil {
		t.Fatalf("create B: %v", err)
	}

	gotA, err := p.GetRankings(tenantA)
	if err != nil {
		t.Fatalf("get A: %v", err)
	}
	attrsA, _ := gotA["attributes"].(map[string]interface{})
	if v, _ := attrsA["recomputeIntervalMinutes"].(float64); v != 15 {
		t.Fatalf("tenant A interval = %v, want 15", v)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-tenants/atlas.com/tenants`): `go test ./configuration/ -run TestRankings -v`
Expected: FAIL — `CreateRankings`, `GetRankings`, etc. undefined.

- [x] **Step 3: rest.go additions**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (mirroring the Route equivalents; `uuid` may need adding to imports):

```go
// RankingsRestModel is the JSON:API resource for the rankings configuration
type RankingsRestModel struct {
	Id                       string `json:"-"`
	RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
}

// GetID returns the resource ID
func (r RankingsRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RankingsRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RankingsRestModel) GetName() string {
	return "rankings"
}

// TransformRankings converts a map[string]interface{} to a RankingsRestModel
func TransformRankings(data map[string]interface{}) (RankingsRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	interval := uint32(0)
	if val, ok := attributes["recomputeIntervalMinutes"].(float64); ok {
		interval = uint32(val)
	}

	return RankingsRestModel{Id: id, RecomputeIntervalMinutes: interval}, nil
}

// ExtractRankings converts a RankingsRestModel to a map[string]interface{}
func ExtractRankings(r RankingsRestModel) (map[string]interface{}, error) {
	id := r.Id
	if id == "" {
		id = uuid.New().String()
	}
	return map[string]interface{}{
		"type": "rankings",
		"id":   id,
		"attributes": map[string]interface{}{
			"recomputeIntervalMinutes": r.RecomputeIntervalMinutes,
		},
	}, nil
}

// CreateSingleRankingsJsonData creates a JSON:API compliant data structure
// for the single-object rankings configuration
func CreateSingleRankingsJsonData(rankings map[string]interface{}) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{"data": rankings})
}
```

- [x] **Step 4: kafka.go additions**

Append to the `const` block in `services/atlas-tenants/atlas.com/tenants/configuration/kafka.go`:

```go
	EventTypeRankingsCreated = "RANKINGS_CREATED"
	EventTypeRankingsUpdated = "RANKINGS_UPDATED"
	EventTypeRankingsDeleted = "RANKINGS_DELETED"
```

Append the provider:

```go
// CreateRankingsStatusEventProvider creates a provider for rankings configuration status events
func CreateRankingsStatusEventProvider(tenantId uuid.UUID, eventType string, rankingsId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "rankings",
		ResourceId:   rankingsId,
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [x] **Step 5: provider.go addition**

Append to `services/atlas-tenants/atlas.com/tenants/configuration/provider.go`:

```go
// GetRankingsProvider returns a provider for the tenant's single-object
// rankings configuration
func GetRankingsProvider(tenantID uuid.UUID) func(db *gorm.DB) model.Provider[map[string]interface{}] {
	return func(db *gorm.DB) model.Provider[map[string]interface{}] {
		entityProvider := GetByTenantIdAndResourceNameProvider(tenantID, "rankings")(db)
		return model.Map(func(e Entity) (map[string]interface{}, error) {
			var resourceData map[string]interface{}
			if err := json.Unmarshal(e.ResourceData, &resourceData); err != nil {
				return nil, err
			}
			if data, ok := resourceData["data"].(map[string]interface{}); ok {
				return data, nil
			}
			return nil, gorm.ErrRecordNotFound
		})(entityProvider)
	}
}
```

- [x] **Step 6: processor.go additions**

Add to the `Processor` interface (after the vessel block, matching comment style):

```go
	// Rankings operations
	// CreateRankings creates (or replaces) the tenant's rankings configuration
	CreateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error)
	// CreateRankingsAndEmit creates the rankings configuration and emits events
	CreateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error)
	// UpdateRankings updates the existing rankings configuration
	UpdateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error)
	// UpdateRankingsAndEmit updates the rankings configuration and emits events
	UpdateRankingsAndEmit(tenantId uuid.UUID, rankings map[string]interface{}) (Model, error)
	// DeleteRankings deletes the rankings configuration
	DeleteRankings(mb *message.Buffer) func(tenantId uuid.UUID) error
	// DeleteRankingsAndEmit deletes the rankings configuration and emits events
	DeleteRankingsAndEmit(tenantId uuid.UUID) error
	// GetRankings gets the rankings configuration for a tenant
	GetRankings(tenantId uuid.UUID) (map[string]interface{}, error)
	// RankingsProvider returns a provider for the rankings configuration
	RankingsProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}]
```

Implementation (append to processor.go; `AndEmit` wrappers must mirror the exact `message.Emit`-based shape used by `CreateRouteAndEmit` in this file):

```go
// CreateRankings creates (or replaces) the tenant's rankings configuration
func (p *ProcessorImpl) CreateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
		return func(rankings map[string]interface{}) (Model, error) {
			rankingsId := ""
			if id, ok := rankings["id"].(string); ok {
				rankingsId = id
			}

			resourceData, err := CreateSingleRankingsJsonData(rankings)
			if err != nil {
				return Model{}, err
			}

			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rankings")(p.db)
			existing, err := existingProvider()
			if err == nil {
				existing.ResourceData = resourceData
				if err := UpdateConfiguration(p.db, existing); err != nil {
					return Model{}, err
				}
				m, err := Make(existing)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsUpdated, rankingsId)); err != nil {
					return Model{}, err
				}
				return m, nil
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				entity := Entity{
					ID:           uuid.New(),
					TenantId:     tenantId,
					ResourceName: "rankings",
					ResourceData: resourceData,
				}
				if err := CreateConfiguration(p.db, entity); err != nil {
					return Model{}, err
				}
				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}
				if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsCreated, rankingsId)); err != nil {
					return Model{}, err
				}
				return m, nil
			}
			return Model{}, err
		}
	}
}

// UpdateRankings updates the existing rankings configuration
func (p *ProcessorImpl) UpdateRankings(mb *message.Buffer) func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
	return func(tenantId uuid.UUID) func(rankings map[string]interface{}) (Model, error) {
		return func(rankings map[string]interface{}) (Model, error) {
			existingProvider := GetByTenantIdAndResourceNameProvider(tenantId, "rankings")(p.db)
			existing, err := existingProvider()
			if err != nil {
				return Model{}, err
			}

			rankingsId := ""
			if id, ok := rankings["id"].(string); ok {
				rankingsId = id
			}

			resourceData, err := CreateSingleRankingsJsonData(rankings)
			if err != nil {
				return Model{}, err
			}
			existing.ResourceData = resourceData
			if err := UpdateConfiguration(p.db, existing); err != nil {
				return Model{}, err
			}
			m, err := Make(existing)
			if err != nil {
				return Model{}, err
			}
			if err := mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsUpdated, rankingsId)); err != nil {
				return Model{}, err
			}
			return m, nil
		}
	}
}

// DeleteRankings deletes the rankings configuration
func (p *ProcessorImpl) DeleteRankings(mb *message.Buffer) func(tenantId uuid.UUID) error {
	return func(tenantId uuid.UUID) error {
		if _, err := DeleteConfigurationByResourceName(p.db, tenantId, "rankings"); err != nil {
			return err
		}
		return mb.Put(EventTopicConfigurationStatus, CreateRankingsStatusEventProvider(tenantId, EventTypeRankingsDeleted, ""))
	}
}

// GetRankings gets the rankings configuration for a tenant
func (p *ProcessorImpl) GetRankings(tenantId uuid.UUID) (map[string]interface{}, error) {
	return p.RankingsProvider(tenantId)()
}

// RankingsProvider returns a provider for the rankings configuration
func (p *ProcessorImpl) RankingsProvider(tenantId uuid.UUID) model.Provider[map[string]interface{}] {
	return GetRankingsProvider(tenantId)(p.db)
}
```

For `CreateRankingsAndEmit` / `UpdateRankingsAndEmit` / `DeleteRankingsAndEmit`: copy the exact wrapper shape of `CreateRouteAndEmit` / `UpdateRouteAndEmit` / `DeleteRouteAndEmit` from this file (they wrap the buffered variant in `message.Emit(p.p)`), substituting the Rankings methods and dropping the id parameter where the Route version takes one.

- [x] **Step 7: mock additions (mandatory — tests fail otherwise)**

In `services/atlas-tenants/atlas.com/tenants/configuration/mock/processor.go`, add to `ProcessorMock`:

```go
	// Rankings operations
	CreateRankingsFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(rankings map[string]interface{}) (configuration.Model, error)
	CreateRankingsAndEmitFunc func(tenantID uuid.UUID, rankings map[string]interface{}) (configuration.Model, error)
	UpdateRankingsFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) func(rankings map[string]interface{}) (configuration.Model, error)
	UpdateRankingsAndEmitFunc func(tenantID uuid.UUID, rankings map[string]interface{}) (configuration.Model, error)
	DeleteRankingsFunc        func(mb *message.Buffer) func(tenantID uuid.UUID) error
	DeleteRankingsAndEmitFunc func(tenantID uuid.UUID) error
	GetRankingsFunc           func(tenantID uuid.UUID) (map[string]interface{}, error)
	RankingsProviderFunc      func(tenantID uuid.UUID) model.Provider[map[string]interface{}]
```

and the eight method implementations following the file's existing nil-check pattern (return the func field's result when set; otherwise zero values / no-op closures, exactly like the Route methods do).

- [x] **Step 8: resource.go handlers + routes**

Append handlers to `services/atlas-tenants/atlas.com/tenants/configuration/resource.go`:

```go
// GetRankingsHandler handles GET /tenants/{tenantId}/configurations/rankings
func GetRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				rankings, err := processor.GetRankings(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := TransformRankings(rankings)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreateRankingsHandler handles POST /tenants/{tenantId}/configurations/rankings
func CreateRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rankings, err := ExtractRankings(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract rankings data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.CreateRankingsAndEmit(tenantId, rankings); err != nil {
					d.Logger().WithError(err).Error("Failed to create rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				created, err := processor.GetRankings(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, err := TransformRankings(created)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateRankingsHandler handles PATCH /tenants/{tenantId}/configurations/rankings
func UpdateRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rankings, err := ExtractRankings(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract rankings data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.UpdateRankingsAndEmit(tenantId, rankings); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to update rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				updated, err := processor.GetRankings(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get updated rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, err := TransformRankings(updated)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// DeleteRankingsHandler handles DELETE /tenants/{tenantId}/configurations/rankings
func DeleteRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				if err := processor.DeleteRankingsAndEmit(tenantId); err != nil {
					d.Logger().WithError(err).Error("Failed to delete rankings configuration")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
```

In `RegisterRoutes`, add after the instance-route endpoints:

```go
			// Rankings endpoints
			registerRankingsInputHandler := rest.RegisterInputHandler[RankingsRestModel](l)(si)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerHandler("get_rankings_config", GetRankingsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerRankingsInputHandler("create_rankings_config", CreateRankingsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerRankingsInputHandler("update_rankings_config", UpdateRankingsHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerHandler("delete_rankings_config", DeleteRankingsHandler(db))).Methods(http.MethodDelete)
```

(Declare `registerRankingsInputHandler` alongside the existing `registerVesselInputHandler` declarations at the top of the function instead if that matches the file layout better.)

- [x] **Step 9: Run tests to verify they pass**

Run from `services/atlas-tenants/atlas.com/tenants`:

```bash
go test -race ./... 
go vet ./...
go build ./...
```

Expected: TestRankings* PASS and the whole existing suite stays green (the mock compile-time check `var _ configuration.Processor = (*ProcessorMock)(nil)` catches any interface mismatch).

- [x] **Step 10: Commit**

```bash
git add services/atlas-tenants
git commit -m "feat(task-143): rankings configuration resource in atlas-tenants"
```

---

### Task 11: atlas-login model rank fields

**Files:**
- Modify: `services/atlas-login/atlas.com/login/character/model.go`
- Test: `services/atlas-login/atlas.com/login/character/model_rank_test.go`

**Interfaces:**
- Produces: `Model` fields `rank uint32`, `rankMove int32`, `jobRank uint32`, `jobRankMove int32`; getters `Rank() uint32`, `RankMove() uint32` (two's-complement pass-through of the signed move), `JobRank() uint32`, `JobRankMove() uint32`; Builder setters `SetRank(uint32)`, `SetRankMove(int32)`, `SetJobRank(uint32)`, `SetJobRankMove(int32)`; `ToBuilder()` round-trips all four. The packet writer (`socket/writer/character_list.go:61`) already consumes these getters — no writer change.

- [x] **Step 1: Write the failing test**

Create `services/atlas-login/atlas.com/login/character/model_rank_test.go`:

```go
package character

import "testing"

func TestRankBuilderRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetRank(17).
		SetRankMove(2).
		SetJobRank(4).
		SetJobRankMove(-1).
		Build()

	if m.Rank() != 17 || m.JobRank() != 4 {
		t.Fatalf("ranks lost: rank=%d jobRank=%d", m.Rank(), m.JobRank())
	}
	if m.RankMove() != 2 {
		t.Fatalf("rankMove = %d, want 2", m.RankMove())
	}

	// The packet field is uint32; the v83 client reinterprets it signed
	// (abs + sign branch). -1 must pass through as two's complement.
	if m.JobRankMove() != 0xFFFFFFFF {
		t.Fatalf("jobRankMove = %#x, want 0xFFFFFFFF", m.JobRankMove())
	}

	rt := m.ToBuilder().Build()
	if rt.Rank() != 17 || rt.RankMove() != 2 || rt.JobRank() != 4 || rt.JobRankMove() != 0xFFFFFFFF {
		t.Fatalf("ToBuilder dropped rank fields: %+v", rt)
	}
}

func TestRankDefaultsToZero(t *testing.T) {
	m := NewBuilder().SetId(1).Build()
	if m.Rank() != 0 || m.RankMove() != 0 || m.JobRank() != 0 || m.JobRankMove() != 0 {
		t.Fatal("unranked character must render all-zero rank fields")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run (from `services/atlas-login/atlas.com/login`): `go test ./character/ -run TestRank -v`
Expected: FAIL — `SetRank` undefined.

- [x] **Step 3: Modify model.go**

In `services/atlas-login/atlas.com/login/character/model.go`:

1. Add to the `Model` struct (after `gm int`):

```go
	rank               uint32
	rankMove           int32
	jobRank            uint32
	jobRankMove        int32
```

2. Replace the four hardcoded getters (`model.go:55-69`):

```go
func (m Model) Rank() uint32 {
	return m.rank
}

// RankMove passes the signed move through as two's complement — the packet
// lib field is uint32 and the client reinterprets it signed (abs + sign).
func (m Model) RankMove() uint32 {
	return uint32(m.rankMove)
}

func (m Model) JobRank() uint32 {
	return m.jobRank
}

func (m Model) JobRankMove() uint32 {
	return uint32(m.jobRankMove)
}
```

3. Add the same four fields to the `Builder` struct, add the four fields to `ToBuilder()`'s literal (`rank: m.rank,` etc.), add them to `Build()`'s literal (`rank: b.rank,` etc.), and add setters alongside the existing ones:

```go
func (b *Builder) SetRank(v uint32) *Builder        { b.rank = v; return b }
func (b *Builder) SetRankMove(v int32) *Builder     { b.rankMove = v; return b }
func (b *Builder) SetJobRank(v uint32) *Builder     { b.jobRank = v; return b }
func (b *Builder) SetJobRankMove(v int32) *Builder  { b.jobRankMove = v; return b }
```

- [x] **Step 4: Run tests to verify they pass**

Run: `go test -race ./character/ -v`
Expected: PASS (new tests plus the package's existing suite).

- [x] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/character
git commit -m "feat(task-143): real rank fields on login character model"
```

---

### Task 12: atlas-login ranking client + character-list decoration

**Files:**
- Create: `services/atlas-login/atlas.com/login/ranking/rest.go`
- Create: `services/atlas-login/atlas.com/login/ranking/model.go`
- Create: `services/atlas-login/atlas.com/login/ranking/requests.go`
- Create: `services/atlas-login/atlas.com/login/ranking/processor.go`
- Modify: `services/atlas-login/atlas.com/login/character/processor.go`
- Test: `services/atlas-login/atlas.com/login/character/processor_rank_test.go`

**Interfaces:**
- Consumes: Task 11 builder setters; atlas-rankings bulk endpoint (Task 7).
- Produces: `ranking.Model` with `CharacterId() uint32`, `Rank() uint32`, `RankMove() int32`, `JobRank() uint32`, `JobRankMove() int32`; `ranking.NewProcessor(l, ctx).GetByCharacterIds(ids []uint32) ([]Model, error)` with a 2s per-call timeout; `character.MergeRankings(cs []Model, rs []ranking.Model) []Model` (exported for tests).
- Env: `RANKINGS_SERVICE_URL` with `BASE_SERVICE_URL` fallback. Do NOT add either to any configmap.

- [x] **Step 1: Write the failing decoration tests**

Create `services/atlas-login/atlas.com/login/character/processor_rank_test.go`:

```go
package character

import (
	"context"
	"errors"
	"testing"

	"atlas-login/ranking"

	"github.com/sirupsen/logrus"
)

func rankingModel(t *testing.T, characterId uint32, rank uint32, rankMove int32) ranking.Model {
	t.Helper()
	rm := ranking.RestModel{Rank: rank, RankMove: rankMove, JobRank: rank, JobRankMove: rankMove}
	rm.Id = characterId
	m, err := ranking.Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

func TestMergeRankings(t *testing.T) {
	cs := []Model{
		NewBuilder().SetId(1).SetName("A").Build(),
		NewBuilder().SetId(2).SetName("B").Build(),
	}
	rs := []ranking.Model{rankingModel(t, 1, 17, -2)}

	got := MergeRankings(cs, rs)
	if len(got) != 2 {
		t.Fatalf("merge changed slice length: %d", len(got))
	}
	if got[0].Rank() != 17 || got[0].RankMove() != uint32(0xFFFFFFFE) {
		t.Fatalf("char 1 not decorated: rank=%d move=%#x", got[0].Rank(), got[0].RankMove())
	}
	if got[0].Name() != "A" {
		t.Fatalf("merge dropped unrelated fields: %+v", got[0])
	}
	if got[1].Rank() != 0 || got[1].RankMove() != 0 {
		t.Fatalf("char 2 without entry must stay zero: %+v", got[1])
	}
}

func TestDecorateRankingsFailsOpen(t *testing.T) {
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		rankings: func(ids []uint32) ([]ranking.Model, error) {
			return nil, errors.New("rankings unavailable")
		},
	}
	cs := []Model{NewBuilder().SetId(1).Build()}

	got := p.decorateRankings(cs)
	if len(got) != 1 || got[0].Rank() != 0 {
		t.Fatalf("fail-open must return originals with zero ranks: %+v", got)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./character/ -run 'TestMergeRankings|TestDecorateRankings' -v`
Expected: FAIL — package `atlas-login/ranking` missing, `MergeRankings` undefined.

- [x] **Step 3: Write the ranking client package**

Create `services/atlas-login/atlas.com/login/ranking/rest.go`:

```go
package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id          uint32    `json:"-"`
	WorldId     world.Id  `json:"worldId"`
	Rank        uint32    `json:"rank"`
	RankMove    int32     `json:"rankMove"`
	JobRank     uint32    `json:"jobRank"`
	JobRankMove int32     `json:"jobRankMove"`
	ComputedAt  time.Time `json:"computedAt"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		characterId: r.Id,
		rank:        r.Rank,
		rankMove:    r.RankMove,
		jobRank:     r.JobRank,
		jobRankMove: r.JobRankMove,
	}, nil
}
```

Create `services/atlas-login/atlas.com/login/ranking/model.go`:

```go
package ranking

type Model struct {
	characterId uint32
	rank        uint32
	rankMove    int32
	jobRank     uint32
	jobRankMove int32
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Rank() uint32        { return m.rank }
func (m Model) RankMove() int32     { return m.rankMove }
func (m Model) JobRank() uint32     { return m.jobRank }
func (m Model) JobRankMove() int32  { return m.jobRankMove }
```

Create `services/atlas-login/atlas.com/login/ranking/requests.go`:

```go
package ranking

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	Resource = "rankings/characters"
	ByIds    = Resource + "?ids=%s"

	// requestTimeout bounds the login-path call — login latency must never
	// ride on atlas-rankings health (FR-11). The lib default is 10s, far
	// too long for a fail-open decoration.
	requestTimeout = 2 * time.Second
)

func getBaseRequest() string {
	return requests.RootUrl("RANKINGS")
}

func requestByCharacterIds(ids []uint32) requests.Request[[]RestModel] {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.FormatUint(uint64(id), 10)
	}
	url := fmt.Sprintf(getBaseRequest()+ByIds, strings.Join(strs, ","))
	return func(l logrus.FieldLogger, ctx context.Context) ([]RestModel, error) {
		sd := requests.AddHeaderDecorator(requests.SpanHeaderDecorator(ctx))
		td := requests.AddHeaderDecorator(requests.TenantHeaderDecorator(ctx))
		return requests.MakeGetRequest[[]RestModel](url, sd, td, requests.SetTimeout(requestTimeout))(l, ctx)
	}
}
```

Create `services/atlas-login/atlas.com/login/ranking/processor.go`:

```go
package ranking

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// ByCharacterIdsProvider returns a provider for rankings of the given characters.
	ByCharacterIdsProvider(ids []uint32) model.Provider[[]Model]
	// GetByCharacterIds bulk-fetches rankings; characters without an entry are absent.
	GetByCharacterIds(ids []uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) ByCharacterIdsProvider(ids []uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterIds(ids), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterIds(ids []uint32) ([]Model, error) {
	return p.ByCharacterIdsProvider(ids)()
}
```

- [x] **Step 4: Wire the decoration into character/processor.go**

In `services/atlas-login/atlas.com/login/character/processor.go`:

1. Add imports `"atlas-login/ranking"`.
2. Add a field to `ProcessorImpl` and wire it in `NewProcessor`:

```go
type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	ip       *inventory.ProcessorImpl
	rankings func(ids []uint32) ([]ranking.Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	rp := ranking.NewProcessor(l, ctx)
	p := &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		ip:       inventory.NewProcessor(l, ctx),
		rankings: rp.GetByCharacterIds,
	}
	return p
}
```

3. Replace `GetForWorld` (both call paths — world-selection `socket/handler/character_list_world.go:48` and view-all `character_view_all.go:38` — converge here, so this is the single integration point; one bulk call per world, FR-8):

```go
func (p *ProcessorImpl) GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error) {
	return func(accountId uint32, worldId world.Id) ([]Model, error) {
		cs, err := p.ByAccountAndWorldProvider(decorators...)(accountId, worldId)()
		if errors.Is(err, requests.ErrNotFound) {
			return make([]Model, 0), nil
		}
		if err != nil {
			return cs, err
		}
		return p.decorateRankings(cs), nil
	}
}

// decorateRankings applies the slice-level rankings decoration: one bulk
// call for the whole character list, failing open to zero-valued rank
// fields (client renders "Ranking not available") on any error.
func (p *ProcessorImpl) decorateRankings(cs []Model) []Model {
	if len(cs) == 0 {
		return cs
	}
	ids := make([]uint32, len(cs))
	for i, c := range cs {
		ids[i] = c.Id()
	}
	rs, err := p.rankings(ids)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch character rankings; character select renders without ranks.")
		return cs
	}
	return MergeRankings(cs, rs)
}

// MergeRankings rebuilds each character with its ranking values; characters
// without a ranking entry keep zero-valued rank fields.
func MergeRankings(cs []Model, rs []ranking.Model) []Model {
	byId := make(map[uint32]ranking.Model, len(rs))
	for _, r := range rs {
		byId[r.CharacterId()] = r
	}
	out := make([]Model, len(cs))
	for i, c := range cs {
		r, ok := byId[c.Id()]
		if !ok {
			out[i] = c
			continue
		}
		out[i] = c.ToBuilder().
			SetRank(r.Rank()).
			SetRankMove(r.RankMove()).
			SetJobRank(r.JobRank()).
			SetJobRankMove(r.JobRankMove()).
			Build()
	}
	return out
}
```

- [x] **Step 5: Run tests to verify they pass**

Run from `services/atlas-login/atlas.com/login`:

```bash
go test -race ./...
go vet ./...
go build ./...
```

Expected: new tests PASS, existing login suite stays green.

- [x] **Step 6: Commit**

```bash
git add services/atlas-login/atlas.com/login
git commit -m "feat(task-143): login fetches character rankings with fail-open decoration"
```

---

### Task 13: final verification sweep

**Files:** none new — verification and any fix-ups it forces.

- [x] **Step 1: Per-module verification**

Run in each of the three changed modules (`services/atlas-rankings/atlas.com/rankings`, `services/atlas-login/atlas.com/login`, `services/atlas-tenants/atlas.com/tenants`):

```bash
go test -race ./...
go vet ./...
go build ./...
```

Expected: all clean.

- [x] **Step 2: Docker bake**

From the repo root:

```bash
docker buildx bake atlas-rankings
```

Expected: builds clean. `go.mod` files of atlas-login/atlas-tenants are only touched if `go mod tidy` changed them — check `git status`; if either module's `go.mod`/`go.sum` changed, also run `docker buildx bake atlas-login atlas-tenants`.

- [x] **Step 3: Redis key guard + ingress sync check**

From the repo root:

```bash
tools/redis-key-guard.sh
./deploy/scripts/sync-k8s-ingress-routes.sh && git diff --exit-code deploy
```

Expected: guard clean (atlas-lock usage is through the lib); sync produces no further diff.

- [x] **Step 4: Fix anything the sweep surfaced, then commit**

```bash
git add -A
git commit -m "chore(task-143): verification sweep fix-ups"
```

(Skip the commit if the sweep was already clean.)

- [x] **Step 5: Code review**

Run `superpowers:requesting-code-review` (dispatches plan-adherence-reviewer + backend-guidelines-reviewer) BEFORE opening any PR. Address findings in `docs/tasks/task-143-character-rankings/audit.md`.

Manual acceptance (post-deploy, not part of this plan's automated gate): on a v83 tenant, a pre-first-cycle character shows the short info board with "Ranking not available"; after a cycle, "Ranked at N" with correct arrows for +/−/0 movement; two tenants' rankings isolated.
