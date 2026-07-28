# Character Rankings Leaderboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a full-stack per-world character leaderboard — a new paginated `GET /rankings` list endpoint in atlas-rankings (with name/level/job stored on each ranking row) and an atlas-ui page that renders ranked characters with a character render image and movement arrows.

**Architecture:** The recompute already scans every character; extend it to persist `name`/`level`/`jobId` on each `character_rankings` row so the leaderboard endpoint is self-contained (one indexed, ordered, paginated query — no read-path fan-out to atlas-character). The atlas-ui page mirrors the existing MarketplacePage (world selector from tenant config, React Query hook, pagination) and reuses the existing `OptimizedCharacterRenderer` per row, fetching each visible character's appearance via `useCharacter` and failing open to a placeholder.

**Tech Stack:** Go (atlas-rankings; GORM, api2go JSON:API, `libs/atlas-rest/server/paginate`, `libs/atlas-model`); atlas-ui (Vite + React Router v7, TanStack React Query 5, shadcn/ui, Vitest + Testing Library).

## Global Constraints

- Backend module: `services/atlas-rankings/atlas.com/rankings` (module name `atlas-rankings`).
- Use shared types from `libs/atlas-constants` — `world.Id`, `job.Id` (DOM-21); never reinvent them.
- All reads/writes tenant-scoped via the existing GORM tenant callback on `db.WithContext(ctx)`; never issue an unscoped query. The single-char lookup path (`/rankings/characters...`) and its `RestModel` MUST NOT change — the login decoration depends on it.
- No packet changes; no change to ranking math (level DESC / experience DESC / characterId ASC; `JobCategory = jobId/100`).
- Pagination uses `libs/atlas-rest/server/paginate` (`DefaultPageSize=50`, `MaxPageSize=250`) and `server.MarshalPaginatedResponse` — same as `atlas-character` `GET /characters`.
- Frontend: React Query hooks under `src/lib/hooks/api/`, services under `src/services/api/`, tenant-first query keys, `vi.*` (not `jest.*`) in new tests, named page exports, lazy route registration in `App.tsx`. World id = index into `tenantConfiguration.attributes.worlds` (MarketplacePage convention).
- UI calls the ingress path `/api/rankings?...` (nginx `^/api/rankings(/.*)?$` → `atlas-rankings:8080`, full URI preserved).
- Per-file verification per repo CLAUDE.md: `go build`/`go vet`/`go test -race` for changed Go module, `docker buildx bake atlas-rankings` (schema columns + new route), `tools/lint.sh --check`; `npm run build` + `npm run test` for atlas-ui.

---

## Backend — atlas-rankings

### Task 1: Read `name` into the ranking character model

**Files:**
- Modify: `services/atlas-rankings/atlas.com/rankings/character/rest.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/character/model.go`
- Test: `services/atlas-rankings/atlas.com/rankings/character/processor_test.go` (existing) — add an Extract case

**Interfaces:**
- Consumes: atlas-character `characters` resource (`json:"name"`, already served).
- Produces: `character.Model.Name() string`; `character.RestModel.Name string` populated by `Extract`.

- [ ] **Step 1: Write the failing test**

Add to `character/processor_test.go`:

```go
func TestExtractCarriesName(t *testing.T) {
	rm := RestModel{Id: 7, Name: "Hero", WorldId: 0, Level: 30, Experience: 100, JobId: 100, Gm: 0}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Name() != "Hero" {
		t.Fatalf("Name = %q, want Hero", m.Name())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./character/ -run TestExtractCarriesName`
Expected: FAIL — `RestModel` has no field `Name` / `Model` has no method `Name`.

- [ ] **Step 3: Add the Name field and getter**

In `character/model.go`, add `name string` to the struct and a getter:

```go
type Model struct {
	id         uint32
	name       string
	worldId    world.Id
	jobId      job.Id
	level      byte
	experience uint32
	gm         int
}

func (m Model) Id() uint32         { return m.id }
func (m Model) Name() string       { return m.name }
func (m Model) WorldId() world.Id  { return m.worldId }
func (m Model) JobId() job.Id      { return m.jobId }
func (m Model) Level() byte        { return m.level }
func (m Model) Experience() uint32 { return m.experience }
func (m Model) Gm() int            { return m.gm }
```

In `character/rest.go`, add `Name` to `RestModel` and populate it in `Extract`:

```go
type RestModel struct {
	Id         uint32   `json:"-"`
	AccountId  uint32   `json:"accountId"`
	Name       string   `json:"name"`
	WorldId    world.Id `json:"worldId"`
	Level      byte     `json:"level"`
	Experience uint32   `json:"experience"`
	JobId      job.Id   `json:"jobId"`
	Gm         int      `json:"gm"`
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:         r.Id,
		name:       r.Name,
		worldId:    r.WorldId,
		jobId:      r.JobId,
		level:      r.Level,
		experience: r.Experience,
		gm:         r.Gm,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./character/ -run TestExtractCarriesName`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/character/
git commit -m "feat(task-143): read character name into ranking character model"
```

---

### Task 2: Thread name/level/job through ranking compute

**Files:**
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/compute.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/compute_test.go` (existing)

**Interfaces:**
- Consumes: `job.Id` from `libs/atlas-constants/job`.
- Produces: `Input{CharacterId, Name, WorldId, JobId, Level, Experience}`; `Ranked{CharacterId, Name, WorldId, JobCategory, Level, JobId, OverallRank, JobRank}`.

- [ ] **Step 1: Write the failing test**

Add to `ranking/compute_test.go`:

