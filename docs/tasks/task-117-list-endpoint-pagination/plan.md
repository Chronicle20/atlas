# List Endpoint Pagination Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every collection GET endpoint in the Go services paginates (JSON:API `page[number]`/`page[size]` + envelope), DB-backed lists page in SQL, internal "need everything" consumers drain page-by-page, and atlas-ui list views page server-side.

**Architecture:** A `model.Paged[T]` container flows through the existing lazy `model.Provider` composition (`MapPaged` lifts item transforms/decorators over it). `database.PagedQuery` runs COUNT + OFFSET/LIMIT on the same scoped `*gorm.DB` with a schema-derived PK tie-break. `paginate.ParseParams` (hoisted from atlas-data) parses params; `paginate.Slice` pages already-materialized slices (registries, doc sub-lists). Client side, `requests.PagedProvider`/`DrainProvider` fetch one page / all pages, tolerating envelope-less responses so consumers can land before servers.

**Tech Stack:** Go (GORM, gorilla/mux, api2go/jsonapi, logrus, httptest, sqlite via `databasetest`), TypeScript/React (Next.js, TanStack React Query) for atlas-ui.

**Spec:** `docs/tasks/task-117-list-endpoint-pagination/design.md` (authoritative), `prd.md`, `endpoint-inventory.md` (endpoint census — the sweep tasks' checklists come from it).

## Global Constraints

- Worktree: all work happens in `.worktrees/task-117-list-endpoint-pagination` on branch `task-117-list-endpoint-pagination`. Every implementer must `cd` there first and verify `git branch --show-current` after each commit.
- Verification gauntlet per touched Go module (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` clean; `docker buildx bake atlas-<svc>` from the worktree root for every service whose `go.mod` was touched; `tools/redis-key-guard.sh` clean from repo root. atlas-ui: `npm run build` (type-checks tests too) + `npm test`.
- Param contract: `page[number]` 1-based, default 1; `page[size]` default per class, both non-integer/`<1`/`>max` → HTTP 400 (JSON:API error object, no silent clamping). Legacy `?limit=` → 400.
- Page-size classes (design §4): standard collections default **50** max **250**; Group C game-capped lists default **250** max **250**; growing logs (`/visits`, `/sessions`, `/history/accounts/{id}`) default **50** max **250**.
- Envelope: `meta: {total, page: {number, size, last}}` + `links: self/first/prev/next/last` — produced only by the existing `paginate.Envelope` / `server.MarshalPaginatedResponse`; never hand-assembled.
- Unfiltered `GetAll`-style processor methods are **deleted**, not shadowed. Filtered fetch-all methods used by same-service internal logic may remain; their REST handlers use paged variants.
- Hard correctness rule (PRD FR-5.2): every internal Go consumer of a converted endpoint that semantically needs the whole collection switches to `requests.DrainProvider`. Within each task, convert consumers **before** (or in the same commit as) the server — no intermediate commit may leave a consumer reading a silently truncated collection.
- DB-backed lists page in SQL (`LIMIT/OFFSET` + `COUNT` on the same scoped query), with total ordering (PK tie-break). Registry/doc-sub-list sources page the materialized slice via `paginate.Slice` over a deterministically ordered slice.
- `include` decorators run only over the returned page's rows.
- No `// TODO`, stubs, or 501s in landed commits. No literal home/absolute paths in committed files.
- Test setup uses the project's Builder pattern; no `*_testhelpers.go` files. External-client tests are httptest-backed (never rely on `mock/` FakeClients for unmarshal coverage — see `libs/atlas-rest/CLAUDE.md`).
- JSON:API target structs decoded by `requests` MUST implement `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs if upstream responses carry `relationships` (existing models already do; new test fixtures must include a `relationships` block to keep this pinned).
- Verify before editing: sweep tasks name expected files/symbols from the endpoint inventory; the implementer must `grep`/read the actual file and adapt names to what is actually there — never invent a symbol. If an inventory claim doesn't match reality, stop and report the discrepancy in the task output rather than guessing.
- Commits: conventional style with service scope, e.g. `feat(atlas-character): paginate GET /characters (task-117)`.

---

## Phase L — Library layer

### Task 1: `model.Page` / `model.Paged` / `MapPaged` (libs/atlas-model)

**Files:**
- Create: `libs/atlas-model/model/paged.go`
- Create: `libs/atlas-model/model/paged_test.go`

**Interfaces:**
- Consumes: existing `model.Provider`, `model.Transformer`, `model.SliceMap`, `model.FixedProvider`, `model.MapFuncConfigurator`, `model.ParallelMap` (`libs/atlas-model/model/processor.go`).
- Produces (relied on by every later task):
  - `type Page struct { Number int; Size int }` — `Number` is 1-based.
  - `type Paged[T any] struct { Items []T; Total int; Page Page }` — `Total` is the pre-paging count of rows matching the scope.
  - `func MapPaged[E any, M any](transformer Transformer[E, M]) func(provider Provider[Paged[E]]) func(configurators ...MapFuncConfigurator) Provider[Paged[M]]`

- [ ] **Step 1: Write the failing tests**

```go
// libs/atlas-model/model/paged_test.go
package model

import (
	"errors"
	"strconv"
	"testing"
)

func TestMapPagedPreservesEnvelope(t *testing.T) {
	src := FixedProvider(Paged[int]{Items: []int{1, 2, 3}, Total: 42, Page: Page{Number: 2, Size: 3}})
	out, err := MapPaged[int, string](func(i int) (string, error) { return strconv.Itoa(i), nil })(src)()()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 42 || out.Page.Number != 2 || out.Page.Size != 3 {
		t.Fatalf("envelope not preserved: %+v", out)
	}
	if len(out.Items) != 3 || out.Items[0] != "1" || out.Items[2] != "3" {
		t.Fatalf("items wrong: %v", out.Items)
	}
}

func TestMapPagedParallelIndexStable(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = i
	}
	src := FixedProvider(Paged[int]{Items: items, Total: 200, Page: Page{Number: 1, Size: 200}})
	out, err := MapPaged[int, int](func(i int) (int, error) { return i * 2, nil })(src)(ParallelMap())()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range out.Items {
		if v != i*2 {
			t.Fatalf("index %d: got %d want %d", i, v, i*2)
		}
	}
}

func TestMapPagedSourceError(t *testing.T) {
	want := errors.New("boom")
	src := ErrorProvider[Paged[int]](want)
	_, err := MapPaged[int, int](func(i int) (int, error) { return i, nil })(src)()()
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}

func TestMapPagedTransformError(t *testing.T) {
	want := errors.New("boom")
	src := FixedProvider(Paged[int]{Items: []int{1}, Total: 1, Page: Page{Number: 1, Size: 1}})
	_, err := MapPaged[int, int](func(int) (int, error) { return 0, want })(src)()()
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-model && go test ./model/ -run TestMapPaged -v`
Expected: compile FAIL — `undefined: Paged`, `undefined: MapPaged`, `undefined: Page`.

- [ ] **Step 3: Implement `paged.go`**

```go
// libs/atlas-model/model/paged.go
package model

// Page identifies one page of a collection. Number is 1-based.
type Page struct {
	Number int
	Size   int
}

// Paged carries one page of items together with the pre-paging total of
// rows matching the scope and the page that produced Items.
type Paged[T any] struct {
	Items []T
	Total int
	Page  Page
}

// MapPaged lifts an item transform over the Paged container, preserving
// Total/Page. Composes exactly like SliceMap:
//
//	MapPaged(f)(provider)(ParallelMap())
//
// Decoration needs no separate primitive:
//
//	MapPaged(Decorate[M](decorators))(p)(ParallelMap())
func MapPaged[E any, M any](transformer Transformer[E, M]) func(provider Provider[Paged[E]]) func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
	return func(provider Provider[Paged[E]]) func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
		return func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
			return func() (Paged[M], error) {
				pe, err := provider()
				if err != nil {
					return Paged[M]{}, err
				}
				items, err := SliceMap[E, M](transformer)(FixedProvider(pe.Items))(configurators...)()
				if err != nil {
					return Paged[M]{}, err
				}
				return Paged[M]{Items: items, Total: pe.Total, Page: pe.Page}, nil
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-model && go test -race ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-model/model/paged.go libs/atlas-model/model/paged_test.go
git commit -m "feat(atlas-model): Page/Paged container and MapPaged combinator (task-117)"
```

---

### Task 2: `database.PagedQuery` (libs/atlas-database)

**Files:**
- Create: `libs/atlas-database/paged.go`
- Create: `libs/atlas-database/paged_test.go` (external test package `database_test` — `databasetest` imports `database`, so an internal test package would cycle)

**Interfaces:**
- Consumes: Task 1 (`model.Page`, `model.Paged`); `databasetest.NewInMemoryTenantDB`, `databasetest.TenantContext` (`libs/atlas-database/databasetest/testdb.go`); GORM `Statement.Parse` / `Schema.PrioritizedPrimaryField`.
- Produces: `func PagedQuery[E any](db *gorm.DB, page model.Page) model.Provider[model.Paged[E]]` — count + page fetch on the same scoped `*gorm.DB`; appends schema-derived PK ordering after any caller-supplied `Order`; errors on `page.Number < 1 || page.Size < 1` or an entity with no primary key.

- [ ] **Step 1: Write the failing tests**

Test entity + migration live inside the test file (no shared helper file):

```go
// libs/atlas-database/paged_test.go
package database_test

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type pagedEntity struct {
	Id        uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId  uuid.UUID `gorm:"not null"`
	Grp       string    // deliberately non-unique to exercise the PK tie-break
	CreatedAt time.Time
}

func (pagedEntity) TableName() string { return "paged_entities" }

func migrate(db *gorm.DB) error { return db.AutoMigrate(&pagedEntity{}) }

func seed(t *testing.T, db *gorm.DB, tenantId uuid.UUID, n int, grp string) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := db.Create(&pagedEntity{TenantId: tenantId, Grp: grp, CreatedAt: time.Unix(int64(1000+i), 0)}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestPagedQueryTenantScopeAgreement(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	t1, t2 := uuid.New(), uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(t1)), t1, 7, "a")
	seed(t, db.WithContext(databasetest.TenantContext(t2)), t2, 5, "a")

	scoped := db.WithContext(databasetest.TenantContext(t1))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 1, Size: 3})()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 7 {
		t.Fatalf("count leaked across tenants: total=%d want 7", p.Total)
	}
	if len(p.Items) != 3 {
		t.Fatalf("items=%d want 3", len(p.Items))
	}
	for _, e := range p.Items {
		if e.TenantId != t1 {
			t.Fatalf("row from wrong tenant: %+v", e)
		}
	}
}

