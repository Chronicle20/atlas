# Seed Counts on Bootstrap UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add eight tenant-scoped `GET .../seed/status` JSON:API endpoints across seven Go services and surface the counts next to each Seed button on `/setup`, polled every 5 seconds.

**Architecture:** Backend — each sub-resource package gets a `Count() (int64, *time.Time, error)` method using the existing GORM tenant auto-filter; each seed package gains a `seed_status.go` (or equivalent) handler plus one new `GET` route. Compound handlers use `errgroup.WithContext` + `sync.Mutex`. Frontend — extract a shared `SetupRow` primitive, add 8 typed status interfaces + getters, 8 polling `useQuery` hooks, wire `onSuccess` invalidation on each existing seed mutation, rewrite the Seed Data grid to match the Game Data row layout.

**Tech Stack:** Go 1.22+, GORM, `errgroup`, api2go/jsonapi, logrus; React 19, TanStack React Query 5, Vitest, TypeScript.

**Pre-flight:** Create a feature branch off `main` (e.g., `task-022-seed-counts`). Branch protection blocks pushes to `main`.

---

## Task 1: atlas-drop-information — `Count()` on `monster/drop`

**Files:**
- Modify: `services/atlas-drop-information/atlas.com/dis/monster/drop/processor.go`
- Test: `services/atlas-drop-information/atlas.com/dis/monster/drop/processor_test.go`

- [ ] **Step 1: Write failing tests**

Append to `processor_test.go`:

```go
func TestProcessorImpl_Count_Empty(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := drop.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	seedTestData(t, db, te.Id(), 100100, []uint32{2000000, 2000001, 2000002})
	seedTestData(t, db, te.Id(), 100101, []uint32{2000003, 2000004})

	p := drop.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
	// monster_drops has no updated_at column; updatedAt must be nil.
	if updated != nil {
		t.Errorf("Expected nil updatedAt for table without updated_at, got %v", updated)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	te1 := testTenant()
	te2 := testTenant()
	ctx1 := tenant.WithContext(context.Background(), te1)
	db := testDatabase(t)

	seedTestData(t, db, te1.Id(), 100100, []uint32{2000000, 2000001})
	seedTestData(t, db, te2.Id(), 100100, []uint32{2000002, 2000003, 2000004})

	p := drop.NewProcessor(l, ctx1, db)
	count, _, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./monster/drop/ -run Count -v
```

Expected: compile error — `p.Count undefined`.

- [ ] **Step 3: Add `Count()` to the interface and impl**

Edit `processor.go`:

```go
package drop

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAll() model.Provider[[]Model]
	GetForMonster(monsterId uint32) model.Provider[[]Model]
	GetForItem(itemId uint32) model.Provider[[]Model]
	Count() (int64, *time.Time, error)
}
```

Append the method below `GetForItem`:

```go
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	// monster_drops has no updated_at column; updatedAt is always nil.
	return count, nil, nil
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./monster/drop/ -run Count -v
```

Expected: 3 PASS.

- [ ] **Step 5: Build + full package test**

```bash
cd services/atlas-drop-information/atlas.com/dis && go build ./... && go test ./monster/drop/...
```