```go
func TestRankCarriesDisplayFields(t *testing.T) {
	inputs := []Input{
		{CharacterId: 1, Name: "Alpha", WorldId: 0, JobId: 110, Level: 40, Experience: 500},
		{CharacterId: 2, Name: "Beta", WorldId: 0, JobId: 210, Level: 30, Experience: 100},
	}
	got := Rank(inputs)
	byId := map[uint32]Ranked{}
	for _, r := range got {
		byId[r.CharacterId] = r
	}
	if byId[1].Name != "Alpha" || byId[1].Level != 40 || byId[1].JobId != 110 {
		t.Fatalf("Ranked[1] display fields not carried: %+v", byId[1])
	}
	if byId[1].OverallRank != 1 || byId[2].OverallRank != 2 {
		t.Fatalf("ordering changed: %+v", byId)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestRankCarriesDisplayFields`
Expected: FAIL — `Input`/`Ranked` have no `Name`/`Level`/`JobId` fields.

- [ ] **Step 3: Add fields and pass them through Rank**

In `ranking/compute.go`, extend the structs and the emit in `Rank`:

```go
type Input struct {
	CharacterId uint32
	Name        string
	WorldId     world.Id
	JobId       job.Id
	Level       byte
	Experience  uint32
}

type Ranked struct {
	CharacterId uint32
	Name        string
	WorldId     world.Id
	JobCategory uint16
	Level       byte
	JobId       job.Id
	OverallRank uint32
	JobRank     uint32
}
```

In the inner loop of `Rank`, extend the appended `Ranked` (leave the sort/`less`/`JobCategory` logic unchanged):

```go
			results = append(results, Ranked{
				CharacterId: c.CharacterId,
				Name:        c.Name,
				WorldId:     wid,
				JobCategory: cat,
				Level:       c.Level,
				JobId:       c.JobId,
				OverallRank: uint32(idx + 1),
				JobRank:     jobPos[cat],
			})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestRankCarriesDisplayFields`
Expected: PASS. Also run the full existing package to confirm no regression: `go test ./ranking/` → PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking/compute.go services/atlas-rankings/atlas.com/rankings/ranking/compute_test.go
git commit -m "feat(task-143): carry name/level/job through ranking compute"
```

---

### Task 3: Persist name/level/job on the ranking row + domain model

**Files:**
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/entity.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/model.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/builder.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/administrator.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/entity_test.go` (existing)

**Interfaces:**
- Produces: `Entity` columns `name`,`level`,`job_id`; `Model.Name()/Level()/JobId()`; `Builder.SetName/SetLevel/SetJobId`; upsert refreshes the three new columns.

- [ ] **Step 1: Write the failing test**

Add to `ranking/entity_test.go`:

```go
func TestMakeCarriesDisplayFields(t *testing.T) {
	e := Entity{CharacterId: 5, Name: "Gamma", Level: 50, JobId: 412, JobCategory: 4, OverallRank: 3}
	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if m.Name() != "Gamma" || m.Level() != 50 || m.JobId() != 412 {
		t.Fatalf("Make dropped display fields: name=%q level=%d job=%d", m.Name(), m.Level(), m.JobId())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestMakeCarriesDisplayFields`
Expected: FAIL — `Entity`/`Model` have no `Name`/`Level`/`JobId`.

- [ ] **Step 3: Add columns, builder setters, model getters, and Make mapping**

In `ranking/model.go`, add fields + getters:

```go
type Model struct {
	characterId     uint32
	name            string
	worldId         world.Id
	jobCategory     uint16
	level           byte
	jobId           job.Id
	overallRank     uint32
	overallRankMove int32
	jobRank         uint32
	jobRankMove     int32
	computedAt      time.Time
}

func (m Model) Name() string  { return m.name }
func (m Model) Level() byte    { return m.level }
func (m Model) JobId() job.Id  { return m.jobId }
```

(Add `"github.com/Chronicle20/atlas/libs/atlas-constants/job"` to the imports. Keep the existing getters.)

In `ranking/builder.go`, add matching builder fields, setters, and Build wiring:

```go
func (b *Builder) SetName(v string) *Builder  { b.name = v; return b }
func (b *Builder) SetLevel(v byte) *Builder    { b.level = v; return b }
func (b *Builder) SetJobId(v job.Id) *Builder  { b.jobId = v; return b }
```

(Add `name string`, `level byte`, `jobId job.Id` to the `Builder` struct, the `job` import, and `name: b.name, level: b.level, jobId: b.jobId` to the returned `Model` in `Build()`.)

In `ranking/entity.go`, add columns to `Entity` and map them in `Make`:

```go
type Entity struct {
	TenantId        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_rankings_tenant_character;index:idx_rankings_tenant_world"`
	Id              uuid.UUID `gorm:"type:uuid;primaryKey"`
	CharacterId     uint32    `gorm:"not null;uniqueIndex:idx_rankings_tenant_character"`
	Name            string    `gorm:"not null;default:''"`
	WorldId         world.Id  `gorm:"not null;index:idx_rankings_tenant_world"`
	JobCategory     uint16    `gorm:"not null"`
	Level           byte      `gorm:"not null;default:0"`
	JobId           job.Id    `gorm:"not null;default:0"`
	OverallRank     uint32    `gorm:"not null"`
	OverallRankMove int32     `gorm:"not null"`
	JobRank         uint32    `gorm:"not null"`
	JobRankMove     int32     `gorm:"not null"`
	ComputedAt      time.Time `gorm:"not null"`
}
```

(Add the `job` import.) Extend `Make`:

```go
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetCharacterId(e.CharacterId).
		SetName(e.Name).
		SetWorldId(e.WorldId).
		SetJobCategory(e.JobCategory).
		SetLevel(e.Level).
		SetJobId(e.JobId).
		SetOverallRank(e.OverallRank).
		SetOverallRankMove(e.OverallRankMove).
		SetJobRank(e.JobRank).
		SetJobRankMove(e.JobRankMove).
		SetComputedAt(e.ComputedAt).
		Build(), nil
}
```

In `ranking/administrator.go`, add the three columns to the upsert `DoUpdates` list so refreshes overwrite them:

```go
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "world_id", "job_category", "level", "job_id",
			"overall_rank", "overall_rank_move",
			"job_rank", "job_rank_move",
			"computed_at",
		}),
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestMakeCarriesDisplayFields` → PASS
Then `go build ./...` → clean (catches any missed import/field).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking/
git commit -m "feat(task-143): persist name/level/job on ranking row and domain model"
```