func TestPagedQueryPagesArePartition(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	// all rows share Grp so any ORDER BY grp alone is non-total; the PK
	// tie-break must make pages disjoint and exhaustive.
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 10, "same")

	seen := map[uint32]bool{}
	for n := 1; n <= 4; n++ {
		scoped := db.WithContext(databasetest.TenantContext(tid)).Order("grp")
		p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: n, Size: 3})()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range p.Items {
			if seen[e.Id] {
				t.Fatalf("row %d appeared on two pages", e.Id)
			}
			seen[e.Id] = true
		}
	}
	if len(seen) != 10 {
		t.Fatalf("pages missed rows: saw %d want 10", len(seen))
	}
}

func TestPagedQueryCallerOrderPreservedAndCountUnaffected(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 6, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid)).Order("created_at desc")
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 1, Size: 6})()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 6 {
		t.Fatalf("count with caller ORDER BY: total=%d want 6", p.Total)
	}
	for i := 1; i < len(p.Items); i++ {
		if p.Items[i].CreatedAt.After(p.Items[i-1].CreatedAt) {
			t.Fatalf("caller order not preserved at %d", i)
		}
	}
}

func TestPagedQueryOffsetLimit(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 9, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 2, Size: 3})()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 3 {
		t.Fatalf("items=%d want 3", len(p.Items))
	}
	// PK order: page 2 of size 3 = ids 4,5,6
	if p.Items[0].Id != 4 || p.Items[2].Id != 6 {
		t.Fatalf("wrong window: %+v", p.Items)
	}
}

func TestPagedQueryPastEndEmpty(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 2, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 5, Size: 50})()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 0 || p.Total != 2 {
		t.Fatalf("past-end: items=%d total=%d", len(p.Items), p.Total)
	}
}

func TestPagedQueryRejectsInvalidPage(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	if _, err := database.PagedQuery[pagedEntity](db, model.Page{Number: 0, Size: 10})(); err == nil {
		t.Fatal("expected error for page.Number=0")
	}
	if _, err := database.PagedQuery[pagedEntity](db, model.Page{Number: 1, Size: 0})(); err == nil {
		t.Fatal("expected error for page.Size=0")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-database && go test ./ -run TestPagedQuery -v`
Expected: compile FAIL — `undefined: database.PagedQuery`.

- [ ] **Step 3: Implement `paged.go`**

```go
// libs/atlas-database/paged.go
package database

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PagedQuery runs a COUNT plus an OFFSET/LIMIT Find against the same scoped
// *gorm.DB, so the tenant-filter callback and all Where clauses apply
// identically to both. A schema-derived primary-key ordering is appended
// after any caller-supplied ordering so pages form a total order.
// page.Number is 1-based. Lazy: nothing executes until the provider is invoked.
func PagedQuery[E any](db *gorm.DB, page model.Page) model.Provider[model.Paged[E]] {
	return func() (model.Paged[E], error) {
		if page.Number < 1 || page.Size < 1 {
			return model.Paged[E]{}, fmt.Errorf("invalid page number=%d size=%d", page.Number, page.Size)
		}

		var e E
		// Count on a session clone with ORDER BY stripped explicitly —
		// GORM's own order-stripping inside Count is an implementation
		// detail we do not rely on (design §3.2).
		countDB := db.Session(&gorm.Session{}).Model(&e)
		delete(countDB.Statement.Clauses, "ORDER BY")
		var total int64
		if err := countDB.Count(&total).Error; err != nil {
			return model.Paged[E]{}, err
		}

		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(&e); err != nil {
			return model.Paged[E]{}, err
		}
		pk := stmt.Schema.PrioritizedPrimaryField
		if pk == nil {
			return model.Paged[E]{}, fmt.Errorf("entity for table %s has no primary key; stable paging requires one", stmt.Schema.Table)
		}

		var results []E
		err := db.Session(&gorm.Session{}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: pk.DBName}}).
			Offset((page.Number - 1) * page.Size).
			Limit(page.Size).
			Find(&results).Error
		if err != nil {
			return model.Paged[E]{}, err
		}
		return model.Paged[E]{Items: results, Total: int(total), Page: page}, nil
	}
}
```

Note for the implementer: if `delete(countDB.Statement.Clauses, "ORDER BY")` turns out to mutate the original `db`'s statement under the installed GORM version (the caller-order test will catch it — page order would be lost), switch the clone to `db.Session(&gorm.Session{NewDB: false}).Model(&e)` with an explicitly copied clause map. The tests are the contract; adjust the clone mechanics until all six pass.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-database && go test -race ./... && go vet ./...`
Expected: PASS (all six new tests + existing tenant_scope tests), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-database/paged.go libs/atlas-database/paged_test.go
git commit -m "feat(atlas-database): PagedQuery with tenant-scoped count and PK tie-break (task-117)"
```

---

### Task 3: `paginate.ParseParams` / `paginate.Slice` / `EnvelopeFor` + `server.WriteBadRequest` (libs/atlas-rest)

**Files:**
- Create: `libs/atlas-rest/server/paginate/params.go`
- Create: `libs/atlas-rest/server/paginate/params_test.go`
- Create: `libs/atlas-rest/server/paginate/slice.go`
- Create: `libs/atlas-rest/server/paginate/slice_test.go`
- Create: `libs/atlas-rest/server/error.go`
- Modify: `libs/atlas-rest/server/paginate/envelope.go` (append `EnvelopeFor`)

**Interfaces:**
- Consumes: Task 1 (`model.Page`, `model.Paged`); existing `paginate.Envelope`.
- Produces (used by every server-side conversion task):
  - `paginate.ErrInvalidPageParam` (sentinel error)
  - `const paginate.DefaultPageSize = 50`, `const paginate.MaxPageSize = 250`
  - `func ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error)`
  - `func Slice[T any](items []T, page model.Page) model.Paged[T]`
  - `func EnvelopeFor[T any](p model.Paged[T]) Envelope`
  - `func server.WriteBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string)` — writes a JSON:API error object with status 400.

- [ ] **Step 1: Write the failing tests**

```go
// libs/atlas-rest/server/paginate/params_test.go
package paginate

import (
	"errors"
	"net/url"
	"testing"
)

func q(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

func TestParseParams(t *testing.T) {
	cases := []struct {
		name       string
		query      url.Values
		wantNumber int
		wantSize   int
		wantErr    bool
	}{
		{"defaults", q(), 1, 50, false},
		{"explicit", q("page[number]", "3", "page[size]", "25"), 3, 25, false},
		{"size at max", q("page[size]", "250"), 1, 250, false},
		{"size over max", q("page[size]", "251"), 0, 0, true},
		{"size zero", q("page[size]", "0"), 0, 0, true},
		{"size negative", q("page[size]", "-5"), 0, 0, true},
		{"size non-integer", q("page[size]", "abc"), 0, 0, true},
		{"number zero", q("page[number]", "0"), 0, 0, true},
		{"number non-integer", q("page[number]", "x"), 0, 0, true},
		{"legacy limit rejected", q("limit", "10"), 0, 0, true},
		{"other params ignored", q("include", "skills", "page[number]", "2"), 2, 50, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseParams(c.query, 50, 250)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidPageParam) {
					t.Fatalf("want ErrInvalidPageParam, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Number != c.wantNumber || p.Size != c.wantSize {
				t.Fatalf("got %+v", p)
			}
		})
	}
}
```

```go
// libs/atlas-rest/server/paginate/slice_test.go
package paginate

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7}
	p1 := Slice(items, model.Page{Number: 1, Size: 3})
	if p1.Total != 7 || len(p1.Items) != 3 || p1.Items[0] != 1 {
		t.Fatalf("page1: %+v", p1)
	}
	p3 := Slice(items, model.Page{Number: 3, Size: 3})
	if len(p3.Items) != 1 || p3.Items[0] != 7 {
		t.Fatalf("partial last page: %+v", p3)
	}
	past := Slice(items, model.Page{Number: 9, Size: 3})
	if len(past.Items) != 0 || past.Total != 7 {
		t.Fatalf("past-end: %+v", past)
	}
	empty := Slice([]int{}, model.Page{Number: 1, Size: 3})
	if len(empty.Items) != 0 || empty.Total != 0 {
		t.Fatalf("empty: %+v", empty)
	}
}