Expected: no errors, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-drop-information/atlas.com/dis/monster/drop/
git commit -m "feat(atlas-drop-information): add Count() to monster drop processor"
```

---

## Task 2: atlas-drop-information — `Count()` on `continent/drop`

**Files:**
- Modify: `services/atlas-drop-information/atlas.com/dis/continent/drop/processor.go`
- Test: `services/atlas-drop-information/atlas.com/dis/continent/drop/processor_test.go`

- [ ] **Step 1: Inspect existing test harness**

Read `processor_test.go` top ~50 lines to confirm the helper names used for continent drops (`testDatabase`, `testTenant`, `seedTestData`). If the seed helper signature differs from monster drops, note it before writing the test.

```bash
cd services/atlas-drop-information/atlas.com/dis && head -60 continent/drop/processor_test.go
```

- [ ] **Step 2: Write failing tests**

Append to `continent/drop/processor_test.go` (adjust `seedTestData` args to the continent-drop signature observed in Step 1 — the table columns are `(tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance)`):

```go
func TestProcessorImpl_Count_Empty(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	p := drop.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 || updated != nil {
		t.Errorf("Expected (0, nil), got (%d, %v)", count, updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	for i, itemId := range []uint32{2000000, 2000001, 2000002} {
		if err := db.Exec(
			"INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			te.Id(), int32(1), itemId, 1, 5, 0, 50000+uint32(i)*1000,
		).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	p := drop.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	te1 := testTenant()
	te2 := testTenant()
	ctx1 := tenant.WithContext(context.Background(), te1)
	db := testDatabase(t)

	for _, tid := range []uuid.UUID{te1.Id(), te2.Id()} {
		if err := db.Exec(
			"INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, ?, ?, ?, ?, ?, ?)",
			tid, int32(1), uint32(2000000), 1, 1, 0, 50000,
		).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	p := drop.NewProcessor(l, ctx1, db)
	count, _, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 for tenant 1, got %d", count)
	}
}
```

If the file lacks `uuid` or `test` imports, add them (they are used in the monster/drop test).

- [ ] **Step 3: Run to verify failure**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./continent/drop/ -run Count -v
```

Expected: compile error.

- [ ] **Step 4: Add `Count()` interface + impl**

Edit `continent/drop/processor.go` — add `Count() (int64, *time.Time, error)` to the `Processor` interface (import `"time"`) and implement:

```go
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
```

- [ ] **Step 5: Run tests + package build**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./continent/drop/... && go build ./...
```

Expected: PASS, no build errors.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-drop-information/atlas.com/dis/continent/drop/
git commit -m "feat(atlas-drop-information): add Count() to continent drop processor"
```

---

## Task 3: atlas-drop-information — `Count()` on `reactor/drop`

**Files:**
- Modify: `services/atlas-drop-information/atlas.com/dis/reactor/drop/processor.go`
- Test: `services/atlas-drop-information/atlas.com/dis/reactor/drop/processor_test.go`

- [ ] **Step 1: Inspect existing test harness**

```bash
head -60 services/atlas-drop-information/atlas.com/dis/reactor/drop/processor_test.go
```

Note the helper names. Columns for `reactor_drops` are `(tenant_id, reactor_id, item_id, quest_id, chance)`.

- [ ] **Step 2: Write failing tests**

Append three `Count` tests following the shape in Task 2, using an inline `INSERT` against `reactor_drops` with the column list above. Assert `(0,nil,nil)` empty, `(N,nil,nil)` populated (no `updated_at` on this table), and tenant isolation.

- [ ] **Step 3: Verify failure**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./reactor/drop/ -run Count -v
```

- [ ] **Step 4: Add `Count()` interface method + impl**

Identical to Task 2: add `Count() (int64, *time.Time, error)` to the interface; impl returns `(count, nil, nil)`.

- [ ] **Step 5: Verify**

```bash
cd services/atlas-drop-information/atlas.com/dis && go test ./reactor/drop/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-drop-information/atlas.com/dis/reactor/drop/
git commit -m "feat(atlas-drop-information): add Count() to reactor drop processor"
```

---

## Task 4: atlas-drop-information — drops seed-status handler + route

**Files:**
- Create: `services/atlas-drop-information/atlas.com/dis/seed/status.go`
- Create: `services/atlas-drop-information/atlas.com/dis/seed/status_test.go`
- Modify: `services/atlas-drop-information/atlas.com/dis/seed/resource.go`

- [ ] **Step 1: Write the REST model + handler in a new file**

Create `seed/status.go`:

```go
package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"
	"atlas-drops-information/rest"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type DropsSeedStatusRestModel struct {
	Id                 string  `json:"-"`
	MonsterDropCount   int64   `json:"monsterDropCount"`
	ContinentDropCount int64   `json:"continentDropCount"`
	ReactorDropCount   int64   `json:"reactorDropCount"`
	UpdatedAt          *string `json:"updatedAt"`
}

func (r DropsSeedStatusRestModel) GetName() string             { return "dropsSeedStatus" }
func (r DropsSeedStatusRestModel) GetID() string               { return r.Id }
func (r *DropsSeedStatusRestModel) SetID(id string) error      { r.Id = id; return nil }
func (r DropsSeedStatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}
func (r DropsSeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}
func (r DropsSeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}
func (r *DropsSeedStatusRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *DropsSeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
func (r *DropsSeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

type subcount struct {
	count     int64
	updatedAt *time.Time
}

func handleGetSeedStatus(_ *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// rest.HandlerDependency wires d.Context()/d.DB()/d.Logger() via RegisterHandler.
		d := rest.FromContext(r.Context())
		l := d.Logger()
		db := d.DB()
		t := tenant.MustFromContext(d.Context())

		var mu sync.Mutex
		var monster, continent, reactor subcount

		g, gctx := errgroup.WithContext(d.Context())
		g.Go(func() error {
			count, updated, err := monsterdrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			monster = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := continentdrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			continent = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := reactordrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			reactor = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})

		if err := g.Wait(); err != nil {
			l.WithError(err).Errorf("Unable to read drops seed status.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		res := DropsSeedStatusRestModel{
			Id:                 t.Id().String(),
			MonsterDropCount:   monster.count,
			ContinentDropCount: continent.count,
			ReactorDropCount:   reactor.count,
			UpdatedAt:          maxUpdatedAtRFC3339(monster.updatedAt, continent.updatedAt, reactor.updatedAt),
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[DropsSeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
	}
}

func maxUpdatedAtRFC3339(parts ...*time.Time) *string {
	var max *time.Time
	for _, p := range parts {
		if p == nil {
			continue
		}
		if max == nil || p.After(*max) {
			max = p
		}
	}
	if max == nil {
		return nil
	}
	s := max.UTC().Format(time.RFC3339)
	return &s
}

// silence unused import of context/logrus/gorm when signatures change.
var _ = func(logrus.FieldLogger, context.Context, *gorm.DB) {}
```

**Note on `rest.FromContext`.** The existing `handleSeedDrops` reads `d *rest.HandlerDependency` directly as the first arg. Prefer that pattern — replace `_ *rest.HandlerDependency` and `rest.FromContext(r.Context())` with named `d` and drop the `FromContext` line. Sketched as above only if `FromContext` is the project convention (check `services/atlas-drop-information/atlas.com/dis/rest/` before using it; if absent, use the `d` arg directly and delete the fallback line and the unused-imports guard).

Final simplified handler signature (recommended):

```go
func handleGetSeedStatus(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := d.Logger()
		db := d.DB()
		t := tenant.MustFromContext(d.Context())
		// …errgroup body unchanged…
	}
}
```

- [ ] **Step 2: Register the GET route**

Edit `seed/resource.go` — inside `InitResource`'s inner function, add the line after the existing `POST /drops/seed` registration:

```go
router.HandleFunc("/drops/seed/status", registerHandler("get_drops_seed_status", handleGetSeedStatus)).Methods(http.MethodGet)
```

- [ ] **Step 3: Write handler test**

Create `seed/status_test.go`. Model it on `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/status_test.go` — open that file first to copy the tenant-header helper:

```bash
cat services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/status_test.go
```

Minimum tests:

```go
package seed_test

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"
	"atlas-drops-information/seed"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenantlib "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	database.RegisterTenantCallbacks(l, db)
	for _, m := range []func(*gorm.DB) error{
		monsterdrop.Migration, continentdrop.Migration, reactordrop.Migration,
	} {
		if err := m(db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	return db
}

func makeTenant() tenantlib.Model {
	tm, _ := tenantlib.Create(uuid.New(), "GMS", 83, 1)
	return tm
}

func doGet(t *testing.T, db *gorm.DB, tm tenantlib.Model) *httptest.ResponseRecorder {
	t.Helper()
	l, _ := test.NewNullLogger()
	si := server.NewServerInformation("http://localhost:8080", "atlas-drop-information")

	req := httptest.NewRequest(http.MethodGet, "/drops/seed/status", nil)
	req.Header.Set("TENANT_ID", tm.Id().String())
	req.Header.Set("REGION", tm.Region())
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")

	router := server.NewRouter()
	seed.InitResource(si)(db)(router, l)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req.WithContext(tenantlib.WithContext(context.Background(), tm)))
	return w
}

func TestStatusHandler_Empty(t *testing.T) {
	db := setupDB(t)
	tm := makeTenant()

	w := doGet(t, db, tm)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				MonsterDropCount   int64   `json:"monsterDropCount"`
				ContinentDropCount int64   `json:"continentDropCount"`
				ReactorDropCount   int64   `json:"reactorDropCount"`
				UpdatedAt          *string `json:"updatedAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&env)
	if env.Data.Type != "dropsSeedStatus" || env.Data.Id != tm.Id().String() {
		t.Fatalf("bad envelope: %+v", env)
	}
	if env.Data.Attributes.MonsterDropCount != 0 ||
		env.Data.Attributes.ContinentDropCount != 0 ||
		env.Data.Attributes.ReactorDropCount != 0 ||
		env.Data.Attributes.UpdatedAt != nil {
		t.Fatalf("expected zero counts + null updatedAt, got %+v", env.Data.Attributes)
	}
}

func TestStatusHandler_Populated(t *testing.T) {
	db := setupDB(t)
	tm := makeTenant()

	// seed 2 monster + 1 continent + 3 reactor
	_ = db.Exec("INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, 100, 1, 1, 1, 0, 1), (?, 100, 2, 1, 1, 0, 1)", tm.Id(), tm.Id()).Error
	_ = db.Exec("INSERT INTO continent_drops (tenant_id, continent_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, 1, 1, 1, 1, 0, 1)", tm.Id()).Error
	_ = db.Exec("INSERT INTO reactor_drops (tenant_id, reactor_id, item_id, quest_id, chance) VALUES (?, 1, 1, 0, 1), (?, 1, 2, 0, 1), (?, 1, 3, 0, 1)", tm.Id(), tm.Id(), tm.Id()).Error

	w := doGet(t, db, tm)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Attributes struct {
				MonsterDropCount   int64   `json:"monsterDropCount"`
				ContinentDropCount int64   `json:"continentDropCount"`
				ReactorDropCount   int64   `json:"reactorDropCount"`
				UpdatedAt          *string `json:"updatedAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&env)
	if env.Data.Attributes.MonsterDropCount != 2 {
		t.Errorf("monster=%d", env.Data.Attributes.MonsterDropCount)
	}
	if env.Data.Attributes.ContinentDropCount != 1 {
		t.Errorf("continent=%d", env.Data.Attributes.ContinentDropCount)
	}
	if env.Data.Attributes.ReactorDropCount != 3 {
		t.Errorf("reactor=%d", env.Data.Attributes.ReactorDropCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("expected nil updatedAt, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestStatusHandler_TenantIsolation(t *testing.T) {
	db := setupDB(t)
	tm1 := makeTenant()
	tm2 := makeTenant()

	_ = db.Exec("INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, 100, 1, 1, 1, 0, 1)", tm1.Id()).Error
	_ = db.Exec("INSERT INTO monster_drops (tenant_id, monster_id, item_id, minimum_quantity, maximum_quantity, quest_id, chance) VALUES (?, 100, 1, 1, 1, 0, 1), (?, 100, 2, 1, 1, 0, 1)", tm2.Id(), tm2.Id()).Error

	w := doGet(t, db, tm1)
	var env struct {
		Data struct {
			Attributes struct {
				MonsterDropCount int64 `json:"monsterDropCount"`
			} `json:"attributes"`
		} `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&env)
	if env.Data.Attributes.MonsterDropCount != 1 {
		t.Errorf("expected 1 for tenant 1, got %d", env.Data.Attributes.MonsterDropCount)
	}
}
```

If `server.NewServerInformation` or `server.NewRouter` signatures differ from the above, copy them from the existing `extraction/status_test.go`.

- [ ] **Step 4: Verify build + tests**

```bash
cd services/atlas-drop-information/atlas.com/dis && go build ./... && go test ./seed/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-drop-information/atlas.com/dis/seed/
git commit -m "feat(atlas-drop-information): add GET /drops/seed/status handler"
```

---

## Task 5: Ingress — broaden drops regex

**Files:**
- Modify: `deploy/shared/routes.conf` (line 151)
- Modify: `deploy/k8s/ingress.yaml` (line 168)

- [ ] **Step 1: Edit `deploy/shared/routes.conf`**

Change block at line 151 from:

```nginx
location ~ ^/api/drops/seed$ {
  set $u "atlas-drop-information:8080";
  proxy_pass http://$u$request_uri;
}
```

to:

```nginx
location ~ ^/api/drops/seed(/.*)?$ {
  set $u "atlas-drop-information:8080";
  proxy_pass http://$u$request_uri;
}
```

- [ ] **Step 2: Edit `deploy/k8s/ingress.yaml`**

Change block at line 168 from:

```yaml
location ~ ^/api/drops/seed$ {
  proxy_pass http://atlas-drop-information:8080;
}
```

to:

```yaml
location ~ ^/api/drops/seed(/.*)?$ {
  proxy_pass http://atlas-drop-information:8080;
}
```

- [ ] **Step 3: Confirm no other ingress edits are needed**

Grep confirms all remaining new paths fall under existing catch-alls:

```bash
grep -n 'gachapons\|npcs/conversations\|quests/conversations\|api/shops\|api/portals\|reactors/actions\|maps/actions' deploy/shared/routes.conf
```

Expected: each matches a `(/.*)?$` block (see design §3.8 table).

- [ ] **Step 4: Commit**

```bash
git add deploy/shared/routes.conf deploy/k8s/ingress.yaml
git commit -m "fix(ingress): broaden /api/drops/seed regex to cover /seed/status"
```

---

## Task 6: atlas-gachapons — `Count()` on `gachapon`

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/gachapon/processor.go`
- Test: `services/atlas-gachapons/atlas.com/gachapons/gachapon/processor_test.go`

- [ ] **Step 1: Inspect existing test harness**

```bash
head -80 services/atlas-gachapons/atlas.com/gachapons/gachapon/processor_test.go
```

Note whether a `testDatabase`/`testTenant` pair exists. If not, follow Task 1 shape and add helpers scoped to the test file.

- [ ] **Step 2: Write failing tests**

Append three `Count` tests (empty / populated / tenant isolation). Seed by constructing domain models through the existing builder (or raw `INSERT INTO gachapons …`). Table has no `updated_at`, so assert `(N, nil, nil)`.

Example populated case using `db.Exec`:

```go
if err := db.Exec(
	"INSERT INTO gachapons (tenant_id, id, name, npc_ids, common_weight, uncommon_weight, rare_weight) VALUES (?, ?, ?, ?, ?, ?, ?)",
	te.Id(), "g1", "test", "{}", 100, 10, 1,
).Error; err != nil { t.Fatalf("%v", err) }
```

(SQLite may not parse `integer[]` literals — if the harness uses sqlite, fall back to creating a gachapon via the processor's `Create` method instead of raw SQL.)

- [ ] **Step 3: Verify failure**

```bash
cd services/atlas-gachapons/atlas.com/gachapons && go test ./gachapon/ -run Count -v
```

- [ ] **Step 4: Add `Count()` to interface + impl**

```go
// gachapon/processor.go
type Processor interface {
	// …existing methods…
	Count() (int64, *time.Time, error)
}

func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
```

Add `"time"` import.

- [ ] **Step 5: Verify**

```bash
cd services/atlas-gachapons/atlas.com/gachapons && go test ./gachapon/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-gachapons/atlas.com/gachapons/gachapon/
git commit -m "feat(atlas-gachapons): add Count() to gachapon processor"
```

---

## Task 7: atlas-gachapons — `Count()` on `item`

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/item/processor.go`
- Test: `services/atlas-gachapons/atlas.com/gachapons/item/processor_test.go`

- [ ] **Step 1: Inspect harness, then copy Task 6 pattern**

- [ ] **Step 2: Write three failing `Count` tests** (empty/populated/isolation). Use the existing `Create` path or raw `INSERT` against the item entity.

- [ ] **Step 3: Verify failure**

- [ ] **Step 4: Add `Count() (int64, *time.Time, error)` to the interface and impl — `(count, nil, nil)`**

- [ ] **Step 5: Verify**

```bash
cd services/atlas-gachapons/atlas.com/gachapons && go test ./item/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-gachapons/atlas.com/gachapons/item/
git commit -m "feat(atlas-gachapons): add Count() to gachapon item processor"
```

---

## Task 8: atlas-gachapons — `Count()` on `global`

**Files:**
- Modify: `services/atlas-gachapons/atlas.com/gachapons/global/processor.go`
- Test: `services/atlas-gachapons/atlas.com/gachapons/global/processor_test.go`

Identical shape to Task 7 against the `global` entity (global items). Implementation returns `(count, nil, nil)`.

- [ ] **Step 1: Inspect harness**
- [ ] **Step 2: Write failing `Count` tests** — empty / populated / tenant isolation.
- [ ] **Step 3: Run — verify failure**
- [ ] **Step 4: Add interface method + impl**
- [ ] **Step 5: `go test ./global/... && go build ./...`**
- [ ] **Step 6: Commit**

```bash
git add services/atlas-gachapons/atlas.com/gachapons/global/
git commit -m "feat(atlas-gachapons): add Count() to gachapon global processor"
```

---

## Task 9: atlas-gachapons — gachapons seed-status handler + route

**Files:**
- Create: `services/atlas-gachapons/atlas.com/gachapons/seed/status.go`
- Create: `services/atlas-gachapons/atlas.com/gachapons/seed/status_test.go`
- Modify: `services/atlas-gachapons/atlas.com/gachapons/seed/resource.go`

- [ ] **Step 1: Write `seed/status.go`**

Follow the Task 4 template verbatim. Differences:

- Package `seed` under `atlas-gachapons`.
- Import `gachapon`, `item`, and `global` sub-packages (match the `import` aliases used elsewhere in the service).
- Resource type `gachaponsSeedStatus`; attributes `GachaponCount`, `ItemCount`, `GlobalItemCount`, `UpdatedAt`.
- `UpdatedAt` is always `nil` (all three tables lack `updated_at`), so `maxUpdatedAtRFC3339` still returns `nil` — but keep it in place for symmetry.

- [ ] **Step 2: Register the route**

Edit `seed/resource.go`:

```go
router.HandleFunc("/gachapons/seed/status", registerHandler("get_gachapons_seed_status", handleGetSeedStatus)).Methods(http.MethodGet)
```

- [ ] **Step 3: Write `seed/status_test.go`**

Mirror Task 4's test file. Cases:

- Empty → 200, zero counts, `updatedAt: null`.
- Populated → 200, correct per-sub-table counts (insert 2 gachapons, 5 items, 1 global via the existing Create path or raw SQL).
- Tenant isolation → tm1 vs tm2; tm1's counts match only its rows.

- [ ] **Step 4: Verify**

```bash
cd services/atlas-gachapons/atlas.com/gachapons && go build ./... && go test ./seed/... -v
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-gachapons/atlas.com/gachapons/seed/
git commit -m "feat(atlas-gachapons): add GET /gachapons/seed/status handler"
```

---

## Task 10: atlas-npc-conversations — `Count()` on `npc`

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor_test.go` (create if absent; an `administrator_test.go` / `provider_test.go` already exist in the package)

The `conversations` table has a real `UpdatedAt` column.

- [ ] **Step 1: Inspect existing tests**

```bash
head -100 services/atlas-npc-conversations/atlas.com/npc/conversation/npc/provider_test.go
head -100 services/atlas-npc-conversations/atlas.com/npc/conversation/npc/administrator_test.go
```

Note the `setup`, tenant, migration helpers. Reuse them if present; otherwise add a small `processor_test.go` with an inline `setupDB(t)` + `makeTenant()` pair.

- [ ] **Step 2: Write failing `Count` tests**

Three cases. Populated case must assert `updated != nil` and that the returned `*time.Time` is within 5 seconds of `time.Now()`:

```go
func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := makeTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := setupDB(t)

	// insert two rows with CURRENT_TIMESTAMP
	for i := 0; i < 2; i++ {
		if err := db.Exec(
			"INSERT INTO conversations (id, tenant_id, npc_id, data) VALUES (?, ?, ?, ?)",
			uuid.New(), te.Id(), uint32(1000+i), `{"id":"00000000-0000-0000-0000-000000000000","npcId":1,"states":[]}`,
		).Error; err != nil {
			t.Fatalf("%v", err)
		}
	}

	p := npc.NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Errorf("count=%d", count)
	}
	if updated == nil {
		t.Fatalf("updatedAt is nil; expected non-nil")
	}
	if time.Since(*updated) > 5*time.Second {
		t.Errorf("updatedAt too old: %v", *updated)
	}
}
```

Plus empty (0,nil,nil) and tenant-isolation cases.

- [ ] **Step 3: Verify failure**

```bash
cd services/atlas-npc-conversations/atlas.com && go test ./npc/conversation/npc/ -run Count -v
```

- [ ] **Step 4: Add `Count()` + impl with MAX(updated_at) path**

Add `Count() (int64, *time.Time, error)` to the interface. Implementation:

```go
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	if count == 0 {
		return 0, nil, nil
	}
	row := p.db.WithContext(p.ctx).Model(&Entity{}).Select("MAX(updated_at)").Row()
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return 0, nil, err
	}
	if !raw.Valid || raw.String == "" {
		return count, nil, nil
	}
	t, err := parseDBTime(raw.String)
	if err != nil || t.IsZero() {
		return count, nil, nil
	}
	return count, &t, nil
}
```

Add the `parseDBTime` helper in the same file (or a new `time_util.go`). Copy verbatim from `services/atlas-data/atlas.com/data/data/status.go:93-108`. Add imports `"database/sql"` and `"time"`.

- [ ] **Step 5: Verify**

```bash
cd services/atlas-npc-conversations/atlas.com && go test ./npc/conversation/npc/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/npc/
git commit -m "feat(atlas-npc-conversations): add Count() to npc conversation processor"
```

---

## Task 11: atlas-npc-conversations — NPC conversations seed-status handler + route

**Files:**
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/seed_status.go`
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/seed_status_test.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/resource.go` (line 33-ish — append after the existing seed POST)

Single-count handler (no errgroup).

- [ ] **Step 1: Write `seed_status.go`**

```go
package npc

import (
	"atlas-npc-conversations/rest"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
)

type SeedStatusRestModel struct {
	Id                string  `json:"-"`
	ConversationCount int64   `json:"conversationCount"`
	UpdatedAt         *string `json:"updatedAt"`
}

func (r SeedStatusRestModel) GetName() string         { return "npcConversationsSeedStatus" }
func (r SeedStatusRestModel) GetID() string           { return r.Id }
func (r *SeedStatusRestModel) SetID(id string) error  { r.Id = id; return nil }
func (r SeedStatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}
func (r SeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}
func (r SeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}
func (r *SeedStatusRestModel) SetToOneReferenceID(_, _ string) error          { return nil }
func (r *SeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r *SeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func SeedStatusHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := d.Logger()
		t := tenant.MustFromContext(d.Context())
		count, updated, err := NewProcessor(l, d.Context(), d.DB()).Count()
		if err != nil {
			l.WithError(err).Errorf("Unable to read npc conversations seed status.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		res := SeedStatusRestModel{
			Id:                t.Id().String(),
			ConversationCount: count,
		}
		if updated != nil {
			s := updated.UTC().Format(time.RFC3339)
			res.UpdatedAt = &s
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[SeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
	}
}
```

- [ ] **Step 2: Register the route**

Edit `resource.go` — after the existing line 33 (`router.HandleFunc("/npcs/conversations/seed", ...)`):

```go
router.HandleFunc("/npcs/conversations/seed/status", registerHandler("get_npc_conversations_seed_status", SeedStatusHandler)).Methods(http.MethodGet)
```

- [ ] **Step 3: Write `seed_status_test.go`**

Follow Task 4's test harness, but only one sub-count. Three cases: empty, populated (insert 3 rows; assert `conversationCount: 3` and non-nil `updatedAt`), tenant isolation.

- [ ] **Step 4: Verify**

```bash
cd services/atlas-npc-conversations/atlas.com && go build ./... && go test ./npc/conversation/npc/... -v
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/npc/
git commit -m "feat(atlas-npc-conversations): add GET /npcs/conversations/seed/status"
```

---

## Task 12: atlas-npc-conversations — `Count()` on `quest`

Symmetric to Task 10 against the `quest_conversations` table.

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/processor.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/processor_test.go` (create or extend)

- [ ] **Step 1: Inspect existing test harness**

- [ ] **Step 2: Write three failing `Count` tests** (empty / populated with MAX(updated_at) asserted non-nil / tenant isolation). Insert rows with explicit fields — columns: `(id, tenant_id, quest_id, npc_id, data)`.

- [ ] **Step 3: Verify failure**

- [ ] **Step 4: Add `Count()` interface method + impl**

Same body as Task 10 (with MAX(updated_at) path). Copy `parseDBTime` into the same package if not already introduced.

- [ ] **Step 5: Verify**

```bash
cd services/atlas-npc-conversations/atlas.com && go test ./npc/conversation/quest/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/quest/
git commit -m "feat(atlas-npc-conversations): add Count() to quest conversation processor"
```

---

## Task 13: atlas-npc-conversations — Quest conversations seed-status handler + route

**Files:**
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/seed_status.go`
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/seed_status_test.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/resource.go`

Mirror Task 11. Differences:

- Package `quest`.
- `SeedStatusRestModel` — resource type string `"questConversationsSeedStatus"`.
- Route: `GET /quests/conversations/seed/status`.

- [ ] **Step 1: Write `seed_status.go`** (copy Task 11, change resource type + handler name).
- [ ] **Step 2: Register route in `resource.go`** — after the existing seed POST registration:

```go
router.HandleFunc("/quests/conversations/seed/status", registerHandler("get_quest_conversations_seed_status", SeedStatusHandler)).Methods(http.MethodGet)
```

- [ ] **Step 3: Write `seed_status_test.go`** — 3 cases as in Task 11.
- [ ] **Step 4: Verify**

```bash
cd services/atlas-npc-conversations/atlas.com && go build ./... && go test ./npc/conversation/quest/... -v
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/quest/
git commit -m "feat(atlas-npc-conversations): add GET /quests/conversations/seed/status"
```

---

## Task 14: atlas-npc-shops — `Count()` on `shops`

**Files:**
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/processor.go`
- Test: `services/atlas-npc-shops/atlas.com/npc/shops/processor_test.go`

Entity embeds `gorm.Model`, so `updated_at` is present.

- [ ] **Step 1: Inspect harness**
- [ ] **Step 2: Write three failing `Count` tests** — empty / populated (assert non-nil `*time.Time`) / tenant isolation.
- [ ] **Step 3: Verify failure**
- [ ] **Step 4: Add interface method + impl** — use the full MAX(updated_at) body from Task 10. Copy `parseDBTime` into a helper file (e.g., `shops/time_util.go`) or inline.
- [ ] **Step 5: Verify**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go test ./shops/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/shops/
git commit -m "feat(atlas-npc-shops): add Count() to shops processor"
```

---

## Task 15: atlas-npc-shops — `Count()` on `commodities`

**Files:**
- Modify: `services/atlas-npc-shops/atlas.com/npc/commodities/processor.go`
- Test: `services/atlas-npc-shops/atlas.com/npc/commodities/processor_test.go`

Entity embeds `gorm.Model`; `updated_at` is present. Same shape as Task 14.

- [ ] **Step 1: Inspect harness**
- [ ] **Step 2: Write three failing tests**
- [ ] **Step 3: Verify failure**
- [ ] **Step 4: Add interface method + impl** (MAX(updated_at) variant)
- [ ] **Step 5: Verify**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go test ./commodities/... && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/commodities/
git commit -m "feat(atlas-npc-shops): add Count() to commodities processor"
```

---

## Task 16: atlas-npc-shops — shops seed-status handler + route

**Files:**
- Create: `services/atlas-npc-shops/atlas.com/npc/seed/status.go`
- Create: `services/atlas-npc-shops/atlas.com/npc/seed/status_test.go`
- Modify: `services/atlas-npc-shops/atlas.com/npc/seed/resource.go`

Compound handler (shops + commodities) — 2 errgroup goroutines.

- [ ] **Step 1: Write `seed/status.go`**

Copy the Task 4 template. Differences:

- Package `seed` under `atlas-npc-shops/npc/seed`.
- Resource type `npcShopsSeedStatus`.
- Attributes: `ShopCount`, `CommodityCount`, `UpdatedAt`.
- Two errgroup goroutines (shops + commodities).
- `UpdatedAt` = `maxUpdatedAtRFC3339(shopsUpdated, commoditiesUpdated)` — will be non-nil when rows exist.

- [ ] **Step 2: Register route**

```go
router.HandleFunc("/shops/seed/status", registerHandler("get_shops_seed_status", handleGetSeedStatus)).Methods(http.MethodGet)
```

- [ ] **Step 3: Write `seed/status_test.go`** — empty, populated (assert `updatedAt != nil`), tenant isolation.
- [ ] **Step 4: Verify**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./seed/... -v
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/seed/
git commit -m "feat(atlas-npc-shops): add GET /shops/seed/status handler"
```

---

## Task 17: atlas-portal-actions — `Count()` + seed-status endpoint

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/processor.go`
- Test: `services/atlas-portal-actions/atlas.com/portal/script/processor_test.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/script/seed_status.go`
- Create: `services/atlas-portal-actions/atlas.com/portal/script/seed_status_test.go`
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/resource.go` (register new GET alongside existing seed POST)

Single-count handler. Entity has `UpdatedAt`.

- [ ] **Step 1: Write `Count` tests** (empty / populated / isolation) in `processor_test.go`. Columns for `portal_scripts`: `(id, tenant_id, portal_id, map_id, data)`; `created_at`/`updated_at` default to CURRENT_TIMESTAMP.

- [ ] **Step 2: Verify tests fail**

```bash
cd services/atlas-portal-actions/atlas.com/portal && go test ./script/ -run Count -v
```

- [ ] **Step 3: Add `Count()` interface method + impl** with MAX(updated_at) path. Copy `parseDBTime` helper.

- [ ] **Step 4: Run tests** — expect pass.

- [ ] **Step 5: Write `seed_status.go`** — single-count variant (like Task 11's NPC seed_status) with resource type `"portalScriptsSeedStatus"` and attribute `ScriptCount`.

- [ ] **Step 6: Register route in `resource.go`**

```go
router.HandleFunc("/portals/scripts/seed/status", registerHandler("get_portal_scripts_seed_status", SeedStatusHandler)).Methods(http.MethodGet)
```

- [ ] **Step 7: Write `seed_status_test.go`** — 3 cases.

- [ ] **Step 8: Verify**

```bash
cd services/atlas-portal-actions/atlas.com/portal && go build ./... && go test ./script/... -v
```

- [ ] **Step 9: Commit**

```bash
git add services/atlas-portal-actions/atlas.com/portal/script/
git commit -m "feat(atlas-portal-actions): add GET /portals/scripts/seed/status"
```

---

## Task 18: atlas-reactor-actions — `Count()` + seed-status endpoint

Same shape as Task 17.

**Files:**
- Modify: `services/atlas-reactor-actions/atlas.com/reactor/script/processor.go`
- Test: `services/atlas-reactor-actions/atlas.com/reactor/script/processor_test.go`
- Create: `services/atlas-reactor-actions/atlas.com/reactor/script/seed_status.go`
- Create: `services/atlas-reactor-actions/atlas.com/reactor/script/seed_status_test.go`
- Modify: `services/atlas-reactor-actions/atlas.com/reactor/script/resource.go`

Entity columns: `(id, tenant_id, reactor_id, data)`. Resource type `"reactorScriptsSeedStatus"`. Route `GET /reactors/actions/seed/status`.

- [ ] **Step 1–8: Mirror Task 17 exactly** against `reactor_scripts`.
- [ ] **Step 9: Commit**

```bash
git add services/atlas-reactor-actions/atlas.com/reactor/script/
git commit -m "feat(atlas-reactor-actions): add GET /reactors/actions/seed/status"
```

---

## Task 19: atlas-map-actions — `Count()` + seed-status endpoint

Same shape.

**Files:**
- Modify: `services/atlas-map-actions/atlas.com/map-actions/script/processor.go`
- Test: `services/atlas-map-actions/atlas.com/map-actions/script/processor_test.go`
- Create: `services/atlas-map-actions/atlas.com/map-actions/script/seed_status.go`
- Create: `services/atlas-map-actions/atlas.com/map-actions/script/seed_status_test.go`
- Modify: `services/atlas-map-actions/atlas.com/map-actions/script/resource.go`

Entity columns: `(id, tenant_id, script_name, script_type, data)`. Resource type `"mapActionScriptsSeedStatus"`. Route `GET /maps/actions/seed/status`.

- [ ] **Step 1–8: Mirror Task 17** against `map_scripts`.
- [ ] **Step 9: Commit**

```bash
git add services/atlas-map-actions/atlas.com/map-actions/script/
git commit -m "feat(atlas-map-actions): add GET /maps/actions/seed/status"
```

---

## Task 20: atlas-ui — Extract `SetupRow` component

**Files:**
- Create: `services/atlas-ui/src/components/features/setup/SetupRow.tsx`
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx`

This is a pure extraction — no behavior change yet.

- [ ] **Step 1: Create `SetupRow.tsx`** with the inline `GameDataRow` body verbatim and the two helpers moved out of `SetupPage.tsx`:

```tsx
import type { ReactNode } from "react";

interface SetupRowProps {
  icon: ReactNode;
  label: ReactNode;
  badge: ReactNode;
  action: ReactNode;
  warning?: ReactNode;
}

export function SetupRow({ icon, label, badge, action, warning }: SetupRowProps) {
  return (
    <div className="flex flex-col gap-2 border-b last:border-0 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="text-muted-foreground">{icon}</div>
          <div>
            <p className="font-medium text-sm">{label}</p>
            <p
              className="text-xs text-muted-foreground"
              aria-live="polite"
            >
              {badge}
            </p>
          </div>
        </div>
        {action}
      </div>
      {warning}
    </div>
  );
}

export function formatCount(n: number): string {
  return new Intl.NumberFormat().format(n);
}

export function pluralize(n: number, singular: string, plural: string): string {
  return n === 1 ? singular : plural;
}
```

- [ ] **Step 2: Update `SetupPage.tsx`**

Remove the inline `GameDataRow` (lines ~89–118), `formatCount` (lines ~81–82), and `pluralize` (lines ~85–86). Replace the three `<GameDataRow …/>` call sites with `<SetupRow …/>`. Add the import:

```tsx
import { SetupRow, formatCount, pluralize } from "@/components/features/setup/SetupRow";
```

Keep `formatBytes` in `SetupPage.tsx` — only the WZ row uses it.

- [ ] **Step 3: Verify lint + build**

```bash
cd services/atlas-ui && npm run lint && npm run build
```

Expected: no new errors; dist built.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/components/features/setup/SetupRow.tsx services/atlas-ui/src/pages/SetupPage.tsx
git commit -m "refactor(atlas-ui): extract SetupRow + formatting helpers to dedicated module"
```

---

## Task 21: atlas-ui — Add seed-status service-layer types + getters

**Files:**
- Modify: `services/atlas-ui/src/services/api/seed.service.ts`

- [ ] **Step 1: Add eight interfaces + eight getters**

After the existing `DataStatus` interface, add:

```ts
export interface DropsSeedStatus {
  monsterDropCount: number;
  continentDropCount: number;
  reactorDropCount: number;
  updatedAt: string | null;
}

export interface GachaponsSeedStatus {
  gachaponCount: number;
  itemCount: number;
  globalItemCount: number;
  updatedAt: string | null;
}

export interface NpcConversationsSeedStatus {
  conversationCount: number;
  updatedAt: string | null;
}

export interface QuestConversationsSeedStatus {
  conversationCount: number;
  updatedAt: string | null;
}

export interface NpcShopsSeedStatus {
  shopCount: number;
  commodityCount: number;
  updatedAt: string | null;
}

export interface PortalScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}

export interface ReactorScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}

export interface MapActionScriptsSeedStatus {
  scriptCount: number;
  updatedAt: string | null;
}
```

Inside `class SeedService`, after `getDataStatus`, add eight getters:

```ts
async getDropsSeedStatus(tenant: Tenant): Promise<DropsSeedStatus> {
  return fetchJsonApi<DropsSeedStatus>('/api/drops/seed/status', tenant);
}

async getGachaponsSeedStatus(tenant: Tenant): Promise<GachaponsSeedStatus> {
  return fetchJsonApi<GachaponsSeedStatus>('/api/gachapons/seed/status', tenant);
}

async getNpcConversationsSeedStatus(tenant: Tenant): Promise<NpcConversationsSeedStatus> {
  return fetchJsonApi<NpcConversationsSeedStatus>('/api/npcs/conversations/seed/status', tenant);
}

async getQuestConversationsSeedStatus(tenant: Tenant): Promise<QuestConversationsSeedStatus> {
  return fetchJsonApi<QuestConversationsSeedStatus>('/api/quests/conversations/seed/status', tenant);
}

async getNpcShopsSeedStatus(tenant: Tenant): Promise<NpcShopsSeedStatus> {
  return fetchJsonApi<NpcShopsSeedStatus>('/api/shops/seed/status', tenant);
}

async getPortalScriptsSeedStatus(tenant: Tenant): Promise<PortalScriptsSeedStatus> {
  return fetchJsonApi<PortalScriptsSeedStatus>('/api/portals/scripts/seed/status', tenant);
}

async getReactorScriptsSeedStatus(tenant: Tenant): Promise<ReactorScriptsSeedStatus> {
  return fetchJsonApi<ReactorScriptsSeedStatus>('/api/reactors/actions/seed/status', tenant);
}

async getMapActionScriptsSeedStatus(tenant: Tenant): Promise<MapActionScriptsSeedStatus> {
  return fetchJsonApi<MapActionScriptsSeedStatus>('/api/maps/actions/seed/status', tenant);
}
```

- [ ] **Step 2: Verify typecheck + build**

```bash
cd services/atlas-ui && npm run build
```

Expected: no TS errors.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-ui/src/services/api/seed.service.ts
git commit -m "feat(atlas-ui): add seed-status typed getters to seed.service"
```

---

## Task 22: atlas-ui — Add seed-status hooks + mutation invalidation

**Files:**
- Modify: `services/atlas-ui/src/lib/hooks/api/useSeed.ts`

- [ ] **Step 1: Add eight query keys**

Immediately after the existing `dataStatusKey` declaration (line 18), insert:

```ts
const dropsSeedStatusKey = (tenantId: string) => ['dropsSeedStatus', tenantId] as const;
const gachaponsSeedStatusKey = (tenantId: string) => ['gachaponsSeedStatus', tenantId] as const;
const npcConversationsSeedStatusKey = (tenantId: string) => ['npcConversationsSeedStatus', tenantId] as const;
const questConversationsSeedStatusKey = (tenantId: string) => ['questConversationsSeedStatus', tenantId] as const;
const npcShopsSeedStatusKey = (tenantId: string) => ['npcShopsSeedStatus', tenantId] as const;
const portalScriptsSeedStatusKey = (tenantId: string) => ['portalScriptsSeedStatus', tenantId] as const;
const reactorScriptsSeedStatusKey = (tenantId: string) => ['reactorScriptsSeedStatus', tenantId] as const;
const mapActionScriptsSeedStatusKey = (tenantId: string) => ['mapActionScriptsSeedStatus', tenantId] as const;
```

- [ ] **Step 2: Rewrite each existing `useSeedXxx` mutation**

Replace each of the eight mutation hooks (lines 20–58) with a version that invalidates the matching status key on success. Example for `useSeedDrops`:

```ts
export function useSeedDrops(): UseMutationResult<void, Error, void> {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => seedService.seedDrops(),
    onSuccess: () => {
      if (!activeTenant) return;
      void queryClient.invalidateQueries({ queryKey: dropsSeedStatusKey(activeTenant.id) });
    },
  });
}
```

Repeat verbatim for the other seven, swapping in the matching status key. `seedNpcConversations` / `seedQuestConversations` / `seedNpcShops` / `seedPortalScripts` / `seedReactorScripts` / `seedMapActionScripts` already return `SeedResult` (not `void`) — keep the existing return-type generic (`UseMutationResult<unknown, Error, void>` for those that returned `unknown`). Match the original return shape of each hook.

- [ ] **Step 3: Add eight new status hooks**

After `useDataStatus` (line 120–129), append one hook per status key. Template:

```ts
export function useDropsSeedStatus(): UseQueryResult<DropsSeedStatus, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant ? dropsSeedStatusKey(activeTenant.id) : ['dropsSeedStatus', 'none'],
    queryFn: () => seedService.getDropsSeedStatus(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
  });
}
```

Add the remaining seven — one per endpoint — using the matching key, getter, and return type.

- [ ] **Step 4: Update imports**

At the top of `useSeed.ts`, extend the existing `seedService` import with the eight new types:

```ts
import {
  seedService,
  type DataStatus,
  type DropsSeedStatus,
  type GachaponsSeedStatus,
  type MapActionScriptsSeedStatus,
  type NpcConversationsSeedStatus,
  type NpcShopsSeedStatus,
  type PortalScriptsSeedStatus,
  type QuestConversationsSeedStatus,
  type ReactorScriptsSeedStatus,
  type WzExtractionStatus,
  type WzInputStatus,
} from '@/services/api/seed.service';
```

- [ ] **Step 5: Verify typecheck**

```bash
cd services/atlas-ui && npm run build
```

Expected: no TS errors.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useSeed.ts
git commit -m "feat(atlas-ui): add seed-status polling hooks and mutation invalidation"
```

---

## Task 23: atlas-ui — Rewire SetupPage Seed Data as `<SetupRow>` grid

**Files:**
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx`

- [ ] **Step 1: Import new hooks**

Extend the `useSeed` import block to add:

```tsx
import {
  // …existing…
  useDropsSeedStatus,
  useGachaponsSeedStatus,
  useNpcConversationsSeedStatus,
  useQuestConversationsSeedStatus,
  useNpcShopsSeedStatus,
  usePortalScriptsSeedStatus,
  useReactorScriptsSeedStatus,
  useMapActionScriptsSeedStatus,
} from "@/lib/hooks/api/useSeed";
```

Also import the seed-status types used in the `formatBadge` signatures:

```tsx
import type {
  DropsSeedStatus,
  GachaponsSeedStatus,
  NpcConversationsSeedStatus,
  QuestConversationsSeedStatus,
  NpcShopsSeedStatus,
  PortalScriptsSeedStatus,
  ReactorScriptsSeedStatus,
  MapActionScriptsSeedStatus,
} from "@/services/api/seed.service";
```

- [ ] **Step 2: Call each status hook at the top of `SetupPage`**

Immediately after `const dataStatus = useDataStatus();`:

```tsx
const dropsSeed = useDropsSeedStatus();
const gachaponsSeed = useGachaponsSeedStatus();
const npcConversationsSeed = useNpcConversationsSeedStatus();
const questConversationsSeed = useQuestConversationsSeedStatus();
const npcShopsSeed = useNpcShopsSeedStatus();
const portalScriptsSeed = usePortalScriptsSeedStatus();
const reactorScriptsSeed = useReactorScriptsSeedStatus();
const mapActionScriptsSeed = useMapActionScriptsSeedStatus();
```

- [ ] **Step 3: Delete the old `SeedButton` component and the current `seedActions` array**

Remove the `SeedButton` component definition (and its `SeedButtonProps` interface) if no other file imports them:

```bash
grep -rn "SeedButton\b" services/atlas-ui/src/
```

If the grep only hits `SetupPage.tsx`, delete the component.

- [ ] **Step 4: Build the row spec with badge formatters**

Replace the existing `seedActions` array with:

```tsx
const seedRows = [
  {
    label: "Monster & Reactor Drops",
    icon: <Database className="h-5 w-5" />,
    mutation: seedDrops,
    status: dropsSeed,
    formatBadge: (d?: DropsSeedStatus) =>
      !d
        ? "—"
        : `${formatCount(d.monsterDropCount)} ${pluralize(d.monsterDropCount, "monster drop", "monster drops")} / ${formatCount(d.continentDropCount)} ${pluralize(d.continentDropCount, "continent drop", "continent drops")} / ${formatCount(d.reactorDropCount)} ${pluralize(d.reactorDropCount, "reactor drop", "reactor drops")}`,
  },
  {
    label: "Gachapons",
    icon: <Package className="h-5 w-5" />,
    mutation: seedGachapons,
    status: gachaponsSeed,
    formatBadge: (d?: GachaponsSeedStatus) =>
      !d
        ? "—"
        : `${formatCount(d.gachaponCount)} ${pluralize(d.gachaponCount, "gachapon", "gachapons")} / ${formatCount(d.itemCount)} ${pluralize(d.itemCount, "item", "items")} / ${formatCount(d.globalItemCount)} ${pluralize(d.globalItemCount, "global item", "global items")}`,
  },
  {
    label: "NPC Conversations",
    icon: <MessageSquare className="h-5 w-5" />,
    mutation: seedNpcConversations,
    status: npcConversationsSeed,
    formatBadge: (d?: NpcConversationsSeedStatus) =>
      !d ? "—" : `${formatCount(d.conversationCount)} ${pluralize(d.conversationCount, "conversation", "conversations")}`,
  },
  {
    label: "Quest Conversations",
    icon: <HelpCircle className="h-5 w-5" />,
    mutation: seedQuestConversations,
    status: questConversationsSeed,
    formatBadge: (d?: QuestConversationsSeedStatus) =>
      !d ? "—" : `${formatCount(d.conversationCount)} ${pluralize(d.conversationCount, "conversation", "conversations")}`,
  },
  {
    label: "NPC Shops",
    icon: <Store className="h-5 w-5" />,
    mutation: seedNpcShops,
    status: npcShopsSeed,
    formatBadge: (d?: NpcShopsSeedStatus) =>
      !d
        ? "—"
        : `${formatCount(d.shopCount)} ${pluralize(d.shopCount, "shop", "shops")} / ${formatCount(d.commodityCount)} ${pluralize(d.commodityCount, "commodity", "commodities")}`,
  },
  {
    label: "Portal Scripts",
    icon: <DoorOpen className="h-5 w-5" />,
    mutation: seedPortalScripts,
    status: portalScriptsSeed,
    formatBadge: (d?: PortalScriptsSeedStatus) =>
      !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
  },
  {
    label: "Reactor Scripts",
    icon: <Zap className="h-5 w-5" />,
    mutation: seedReactorScripts,
    status: reactorScriptsSeed,
    formatBadge: (d?: ReactorScriptsSeedStatus) =>
      !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
  },
  {
    label: "Map Action Scripts",
    icon: <Map className="h-5 w-5" />,
    mutation: seedMapActionScripts,
    status: mapActionScriptsSeed,
    formatBadge: (d?: MapActionScriptsSeedStatus) =>
      !d ? "—" : `${formatCount(d.scriptCount)} ${pluralize(d.scriptCount, "script", "scripts")}`,
  },
];
```

- [ ] **Step 5: Replace the Seed Data render block**

Replace the current `seedActions.map(...)` block (lines ~373–383) with:

```tsx
<div className="grid gap-0">
  {seedRows.map((row) => (
    <SetupRow
      key={row.label}
      icon={row.icon}
      label={row.label}
      badge={row.formatBadge(row.status.data as never)}
      action={
        <Button
          size="sm"
          variant="outline"
          onClick={() => handleSeed(row.mutation, row.label)}
          disabled={row.mutation.isPending}
        >
          {row.mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Seed"}
        </Button>
      }
    />
  ))}
</div>
```

(`as never` is the TypeScript escape hatch needed because `row.formatBadge` is the union of eight different function signatures — one per status shape — and `row.status.data` is the matching union of statuses. The per-row pairing is safe; the cast silences the variance check.)

- [ ] **Step 6: Manually verify in browser**

```bash
cd services/atlas-ui && npm run dev
```

Open `http://localhost:5173/setup` (or `http://localhost:3000` if routing through the compose stack). With the backend running and a tenant selected, each seed row should show a count badge. Clicking a Seed button should:

- Invalidate the row's status query immediately → a poll fires.
- For sync seeds (shops, conversations, scripts): badge updates within ~1 s.
- For async seeds (drops, gachapons): badge numbers climb across 5-second ticks.

If no tenant is selected or a backend is unreachable, that row's badge reads `"—"`.

- [ ] **Step 7: Run test suite + build**

```bash
cd services/atlas-ui && npm run lint && npm run test && npm run build
```

- [ ] **Step 8: Commit**

```bash
git add services/atlas-ui/src/pages/SetupPage.tsx
git commit -m "feat(atlas-ui): wire seed-status badges into SetupPage Seed Data rows"
```

---

## Task 24: atlas-ui — Seed-status hook tests

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.ts`

- [ ] **Step 1: Scaffold the test file with shared harness**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import {
  useDropsSeedStatus,
  useSeedDrops,
  // …import remaining seven status hooks + seven mutation hooks…
} from '../useSeed';
import { seedService } from '@/services/api/seed.service';
import * as tenantContext from '@/context/tenant-context';

vi.mock('@/services/api/seed.service', () => ({
  seedService: {
    getDropsSeedStatus: vi.fn(),
    getGachaponsSeedStatus: vi.fn(),
    getNpcConversationsSeedStatus: vi.fn(),
    getQuestConversationsSeedStatus: vi.fn(),
    getNpcShopsSeedStatus: vi.fn(),
    getPortalScriptsSeedStatus: vi.fn(),
    getReactorScriptsSeedStatus: vi.fn(),
    getMapActionScriptsSeedStatus: vi.fn(),
    seedDrops: vi.fn(),
    seedGachapons: vi.fn(),
    seedNpcConversations: vi.fn(),
    seedQuestConversations: vi.fn(),
    seedNpcShops: vi.fn(),
    seedPortalScripts: vi.fn(),
    seedReactorScripts: vi.fn(),
    seedMapActionScripts: vi.fn(),
  },
}));

vi.mock('@/context/tenant-context', () => ({
  useTenant: vi.fn(),
}));

const fakeTenant = { id: 'tenant-1', region: 'GMS', majorVersion: 83, minorVersion: 1 };

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return {
    qc,
    wrapper: ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    ),
  };
}

beforeEach(() => vi.clearAllMocks());
```

- [ ] **Step 2: Polling-enabled-when-tenant-active test (one per hook, or use `describe.each` to DRY)**

```ts
describe.each([
  ['useDropsSeedStatus', useDropsSeedStatus, 'getDropsSeedStatus', 'dropsSeedStatus'],
  ['useGachaponsSeedStatus', useGachaponsSeedStatus, 'getGachaponsSeedStatus', 'gachaponsSeedStatus'],
  // … list the remaining six …
] as const)('%s', (_, hook, method, key) => {
  it('enables polling and keys by tenant id when a tenant is active', async () => {
    (tenantContext.useTenant as any).mockReturnValue({ activeTenant: fakeTenant });
    (seedService as any)[method].mockResolvedValue({ /* minimal valid payload */ });

    const { wrapper, qc } = makeWrapper();
    const { result } = renderHook(() => hook(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect((seedService as any)[method]).toHaveBeenCalledWith(fakeTenant);
    expect(qc.getQueryData([key, fakeTenant.id])).toBeDefined();
  });

  it('disables polling when no tenant is active', () => {
    (tenantContext.useTenant as any).mockReturnValue({ activeTenant: null });

    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => hook(), { wrapper });

    expect(result.current.fetchStatus).toBe('idle');
    expect((seedService as any)[method]).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 3: `onSuccess` invalidation test per mutation hook**

```ts
describe.each([
  ['useSeedDrops', useSeedDrops, 'seedDrops', 'dropsSeedStatus'],
  // … list the remaining seven …
] as const)('%s mutation', (_, hook, method, statusKeyRoot) => {
  it('invalidates the matching status key on success', async () => {
    (tenantContext.useTenant as any).mockReturnValue({ activeTenant: fakeTenant });
    (seedService as any)[method].mockResolvedValue(undefined);

    const { wrapper, qc } = makeWrapper();
    const invalidate = vi.spyOn(qc, 'invalidateQueries');
    const { result } = renderHook(() => hook(), { wrapper });

    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidate).toHaveBeenCalledWith({ queryKey: [statusKeyRoot, fakeTenant.id] });
  });
});
```

- [ ] **Step 4: Run + debug**

```bash
cd services/atlas-ui && npm run test -- useSeed
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.ts
git commit -m "test(atlas-ui): add seed-status polling + mutation invalidation tests"
```

---

## Task 25: End-to-end smoke — ingress, curl, browser

**Files:** none (verification-only)

- [ ] **Step 1: Bring the stack up**

```bash
docker compose -f deploy/compose/docker-compose.core.yml up -d --build
```

Wait for services to be healthy.

- [ ] **Step 2: `curl` each new endpoint**

With `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` set to your local tenant's values:

```bash
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/drops/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/gachapons/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/npcs/conversations/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/quests/conversations/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/shops/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/portals/scripts/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/reactors/actions/seed/status | jq
curl -s -H "TENANT_ID: $TID" -H "REGION: $REG" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" \
  http://localhost:8080/api/maps/actions/seed/status | jq
```

Expected for each: 200 with the envelope documented in `api-contracts.md`. `type` matches the expected resource-type string and `id` equals `$TID`.

- [ ] **Step 3: Verify portals routing lands on atlas-portal-actions (not atlas-portals)**

```bash
docker compose logs atlas-portal-actions | grep 'get_portal_scripts_seed_status'
```

Expected: at least one entry after the curl.

- [ ] **Step 4: Browser smoke**

Open `http://localhost:3000/setup`. Confirm:

- All 11 rows (3 Game Data + 8 Seed Data) use the same layout.
- Every Seed row shows a count badge (initially `"—"`, then a real number within 5 s).
- Clicking Seed on NPC Shops updates its badge within 1 tick.
- Clicking Seed on Drops causes the monster/continent/reactor numbers to climb over several polls.
- Stopping `atlas-drop-information` (`docker compose stop atlas-drop-information`) makes only the drops badge revert to `"—"` without any toast or banner; restarting the service restores the badge.

- [ ] **Step 5: Run all affected service tests one more time**

```bash
for svc in atlas-drop-information atlas-gachapons atlas-npc-conversations atlas-npc-shops atlas-portal-actions atlas-reactor-actions atlas-map-actions; do
  dir=$(ls -d services/$svc/atlas.com/*/ | head -1)
  echo "==> $dir"
  (cd "$dir" && go test ./...) || break
done
(cd services/atlas-ui && npm run test && npm run build)
```

Expected: all green.

- [ ] **Step 6: No commit — this task is verification-only**

If any step failed, re-open the owning task and fix before merging.

---

## Self-review checklist

This plan has been checked for:

- [x] **Spec coverage** — each PRD §4 sub-section maps to a task:
  - PRD §4.1 endpoint table → Tasks 4, 9, 11, 13, 16, 17, 18, 19 (one per endpoint).
  - PRD §4.2 envelope shape → implemented in every handler task.
  - PRD §4.3 ingress → Task 5 (drops broadening) + Task 25 (manual verification of the other seven).
  - PRD §4.4 service layer → Task 21.
  - PRD §4.5 React Query hooks → Task 22.
  - PRD §4.6 SetupPage → Tasks 20 + 23.
  - PRD §4.7 error-state rendering → Task 23 (formatBadge `!d → "—"`).
  - PRD §4.8 pluralisation → Tasks 20 + 23.
  - PRD §6 tests (Count methods + handlers + hooks) → Tasks 1–3, 6–8, 10, 12, 14–15, 17–19 (unit), 4/9/11/13/16/17/18/19 (handler), 24 (hook).
- [x] **Placeholder scan** — no `TBD`, no "similar to Task N" without code. Tasks 7, 8, 12, 15, 18, 19 explicitly reuse the shape of an earlier task and inline the differences; the canonical code remains in the earlier task. Tasks 2 and 3 reuse Task 1's test shape but with table-specific `INSERT` statements shown explicitly.
- [x] **Type consistency** — interface names (`Processor`, `ScriptProcessor`) and method signature `Count() (int64, *time.Time, error)` are identical across all 13 sub-resource tasks. Resource-type strings (`dropsSeedStatus`, `gachaponsSeedStatus`, `npcConversationsSeedStatus`, `questConversationsSeedStatus`, `npcShopsSeedStatus`, `portalScriptsSeedStatus`, `reactorScriptsSeedStatus`, `mapActionScriptsSeedStatus`) match PRD §4.1 + api-contracts.md. Hook names (`useDropsSeedStatus`, etc.) and query-key strings match the resource types. Service-layer type names (`DropsSeedStatus`, etc.) match the hook return types.