---

### Task 4: Populate name/level/job in Recompute

**Files:**
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/processor.go` (the `Recompute` Input build + Entity build, ~lines 107-151)
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/processor_test.go` (existing)

**Interfaces:**
- Consumes: `character.Model.Name()/Level()/JobId()` (Task 1), `Input`/`Ranked`/`Entity` display fields (Tasks 2-3).
- Produces: persisted `character_rankings` rows carrying name/level/jobId after a cycle.

- [ ] **Step 1: Write the failing test**

Add to `ranking/processor_test.go` (follow the existing test's `WithCharacterSupplier` + tenant-context DB harness — reuse whatever `newTestProcessor`/fixture helper the file already defines; the assertion is the new part):

```go
func TestRecomputePersistsDisplayFields(t *testing.T) {
	db, ctx := newRankingTestDB(t) // existing helper in this test file
	supplier := func() ([]character.Model, error) {
		return []character.Model{
			mustCharacter(t, 1, "Alpha", 0, 110, 40, 500), // id,name,world,job,level,exp
		}, nil
	}
	p := NewProcessor(testLogger(t), ctx, db).WithCharacterSupplier(supplier)
	if err := p.Recompute(time.Unix(1000, 0)); err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	rows, err := allEntityProvider()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "Alpha" || rows[0].Level != 40 || rows[0].JobId != 110 {
		t.Fatalf("row display fields wrong: %+v", rows)
	}
}
```

If the file lacks `mustCharacter`, build the `character.Model` via `character.Extract(character.RestModel{Id:1, Name:"Alpha", WorldId:0, JobId:110, Level:40, Experience:500})`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestRecomputePersistsDisplayFields`
Expected: FAIL — persisted `Name` is empty / `Level`/`JobId` are 0 (Recompute doesn't set them yet).

- [ ] **Step 3: Populate the fields in Recompute**

In `ranking/processor.go`, extend the `Input` build (in the character loop):

```go
		inputs = append(inputs, Input{
			CharacterId: c.Id(),
			Name:        c.Name(),
			WorldId:     c.WorldId(),
			JobId:       c.JobId(),
			Level:       c.Level(),
			Experience:  c.Experience(),
		})
```

And the `Entity` build (in the ranked loop):

```go
		entities = append(entities, Entity{
			CharacterId:     r.CharacterId,
			Name:            r.Name,
			WorldId:         r.WorldId,
			JobCategory:     r.JobCategory,
			Level:           r.Level,
			JobId:           r.JobId,
			OverallRank:     r.OverallRank,
			OverallRankMove: Move(prevOverall, r.OverallRank),
			JobRank:         r.JobRank,
			JobRankMove:     Move(prevJob, r.JobRank),
			ComputedAt:      now,
		})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestRecomputePersistsDisplayFields` → PASS
Then `go test ./ranking/` → all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking/processor.go services/atlas-rankings/atlas.com/rankings/ranking/processor_test.go
git commit -m "feat(task-143): populate name/level/job during rankings recompute"
```

---

### Task 5: Leaderboard paged provider (by world, optional category, ordered)

**Files:**
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/provider.go`
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/provider_test.go` (create)

**Interfaces:**
- Consumes: `database.PagedQuery[Entity]`, `model.Page`, `model.Paged`.
- Produces: `byWorldPagedEntityProvider(worldId world.Id, jobCategory *uint16, page model.Page) database.EntityProvider[model.Paged[Entity]]` — filtered by world (and category when non-nil), ordered by `overall_rank ASC` (no category) or `job_rank ASC` (category filter).

- [ ] **Step 1: Write the failing test**

Create `ranking/provider_test.go` (reuse the package's existing test DB helper; if none exists in provider scope, use the same harness `processor_test.go` uses):

```go
func TestByWorldPagedOrdersAndFilters(t *testing.T) {
	db, ctx := newRankingTestDB(t)
	tdb := db.WithContext(ctx)
	seed := []Entity{
		{CharacterId: 1, WorldId: 0, JobCategory: 1, OverallRank: 2, JobRank: 1, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, WorldId: 0, JobCategory: 1, OverallRank: 1, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 3, WorldId: 1, JobCategory: 1, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}
	if err := upsertBatch(tdb, tenantIdFrom(ctx), seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	page := model.Page{Number: 1, Size: 10}
	paged, err := byWorldPagedEntityProvider(0, nil, page)(tdb)()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if paged.Total != 2 {
		t.Fatalf("Total = %d, want 2 (world 0 only)", paged.Total)
	}
	if paged.Items[0].CharacterId != 2 || paged.Items[1].CharacterId != 1 {
		t.Fatalf("overall order wrong: %+v", paged.Items)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestByWorldPagedOrdersAndFilters`
Expected: FAIL — `byWorldPagedEntityProvider` undefined.

- [ ] **Step 3: Implement the provider**

Append to `ranking/provider.go`:

```go
// byWorldPagedEntityProvider reads one page of ranking rows for a world,
// ordered for a leaderboard: overall_rank ASC for the overall view, or
// job_rank ASC when a job category filter is supplied. Tenant scoping comes
// from the GORM query callback on the context-bearing db handle.
func byWorldPagedEntityProvider(worldId world.Id, jobCategory *uint16, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		q := db.Where("world_id = ?", worldId)
		if jobCategory != nil {
			q = q.Where("job_category = ?", *jobCategory).Order("job_rank ASC")
		} else {
			q = q.Order("overall_rank ASC")
		}
		return database.PagedQuery[Entity](q, page)
	}
}
```

(Add `world` to the imports if not already present in this file — it is used by `world.Id`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestByWorldPagedOrdersAndFilters` → PASS

Add and run a second case for the category filter + ordering by `job_rank`:

```go
func TestByWorldPagedCategoryFilter(t *testing.T) {
	db, ctx := newRankingTestDB(t)
	tdb := db.WithContext(ctx)
	cat := uint16(1)
	seed := []Entity{
		{CharacterId: 1, WorldId: 0, JobCategory: 1, OverallRank: 3, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, WorldId: 0, JobCategory: 1, OverallRank: 4, JobRank: 1, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 3, WorldId: 0, JobCategory: 2, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	}
	if err := upsertBatch(tdb, tenantIdFrom(ctx), seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	paged, err := byWorldPagedEntityProvider(0, &cat, model.Page{Number: 1, Size: 10})(tdb)()
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if paged.Total != 2 || paged.Items[0].CharacterId != 2 || paged.Items[1].CharacterId != 1 {
		t.Fatalf("category filter/order wrong: %+v", paged.Items)
	}
}
```

Run: `go test ./ranking/ -run TestByWorldPaged` → PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking/provider.go services/atlas-rankings/atlas.com/rankings/ranking/provider_test.go
git commit -m "feat(task-143): paged per-world leaderboard provider"
```

---

### Task 6: Leaderboard endpoint — RestModel, processor method, handler, route

**Files:**
- Create: `services/atlas-rankings/atlas.com/rankings/ranking/leaderboard_rest.go`
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/processor.go` (add interface method + impl)
- Modify: `services/atlas-rankings/atlas.com/rankings/ranking/resource.go` (route + handler)
- Test: `services/atlas-rankings/atlas.com/rankings/ranking/resource_test.go` (existing) — add leaderboard cases

**Interfaces:**
- Consumes: `byWorldPagedEntityProvider` (Task 5), `paginate.ParseParams`, `server.MarshalPaginatedResponse`, `paginate.EnvelopeFor`.
- Produces: `GET /rankings?filter[worldId]=&filter[jobCategory]=&page[number]=&page[size]=` → JSON:API paginated `LeaderboardRestModel` list; `Processor.LeaderboardProvider(worldId, jobCategory, page)`.

- [ ] **Step 1: Write the failing test**

Add to `ranking/resource_test.go` (reuse the file's existing router/tenant test harness that already exercises `/rankings/characters`):

```go
func TestLeaderboardOrdersByOverallRank(t *testing.T) {
	router, db, ctx := newRankingRouter(t) // existing helper style in this file
	seedRankings(t, db, ctx, []Entity{
		{CharacterId: 1, Name: "A", WorldId: 0, JobCategory: 1, Level: 40, JobId: 110, OverallRank: 2, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, Name: "B", WorldId: 0, JobCategory: 1, Level: 50, JobId: 110, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	})
	rr := doGet(t, router, "/rankings?filter[worldId]=0&page[number]=1&page[size]=10")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	// First data element must be character 2 (overall_rank 1).
	if !strings.Contains(rr.Body.String(), `"characterId":2`) {
		t.Fatalf("missing characterId 2 in body: %s", rr.Body.String())
	}
}

func TestLeaderboardRequiresWorldId(t *testing.T) {
	router, _, _ := newRankingRouter(t)
	rr := doGet(t, router, "/rankings?page[number]=1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestLeaderboard`
Expected: FAIL — route not registered (404) / handler missing.

- [ ] **Step 3: Add the RestModel**

Create `ranking/leaderboard_rest.go`:

```go
package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// LeaderboardRestModel is the list projection for the per-world leaderboard.
// It intentionally differs from the single-character RestModel (which the
// login decoration depends on) by carrying the display fields.
type LeaderboardRestModel struct {
	Id          uint32    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	Name        string    `json:"name"`
	WorldId     world.Id  `json:"worldId"`
	JobId       job.Id    `json:"jobId"`
	JobCategory uint16    `json:"jobCategory"`
	Level       byte      `json:"level"`
	Rank        uint32    `json:"rank"`
	RankMove    int32     `json:"rankMove"`
	JobRank     uint32    `json:"jobRank"`
	JobRankMove int32     `json:"jobRankMove"`
	ComputedAt  time.Time `json:"computedAt"`
}

func (r LeaderboardRestModel) GetName() string { return "rankings" }
func (r LeaderboardRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *LeaderboardRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func TransformLeaderboard(m Model) (LeaderboardRestModel, error) {
	return LeaderboardRestModel{
		Id:          m.CharacterId(),
		CharacterId: m.CharacterId(),
		Name:        m.Name(),
		WorldId:     m.WorldId(),
		JobId:       m.JobId(),
		JobCategory: m.JobCategory(),
		Level:       m.Level(),
		Rank:        m.OverallRank(),
		RankMove:    m.OverallRankMove(),
		JobRank:     m.JobRank(),
		JobRankMove: m.JobRankMove(),
		ComputedAt:  m.ComputedAt(),
	}, nil
}
```

- [ ] **Step 4: Add the processor method**

In `ranking/processor.go`, add to the `Processor` interface and impl:

```go
	// LeaderboardProvider returns one page of ranked characters for a world,
	// ordered overall (jobCategory nil) or within a job category.
	LeaderboardProvider(worldId world.Id, jobCategory *uint16, page model.Page) model.Provider[model.Paged[Model]]
```

```go
func (p *ProcessorImpl) LeaderboardProvider(worldId world.Id, jobCategory *uint16, page model.Page) model.Provider[model.Paged[Model]] {
	ep := byWorldPagedEntityProvider(worldId, jobCategory, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}
```

(Add the `world` import to processor.go if not present.)

- [ ] **Step 5: Add the handler + route**

In `ranking/resource.go`, register the bare-collection route on the existing `/rankings` subrouter (add it BEFORE the `/characters` routes is fine; paths don't overlap):

```go
			r.HandleFunc("", registerGet("get_leaderboard", handleGetLeaderboard)).Methods(http.MethodGet)
```

Add the handler (imports: `strconv`, `github.com/Chronicle20/atlas/libs/atlas-model/model`, `github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate`, `github.com/Chronicle20/atlas/libs/atlas-constants/world`):

```go
func handleGetLeaderboard(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		rawWorld := q.Get("filter[worldId]")
		if rawWorld == "" {
			server.WriteBadRequest(d.Logger(), w, "filter[worldId] query parameter is required")
			return
		}
		wid64, err := strconv.ParseUint(rawWorld, 10, 8)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "filter[worldId] must be a valid world id")
			return
		}
		worldId := world.Id(wid64)

		var jobCategory *uint16
		if rawCat := q.Get("filter[jobCategory]"); rawCat != "" {
			cat64, err := strconv.ParseUint(rawCat, 10, 16)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "filter[jobCategory] must be a valid job category")
				return
			}
			cat := uint16(cat64)
			jobCategory = &cat
		}

		page, err := paginate.ParseParams(q, paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, err.Error())
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).LeaderboardProvider(worldId, jobCategory, page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to read leaderboard for world [%d].", worldId)
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		res, err := model.SliceMap(func(m Model) (LeaderboardRestModel, error) { return TransformLeaderboard(m) })(model.FixedProvider(paged.Items))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating leaderboard REST model.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		queryParams := jsonapi.ParseQueryFields(&q)
		server.MarshalPaginatedResponse[[]LeaderboardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-rankings/atlas.com/rankings && go test ./ranking/ -run TestLeaderboard` → PASS
Then `go test ./...` → all PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-rankings/atlas.com/rankings/ranking/
git commit -m "feat(task-143): per-world leaderboard REST endpoint"
```

---

### Task 7: Backend verification gate

**Files:** none (verification only)

- [ ] **Step 1: Build, vet, race-test the module**

```bash
cd services/atlas-rankings/atlas.com/rankings
go build ./... && go vet ./... && go test -race ./...
```
Expected: all clean/PASS.

- [ ] **Step 2: Docker bake (schema columns + new route reach the image)**

```bash
cd "$(git rev-parse --show-toplevel)"
docker buildx bake atlas-rankings
```
Expected: image builds (`naming to docker.io/library/atlas-rankings:local done`).

- [ ] **Step 3: Lint/format guard**

```bash
tools/lint.sh --check
```
Expected: clean. If it rewrites files, run `tools/lint.sh` (fix mode), review, and amend the relevant commit.

- [ ] **Step 4: Commit any formatting fixups**

```bash
git commit -am "style(task-143): lint/format leaderboard backend" # only if fix mode changed files
```

---

## Frontend — atlas-ui

### Task 8: Rankings service + types

**Files:**
- Create: `services/atlas-ui/src/services/api/rankings.service.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/rankings.service.test.ts`

**Interfaces:**
- Consumes: `apiClient` from `@/lib/api/client`, `ApiPagedResponse` from `@/types/api/responses`, `ServiceOptions` from `@/lib/api/query-params`.
- Produces: `rankingsService.leaderboard(worldId, filter, options?) => Promise<RankingPage>`; types `RankingEntry`, `RankingPage`, `LeaderboardFilter`.

- [ ] **Step 1: Write the failing test**

Create `__tests__/rankings.service.test.ts` (mirror `mts-listings.service.test.ts` — mock `apiClient.get`):

```ts
import { describe, expect, it, vi, beforeEach } from "vitest";
import { apiClient } from "@/lib/api/client";
import { rankingsService } from "@/services/api/rankings.service";

vi.mock("@/lib/api/client", () => ({
  apiClient: { get: vi.fn() },
}));

describe("rankingsService.leaderboard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("requests the leaderboard with world + paging params and returns total/lastPage from meta", async () => {
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: [{ id: "2", attributes: { characterId: 2, name: "B", worldId: 0, level: 50, jobId: 110, jobCategory: 1, rank: 1, rankMove: 0, jobRank: 1, jobRankMove: 0, computedAt: "" } }],
      meta: { total: 1, page: { last: 1 } },
    });
    const res = await rankingsService.leaderboard(0, { page: 0, pageSize: 25 });
    expect(apiClient.get).toHaveBeenCalledWith(
      "/api/rankings?filter%5BworldId%5D=0&page%5Bnumber%5D=1&page%5Bsize%5D=25",
      undefined,
    );
    expect(res.total).toBe(1);
    expect(res.entries[0].attributes.characterId).toBe(2);
  });

  it("includes the job category filter when set", async () => {
    (apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [], meta: { total: 0, page: { last: 1 } } });
    await rankingsService.leaderboard(0, { jobCategory: 1, page: 0, pageSize: 25 });
    expect(apiClient.get).toHaveBeenCalledWith(
      expect.stringContaining("filter%5BjobCategory%5D=1"),
      undefined,
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- rankings.service`
Expected: FAIL — module `rankings.service` not found.

- [ ] **Step 3: Implement the service**

Create `services/atlas-ui/src/services/api/rankings.service.ts`:

```ts
import { apiClient } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import type { ApiPagedResponse } from "@/types/api/responses";

/**
 * Read-only per-world character leaderboard, backed by atlas-rankings:
 *   GET /api/rankings?filter[worldId]=&filter[jobCategory]=&page[number]=&page[size]=
 *
 * The response is a JSON:API list of `rankings` resources with a pagination
 * `meta` block (`meta.total`, `meta.page.last`), so total/lastPage are
 * authoritative — never inferred from the returned length.
 */

export interface RankingEntryAttributes {
  characterId: number;
  name: string;
  worldId: number;
  level: number;
  jobId: number;
  jobCategory: number;
  rank: number;
  rankMove: number;
  jobRank: number;
  jobRankMove: number;
  computedAt: string;
}

export interface RankingEntry {
  id: string;
  attributes: RankingEntryAttributes;
}

export interface RankingPage {
  entries: RankingEntry[];
  total: number;
  lastPage: number;
}

export interface LeaderboardFilter {
  /** Overall leaderboard when undefined; otherwise restricts to jobId/100. */
  jobCategory?: number | undefined;
  /** ZERO-BASED caller page (page=0 is the first page). */
  page?: number | undefined;
  pageSize?: number | undefined;
}

export function buildLeaderboardQuery(worldId: number, filter: LeaderboardFilter): string {
  const params = new URLSearchParams();
  params.set("filter[worldId]", String(worldId));
  if (filter.jobCategory !== undefined)
    params.set("filter[jobCategory]", String(filter.jobCategory));
  if (filter.page !== undefined)
    params.set("page[number]", String(filter.page + 1));
  if (filter.pageSize !== undefined)
    params.set("page[size]", String(filter.pageSize));
  return `?${params.toString()}`;
}

export const rankingsService = {
  async leaderboard(
    worldId: number,
    filter: LeaderboardFilter,
    options?: ServiceOptions,
  ): Promise<RankingPage> {
    const query = buildLeaderboardQuery(worldId, filter);
    const resp = await apiClient.get<ApiPagedResponse<RankingEntry>>(
      `/api/rankings${query}`,
      options,
    );
    const total = resp.meta?.total ?? resp.data.length;
    const lastPage = resp.meta?.page?.last ?? 1;
    return { entries: resp.data, total, lastPage };
  },
};
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- rankings.service`
Expected: PASS. (If the exact URL-encoding assertion differs, align the test's expected string to `buildLeaderboardQuery`'s output — the encoding, not the params, is what varies.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/rankings.service.ts services/atlas-ui/src/services/api/__tests__/rankings.service.test.ts
git commit -m "feat(task-143): atlas-ui rankings leaderboard service"
```

---

### Task 9: useRankings hook

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/useRankings.ts`
- Test: `services/atlas-ui/src/lib/hooks/api/__tests__/useRankings.test.tsx`

**Interfaces:**
- Consumes: `rankingsService.leaderboard`, `LeaderboardFilter`, `RankingPage`.
- Produces: `useRankings(tenantId, worldId, filter, enabled?) => UseQueryResult<RankingPage, Error>`; `rankingsKeys`.

- [ ] **Step 1: Write the failing test**

Create `__tests__/useRankings.test.tsx` (mirror `useMtsListings` test if present; otherwise a minimal render-hook test):

```tsx
import { describe, expect, it, vi } from "vitest";
import { rankingsKeys } from "@/lib/hooks/api/useRankings";

vi.mock("@/services/api/rankings.service", () => ({
  rankingsService: { leaderboard: vi.fn() },
}));

describe("rankingsKeys", () => {
  it("scopes the cache by tenant, world and filter", () => {
    const key = rankingsKeys.leaderboard("t1", 0, { jobCategory: 1, page: 0, pageSize: 25 });
    expect(key[0]).toBe("rankings");
    expect(key).toContain("t1");
    expect(key).toContain(0);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- useRankings`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the hook**

Create `services/atlas-ui/src/lib/hooks/api/useRankings.ts`:

```tsx
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  rankingsService,
  type LeaderboardFilter,
  type RankingPage,
} from "@/services/api/rankings.service";

export const rankingsKeys = {
  all: ["rankings"] as const,
  // Tenant id is the FIRST segment: rankings are tenant-scoped only via the
  // mutable global apiClient tenant header, so without it two tenants sharing
  // (worldId, filter) would collide in cache. Mirrors mtsListingsKeys.
  leaderboard: (tenantId: string, worldId: number, filter: LeaderboardFilter) =>
    [...rankingsKeys.all, "leaderboard", tenantId, worldId, filter] as const,
};

export function useRankings(
  tenantId: string,
  worldId: number,
  filter: LeaderboardFilter,
  enabled = true,
): UseQueryResult<RankingPage, Error> {
  return useQuery({
    queryKey: rankingsKeys.leaderboard(tenantId, worldId, filter),
    queryFn: () => rankingsService.leaderboard(worldId, filter),
    enabled,
  });
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- useRankings` → PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useRankings.ts services/atlas-ui/src/lib/hooks/api/__tests__/useRankings.test.tsx
git commit -m "feat(task-143): useRankings leaderboard hook"
```

---

### Task 10: LeaderboardRow component (image + movement arrow, fail-open)

**Files:**
- Create: `services/atlas-ui/src/components/features/rankings/LeaderboardRow.tsx`
- Test: `services/atlas-ui/src/components/features/rankings/__tests__/LeaderboardRow.test.tsx`

**Interfaces:**
- Consumes: `RankingEntry` (service), `useCharacter` from `@/lib/hooks/api/useCharacters`, `OptimizedCharacterRenderer` from `@/components/features/characters/OptimizedCharacterRenderer`.
- Produces: `<LeaderboardRow entry={RankingEntry} view="overall" | "job" />` — a `<tr>` showing rank #, image, name, level, job, movement arrow. One `useCharacter(entry.attributes.characterId)` per row (lazy per-row appearance fetch); image fails open to the renderer's own fallback.

- [ ] **Step 1: Write the failing test**

Create `__tests__/LeaderboardRow.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { LeaderboardRow } from "@/components/features/rankings/LeaderboardRow";

vi.mock("@/lib/hooks/api/useCharacters", () => ({
  useCharacter: () => ({ data: undefined, isLoading: false, isError: true }),
}));
vi.mock("@/components/features/characters/OptimizedCharacterRenderer", () => ({
  OptimizedCharacterRenderer: () => <div data-testid="renderer" />,
}));

function row(over: number, move: number) {
  return {
    id: String(over),
    attributes: {
      characterId: 2, name: "B", worldId: 0, level: 50, jobId: 110,
      jobCategory: 1, rank: over, rankMove: move, jobRank: 1, jobRankMove: 0, computedAt: "",
    },
  };
}

describe("LeaderboardRow", () => {
  it("renders rank, name and level even when the character render is unavailable", () => {
    render(<table><tbody><LeaderboardRow entry={row(1, 0)} view="overall" /></tbody></table>);
    expect(screen.getByText("B")).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("#1")).toBeInTheDocument();
  });

  it("shows an up arrow when rankMove is positive", () => {
    render(<table><tbody><LeaderboardRow entry={row(1, 3)} view="overall" /></tbody></table>);
    expect(screen.getByLabelText("moved up")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- LeaderboardRow`
Expected: FAIL — component not found.

- [ ] **Step 3: Implement the component**

Create `services/atlas-ui/src/components/features/rankings/LeaderboardRow.tsx`:

```tsx
import { ArrowDown, ArrowUp, Minus } from "lucide-react";
import { useCharacter } from "@/lib/hooks/api/useCharacters";
import { OptimizedCharacterRenderer } from "@/components/features/characters/OptimizedCharacterRenderer";
import type { RankingEntry } from "@/services/api/rankings.service";

interface LeaderboardRowProps {
  entry: RankingEntry;
  /** "overall" uses rank/rankMove; "job" uses jobRank/jobRankMove. */
  view: "overall" | "job";
}

function MoveArrow({ move }: { move: number }) {
  if (move > 0) return <ArrowUp className="h-4 w-4 text-green-600" aria-label="moved up" />;
  if (move < 0) return <ArrowDown className="h-4 w-4 text-red-600" aria-label="moved down" />;
  return <Minus className="h-4 w-4 text-muted-foreground" aria-label="no change" />;
}

export function LeaderboardRow({ entry, view }: LeaderboardRowProps) {
  const a = entry.attributes;
  const rank = view === "overall" ? a.rank : a.jobRank;
  const move = view === "overall" ? a.rankMove : a.jobRankMove;
  const characterQuery = useCharacter(a.characterId);

  return (
    <tr className="border-b">
      <td className="px-3 py-2 font-mono">#{rank}</td>
      <td className="px-3 py-2">
        {characterQuery.data ? (
          <OptimizedCharacterRenderer
            character={characterQuery.data}
            size="small"
            lazy
            fallbackAvatar="/logo.png"
          />
        ) : (
          <div className="h-12 w-12 rounded bg-muted" aria-hidden="true" />
        )}
      </td>
      <td className="px-3 py-2 font-medium">{a.name}</td>
      <td className="px-3 py-2">{a.level}</td>
      <td className="px-3 py-2">{a.jobId}</td>
      <td className="px-3 py-2"><MoveArrow move={move} /></td>
    </tr>
  );
}
```

Notes for the implementer:
- Confirm `useCharacter`'s return shape (`.data` is a `Character`) and that `OptimizedCharacterRenderer` accepts `character` + optional `size`/`lazy`/`fallbackAvatar` (verified props). If `useCharacter(id)` requires additional args (e.g. tenant), pass them per its signature.
- If a job **label** (not raw id) is wanted, map `a.jobId` via the existing jobs data used by `JobsPage`; raw id is acceptable for v1 and keeps the row dependency-free.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- LeaderboardRow` → PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/rankings/
git commit -m "feat(task-143): leaderboard row with character render and move arrow"
```

---

### Task 11: RankingsPage + route registration

**Files:**
- Create: `services/atlas-ui/src/pages/RankingsPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx` (lazy import + `<Route>`)
- Test: `services/atlas-ui/src/pages/__tests__/RankingsPage.test.tsx`

**Interfaces:**
- Consumes: `useTenant` (`@/context/tenant-context`), `useTenantConfiguration` (`@/lib/hooks/api/useTenants`), `useRankings` (Task 9), `LeaderboardRow` (Task 10), shadcn `Select`/`Table`/`Button`.
- Produces: named export `RankingsPage`; route `path="rankings"` under `AppShell`.

- [ ] **Step 1: Write the failing test**

Create `__tests__/RankingsPage.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { RankingsPage } from "@/pages/RankingsPage";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: () => ({ data: { attributes: { worlds: [{ name: "Scania" }] } } }),
}));
vi.mock("@/lib/hooks/api/useRankings", () => ({
  useRankings: () => ({
    data: { entries: [], total: 0, lastPage: 1 },
    isLoading: false,
    isError: false,
  }),
}));

describe("RankingsPage", () => {
  it("renders the leaderboard heading and world selector", () => {
    render(<RankingsPage />);
    expect(screen.getByText(/Rankings/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- RankingsPage`
Expected: FAIL — page not found.

- [ ] **Step 3: Implement the page**

Create `services/atlas-ui/src/pages/RankingsPage.tsx` (mirror MarketplacePage's world selector + pagination controls; use the shadcn primitives already imported elsewhere):

```tsx
import { useState } from "react";
import { useTenant } from "@/context/tenant-context";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useRankings } from "@/lib/hooks/api/useRankings";
import { LeaderboardRow } from "@/components/features/rankings/LeaderboardRow";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";

const JOB_CATEGORIES: { label: string; value: number | undefined }[] = [
  { label: "All jobs", value: undefined },
  { label: "Beginner", value: 0 },
  { label: "Warrior", value: 1 },
  { label: "Magician", value: 2 },
  { label: "Bowman", value: 3 },
  { label: "Thief", value: 4 },
  { label: "Pirate", value: 5 },
];

const PAGE_SIZE = 25;

export function RankingsPage() {
  const { activeTenant } = useTenant();
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const worlds = tenantConfigQuery.data?.attributes.worlds ?? [];

  const [worldId, setWorldId] = useState(0);
  const [jobCategory, setJobCategory] = useState<number | undefined>(undefined);
  const [page, setPage] = useState(0); // zero-based

  const view = jobCategory === undefined ? "overall" : "job";
  const filter = { jobCategory, page, pageSize: PAGE_SIZE };
  const query = useRankings(activeTenant?.id ?? "", worldId, filter, !!activeTenant);

  const total = query.data?.total ?? 0;
  const lastPage = query.data?.lastPage ?? 1;

  return (
    <div className="space-y-4 p-4">
      <h1 className="text-2xl font-semibold">Rankings</h1>

      <div className="flex flex-wrap gap-4">
        <div className="space-y-1">
          <label className="text-sm font-medium">World</label>
          <Select value={String(worldId)} onValueChange={(v) => { setWorldId(parseInt(v, 10)); setPage(0); }}>
            <SelectTrigger className="w-40"><SelectValue placeholder="Select a world" /></SelectTrigger>
            <SelectContent>
              {worlds.length > 0 ? (
                worlds.map((world, index) => (
                  <SelectItem key={index} value={String(index)}>{world.name || `World ${index}`}</SelectItem>
                ))
              ) : (
                <SelectItem value="0">World 0</SelectItem>
              )}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1">
          <label className="text-sm font-medium">Job</label>
          <Select
            value={jobCategory === undefined ? "all" : String(jobCategory)}
            onValueChange={(v) => { setJobCategory(v === "all" ? undefined : parseInt(v, 10)); setPage(0); }}
          >
            <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
            <SelectContent>
              {JOB_CATEGORIES.map((c) => (
                <SelectItem key={c.label} value={c.value === undefined ? "all" : String(c.value)}>{c.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {query.isError ? (
        <p className="text-red-600">Failed to load rankings.</p>
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left text-muted-foreground">
              <th className="px-3 py-2">Rank</th>
              <th className="px-3 py-2">Character</th>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">Level</th>
              <th className="px-3 py-2">Job</th>
              <th className="px-3 py-2">Move</th>
            </tr>
          </thead>
          <tbody>
            {(query.data?.entries ?? []).map((entry) => (
              <LeaderboardRow key={entry.id} entry={entry} view={view} />
            ))}
          </tbody>
        </table>
      )}

      <div className="flex items-center gap-3">
        <Button variant="outline" size="sm" disabled={page <= 0} onClick={() => setPage((p) => Math.max(0, p - 1))}>Previous</Button>
        <span className="text-sm text-muted-foreground">Page {page + 1} of {lastPage} ({total} ranked)</span>
        <Button variant="outline" size="sm" disabled={page + 1 >= lastPage} onClick={() => setPage((p) => p + 1)}>Next</Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Register the route in App.tsx**

Add the lazy import alongside the other page imports:

```tsx
const RankingsPage = lazy(() =>
  import("@/pages/RankingsPage").then((m) => ({ default: m.RankingsPage })),
);
```

Add the route inside the `AppShell` `<Route>` block (next to the other tenant pages):

```tsx
<Route path="rankings" element={<RankingsPage />} />
```

(If the sidebar navigation is data-driven from a nav config rather than App.tsx, add a "Rankings" entry there too — follow whatever the MarketplacePage/`marketplace` route did for nav.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ui && npm run test -- RankingsPage` → PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/RankingsPage.tsx services/atlas-ui/src/pages/__tests__/RankingsPage.test.tsx services/atlas-ui/src/App.tsx
git commit -m "feat(task-143): rankings leaderboard page and route"
```

---

### Task 12: Frontend verification gate

**Files:** none (verification only)

- [ ] **Step 1: Type-check + build**

```bash
cd services/atlas-ui
npm run build
```
Expected: `tsc -b` clean and Vite build succeeds. Fix any type errors surfaced in the new files.

- [ ] **Step 2: Run the full frontend test suite for the new areas**

```bash
cd services/atlas-ui
npm run test -- rankings.service useRankings LeaderboardRow RankingsPage
```
Expected: all PASS.

- [ ] **Step 3: Lint/format guard (repo root)**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/lint.sh --check
```
Expected: clean (Prettier + ESLint for atlas-ui). If fix mode is needed, run `tools/lint.sh`, review, and amend.

- [ ] **Step 4: Commit any fixups**

```bash
git commit -am "style(task-143): lint/format leaderboard frontend" # only if fix mode changed files
```

---

## Post-implementation

- Run the code-review step (`superpowers:requesting-code-review` → backend + frontend guideline reviewers + plan-adherence) before opening the PR, per repo CLAUDE.md.
- The leaderboard reads name/level/job that are only populated on the NEXT recompute cycle after deploy; existing rows show empty name/level 0 until then. This is self-healing within one recompute interval and needs no migration/backfill (absent-config default cadence applies).