func TestEnvelopeFor(t *testing.T) {
	env := EnvelopeFor(model.Paged[int]{Items: []int{1}, Total: 9, Page: model.Page{Number: 2, Size: 4}})
	if env.Total != 9 || env.PageNumber != 2 || env.PageSize != 4 {
		t.Fatalf("%+v", env)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-rest && go test ./server/paginate/ -v`
Expected: compile FAIL — `undefined: ParseParams`, `undefined: Slice`, `undefined: EnvelopeFor`.

- [ ] **Step 3: Implement**

```go
// libs/atlas-rest/server/paginate/params.go
package paginate

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ErrInvalidPageParam is returned for non-integer, out-of-range, or legacy
// paging parameters. Handlers map it to HTTP 400.
var ErrInvalidPageParam = errors.New("invalid page parameter")

// Repo-wide defaults (docs/rest-pagination.md). Group C game-capped lists
// pass MaxPageSize as their default so the common case fits one page.
const (
	DefaultPageSize = 50
	MaxPageSize     = 250
)

// ParseParams parses JSON:API page[number]/page[size] query params.
// Defaults: number=1, size=defaultSize. Invalid values (non-integer,
// number<1, size<1, size>maxSize) are an error, not silently clamped.
// The legacy ?limit= param is rejected outright, enforcing that paging is
// expressed only via page[*] repo-wide.
func ParseParams(query url.Values, defaultSize, maxSize int) (model.Page, error) {
	size := defaultSize
	if raw := query.Get("page[size]"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxSize {
			return model.Page{}, ErrInvalidPageParam
		}
		size = parsed
	}
	number := 1
	if raw := query.Get("page[number]"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return model.Page{}, ErrInvalidPageParam
		}
		number = parsed
	}
	if _, hasLimit := query["limit"]; hasLimit {
		return model.Page{}, ErrInvalidPageParam
	}
	return model.Page{Number: number, Size: size}, nil
}
```

```go
// libs/atlas-rest/server/paginate/slice.go
package paginate

import "github.com/Chronicle20/atlas/libs/atlas-model/model"

// Slice pages an already-materialized collection (runtime registries,
// document sub-lists). items MUST already be deterministically ordered by
// the caller. Past-end pages return empty Items with the correct Total —
// the Envelope's recovery links handle the UX.
func Slice[T any](items []T, page model.Page) model.Paged[T] {
	total := len(items)
	start := (page.Number - 1) * page.Size
	if start >= total {
		return model.Paged[T]{Items: []T{}, Total: total, Page: page}
	}
	end := start + page.Size
	if end > total {
		end = total
	}
	return model.Paged[T]{Items: items[start:end], Total: total, Page: page}
}
```

Append to `libs/atlas-rest/server/paginate/envelope.go`:

```go
// EnvelopeFor builds the response envelope for one fetched page.
func EnvelopeFor[T any](p model.Paged[T]) Envelope {
	return Envelope{Total: p.Total, PageNumber: p.Page.Number, PageSize: p.Page.Size}
}
```

(`envelope.go` gains the import `"github.com/Chronicle20/atlas/libs/atlas-model/model"`.)

```go
// libs/atlas-rest/server/error.go
package server

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// WriteBadRequest writes a JSON:API error object with HTTP 400.
func WriteBadRequest(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	body := fmt.Sprintf(`{"errors":[{"status":"400","title":"Bad Request","detail":%q}]}`, detail)
	if _, err := w.Write([]byte(body)); err != nil {
		l.WithError(err).Errorf("Unable to write error response.")
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-rest && go test -race ./... && go vet ./...`
Expected: PASS (new params/slice tests + existing envelope tests), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-rest/server/paginate/params.go libs/atlas-rest/server/paginate/params_test.go libs/atlas-rest/server/paginate/slice.go libs/atlas-rest/server/paginate/slice_test.go libs/atlas-rest/server/paginate/envelope.go libs/atlas-rest/server/error.go
git commit -m "feat(atlas-rest): hoisted ParseParams, Slice adapter, EnvelopeFor, WriteBadRequest (task-117)"
```

---

### Task 4: `requests.PagedGetRequest` / `PagedProvider` / `DrainProvider` (libs/atlas-rest)

**Files:**
- Modify: `libs/atlas-rest/requests/get.go` (extract `getBody` from `get`; behavior of `get` unchanged)
- Create: `libs/atlas-rest/requests/paged.go`
- Create: `libs/atlas-rest/requests/paged_test.go`

**Interfaces:**
- Consumes: Task 1 (`model.Page`/`Paged`); existing `Request[A]`, `Configurator`, `unmarshalResponse`, `ErrBadRequest`/`ErrNotFound`, retry plumbing in `get.go`.
- Produces (used by every consumer-conversion task):
  - `type PageMetaPage struct { Number, Size, Last int }` (json tags `number`/`size`/`last`)
  - `type PageMeta struct { Total int; Page PageMetaPage }` (json tags `total`/`page`)
  - `type PagedResponse[A any] struct { Data []A; Meta *PageMeta }` — `Meta == nil` ⇔ no pagination envelope (unconverted server).
  - `func PagedGetRequest[A any](rawUrl string, page model.Page, configurators ...Configurator) Request[PagedResponse[A]]`
  - `func PagedProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, page model.Page, t model.Transformer[A, M], configurators ...Configurator) model.Provider[model.Paged[M]]`
  - `func DrainProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M], configurators ...Configurator) model.Provider[[]M]`

- [ ] **Step 1: Refactor `get.go` — extract the body fetch**

Split the existing private `get[A]` into `getBody` (everything through the status-code mapping, returning `([]byte, error)`) and a thin `get[A]` that calls `getBody` then `unmarshalResponse[A]`. The existing debug log of the decoded response stays in `get[A]`. No exported-surface change.

```go
func getBody(l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) ([]byte, error) {
	return func(url string, configurators ...Configurator) ([]byte, error) {
		// ... body of the current get[A] verbatim, up to and including the
		// retry.Try call and status-code mapping ...
		// on 200/202: return body, nil
		// on 400: return nil, ErrBadRequest
		// on 404: return nil, ErrNotFound
		// otherwise: return nil, errors.New("unknown error")
	}
}

func get[A any](l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) (A, error) {
	return func(url string, configurators ...Configurator) (A, error) {
		var resp A
		body, err := getBody(l, ctx)(url, configurators...)
		if err != nil {
			return resp, err
		}
		resp, err = unmarshalResponse[A](body)
		l.WithFields(logrus.Fields{"method": http.MethodGet, "path": url, "response": resp}).Debugf("Printing request.")
		return resp, err
	}
}
```

Run: `cd libs/atlas-rest && go test -race ./...`
Expected: PASS — pure refactor, existing client tests green.

- [ ] **Step 2: Write the failing paged-client tests**

httptest-backed per `libs/atlas-rest/CLAUDE.md`. The fixture type implements the full jsonapi interface set **including relationship stubs**, and the multi-page fixture carries a `relationships` block to pin the UnmarshalToManyRelations gotcha.

```go
// libs/atlas-rest/requests/paged_test.go
package requests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
)

type pagedFixture struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (f pagedFixture) GetName() string { return "fixtures" }
func (f pagedFixture) GetID() string   { return strconv.Itoa(int(f.Id)) }
func (f *pagedFixture) SetID(id string) error {
	v, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	f.Id = uint32(v)
	return nil
}
func (f *pagedFixture) SetToOneReferenceID(_, _ string) error            { return nil }
func (f *pagedFixture) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func extractFixture(f pagedFixture) (uint32, error) { return f.Id, nil }

// pageDoc renders a JSON:API page. ids come from [from, to). last/total per args.
// Each resource carries a relationships block to pin the api2go stub requirement.
func pageDoc(from, to, total, number, size, last int) string {
	data := ""
	for i := from; i < to; i++ {
		if data != "" {
			data += ","
		}
		data += fmt.Sprintf(`{"id":"%d","type":"fixtures","attributes":{"name":"n%d"},"relationships":{"tags":{"data":[{"id":"1","type":"tags"}]}}}`, i, i)
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`, data, total, number, size, last)
}

func servePages(t *testing.T, totalItems, pageSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		number, _ := strconv.Atoi(q.Get("page[number]"))
		size, _ := strconv.Atoi(q.Get("page[size]"))
		if number < 1 || size < 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		last := (totalItems + size - 1) / size
		if last < 1 {
			last = 1
		}
		from := (number-1)*size + 1
		to := from + size
		if from > totalItems {
			from, to = 1, 1 // empty page
		} else if to > totalItems+1 {
			to = totalItems + 1
		}
		_, _ = w.Write([]byte(pageDoc(from, to, totalItems, number, size, last)))
	}))
}

func TestPagedGetRequestDecodesItemsAndMeta(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	resp, err := PagedGetRequest[pagedFixture](srv.URL+"/fixtures", model.Page{Number: 2, Size: 2})(l, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 || resp.Data[0].Id != 3 {
		t.Fatalf("data: %+v", resp.Data)
	}
	if resp.Meta == nil || resp.Meta.Total != 5 || resp.Meta.Page.Last != 3 {
		t.Fatalf("meta: %+v", resp.Meta)
	}
}

func TestPagedGetRequestPreservesExistingQuery(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter[name]")
		_, _ = w.Write([]byte(pageDoc(1, 1, 0, 1, 50, 1)))
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	_, err := PagedGetRequest[pagedFixture](srv.URL+"/fixtures?filter[name]=bob", model.Page{Number: 1, Size: 50})(l, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotFilter != "bob" {
		t.Fatalf("existing query param lost: %q", gotFilter)
	}
}

func TestDrainProviderMultiPage(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 2, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 5 || ms[0] != 1 || ms[4] != 5 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderSinglePage(t *testing.T) {
	srv := servePages(t, 3, 250)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 250, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 3 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderEmptyCollection(t *testing.T) {
	srv := servePages(t, 0, 50)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 50, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 0 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderNoEnvelopeCompat(t *testing.T) {
	// Unconverted server: plain document, no meta. The single response IS
	// the complete collection.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"fixtures","attributes":{"name":"a"},"relationships":{"tags":{"data":[]}}},{"id":"2","type":"fixtures","attributes":{"name":"b"},"relationships":{"tags":{"data":[]}}}]}`))
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 50, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("compat drain: %v", ms)
	}
}

func TestDrainProviderWarnsPast20Pages(t *testing.T) {
	srv := servePages(t, 45, 2) // 23 pages
	defer srv.Close()
	l, hook := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 2, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 45 {
		t.Fatalf("drained %d", len(ms))
	}
	warned := false
	for _, e := range hook.AllEntries() {
		if e.Level.String() == "warning" {
			warned = true
		}
	}
	if !warned {
		t.Fatal("expected a warning for a >20-page drain")
	}
}

func TestPagedProviderErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	_, err := PagedProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", model.Page{Number: 1, Size: 50}, extractFixture)()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPagedProviderReturnsPagedModel(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	p, err := PagedProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", model.Page{Number: 1, Size: 2}, extractFixture)()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 5 || p.Page.Number != 1 || len(p.Items) != 2 {
		t.Fatalf("%+v", p)
	}
}
```

Run: `cd libs/atlas-rest && go test ./requests/ -run 'TestPaged|TestDrain' -v`
Expected: compile FAIL — `undefined: PagedGetRequest` etc.

- [ ] **Step 3: Implement `paged.go`**

```go
// libs/atlas-rest/requests/paged.go
package requests

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// drainWarnPages is the page count past which a single drain logs a warning.
const drainWarnPages = 20

type PageMetaPage struct {
	Number int `json:"number"`
	Size   int `json:"size"`
	Last   int `json:"last"`
}

type PageMeta struct {
	Total int          `json:"total"`
	Page  PageMetaPage `json:"page"`
}

// PagedResponse carries one decoded page. Meta == nil means the response
// had no pagination envelope (an unconverted server): the caller must treat
// Data as the complete collection.
type PagedResponse[A any] struct {
	Data []A
	Meta *PageMeta
}

func withPageParams(rawUrl string, page model.Page) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("page[number]", strconv.Itoa(page.Number))
	q.Set("page[size]", strconv.Itoa(page.Size))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// PagedGetRequest issues a GET with page[number]/page[size] appended
// (existing query params preserved) and decodes both the JSON:API data
// array and the pagination envelope from the same body.
func PagedGetRequest[A any](rawUrl string, page model.Page, configurators ...Configurator) Request[PagedResponse[A]] {
	return func(l logrus.FieldLogger, ctx context.Context) (PagedResponse[A], error) {
		u, err := withPageParams(rawUrl, page)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		body, err := getBody(l, ctx)(u, configurators...)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		var env struct {
			Meta *PageMeta `json:"meta"`
		}
		if err = json.Unmarshal(body, &env); err != nil {
			return PagedResponse[A]{}, err
		}
		items, err := unmarshalResponse[[]A](body)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		return PagedResponse[A]{Data: items, Meta: env.Meta}, nil
	}
}

// PagedProvider fetches one page and transforms it, returning the paged
// container. If the server sent no envelope, Total falls back to the item
// count and Page to the requested page.
func PagedProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, page model.Page, t model.Transformer[A, M], configurators ...Configurator) model.Provider[model.Paged[M]] {
	return func(url string, page model.Page, t model.Transformer[A, M], configurators ...Configurator) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			resp, err := PagedGetRequest[A](url, page, configurators...)(l, ctx)
			if err != nil {
				return model.Paged[M]{}, err
			}
			ms, err := model.SliceMap[A, M](t)(model.FixedProvider(resp.Data))(model.ParallelMap())()
			if err != nil {
				return model.Paged[M]{}, err
			}
			total, pg := len(ms), page
			if resp.Meta != nil {
				total = resp.Meta.Total
				pg = model.Page{Number: resp.Meta.Page.Number, Size: resp.Meta.Page.Size}
			}
			return model.Paged[M]{Items: ms, Total: total, Page: pg}, nil
		}
	}
}

// DrainProvider is the semantic-"all" fetch: it requests page 1 at pageSize
// and iterates page[number] 2..meta.page.last (re-read each response),
// stopping early on an empty page. If a response carries no envelope, that
// single response is treated as the complete collection — this makes
// consumer-first rollout against unconverted servers safe. The (t, filters)
// tail matches SliceProvider so call-site conversion is mechanical.
func DrainProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M], configurators ...Configurator) model.Provider[[]M] {
	return func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M], configurators ...Configurator) model.Provider[[]M] {
		return func() ([]M, error) {
			var out []M
			last := 1
			for number := 1; number <= last; number++ {
				resp, err := PagedGetRequest[A](url, model.Page{Number: number, Size: pageSize}, configurators...)(l, ctx)
				if err != nil {
					return nil, err
				}
				ms, err := model.SliceMap[A, M](t)(model.FixedProvider(resp.Data))(model.ParallelMap())()
				if err != nil {
					return nil, err
				}
				out = append(out, ms...)
				if resp.Meta == nil {
					break
				}
				if len(resp.Data) == 0 {
					break
				}
				last = resp.Meta.Page.Last
				if number == drainWarnPages && last > drainWarnPages {
					l.Warnf("Drain of [%s] exceeds [%d] pages (total [%d]); consider whether this consumer really needs the full collection.", url, drainWarnPages, resp.Meta.Total)
				}
			}
			return model.FilteredProvider(model.FixedProvider(out), filters)()
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-rest && go test -race ./... && go vet ./...`
Expected: PASS (all new paged tests + existing suite), vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-rest/requests/get.go libs/atlas-rest/requests/paged.go libs/atlas-rest/requests/paged_test.go
git commit -m "feat(atlas-rest): PagedGetRequest, PagedProvider, DrainProvider with no-envelope compat (task-117)"
```

---

### Task 5: Refactor atlas-data item-string search onto `paginate.ParseParams`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/item/string_resource.go` (delete private `parsePagingParams` at ~line 161; call site earlier in the handler)
- Modify: its tests (grep `parsePagingParams` under `services/atlas-data/atlas.com/data/item/` — a `string_resource_test.go` exercises the 400 cases; update to exercise the route-level behavior, which must not change)

**Interfaces:**
- Consumes: Task 3 `paginate.ParseParams`; existing `searchindex.MaxLimit` (= 50).
- Produces: no new surface. Wire contract of `GET /data/items/strings` unchanged (defaults `page[size]=searchindex.MaxLimit`, 400 on invalid/legacy `limit`).

- [ ] **Step 1: Locate the call site and existing tests**

Run: `grep -rn "parsePagingParams" services/atlas-data/atlas.com/data/`
Expected: the definition (`string_resource.go:161` area) plus its handler call site and any test references. Read them.

- [ ] **Step 2: Replace the call site**

In the handler, replace:

```go
pageNumber, pageSize, errCode := parsePagingParams(query)
if errCode != 0 {
    w.WriteHeader(errCode)
    return
}
```

(adapt to the actual code found in Step 1) with:

```go
page, err := paginate.ParseParams(query, searchindex.MaxLimit, searchindex.MaxLimit)
if err != nil {
    server.WriteBadRequest(d.Logger(), w, err.Error())
    return
}
pageNumber, pageSize := page.Number, page.Size
```

Delete `parsePagingParams` entirely. Add the `paginate` import (`github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate`).

- [ ] **Step 3: Update tests and run**

Any test that called `parsePagingParams` directly moves to route-level assertions (existing httptest harness in that package) covering: defaults, explicit params, `page[size]>50` → 400, non-integer → 400, `?limit=` → 400.

Run: `cd services/atlas-data/atlas.com/data && go test -race ./item/... && go vet ./... && go build ./...`
Expected: PASS.

- [ ] **Step 4: Bake and commit**

Run: `docker buildx bake atlas-data` (from worktree root) — expected: success.

```bash
git add services/atlas-data/atlas.com/data/item/
git commit -m "refactor(atlas-data): item-string search uses hoisted paginate.ParseParams (task-117)"
```

---

### Task 6: atlas-data document store — `AllPagedProvider` + drain

**Files:**
- Modify: `services/atlas-data/atlas.com/data/document/db_storage.go` (add `AllPaged`)
- Modify: `services/atlas-data/atlas.com/data/document/storage.go` (add `AllPagedProvider`, `DrainAllProvider`)
- Create/Modify: tests alongside (the package has existing tests — check `ls services/atlas-data/atlas.com/data/document/*_test.go` and extend in the same style; use the existing test DB setup found there)

**Interfaces:**
- Consumes: Tasks 1–3; existing `DbStorage.All` scoping (`type = ?` + tenant callback), `canonical.TenantId` fallback logic in `Storage.AllProvider` (`storage.go:71-101`), `database.PagedQuery`, `paginate` is NOT needed here (SQL paging).
- Produces:
  - `func (s *DbStorage[I, M]) AllPaged(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]]` — SQL-paged over the `documents` table, `ORDER BY document_id` (+ PK tie-break from `PagedQuery`), same `type = ?` + tenant scope as `All`.
  - `func (s *Storage[I, M]) AllPagedProvider(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]]` — tenant-scoped paged fetch; **if the tenant-scoped `Total` is 0, falls back to the version-scoped canonical tenant and pages that** (regression guard for the batch-GetAll-skips-fallback bug class).
  - `func (s *Storage[I, M]) DrainAllProvider(ctx context.Context) model.Provider[[]M]` — in-process page loop (page size 1000) over `AllPagedProvider`, for internal callers that genuinely need every document. `Storage.GetAll`/`Storage.AllProvider` are **deleted later** (Task 15, after all callers are converted).

- [ ] **Step 1: Write the failing tests**

In the document package's existing test style (discover the harness first). Required cases:

```go
func TestAllPagedProviderPagesTenantDocs(t *testing.T)      // seed 7 docs one tenant; page 2 size 3 → 3 docs, Total 7, ordered by document_id
func TestAllPagedProviderCanonicalFallbackOnEmpty(t *testing.T) // tenant has 0 docs, canonical tenant (canonical.TenantId(region, major, minor)) has 4 → page 1 returns canonical docs, Total 4
func TestAllPagedProviderNoFallbackWhenTenantHasDocs(t *testing.T) // tenant has 2, canonical has 4 → Total 2
func TestDrainAllProviderReturnsAll(t *testing.T)           // seed 12 docs; drain returns all 12
```

Run: `cd services/atlas-data/atlas.com/data && go test ./document/ -run 'AllPaged|DrainAll' -v`
Expected: compile FAIL.

- [ ] **Step 2: Implement**

```go
// db_storage.go
func (s *DbStorage[I, M]) AllPaged(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]] {
	return func(page model.Page) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			scoped := s.db.WithContext(ctx).Where("type = ?", s.docType).Order("document_id")
			pe, err := database.PagedQuery[Entity](scoped, page)()
			if err != nil {
				return model.Paged[M]{}, err
			}
			ms := make([]M, 0, len(pe.Items))
			for _, doc := range pe.Items {
				var rm M
				if err := jsonapi.Unmarshal(doc.Content, &rm); err != nil {
					return model.Paged[M]{}, err
				}
				ms = append(ms, rm)
			}
			return model.Paged[M]{Items: ms, Total: pe.Total, Page: pe.Page}, nil
		}
	}
}
```

```go
// storage.go
// AllPagedProvider pages this document type for the context tenant. A tenant
// with no rows falls back to the version-scoped canonical dataset, mirroring
// AllProvider — the paged variant must not reintroduce the batch-skips-
// fallback asymmetry (see AllProvider's comment).
func (s *Storage[I, M]) AllPagedProvider(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]] {
	t := tenant.MustFromContext(ctx)
	return func(page model.Page) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			p, err := s.dbSto.AllPaged(ctx)(page)()
			if err != nil {
				return model.Paged[M]{}, err
			}
			if p.Total > 0 {
				return p, nil
			}
			nt, cerr := tenant.Create(canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()), t.Region(), t.MajorVersion(), t.MinorVersion())
			if cerr != nil {
				return model.Paged[M]{}, cerr
			}
			nctx := tenant.WithContext(ctx, nt)
			cp, cerr := s.dbSto.AllPaged(nctx)(page)()
			if cerr != nil {
				return model.Paged[M]{}, cerr
			}
			if cp.Total > 0 {
				return cp, nil
			}
			return p, nil
		}
	}
}

// DrainAllProvider accumulates every document of this type (tenant scope
// with canonical fallback) by paging internally. For in-process callers
// that genuinely need the full set (e.g. search-index builds).
func (s *Storage[I, M]) DrainAllProvider(ctx context.Context) model.Provider[[]M] {
	return func() ([]M, error) {
		const drainPageSize = 1000
		var out []M
		for number := 1; ; number++ {
			p, err := s.AllPagedProvider(ctx)(model.Page{Number: number, Size: drainPageSize})()
			if err != nil {
				return nil, err
			}
			out = append(out, p.Items...)
			if len(p.Items) == 0 || len(out) >= p.Total {
				return out, nil
			}
		}
	}
}
```

(Add imports: `database "github.com/Chronicle20/atlas/libs/atlas-database"` in db_storage.go if absent.)

- [ ] **Step 3: Run tests, gauntlet, commit**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...` then `docker buildx bake atlas-data`.
Expected: PASS.

```bash
git add services/atlas-data/atlas.com/data/document/
git commit -m "feat(atlas-data): paged document storage with canonical fallback and drain (task-117)"
```

---

## Phase A — Group A servers, their consumers, core UI

**Ordering rule:** Tasks 7–8 (consumer drains) land before Task 9 (accounts server) — the compat rule makes the drains correct against the unconverted server, and nothing ever reads a truncated page.

### Task 7: atlas-login — account drain

**Files:**
- Modify: `services/atlas-login/atlas.com/login/account/processor.go` (`AllProvider`, `processor.go:61-63`)
- Modify: `services/atlas-login/atlas.com/login/account/requests.go` (drop `requestAccounts` if now unused)
- Create: `services/atlas-login/atlas.com/login/account/processor_drain_test.go`

**Interfaces:**
- Consumes: Task 4 `requests.DrainProvider`; existing `Extract`, `RestModel`, `getBaseRequest()`, `AccountsResource`, registry `GetRegistry().Init` / `LoggedIn`, `KeyForTenantFunc`, `IsLogged`.
- Produces: `account.Processor` interface **unchanged** (`AllProvider() model.Provider[[]Model]` still means "all accounts"); `InitializeRegistry()` semantics unchanged. Mock in `account/mock` untouched.

- [ ] **Step 1: Write the failing registry-seed test**

httptest server serving two envelope pages of accounts; point the client at it via `t.Setenv`. Read `services/atlas-login/atlas.com/login/account/rest.go` first for the account `RestModel` attribute names, and reuse an existing account fixture pattern if one exists in the package tests. Shape:

```go
func TestInitializeRegistrySeedsAcrossPages(t *testing.T) {
	// two pages: ids 1..250 (page 1), 251..300 (page 2); loggedIn=1 for id 1 and 300
	srv := httptest.NewServer(/* handler emitting JSON:API account docs with
	   meta {total:300, page:{number,size:250,last:2}} keyed off page[number] */)
	defer srv.Close()
	t.Setenv("ACCOUNTS_SERVICE_URL", srv.URL+"/")

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	if err := NewProcessor(l, ctx).InitializeRegistry(); err != nil {
		t.Fatal(err)
	}
	p := NewProcessor(l, ctx)
	if !p.IsLoggedIn(1) || !p.IsLoggedIn(300) {
		t.Fatal("accounts from both pages must land in the registry")
	}
	if p.IsLoggedIn(2) {
		t.Fatal("logged-out account marked logged in")
	}
}
```

The fixture generator must emit the real account RestModel attributes (copy from `rest.go`) and a fresh tenant uuid per test (the registry is a singleton).

Run: `cd services/atlas-login/atlas.com/login && go test ./account/ -run TestInitializeRegistrySeedsAcrossPages -v`
Expected: FAIL — current `AllProvider` uses plain `requestAccounts()`, which ignores `page[number]` and returns whatever the handler sends for page 1 only (make the fixture handler return page 1 content unless `page[number]=2` so the test genuinely distinguishes drain from single-fetch).

- [ ] **Step 2: Convert `AllProvider` to drain**

```go
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(getBaseRequest()+AccountsResource, 250, Extract, model.Filters[Model]())
}
```

Delete `requestAccounts()` from `requests.go` if nothing else references it (`grep -n requestAccounts services/atlas-login -r`).

- [ ] **Step 3: Run tests, gauntlet, commit**

Run: `cd services/atlas-login/atlas.com/login && go test -race ./... && go vet ./... && go build ./...` then `docker buildx bake atlas-login`.

```bash
git add services/atlas-login/atlas.com/login/account/
git commit -m "feat(atlas-login): drain accounts page-by-page for registry seed (task-117)"
```

### Task 8: atlas-channel — account drain

Identical shape to Task 7 against `services/atlas-channel/atlas.com/channel/account/` (`AllProvider` at `processor.go:42`, `GetAll` wrapper at `:51`, `InitializeRegistry` at `:58`, seed call at `main.go:386`).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/account/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/account/requests.go` (same dead-request cleanup)
- Create: `services/atlas-channel/atlas.com/channel/account/processor_drain_test.go`

**Interfaces:** as Task 7; note atlas-channel's interface also exposes `GetAll()` (`processor.go:51`) delegating to `AllProvider()()`. It is REST-backed and now drains (semantic-all is correct), but **rename it to `GetAllAccounts()`** (interface + impl + every caller the compiler surfaces + the `account/mock` package if it mirrors the interface) so the Task 29 acceptance grep for unfiltered-`GetAll` methods stays clean without an exception list.

- [ ] **Step 1: Write the failing two-page registry-seed test** (same fixture approach as Task 7, adapted to this package's `RestModel`)
- [ ] **Step 2: Convert `AllProvider` to `requests.DrainProvider[RestModel, Model](p.l, p.ctx)(getBaseRequest()+AccountsResource, 250, Extract, model.Filters[Model]())`** (verify the actual base-url helper/constant names in this package's `requests.go` first)
- [ ] **Step 3: Run `go test -race ./... && go vet ./... && go build ./...` in `services/atlas-channel/atlas.com/channel`, `docker buildx bake atlas-channel`, commit**

```bash
git commit -m "feat(atlas-channel): drain accounts page-by-page for registry seed (task-117)"
```

---

### Task 9: atlas-account — paginate `GET /accounts`

**Files:**
- Modify: `services/atlas-account/atlas.com/account/account/provider.go` (full-table `getAll` at `provider.go:35` → paged)
- Modify: `services/atlas-account/atlas.com/account/account/processor.go` (delete unfiltered `GetAll`-style method; add `AllProvider(page, decorators...)`)
- Modify: `services/atlas-account/atlas.com/account/account/resource.go` (list handler → §5 pattern)
- Create/Modify: resource/processor tests in the same package

**Interfaces:**
- Consumes: Tasks 1–3 (`model.Page`/`Paged`/`MapPaged`, `database.PagedQuery`, `paginate.ParseParams`/`EnvelopeFor`, `server.MarshalPaginatedResponse`, `server.WriteBadRequest`).
- Produces: `GET /accounts` returns the envelope, defaults 50/250, 400 on bad params. This is **the canonical Group A conversion** — later tasks follow exactly this recipe.

- [ ] **Step 1: Read the three files.** Identify: the entity provider (`getAll`), the processor list method and its interface entry, every internal caller of that method (`grep -rn "GetAll\|AllProvider" services/atlas-account/atlas.com/account/`), and the list handler + route registration. Any *internal* caller that needs all rows must be re-pointed at a filtered provider or an in-service drain loop — enumerate and handle each; do not leave one calling a paged method with page 1 only.

- [ ] **Step 2: Write the failing httptest resource test** (envelope shape + 400):

```go
func TestGetAccountsPaginates(t *testing.T) {
	// seed 3 accounts via the package's Builder-based test setup against a
	// databasetest in-memory DB; mount the resource router with httptest.
	// GET /accounts?page[number]=1&page[size]=2
	//   → 200; data len 2; meta.total 3; meta.page.last 2; links.next present
	// GET /accounts?page[size]=0     → 400
	// GET /accounts?limit=5          → 400
	// GET /accounts?page[number]=99  → 200; data empty; links.prev = last page
}
```

(Write it fully against the package's actual router-mounting pattern — look at how existing resource tests in this service or a sibling service mount `InitResource`; if none exists, invoke the handler funcs directly with `mux.SetURLVars` as other Atlas resource tests do.)

- [ ] **Step 3: Convert provider → processor → resource**

Provider (replace the full-table body):

```go
func getAll(page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db, page)
	}
}
```

Processor (delete the old unfiltered method from interface + impl; add):

```go
AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]]

func (p *ProcessorImpl) AllProvider(page model.Page, decorators ...model.Decorator[Model]) model.Provider[model.Paged[Model]] {
	ep := getAll(page)(p.db.WithContext(p.ctx))
	mp := model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
	return model.MapPaged(model.Decorate[Model](decorators))(mp)(model.ParallelMap())
}
```

Resource handler:

```go
page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
if err != nil {
	server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
	return
}
paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).AllProvider(page /* , decorators per this service's include handling */)()
if err != nil {
	d.Logger().WithError(err).Errorf("Unable to get accounts.")
	w.WriteHeader(http.StatusInternalServerError)
	return
}
res, err := model.SliceMap(Transform /* actual transformer signature */)(model.FixedProvider(paged.Items))(model.ParallelMap())()
if err != nil {
	d.Logger().WithError(err).Errorf("Creating REST model.")
	w.WriteHeader(http.StatusInternalServerError)
	return
}
query := r.URL.Query()
queryParams := jsonapi.ParseQueryFields(&query)
server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
```

Update any mock processor in this service if the interface changed (`grep -rn "GetAll" services/atlas-account/atlas.com/account/account/mock/ 2>/dev/null`).

- [ ] **Step 4: Run tests, gauntlet, commit**

`cd services/atlas-account/atlas.com/account && go test -race ./... && go vet ./... && go build ./...`; `docker buildx bake atlas-account`.

```bash
git commit -m "feat(atlas-account): paginate GET /accounts, delete unfiltered GetAll (task-117)"
```

---

### Task 10: atlas-character — paginate `/characters` (+ sibling list routes, `/characters/{id}/sessions`)

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/provider.go` (`getAll` at `:40-48` → paged; `getForAccount`/`getForAccountInWorld`/`getForName` gain paged siblings used by REST only)
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (interface at `:65` — delete `GetAll`, add `AllProvider(page, decorators...)`; keep `GetForAccountInWorld`/`GetForName` for internal callers, add paged providers for their handlers)
- Modify: `services/atlas-character/atlas.com/character/character/resource.go` (`handleGetCharacters` at `:39-60`; `handleGetCharactersForAccountInWorld`, `handleGetCharactersByName` gain envelope via their paged providers; route registration unchanged)
- Modify: the sessions list resource (locate: `grep -rn "sessions" services/atlas-character/atlas.com/character --include=resource.go -l`) — growing log, defaults 50/250
- Create/Modify: package tests

**Interfaces:**
- Consumes: Tasks 1–3; recipe from Task 9.
- Produces: all `/characters` list forms + `/characters/{id}/sessions` enveloped. `Processor.GetAll` gone. Internal Go callers of `GetForAccountInWorld`/`GetForName`/`GetForAccount` (character-factory flows, deletion) keep their existing unpaged *filtered* methods — verify with `grep -rn "GetForAccount\|GetForName" services/atlas-character` that no signature they use changed.

- [ ] **Step 1: Write failing resource tests** — same four assertions as Task 9 Step 2 for `GET /characters`; plus one for `?name=` form keeping its filter while accepting `page[*]`.
- [ ] **Step 2: Convert.** `getAll(page)` per Task 9. Paged siblings keep the `Where` scope, e.g.:

```go
func getForAccountInWorldPaged(accountId uint32, worldId world.Id, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where("account_id = ? AND world = ?", accountId, worldId), page)
	}
}
```

`AllProvider` exactly as Task 9 (this service's `decoratorsFromInclude` already exists — pass it through; decorators now run per page by construction). Handlers per the Task 9 Step 3 handler block with `Transform(d.Logger(), d.Context())`.
- [ ] **Step 3: UI consumer check** — `charactersService` is converted in Task 16; no Go consumer of the bare form exists (inventory row 1) — verify: `grep -rn "\"/characters\"\|characters?" services/*/atlas.com/*/*/requests.go | grep -v atlas-character` and confirm all hits carry a filter or id.
- [ ] **Step 4: Gauntlet (`services/atlas-character/atlas.com/character` module), `docker buildx bake atlas-character`, commit**

```bash
git commit -m "feat(atlas-character): paginate character list routes and sessions (task-117)"
```

---

### Task 11: atlas-guilds — paginate `GET /guilds` + server-side `filter[name]`

**Files:**
- Modify: `services/atlas-guilds/atlas.com/guilds/guild/provider.go` (`getAll` at `:11-19` → paged, keeping `Preload("Members").Preload("Titles")`; add `getByNameLike`)
- Modify: `services/atlas-guilds/atlas.com/guilds/guild/processor.go` (delete unfiltered list method; add `AllProvider(page, ...)` and `ByNameLikeProvider(name string, page model.Page)`)
- Modify: the guild resource file (locate route registration: `grep -rn "PathPrefix(\"/guilds\")" services/atlas-guilds -r`)
- Create/Modify: provider/resource tests

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: `GET /guilds` enveloped (50/250); `GET /guilds?filter[name]=<substring>` case-insensitive substring match, composable with `page[*]`, empty value → 400; `?filter[members.id]=` form keeps its shape and additionally accepts `page[*]`.

- [ ] **Step 1: Write failing tests**, including the filter cases (databasetest-backed provider tests):
  - match: guilds `Alpha`, `alphabet`, `Beta` — `filter[name]=alpha` → 2 rows, Total 2.
  - escaping: guild named `100%_raw`; `filter[name]=0%_r` matches it and does NOT match `100xraw` (pins `%`/`_` escaping).
  - paging composition: 5 matching guilds, page 2 size 2 → 2 rows, Total 5.
  - resource-level: empty `filter[name]=` → 400; bad `page[*]` → 400.
- [ ] **Step 2: Implement**

```go
// provider.go
func getAll(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Preload("Members").Preload("Titles"), page)
	}
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func getByNameLike(name string, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		pattern := "%" + escapeLike(name) + "%"
		return database.PagedQuery[Entity](
			db.Preload("Members").Preload("Titles").Where(`LOWER(name) LIKE LOWER(?) ESCAPE '\'`, pattern), page)
	}
}
```

Processor + resource per the Task 9 recipe (adapt transformer names from the actual files). Route registration adds the filter form **before** the bare form (mux matches in order):

```go
r.HandleFunc("", registerGet("get_guilds_by_name_filter", handleGetGuildsByNameFilter)).Methods(http.MethodGet).Queries("filter[name]", "{name}")
```

Handler: reject `name == ""` with `server.WriteBadRequest`, else `ByNameLikeProvider(name, page)`.
- [ ] **Step 3: Verify the members filter form** (`?filter[members.id]=`) — find its handler, keep its query shape, page its result (it's a filtered DB query → paged provider with the same `Where`; consumers pass no `page[*]` and get page 1 @ 50, which bounds a per-member guild list trivially — confirm consumers: `grep -rn "filter\[members.id\]" services/*/atlas.com -r` and convert any semantic-all Go consumer to `DrainProvider`).
- [ ] **Step 4: Gauntlet (`services/atlas-guilds/atlas.com/guilds`), `docker buildx bake atlas-guilds`, commit**

```bash
git commit -m "feat(atlas-guilds): paginate guild list, add server-side filter[name] (task-117)"
```

---

### Task 12: atlas-ban — paginate `/bans/`, `/history/`, `/history/accounts/{id}`

**Files:**
- Modify: `services/atlas-ban/atlas.com/ban/ban/provider.go` (`:25` full-table), processor, resource
- Modify: `services/atlas-ban/atlas.com/ban/history/provider.go` (`:46` full-table with `Order("created_at desc")`), processor, resource (both the bare `/history/` and `/history/accounts/{accountId}` routes)
- Create/Modify: tests

**Interfaces:** consumes Tasks 1–3; Task 9 recipe. History keeps `Order("created_at desc")` — pass the ordered `*gorm.DB` into `PagedQuery`, which appends the PK tie-break (this is the caller-order case Task 2 tests). Defaults: `/bans/` 50/250; `/history/` and `/history/accounts/{id}` 50/250 (growing logs).

- [ ] **Step 1: Read the two domains' provider/processor/resource files; write failing resource tests (envelope + 400 + created_at-desc order preserved on page 1).**
- [ ] **Step 2: Convert all three list routes per the Task 9 recipe** (history provider: `database.PagedQuery[entity](db.Where(...).Order("created_at desc"), page)`).
- [ ] **Step 3: Consumers** — inventory says UI-only (`bansService`, Task 16) and none for history. Verify: `grep -rn "bans\|history" services/*/atlas.com/*/*/requests.go | grep -vi ban/`.
- [ ] **Step 4: Gauntlet (`services/atlas-ban/atlas.com/ban`), `docker buildx bake atlas-ban`, commit**

```bash
git commit -m "feat(atlas-ban): paginate ban and history lists (task-117)"
```

---

### Task 13: atlas-notes — paginate `GET /notes`

**Files:** `services/atlas-notes/atlas.com/notes/note/provider.go` (`:31-34`), processor, resource, tests.

**Interfaces:** Task 9 recipe verbatim; defaults 50/250. No known consumer (convert-don't-remove per design §6.1 — the convention doc flags it as a removal candidate in Task 29).

- [ ] **Step 1: Failing resource test (envelope + 400).**
- [ ] **Step 2: Convert per Task 9 recipe.**
- [ ] **Step 3: Gauntlet (`services/atlas-notes/atlas.com/notes`), `docker buildx bake atlas-notes`, commit**

```bash
git commit -m "feat(atlas-notes): paginate GET /notes (task-117)"
```

---

### Task 14: atlas-merchant — paginate list routes + consumer verification

**Files:** under `services/atlas-merchant/atlas.com/merchant/` — every slice-marshaling GET (find them: `grep -rn "MarshalResponse\[\[\]" services/atlas-merchant --include='*.go' | grep -v _test`): `GET /merchants` (full-table, 50/250), per-character/per-instance forms and `GET /merchants/search/listings` (filtered → paged providers keeping their `Where`, 250/250 game-capped). Tests alongside.

**Interfaces:** Task 9 recipe. Plus the design §6.1 verification duty:

- [ ] **Step 1: Verify the suspected merchant web UI consumer.** `grep -rn "merchants" services/atlas-ui/src/services/api/merchants.service.ts` and `grep -rn "merchants" docs/tasks/legacy-merchant-web-ui/ 2>/dev/null`; also sweep Go consumers: `grep -rln "MERCHANT" services/*/atlas.com/*/*/requests.go services/*/atlas.com/*/*/*/requests.go 2>/dev/null`. Record findings in the commit message. If a consumer is in-repo, convert it (UI merchants view → Task 16 list; Go semantic-all consumer → `DrainProvider`). If evidence points outside the repo, STOP and escalate per PRD open question 3 before converting the endpoint.
- [ ] **Step 2: Failing resource tests, convert all list routes, consumer conversions in the same commit.**
- [ ] **Step 3: Gauntlet (`services/atlas-merchant/atlas.com/merchant`), `docker buildx bake atlas-merchant`, commit**

```bash
git commit -m "feat(atlas-merchant): paginate merchant list routes (task-117)"
```

#### Task 14 addendum — routes pulled in by the rebase over task-127 (owl shop scanner)

Rebasing onto main (2026-07-15) pulled in task-127, which added four new GET
routes to atlas-merchant after Task 14 was executed. Folding them into the
convention (same recipe, same page-size decisions as their siblings):

- [x] `GET /merchants/search/listings` — task-127 rewrote the search with
  criteria (`worldId` filter, `order` asc/desc, explicit tenant predicates)
  and a `MaxSearchResults=200` game cap. Merge resolution: keep the criteria,
  make the game cap the route's default/max page size (200/200), pagination
  hand-rolled with a qualified `listings.id` tiebreaker (the JOIN makes
  `database.PagedQuery`'s unqualified `ORDER BY id` ambiguous). The
  page-param-less atlas-channel owl consumer gets the capped top-N in one
  response, unchanged.
- [x] `GET /merchants/{shopId}/blacklist` — DB-paged 250/250
  (`blacklist.NamesPaged` via PagedQuery, `name ASC`); unpaged form deleted
  (in-process ban checks use `IsBlacklisted`, not the list). atlas-channel
  dialog consumer converted to `DrainProvider`.
- [x] `GET /merchants/{shopId}/visits` — DB-paged 250/250
  (`visit.ListPaged`, `count DESC` + PK tiebreak); the visit log grows with
  unique visitor names, so this is a genuine unbounded list. atlas-channel
  dialog consumer converted to `DrainProvider`.
- [x] `GET /worlds/{worldId}/shop-searches/top` — bounded top-N (`LIMIT 10`,
  now with `item_id ASC` tiebreak for a total order); envelope via
  materialize + `paginate.Slice`. atlas-channel consumer deliberately stays
  a single-page fetch (page 1 at route default is always the whole ranking).
- [x] `GET /characters/{characterId}/frederick` — single-resource status
  document, not a list; out of scope by FR definition.

Route tests in `shop/resource_test.go` (`TestGetMerchantBlacklistPaginates`,
`TestGetMerchantVisitsPaginates`, `TestGetTopShopSearchesPaginates`).

---

### Task 15: atlas-ui — shared pagination utility + envelope types

**Files:**
- Create: `services/atlas-ui/src/services/api/pagination.ts`
- Create: `services/atlas-ui/src/services/api/pagination.test.ts` (match the existing test-file convention — check `ls services/atlas-ui/src/services/api/*.test.ts` and mirror it; if tests live elsewhere, follow that)
- Modify: `services/atlas-ui/src/services/api/index.ts` (export)

**Interfaces:**
- Consumes: `api` client (`services/atlas-ui/src/lib/api/client.ts` — `api.get<T>` returns the parsed body; `api.getList` strips to `data` and is NOT sufficient here).
- Produces (used by Tasks 16, 17, 22):

```ts
export interface PageMeta { total: number; page: { number: number; size: number; last: number } }
export interface PagedResult<T> { data: T[]; meta: PageMeta | null }

// Appends page[number]/page[size] via URLSearchParams (preserving existing
// query params on `url`), decodes meta; meta === null when the server sent
// no envelope (unconverted endpoint).
export async function fetchPaged<T>(url: string, page: { number: number; size: number }, options?: ApiRequestOptions): Promise<PagedResult<T>>

// Drain: page 1 at `size`; if meta is null the single response is the whole
// collection; else iterate 2..meta.page.last, stop early on an empty page.
export async function fetchAll<T>(url: string, size?: number, options?: ApiRequestOptions): Promise<T[]>
```

- [ ] **Step 1: Write failing tests** — mock the api client module (follow the repo's existing service-test mocking pattern): multi-page drain, no-envelope compat, existing-query-param preservation, past-end empty page.
- [ ] **Step 2: Implement.** Internally call `api.get<{ data: T[]; meta?: PageMeta }>(pagedUrl, options)`; return `{ data: doc.data ?? [], meta: doc.meta ?? null }`. Default drain size 250.
- [ ] **Step 3: Run `npm test` and `npm run build` in `services/atlas-ui`; commit**

```bash
git commit -m "feat(atlas-ui): shared fetchPaged/fetchAll pagination utility (task-117)"
```

---

### Task 16: atlas-ui — characters/accounts/bans (+ merchants if in-repo) paged list views

**Files:**
- Modify: `services/atlas-ui/src/services/api/characters.service.ts`, `accounts.service.ts`, `bans.service.ts` (+ `merchants.service.ts` per Task 14's finding)
- Modify: their list-view pages/components and React Query hooks (locate per service: `grep -rln "charactersService.getAll\|getAllAccounts\|getAllBans" services/atlas-ui/src`)
- Modify: test call sites in the same commit (`npm run build` type-checks tests)

**Interfaces:**
- Consumes: Task 15 `fetchPaged`/`PagedResult`.
- Produces: each service gains `getPage(page: {number, size}, options?): Promise<PagedResult<T>>`; list views use page-keyed query keys `[resource, tenant, page, size]` with `placeholderData: keepPreviousData` and a total-driven pager (`meta.total`/`meta.page.last`). Existing `getAll()`-style methods that other code depends on for semantic-all switch to `fetchAll` internally.

- [ ] **Step 1: For each service, enumerate call sites** (`grep -rn "<service>\." services/atlas-ui/src --include='*.ts*' | grep -v services/api`) and classify: list-view (→ `getPage`) vs semantic-all (→ `fetchAll`).
- [ ] **Step 2: Convert services + views + pager UI.** Follow the existing table/pager component patterns in the repo (check how the item-strings search view pages — it already consumes this envelope). Update tests in the same commit.
- [ ] **Step 3: `npm run build` + `npm test` green; commit**

```bash
git commit -m "feat(atlas-ui): server-side paging for characters/accounts/bans views (task-117)"
```

---

### Task 17: atlas-ui — guilds service on server-side filter + paging

**Files:**
- Modify: `services/atlas-ui/src/services/api/guilds.service.ts` (delete the fetch-all-then-filter pattern: `search()`, `getByWorld()`, `getWithSpace()`, `getRankings()` internals at `guilds.service.ts:28-81`)
- Modify: guild list/search views + tests

**Interfaces:**
- Consumes: Task 15; Task 11's `?filter[name]=` contract.
- Produces: `guildsService.search(term, ...)` → `fetchPaged` with `filter[name]=<term>`; `getByMemberId` keeps `filter[members.id]`; browse view pages via `getPage`. `getAll()` is deleted; any remaining semantic-all internal use (e.g. rankings needing the full set) uses `fetchAll` — but prefer paged UI where the view only displays a page. World filtering that was client-side either stays client-side *within a page* only if the view is per-world browse — check whether the backend guild list supports a world filter (`grep -rn "worldId" services/atlas-guilds/atlas.com/guilds/guild/resource*.go`); if it does not, `getByWorld`/`getWithSpace`/`getRankings` use `fetchAll` (drain) and filter in memory — correct, just not page-bounded — and note it in context.md for a future `filter[worldId]`.

- [ ] **Step 1: Enumerate guild view call sites; write/adjust tests for the new service shape.**
- [ ] **Step 2: Convert; delete `getAll()`; `npm run build` + `npm test`; commit**

```bash
git commit -m "feat(atlas-ui): guild search via server-side filter[name], paged browse (task-117)"
```

---

## Phase B — atlas-data routes + script/config stores + UI data browsers

### Task 18: atlas-data — paginate core doc-store list routes

**Files:** the resource files for: monsters, npcs, maps, reactors, skills, mobskills, quests (incl. `/quests/auto-start`) under `services/atlas-data/atlas.com/data/<type>/resource.go`, plus their by-parent variants (`/monsters/{id}/loseItems`, `/monsters/{id}/maps`, `/npcs/{id}/maps`, `/npcs/{id}/quests`, `/maps/{id}/portals`). Reference for the current shape: `monster/resource.go:74-140`.

**Interfaces:**
- Consumes: Task 6 `Storage.AllPagedProvider`; Task 3 `ParseParams`/`EnvelopeFor`/`WriteBadRequest`; `paginate.Slice` for by-parent sub-lists.
- Produces: every bare list route returns the envelope (50/250 defaults). Routes with a `?search=` arm present the identical envelope + 400 semantics on both arms (the search arm already uses `searchindex` + envelope; normalize its param parsing onto `ParseParams` exactly as Task 5 did for item strings).

Recipe per bare route (instantiate with each type's actual storage accessor and RestModel — read the resource file first):

```go
page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
if err != nil {
	server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
	return
}
paged, err := <storage>.AllPagedProvider(d.Context())(page)()
if err != nil { /* 500 */ }
res, err := model.SliceMap(Transform...)(model.FixedProvider(paged.Items))(model.ParallelMap())()
if err != nil { /* 500 */ }
server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
```

By-parent variants fetch one parent document's sub-list (already bounded): keep the fetch, then `paged := paginate.Slice(subItems, page)` (sub-list order is the document's stored order — deterministic) and marshal with the envelope.

- [ ] **Step 1: Enumerate this task's handlers**: `grep -rn "MarshalResponse\[\[\]" services/atlas-data/atlas.com/data/{monster,npc,map,reactor,skill,mobskill,quest}* --include='*.go' | grep -v _test` — every hit is in scope here.
- [ ] **Step 2: Per type: failing resource test (envelope + 400 + search-arm/no-search-arm contract equality where applicable), convert, test.**
- [ ] **Step 3: Go-consumer gate (design §6.2 hard gate):** `grep -rn "data/monsters\|data/npcs\|data/maps\|data/reactors\|data/skills\|data/mobskills\|data/quests" services/*/atlas.com -r --include=requests.go | grep -v "%d\|{.*}"` — any consumer of a **bare** list converts to `requests.DrainProvider` in this same task; by-id fetches are untouched. Record the audited list in the commit body.
- [ ] **Step 4: Gauntlet (`services/atlas-data/atlas.com/data`), `docker buildx bake atlas-data`, commit**

```bash
git commit -m "feat(atlas-data): paginate core game-data list routes (task-117)"
```

### Task 19: atlas-data — remaining doc routes + delete unpaged storage list

**Files:** resource files for consumables, etcs, setups, cash, pets, cosmetics (`/cosmetics/hairs`, `/cosmetics/faces`), character templates, commodities (`/commodities/items`); then `services/atlas-data/atlas.com/data/document/storage.go`.

- [ ] **Step 1: Convert the remaining bare list routes** exactly per Task 18's recipe (tests first per type).
- [ ] **Step 2: Delete `Storage.GetAll` and `Storage.AllProvider`.** Then `grep -rn "\.GetAll(ctx)\|\.AllProvider(ctx)" services/atlas-data/atlas.com/data/ | grep -v _test` — every remaining internal caller (search-index builds, seeding, exports) moves to `DrainAllProvider` (semantic-all) — convert each; the build failing on a missed caller is the gate.
- [ ] **Step 3: Gauntlet + bake atlas-data; commit**

```bash
git commit -m "feat(atlas-data): paginate remaining list routes, delete unpaged doc-store GetAll (task-117)"
```

### Task 20: script/config stores A — atlas-map-actions, atlas-reactor-actions, atlas-portal-actions

**Files:** per service under `services/atlas-<name>/atlas.com/<name>/`: the list route named in the inventory (`GET /maps/actions`, `GET /reactors/actions`, `GET /portals/scripts`), its provider/processor/resource, tests.

**Interfaces:** consumes Tasks 1–3. Backing varies: DB-backed → `database.PagedQuery` (Task 9 recipe); registry/in-memory-cached → materialize, sort by primary id, `paginate.Slice` (design §6.2). Defaults 50/250.

- [ ] **Step 1: Per service, find the list handler** (`grep -rn "MarshalResponse\[\[\]" services/atlas-map-actions services/atlas-reactor-actions services/atlas-portal-actions --include='*.go' | grep -v _test`), read the backing fetch, pick the adapter (SQL vs Slice).
- [ ] **Step 2: Failing test → convert → pass, per service.** Delete each service's unfiltered `GetAll`-style processor method.
- [ ] **Step 3: Consumer gate per service:** `grep -rn "<route fragment>" services/*/atlas.com -r --include='requests.go'`; semantic-all Go consumers (script sync/caching in channel etc.) → `DrainProvider` in the same commit. **Do this before or with the server conversion.**
- [ ] **Step 4: Gauntlet each touched module; `docker buildx bake atlas-map-actions atlas-reactor-actions atlas-portal-actions`; one commit per service**

```bash
git commit -m "feat(<service>): paginate list routes (task-117)"
```

### Task 21: script/config stores B — atlas-npc-conversations, atlas-gachapons, atlas-drop-information, atlas-party-quests (definitions)

Same structure as Task 20 for: `GET /npcs/conversations`, `GET /quests/conversations`, `GET /gachapons`, `GET /global-items`, `GET /continents/drops`, `GET /party-quests/definitions`. Steps identical (discover → test → convert → consumer gate → gauntlet + bake per service → one commit per service). Note atlas-drop-information and atlas-npc-conversations have likely channel-side consumers — the consumer gate is not optional; enumerate hits and convert each to `DrainProvider`, citing file:line in the commit body.

### Task 22: atlas-ui — data browser views page server-side

**Files:** the data browser services/views: `monsters.service.ts`, `npcs.service.ts`, `maps.service.ts`, `items.service.ts`, `skills.service.ts`, `mob-skills.service.ts`, `quests.service.ts`, `reactors.service.ts`, `commodities.service.ts`, `templates.service.ts`, `gachapons.service.ts`, `conversations.service.ts`, `quest-conversations.service.ts`, `portal-scripts.service.ts`, `reactor-scripts.service.ts`, `drops.service.ts` + their list views/hooks/tests.

- [ ] **Step 1: Enumerate which of these services' list methods feed browse views vs semantic-all logic** (same classification grep as Task 16 Step 1).
- [ ] **Step 2: Browse views → `getPage` + total-driven pager (Task 15/16 pattern); semantic-all internals → `fetchAll`. Tests updated in the same commits.**
- [ ] **Step 3: `npm run build` + `npm test`; commit**

```bash
git commit -m "feat(atlas-ui): server-side paging for data browser views (task-117)"
```

---

## Phase C — filtered-but-unbounded sweeps + consumer drains

**The per-service recipe for every Task 23–26 conversion** (all four tasks share it — it is restated here once because these tasks are executed by different subagents; each task references THIS block, which is complete):

1. **Enumerate**: `grep -rn "MarshalResponse\[\[\]" services/<svc> --include='*.go' | grep -v _test` — every hit is a list handler in scope.
2. **Classify backing** by reading the provider chain: DB `Where` query → paged provider via `database.PagedQuery[entity](db.Where(...), page)` (Task 9/10 recipe); Redis/in-memory registry or single-document sub-list → materialize exactly as today, sort deterministically by primary id, `paginate.Slice(items, page)`.
3. **Defaults**: game-capped per-character/per-map lists → `paginate.ParseParams(q, paginate.MaxPageSize, paginate.MaxPageSize)` (250/250); growing logs (visits, sessions, account ban history) → `(q, paginate.DefaultPageSize, paginate.MaxPageSize)` (50/250). Before assigning 250/250, check `libs/atlas-constants` for a mechanical cap on that domain (e.g. compartment capacity, buddy capacity); if a cap above 250 exists, use `ParseParams(q, cap, cap)` and record the override for the convention doc (Task 29).
4. **Processor**: unfiltered `GetAll`-style methods deleted; filtered methods keep their unpaged form for same-service internal callers and gain a paged provider used only by the REST handler (naming: `XxxPagedProvider(args..., page model.Page)`).
5. **Handler**: exact Task 9 Step 3 handler block (ParseParams → provider → SliceMap Transform → `MarshalPaginatedResponse` with `EnvelopeFor`), 400 via `server.WriteBadRequest`.
6. **Consumer drains (hard gate, PRD FR-5.2)**: for each converted route, find every cross-service Go consumer: `grep -rn "<route fragment>" services/*/atlas.com -r --include='requests.go'`. For each hit, resolve the call chain to the consuming processor's provider (typically `requests.SliceProvider(...)(requestXxx(...), Extract, filters)`) and convert mechanically to `requests.DrainProvider[RestModel, Model](l, ctx)(<url>, 250, Extract, filters)`. **Verify by receiver type** (consumer-audit lesson): confirm each `.Xxx()` reader you touch is on the REST-consumer model, not an unrelated mirror type. Consumers convert BEFORE or WITH the server in the same task; list every converted call site as `file:line` in the commit body — this is the plan's auditable call-site checklist.
7. **Tests**: per converted service, at least one resource-level envelope + 400 test; consumer conversions covered by that service's existing tests plus a targeted httptest (two-page fixture) where a consumer had none.
8. **Gauntlet**: `go test -race ./... && go vet ./... && go build ./...` in every touched module, `docker buildx bake atlas-<svc>` for every touched `go.mod`, one commit per service: `feat(<svc>): paginate list routes; drain consumers (task-117)`.

### Task 23: atlas-inventory, atlas-storage, atlas-buddies, atlas-skills, atlas-keys

- [ ] Apply the Phase C recipe to each, in this order. Routes (from the inventory): `/characters/{id}/inventory/compartments/{cid}/assets`, `/storage/accounts/{id}/assets`, `/characters/{id}/buddy-list/buddies`, `/characters/{id}/skills`, `/characters/{id}/macros`, `/characters/{id}/keys`. All 250/250. Expected heavy consumers: atlas-channel (buddies, skills, keys, inventory), atlas-login, atlas-cashshop, atlas-asset-expiration — the recipe's step-6 grep is authoritative, not this list.
- [ ] One commit per service; gauntlet + bake per service.

### Task 24: atlas-pets, atlas-cashshop, atlas-quest, atlas-monster-book

- [ ] Phase C recipe. Routes: `/characters/{id}/pets`; `/accounts/{id}/cash-shop/inventory/compartments?type=`, `/characters/{id}/cash-shop/wishlist`; `/characters/{id}/quests` (+ `/started`, `/completed`, `/{qid}/progress`); `/characters/{id}/monster-book/cards`. All 250/250. Quest routes are hot game paths — the channel consumer MUST drain (recipe step 6), keeping exact semantics.
- [ ] One commit per service; gauntlet + bake per service.

### Task 25: atlas-marriages, atlas-families, atlas-invites, atlas-buffs, atlas-npc-shops

- [ ] Phase C recipe. Routes: `/characters/{id}/marriage/history`, `/proposals`; `/families/tree/{id}`; `/characters/{id}/invites`; `/characters/{id}/buffs` (registry-backed → `paginate.Slice`); `/npcs/{id}/shop/characters`, `/commodities/items/{id}`, `/shops` (content full-table → `PagedQuery`, 50/250). Others 250/250.
- [ ] One commit per service; gauntlet + bake per service.

### Task 26: atlas-maps + in-field registries (atlas-monsters, atlas-drops, atlas-reactors, atlas-summons, atlas-doors, atlas-chairs, atlas-chalkboards)

- [ ] Phase C recipe. atlas-maps: `/characters/{id}/visits` (**50/250 — growing log**), map/instance character lists (registry → `Slice`, 250/250). Field registries: instance-scoped monster/drop/reactor/summon/door/chair/chalkboard lists + `/characters/{id}/doors` and `/in-rect` variants — all registry-backed → materialize (existing registry read), sort by object id, `paginate.Slice`, 250/250.
- [ ] **atlas-channel is the hot consumer of nearly all of these** (in-map monster/drop/reactor state). Recipe step 6 applies per route; the drains are single-round-trip in the common case (page size ≥ spawn caps). Cite every converted channel call site in the commit body.
- [ ] One commit per service; gauntlet + bake per service (including atlas-channel each time its consumers change).

---

## Phase D — runtime registry dumps + LOW sweep

### Task 27: atlas-parties, atlas-messengers, atlas-saga-orchestrator, atlas-party-quests (instances), atlas-portals

- [ ] Phase C recipe with the Group D adapter: bare `GET /parties`, `/messengers`, `/sagas`, `/party-quests/instances`, `/portals/blocked` materialize the registry slice exactly as today, sort by primary id, `paginate.Slice`, envelope, 50/250. Filtered forms (`?filter[members.id]=`, `?characterId=`) keep their shape and additionally accept `page[*]` (paged via `Slice` on the filtered slice).
- [ ] Consumer gate: inventory says every real consumer uses the filtered forms — verify with the recipe step-6 grep anyway; convert any semantic-all hit.
- [ ] One commit per service; gauntlet + bake per service.

### Task 28: LOW sweep — atlas-world, atlas-tenants, atlas-configurations, atlas-transports

- [ ] Phase C recipe (trivial uniformity pass): `GET /worlds`, `/worlds/{id}/channels`, `/tenants`, configuration `routes`/`vessels`/`instance-routes`, `/configurations/templates`, `/services`, `/tenants`, `/transports/routes`, `/instance-routes`, `/instance-routes/{id}/status` — registry/config-backed → `Slice`; DB-backed → `PagedQuery`; 50/250.
- [ ] **Consumer caution:** worlds/channels/tenants lists are consumed at startup by login/channel/world services and the UI — recipe step 6 is mandatory; with ≤ a few dozen rows these are single-page, but drains are still required for semantic-all consumers (e.g. tenant enumeration in provisioning). atlas-tenants configuration resources have a mock (`configuration/mock/processor.go` — update it if the interface changes).
- [ ] One commit per service; gauntlet + bake per service.

---

## Phase Docs — convention, resolution, repo-wide acceptance

### Task 29: convention doc, PS-5 resolution, acceptance sweep

**Files:**
- Create: `docs/rest-pagination.md`
- Modify: `docs/architectural-improvements.md` (PS-5 → ✓ resolved, referencing task-117)
- Modify: `.claude/skills/backend-dev-guidelines/` source (locate: `grep -rn "pagination" .claude/skills/backend-dev-guidelines/ 2>/dev/null; ls .claude/skills/backend-dev-guidelines/`) — add a pointer to `docs/rest-pagination.md` in the REST section
- Modify: `docs/tasks/task-117-list-endpoint-pagination/endpoint-inventory.md` (append a "Disposition" column/section marking every inventory row converted, with the task that did it)

- [ ] **Step 1: Write `docs/rest-pagination.md`** covering exactly (design §4): param names/defaults/400 semantics (incl. legacy `limit` rejection); envelope shape with a worked JSON example; the default/max size table (standard 50/250, game-capped 250/250 + any per-endpoint overrides recorded during Phase C, growing logs 50/250); the `AllProvider(page, decorators...)` processor pattern with a code example; "DB-backed lists page in SQL" rule; consumer rules (UI `fetchPaged`/`fetchAll`; Go semantic-all uses `requests.DrainProvider`; no-envelope compat rule); `/notes` and bare `/history/` flagged as consumer-less removal candidates.
- [ ] **Step 2: Acceptance greps (all must be clean):**

```bash
# 1. No unpaginated slice marshal on any list route:
grep -rn "MarshalResponse\[\[\]" services/*/atlas.com --include='*.go' | grep -v _test
# expected: no output
# 2. No unfiltered GetAll-style processor methods (manual review of any hits):
grep -rn "func (p \*ProcessorImpl) GetAll(" services/*/atlas.com --include='*.go' | grep -v _test
# expected: no output (REST-backed drains like atlas-channel account.GetAll are renamed or justified in the doc)
# 3. Every SliceProvider consumer of a converted list is gone or filtered-by-construction:
grep -rn "requests.SliceProvider" services/*/atlas.com --include='*.go' | grep -v _test
# review each remaining hit: must target a filtered/by-id endpoint, not a converted collection
```

Fix anything the greps surface (that's the point of the sweep — do not rationalize hits away).
- [ ] **Step 3: Full-repo verification:** `go build ./...` per changed module list (from `git diff --name-only main | grep go.mod` union of touched services), `tools/redis-key-guard.sh`, `docker buildx bake all-go-services` (final belt-and-braces), atlas-ui `npm run build` + `npm test`.
- [ ] **Step 4: Commit**

```bash
git add docs/rest-pagination.md docs/architectural-improvements.md docs/tasks/task-117-list-endpoint-pagination/endpoint-inventory.md .claude/skills/backend-dev-guidelines/
git commit -m "docs: rest pagination convention; resolve PS-5 (task-117)"
```

---

## Acceptance Criteria Cross-Check (PRD §10 → tasks)

| PRD criterion | Tasks |
|---|---|
| Lib pieces + unit tests (incl. no-envelope drain compat) | 1, 2, 3, 4, 6 |
| Convention doc; PS-5 resolved | 29 |
| No unbounded collection GET anywhere; page[*] parsed; 400 on invalid | 9–14, 18–28, gated by 29's grep 1 |
| No unfiltered `GetAll`; grep clean | 9–14, 18–28, gated by 29's grep 2 |
| DB lists page in SQL, stable total order, tested | 2 (lib proof), 9–14, 20–26 per-service tests |
| atlas-data string search on hoisted ParseParams; both arms same envelope | 5, 18 |
| login/channel registry drains verified by test | 7, 8 |
| Every internal consumer drains or passes paging; audited call-site list | 7, 8, 18–28 (recipe step 6 — call sites cited per commit) |
| guilds `filter[name]`; UI uses it; fetch-all-then-filter gone | 11, 17 |
| UI views page server-side; build/tests green | 15, 16, 17, 22 |
| Full gauntlet + bake + redis-key-guard | every task + 29 Step 3 |
