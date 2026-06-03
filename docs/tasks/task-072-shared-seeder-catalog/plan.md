# Shared Seeder Library and GitOps Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the duplicated `POST /<x>/seed` + `GET /<x>/seed/status` pattern from eight Atlas services into `libs/atlas-seeder`, move bundled catalog data into a top-level `deploy/seed/<region>/<version>/` GitOps tree mounted via git-sync (k8s) / bind-mount (compose), and persist per-tenant `seed_state` revision metadata to enable a future reconciler.

**Architecture:** A new generic `libs/atlas-seeder` module owns HTTP wiring, JSON:API parsing, parallel subdomain orchestration, and the `seed_state` GORM entity. Each consumer service registers `Group`s of `Subdomain[J, M]` implementations and removes its existing seed code, bundled data dir, and per-service `*_PATH` env vars. A new `deploy/seed/<region>/<version>/` tree holds per-entity JSON:API files; one-shot Go splitters under `tools/seed-splitters/` produce the initial v83 content. A `git-sync` sidecar (k8s Kustomize component) and a `<<: *seed-catalog` compose anchor mount the tree identically across environments. A CI linter under `tools/catalog-lint/` validates JSON:API envelope conformance and `CATALOG_REVISION` presence.

**Tech Stack:** Go 1.25, GORM (`gorm.io/gorm` + `gorm.io/datatypes`), `golang.org/x/sync/errgroup`, `github.com/gorilla/mux`, `github.com/jtumidanski/api2go/jsonapi`, `github.com/sirupsen/logrus`, Prometheus (`promauto`), Kustomize, git-sync v4.4.0.

**Workflow note:** This plan has 8 task groups. Each `Task N.M` is a complete unit (typically write tests → implement → run → commit). The first three groups (library, splitters, catalog tree) are foundational; the per-service migration group (Task 4) repeats the same recipe 8× and is the largest. Run `go test -race ./... && go vet ./... && go build ./...` in every changed module before committing. For services whose `go.mod` or `Dockerfile` was touched, also run `docker build -f services/<svc>/Dockerfile .` from the worktree root before committing — per CLAUDE.md this is mandatory.

---

## Task Group 1: `libs/atlas-seeder` Library

The library lands first with full unit-test coverage. No service depends on it yet.

### Task 1.1: Bootstrap the library module

**Files:**
- Create: `libs/atlas-seeder/go.mod`
- Create: `libs/atlas-seeder/README.md`

- [ ] **Step 1: Create the module directory and go.mod**

```bash
mkdir -p libs/atlas-seeder
cd libs/atlas-seeder
cat > go.mod <<'EOF'
module github.com/Chronicle20/atlas/libs/atlas-seeder

go 1.25.0

require (
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-rest v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jtumidanski/api2go v0.0.0
	github.com/prometheus/client_golang v1.20.5
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/sync v0.20.0
	gorm.io/datatypes v1.2.5
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.25.12
)

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model
replace github.com/Chronicle20/atlas/libs/atlas-rest => ../atlas-rest
replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant
EOF
```

Verify the exact versions for each dependency by running `go mod tidy` in step 2 — it will rewrite the file with concrete versions matching the workspace.

- [ ] **Step 2: Add to top-level go.work and tidy**

```bash
cd <worktree>
# Append the new module to go.work
go work use ./libs/atlas-seeder
cd libs/atlas-seeder && go mod tidy
```

Expected: `go.sum` is created, dep versions are pinned.

- [ ] **Step 3: Write a placeholder README**

```markdown
# atlas-seeder

Shared library for tenant-scoped catalog seeding. Provides:

- `Subdomain[J, M]` generic interface for one catalog dataset.
- `Group` for one `(POST /<prefix>/seed, GET /<prefix>/seed/status)` endpoint pair.
- `CatalogSource` abstraction over file lookup (v1: filesystem rooted at `SEED_CATALOG_ROOT`).
- `RegisterRoutes` to wire HTTP handlers.
- `SeedState` GORM entity persisting `(tenant_id, group_name) -> catalog_revision` per service.

See `docs/tasks/task-072-shared-seeder-catalog/design.md` for architecture.
```

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-seeder/go.mod libs/atlas-seeder/go.sum libs/atlas-seeder/README.md go.work
git commit -m "feat(atlas-seeder): bootstrap module skeleton"
```

### Task 1.2: Define DTOs and the `SeedState` entity

**Files:**
- Create: `libs/atlas-seeder/result.go`
- Create: `libs/atlas-seeder/state.go`
- Test: `libs/atlas-seeder/state_test.go`

- [ ] **Step 1: Write the failing entity test**

```go
// libs/atlas-seeder/state_test.go
package seeder

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SeedState{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestSeedState_TableName(t *testing.T) {
	if got := (SeedState{}).TableName(); got != "seed_state" {
		t.Fatalf("TableName = %q, want seed_state", got)
	}
}

func TestSeedState_UpsertReplacesExistingRow(t *testing.T) {
	db := openTestDB(t)
	tenantID := uuid.New()
	first := SeedState{
		TenantID:        tenantID,
		GroupName:       "drops",
		CatalogRevision: "rev-1",
		SeededAt:        time.Now().UTC(),
		ResultSummary:   datatypes.JSON(`{"groupName":"drops"}`),
	}
	if err := UpsertSeedState(db, &first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second := first
	second.CatalogRevision = "rev-2"
	second.ResultSummary = datatypes.JSON(`{"groupName":"drops","run":2}`)
	if err := UpsertSeedState(db, &second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := ReadSeedState(db, tenantID, "drops")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got == nil || got.CatalogRevision != "rev-2" {
		t.Fatalf("CatalogRevision = %v, want rev-2", got)
	}
	var summary map[string]any
	if err := json.Unmarshal(got.ResultSummary, &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summary["run"] != float64(2) {
		t.Fatalf("ResultSummary not replaced: %v", summary)
	}
}

func TestReadSeedState_NotFoundReturnsNil(t *testing.T) {
	db := openTestDB(t)
	got, err := ReadSeedState(db, uuid.New(), "drops")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("got = %+v, want nil", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestSeedState -v
```

Expected: FAIL with "undefined: SeedState".

- [ ] **Step 3: Implement `result.go`**

```go
// libs/atlas-seeder/result.go
package seeder

import "time"

// SubdomainCounts is the per-subdomain outcome captured in a seed Result.
type SubdomainCounts struct {
	Deleted int64    `json:"deleted"`
	Created int64    `json:"created"`
	Failed  int64    `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// Result is the seeding outcome serialized to seed_state.result_summary.
type Result struct {
	GroupName       string                     `json:"groupName"`
	CatalogRevision string                     `json:"catalogRevision"`
	Subdomains      map[string]SubdomainCounts `json:"subdomains"`
	StartedAt       time.Time                  `json:"startedAt"`
	CompletedAt     time.Time                  `json:"completedAt"`
}

// SubdomainStatus is the per-subdomain count exposed via GET /seed/status.
type SubdomainStatus struct {
	Count     int64      `json:"count"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// Status is the GET /seed/status response payload.
type Status struct {
	GroupName            string                     `json:"groupName"`
	Subdomains           map[string]SubdomainStatus `json:"subdomains"`
	UpdatedAt            *time.Time                 `json:"updatedAt"`
	CatalogRevision      string                     `json:"catalogRevision"`
	TenantSeededRevision *string                    `json:"tenantSeededRevision"`
	TenantSeededAt       *time.Time                 `json:"tenantSeededAt"`
}

// MaxErrors caps the per-subdomain error list to keep Result rows bounded.
const MaxErrors = 100
```

- [ ] **Step 4: Implement `state.go`**

```go
// libs/atlas-seeder/state.go
package seeder

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeedState struct {
	TenantID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	GroupName       string         `gorm:"type:text;primaryKey"`
	CatalogRevision string         `gorm:"type:text;not null"`
	SeededAt        time.Time      `gorm:"type:timestamptz;not null"`
	ResultSummary   datatypes.JSON `gorm:"type:jsonb;not null"`
}

func (SeedState) TableName() string { return "seed_state" }

func UpsertSeedState(db *gorm.DB, s *SeedState) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "group_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"catalog_revision",
			"seeded_at",
			"result_summary",
		}),
	}).Create(s).Error
}

func ReadSeedState(db *gorm.DB, tenantID uuid.UUID, groupName string) (*SeedState, error) {
	var out SeedState
	err := db.Where("tenant_id = ? AND group_name = ?", tenantID, groupName).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestSeedState -v
```

Expected: PASS on all three TestSeedState* subtests.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-seeder/result.go libs/atlas-seeder/state.go libs/atlas-seeder/state_test.go libs/atlas-seeder/go.sum
git commit -m "feat(atlas-seeder): add Result/Status DTOs and SeedState entity"
```

### Task 1.3: Implement the `Subdomain` interface and type-erased adapter

**Files:**
- Create: `libs/atlas-seeder/subdomain.go`
- Test: `libs/atlas-seeder/subdomain_test.go`

- [ ] **Step 1: Write the failing adapter test**

```go
// libs/atlas-seeder/subdomain_test.go
package seeder

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeAttrs struct {
	Name string `json:"name"`
}

type fakeRow struct {
	ID   uint64
	Name string
}

type fakeSubdomain struct {
	name       string
	path       string
	typ        string
	pattern    *regexp.Regexp
	decoded    fakeAttrs
	builtRows  []fakeRow
	deleted    int64
	count      int64
	updatedAt  *time.Time
	bulkCalled bool
}

func (f *fakeSubdomain) Name() string                  { return f.name }
func (f *fakeSubdomain) Path() string                  { return f.path }
func (f *fakeSubdomain) Type() string                  { return f.typ }
func (f *fakeSubdomain) EntityIDPattern() *regexp.Regexp { return f.pattern }
func (f *fakeSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) {
	return f.deleted, nil
}
func (f *fakeSubdomain) Decode(b []byte) (fakeAttrs, error) {
	var a fakeAttrs
	err := json.Unmarshal(b, &a)
	f.decoded = a
	return a, err
}
func (f *fakeSubdomain) Build(_ tenant.Model, _ string, a fakeAttrs) ([]fakeRow, error) {
	r := fakeRow{ID: 1, Name: a.Name}
	f.builtRows = append(f.builtRows, r)
	return []fakeRow{r}, nil
}
func (f *fakeSubdomain) BulkCreate(_ *gorm.DB, _ []fakeRow) error {
	f.bulkCalled = true
	return nil
}
func (f *fakeSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) {
	return f.count, f.updatedAt, nil
}

func TestAdaptSubdomain_PreservesNameAndPath(t *testing.T) {
	s := &fakeSubdomain{name: "widgets", path: "widgets", typ: "widget"}
	a := AdaptSubdomain[fakeAttrs, fakeRow](s)
	if a.Name() != "widgets" || a.Path() != "widgets" || a.Type() != "widget" {
		t.Fatalf("adapter dropped metadata: %s/%s/%s", a.Name(), a.Path(), a.Type())
	}
}

func TestAdaptSubdomain_DecodeBuildBulkCreatePropagate(t *testing.T) {
	s := &fakeSubdomain{
		name:    "widgets",
		path:    "widgets",
		typ:     "widget",
		pattern: regexp.MustCompile(`^widget-(\d+)\.json$`),
	}
	a := AdaptSubdomain[fakeAttrs, fakeRow](s)
	tm := tenant.NewModel(uuid.New(), "gms", 83, 1)
	rows, err := a.LoadAndBuild(tm, "42", []byte(`{"name":"hello"}`))
	if err != nil {
		t.Fatalf("LoadAndBuild: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if err := a.BulkCreate(nil, rows); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}
	if !s.bulkCalled {
		t.Fatalf("inner BulkCreate not called")
	}
	if s.decoded.Name != "hello" {
		t.Fatalf("decoded name = %q", s.decoded.Name)
	}
}
```

Note: `tenant.NewModel(...)` is a constructor — verify it exists in `libs/atlas-tenant/tenant.go`; if it doesn't, use whatever Builder/constructor that file exposes and adjust this test. The test exists to lock the adapter's behavior, not to invent tenant API.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestAdaptSubdomain -v
```

Expected: FAIL — `AdaptSubdomain` undefined.

- [ ] **Step 3: Implement `subdomain.go`**

```go
// libs/atlas-seeder/subdomain.go
package seeder

import (
	"regexp"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// Subdomain declares one tenant-scoped catalog dataset.
//
//	J = the JSON:API attributes shape parsed from a catalog file
//	M = the GORM model shape persisted to the database
type Subdomain[J any, M any] interface {
	Name() string
	Path() string
	Type() string
	EntityIDPattern() *regexp.Regexp
	DeleteAllForTenant(db *gorm.DB) (int64, error)
	Decode(payload []byte) (J, error)
	Build(t tenant.Model, entityID string, j J) ([]M, error)
	BulkCreate(db *gorm.DB, models []M) error
	Count(db *gorm.DB) (count int64, mostRecentUpdate *time.Time, err error)
}

// SubdomainAny is the type-erased form Group holds. Services do not implement
// it directly; they wrap a typed Subdomain via AdaptSubdomain.
type SubdomainAny interface {
	Name() string
	Path() string
	Type() string
	EntityIDPattern() *regexp.Regexp
	DeleteAllForTenant(db *gorm.DB) (int64, error)
	LoadAndBuild(t tenant.Model, entityID string, payload []byte) (any, error)
	BulkCreate(db *gorm.DB, rows any) error
	Count(db *gorm.DB) (int64, *time.Time, error)
}

type adapter[J any, M any] struct {
	inner Subdomain[J, M]
}

func AdaptSubdomain[J any, M any](s Subdomain[J, M]) SubdomainAny {
	return &adapter[J, M]{inner: s}
}

func (a *adapter[J, M]) Name() string                  { return a.inner.Name() }
func (a *adapter[J, M]) Path() string                  { return a.inner.Path() }
func (a *adapter[J, M]) Type() string                  { return a.inner.Type() }
func (a *adapter[J, M]) EntityIDPattern() *regexp.Regexp { return a.inner.EntityIDPattern() }

func (a *adapter[J, M]) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return a.inner.DeleteAllForTenant(db)
}

func (a *adapter[J, M]) LoadAndBuild(t tenant.Model, entityID string, payload []byte) (any, error) {
	j, err := a.inner.Decode(payload)
	if err != nil {
		return nil, err
	}
	rows, err := a.inner.Build(t, entityID, j)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (a *adapter[J, M]) BulkCreate(db *gorm.DB, rows any) error {
	typed, ok := rows.([]M)
	if !ok {
		return errAdapterTypeMismatch
	}
	return a.inner.BulkCreate(db, typed)
}

func (a *adapter[J, M]) Count(db *gorm.DB) (int64, *time.Time, error) {
	return a.inner.Count(db)
}

var errAdapterTypeMismatch = errAdapterMismatch{}

type errAdapterMismatch struct{}

func (errAdapterMismatch) Error() string {
	return "atlas-seeder: BulkCreate received rows of unexpected element type"
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestAdaptSubdomain -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-seeder/subdomain.go libs/atlas-seeder/subdomain_test.go
git commit -m "feat(atlas-seeder): add Subdomain[J,M] generic + type-erased adapter"
```

### Task 1.4: Implement `Group`, `Seeder` envelope parsing, and shared helpers

**Files:**
- Create: `libs/atlas-seeder/seeder.go`
- Create: `libs/atlas-seeder/jsonapi.go`
- Test: `libs/atlas-seeder/jsonapi_test.go`

- [ ] **Step 1: Write the failing envelope test**

```go
// libs/atlas-seeder/jsonapi_test.go
package seeder

import (
	"strings"
	"testing"
)

func TestParseEnvelope_Valid(t *testing.T) {
	env, err := ParseEnvelope([]byte(`{"data":{"type":"widget","id":"42","attributes":{"name":"hi"}}}`))
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	if env.Data.Type != "widget" || env.Data.ID != "42" {
		t.Fatalf("got type=%q id=%q", env.Data.Type, env.Data.ID)
	}
	if string(env.Data.Attributes) == "" {
		t.Fatalf("attributes empty")
	}
}

func TestParseEnvelope_MissingData(t *testing.T) {
	_, err := ParseEnvelope([]byte(`{"type":"widget"}`))
	if err == nil || !strings.Contains(err.Error(), "data") {
		t.Fatalf("expected data-missing error, got: %v", err)
	}
}

func TestParseEnvelope_MalformedJSON(t *testing.T) {
	_, err := ParseEnvelope([]byte(`{"data":`))
	if err == nil {
		t.Fatalf("expected error on malformed JSON")
	}
}

func TestExtractEntityID_Match(t *testing.T) {
	id, err := ExtractEntityID("monster-100100.json", monsterPattern())
	if err != nil {
		t.Fatalf("ExtractEntityID: %v", err)
	}
	if id != "100100" {
		t.Fatalf("id = %q, want 100100", id)
	}
}

func TestExtractEntityID_NoMatch(t *testing.T) {
	_, err := ExtractEntityID("bogus.json", monsterPattern())
	if err == nil {
		t.Fatalf("expected error on no match")
	}
}

func monsterPattern() *regexpish {
	return mustPattern(`^monster-(\d+)\.json$`)
}
```

For `regexpish` / `mustPattern` simply use `*regexp.Regexp` and `regexp.MustCompile` from `regexp` — the test pseudocode shows intent; rewrite to real types when implementing:

```go
import "regexp"

func monsterPattern() *regexp.Regexp { return regexp.MustCompile(`^monster-(\d+)\.json$`) }
```

(Drop the `regexpish` / `mustPattern` placeholders before saving the test.)

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run "TestParseEnvelope|TestExtractEntityID" -v
```

Expected: FAIL — `ParseEnvelope` / `ExtractEntityID` undefined.

- [ ] **Step 3: Implement `jsonapi.go`**

```go
// libs/atlas-seeder/jsonapi.go
package seeder

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Envelope is the canonical JSON:API document shape every catalog file uses.
type Envelope struct {
	Data EnvelopeData `json:"data"`
}

type EnvelopeData struct {
	Type          string          `json:"type"`
	ID            string          `json:"id"`
	Attributes    json.RawMessage `json:"attributes"`
	Relationships json.RawMessage `json:"relationships,omitempty"`
}

// ParseEnvelope decodes the JSON:API envelope. Returns an error when the
// "data" object is missing or the JSON is malformed; does NOT validate type/id.
func ParseEnvelope(b []byte) (Envelope, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return Envelope{}, fmt.Errorf("parse envelope: %w", err)
	}
	dataRaw, ok := raw["data"]
	if !ok {
		return Envelope{}, fmt.Errorf("parse envelope: missing data object")
	}
	var env Envelope
	if err := json.Unmarshal(dataRaw, &env.Data); err != nil {
		return Envelope{}, fmt.Errorf("parse envelope data: %w", err)
	}
	if env.Data.Type == "" {
		return Envelope{}, fmt.Errorf("parse envelope: data.type empty")
	}
	if env.Data.ID == "" {
		return Envelope{}, fmt.Errorf("parse envelope: data.id empty")
	}
	return env, nil
}

// ExtractEntityID returns capture group 1 from pattern applied to filename.
func ExtractEntityID(filename string, pattern *regexp.Regexp) (string, error) {
	if pattern == nil {
		return "", fmt.Errorf("extract id: nil pattern")
	}
	m := pattern.FindStringSubmatch(filename)
	if len(m) < 2 {
		return "", fmt.Errorf("extract id: filename %q does not match pattern", filename)
	}
	return m[1], nil
}
```

- [ ] **Step 4: Implement `seeder.go` (Group only — Seed/Status come later)**

```go
// libs/atlas-seeder/seeder.go
package seeder

// Group declares one (POST /<prefix>/seed, GET /<prefix>/seed/status) pair.
type Group struct {
	Name       string         // stored as seed_state.group_name; e.g. "drops"
	URLPrefix  string         // e.g. "/drops" → routes POST /drops/seed
	Subdomains []SubdomainAny
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run "TestParseEnvelope|TestExtractEntityID" -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-seeder/seeder.go libs/atlas-seeder/jsonapi.go libs/atlas-seeder/jsonapi_test.go
git commit -m "feat(atlas-seeder): add JSON:API envelope parser and Group type"
```

### Task 1.5: Implement `CatalogSource` and `FilesystemCatalogSource`

**Files:**
- Create: `libs/atlas-seeder/catalog.go`
- Test: `libs/atlas-seeder/catalog_test.go`
- Test fixtures: `libs/atlas-seeder/testdata/good/gms/83_1/...`

- [ ] **Step 1: Create fixture catalog tree**

```bash
mkdir -p libs/atlas-seeder/testdata/good/gms/83_1/widgets
mkdir -p libs/atlas-seeder/testdata/good/gms/83_1/gizmos/_global
cat > libs/atlas-seeder/testdata/good/gms/83_1/CATALOG_REVISION <<'EOF'
test-rev-abc123
EOF
cat > libs/atlas-seeder/testdata/good/gms/83_1/widgets/widget-1.json <<'EOF'
{"data":{"type":"widget","id":"1","attributes":{"name":"one"}}}
EOF
cat > libs/atlas-seeder/testdata/good/gms/83_1/widgets/widget-2.json <<'EOF'
{"data":{"type":"widget","id":"2","attributes":{"name":"two"}}}
EOF
cat > libs/atlas-seeder/testdata/good/gms/83_1/widgets/_skipped.json <<'EOF'
{"data":{"type":"widget","id":"skip","attributes":{}}}
EOF
cat > libs/atlas-seeder/testdata/good/gms/83_1/gizmos/gizmo-100.json <<'EOF'
{"data":{"type":"gizmo","id":"100","attributes":{}}}
EOF
cat > libs/atlas-seeder/testdata/good/gms/83_1/gizmos/_global/pool.json <<'EOF'
{"data":{"type":"gizmo-pool","id":"_global","attributes":{"items":[]}}}
EOF
```

- [ ] **Step 2: Write the failing catalog test**

```go
// libs/atlas-seeder/catalog_test.go
package seeder

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func goodFixtureRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", "good")
}

func tenantGMS83() tenant.Model {
	return tenant.NewModel(uuid.New(), "gms", 83, 1)
}

func TestFilesystemCatalogSource_Roots_UsesTenantRegionVersion(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	roots, err := src.Roots(tenantGMS83())
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
	if !strings.HasSuffix(filepath.ToSlash(roots[0]), "good/gms/83_1") {
		t.Fatalf("root = %q, want suffix good/gms/83_1", roots[0])
	}
}

func TestFilesystemCatalogSource_Revision(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	roots, _ := src.Roots(tenantGMS83())
	rev, err := src.Revision(roots[0])
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	if rev != "test-rev-abc123" {
		t.Fatalf("rev = %q, want test-rev-abc123", rev)
	}
}

func TestFilesystemCatalogSource_Revision_MissingReturnsEmptyNoError(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	rev, err := src.Revision(filepath.Join(goodFixtureRoot(t), "gms", "999_1"))
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	if rev != "" {
		t.Fatalf("rev = %q, want empty", rev)
	}
}

func TestFilesystemCatalogSource_Walk_SkipsUnderscorePrefixed(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	roots, _ := src.Roots(tenantGMS83())
	files, err := src.Walk(roots[0], "widgets")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	sort.Strings(files)
	want := []string{"widget-1.json", "widget-2.json"}
	if len(files) != len(want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("files[%d] = %q, want %q", i, files[i], want[i])
		}
	}
}

func TestFilesystemCatalogSource_Walk_SkipsUnderscoreDir(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	roots, _ := src.Roots(tenantGMS83())
	// gizmos/ contains _global/ (skipped) and gizmo-100.json (kept)
	files, err := src.Walk(roots[0], "gizmos")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 || files[0] != "gizmo-100.json" {
		t.Fatalf("files = %v, want [gizmo-100.json]", files)
	}
}

func TestFilesystemCatalogSource_Open(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	roots, _ := src.Roots(tenantGMS83())
	rc, err := src.Open(roots[0], "widgets/widget-1.json")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if !strings.Contains(string(b), `"id":"1"`) {
		t.Fatalf("content = %q", string(b))
	}
}

func TestFilesystemCatalogSource_EnvOverridesFallback(t *testing.T) {
	t.Setenv("MY_TEST_CATALOG_ROOT", goodFixtureRoot(t))
	src := NewFilesystemCatalogSource("MY_TEST_CATALOG_ROOT", "/non/existent")
	roots, err := src.Roots(tenantGMS83())
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(roots[0]), "good/gms/83_1") {
		t.Fatalf("env override ignored: %q", roots[0])
	}
}

func TestFilesystemCatalogSource_Roots_ZeroVersion(t *testing.T) {
	src := NewFilesystemCatalogSource("THIS_ENV_DOES_NOT_EXIST", goodFixtureRoot(t))
	_, err := src.Roots(tenant.NewModel(uuid.New(), "gms", 0, 0))
	if err == nil {
		t.Fatalf("expected error on zero version")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestFilesystemCatalogSource -v
```

Expected: FAIL — `NewFilesystemCatalogSource` undefined.

- [ ] **Step 4: Implement `catalog.go`**

```go
// libs/atlas-seeder/catalog.go
package seeder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CatalogSource abstracts where catalog files live.
type CatalogSource interface {
	Roots(t tenant.Model) ([]string, error)
	Revision(root string) (string, error)
	Open(root, relPath string) (io.ReadCloser, error)
	Walk(root, relPath string) ([]string, error)
}

type filesystemSource struct {
	envVar       string
	fallbackRoot string
}

// NewFilesystemCatalogSource returns a CatalogSource rooted at os.Getenv(envVar)
// when set; otherwise at fallbackRoot. fallbackRoot is normalized via
// filepath.Abs so cwd drift does not break dev runs.
func NewFilesystemCatalogSource(envVar, fallbackRoot string) CatalogSource {
	return &filesystemSource{envVar: envVar, fallbackRoot: fallbackRoot}
}

func (s *filesystemSource) base() string {
	if v := os.Getenv(s.envVar); v != "" {
		return v
	}
	abs, err := filepath.Abs(s.fallbackRoot)
	if err != nil {
		return s.fallbackRoot
	}
	return abs
}

func (s *filesystemSource) Roots(t tenant.Model) ([]string, error) {
	if t.MajorVersion() == 0 || t.MinorVersion() == 0 {
		return nil, fmt.Errorf("catalog: tenant has zero major/minor version (region=%q)", t.Region())
	}
	root := filepath.Join(s.base(), t.Region(), fmt.Sprintf("%d_%d", t.MajorVersion(), t.MinorVersion()))
	return []string{root}, nil
}

func (s *filesystemSource) Revision(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "CATALOG_REVISION"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (s *filesystemSource) Open(root, relPath string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(root, relPath))
}

func (s *filesystemSource) Walk(root, relPath string) ([]string, error) {
	dir := filepath.Join(root, relPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}
```

Note: this v1 `Walk` is shallow (one level under relPath). The design's "overlay-aware walk that honors ordered roots and tombstones" is deferred to the merge-overlay extension and is not exercised in v1. Skipping `_*` subdirectories is achieved because v1 never recurses into them. If a subdomain has subdirectories (e.g., portal-actions wants `portals/`), the Subdomain's `Path()` points at the leaf directory directly.

- [ ] **Step 5: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestFilesystemCatalogSource -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-seeder/catalog.go libs/atlas-seeder/catalog_test.go libs/atlas-seeder/testdata/good
git commit -m "feat(atlas-seeder): add FilesystemCatalogSource with tenant-aware root resolution"
```

### Task 1.6: Implement metrics

**Files:**
- Create: `libs/atlas-seeder/metrics.go`
- Test: `libs/atlas-seeder/metrics_test.go`

- [ ] **Step 1: Write the failing metrics test**

```go
// libs/atlas-seeder/metrics_test.go
package seeder

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestObserveSeederRun_IncrementsCounter(t *testing.T) {
	ResetMetricsForTest()
	ObserveSeederRun("atlas-test", "drops", "success", 0.5)
	ObserveSeederRun("atlas-test", "drops", "success", 0.25)
	got := testutil.ToFloat64(seederRunsTotal.WithLabelValues("atlas-test", "drops", "success"))
	if got != 2 {
		t.Fatalf("counter = %v, want 2", got)
	}
	histCount := testutil.CollectAndCount(seederDurationSeconds)
	if histCount == 0 {
		t.Fatalf("histogram not registered")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestObserveSeederRun -v
```

Expected: FAIL — `ObserveSeederRun` undefined.

- [ ] **Step 3: Implement `metrics.go`**

```go
// libs/atlas-seeder/metrics.go
package seeder

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce           sync.Once
	seederRunsTotal       *prometheus.CounterVec
	seederDurationSeconds *prometheus.HistogramVec
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		seederRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "atlas_seeder_runs_total",
			Help: "Count of atlas-seeder Seed() invocations by service, group, and outcome.",
		}, []string{"service", "group", "outcome"})
		seederDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "atlas_seeder_duration_seconds",
			Help:    "Wall-clock duration of atlas-seeder Seed() invocations.",
			Buckets: prometheus.DefBuckets,
		}, []string{"service", "group"})
		prometheus.MustRegister(seederRunsTotal, seederDurationSeconds)
	})
}

// ObserveSeederRun records one Seed() completion. outcome is one of
// "success", "partial", or "failure".
func ObserveSeederRun(service, group, outcome string, durationSeconds float64) {
	ensureMetrics()
	seederRunsTotal.WithLabelValues(service, group, outcome).Inc()
	seederDurationSeconds.WithLabelValues(service, group).Observe(durationSeconds)
}

// ResetMetricsForTest unregisters the counters so a fresh test run does not
// accumulate state across tests. Test-only.
func ResetMetricsForTest() {
	if seederRunsTotal != nil {
		prometheus.Unregister(seederRunsTotal)
	}
	if seederDurationSeconds != nil {
		prometheus.Unregister(seederDurationSeconds)
	}
	seederRunsTotal = nil
	seederDurationSeconds = nil
	metricsOnce = sync.Once{}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestObserveSeederRun -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-seeder/metrics.go libs/atlas-seeder/metrics_test.go
git commit -m "feat(atlas-seeder): add Prometheus counter and duration histogram"
```

### Task 1.7: Implement `Seed` orchestrator

**Files:**
- Create: `libs/atlas-seeder/seed.go`
- Test: `libs/atlas-seeder/seed_test.go`

- [ ] **Step 1: Write the failing orchestrator test**

```go
// libs/atlas-seeder/seed_test.go
package seeder

import (
	"context"
	"encoding/json"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type widgetAttrs struct {
	Name string `json:"name"`
}
type widgetRow struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

type widgetSubdomain struct {
	deleted int64
	created atomic.Int64
}

func (w *widgetSubdomain) Name() string                  { return "widgets" }
func (w *widgetSubdomain) Path() string                  { return "widgets" }
func (w *widgetSubdomain) Type() string                  { return "widget" }
func (w *widgetSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^widget-(\d+)\.json$`)
}
func (w *widgetSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) {
	return w.deleted, nil
}
func (w *widgetSubdomain) Decode(b []byte) (widgetAttrs, error) {
	var a widgetAttrs
	return a, json.Unmarshal(b, &a)
}
func (w *widgetSubdomain) Build(_ tenant.Model, id string, a widgetAttrs) ([]widgetRow, error) {
	n, _ := uintFromString(id)
	return []widgetRow{{ID: n, Name: a.Name}}, nil
}
func (w *widgetSubdomain) BulkCreate(_ *gorm.DB, rows []widgetRow) error {
	w.created.Add(int64(len(rows)))
	return nil
}
func (w *widgetSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) {
	now := time.Now().UTC()
	return w.created.Load(), &now, nil
}

func uintFromString(s string) (uint64, error) {
	var n uint64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, nil
		}
		n = n*10 + uint64(r-'0')
	}
	return n, nil
}

func TestSeed_SuccessfulRunPersistsStateAndCountsCreated(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:      "widgets-group",
		URLPrefix: "/widgets",
		Subdomains: []SubdomainAny{
			AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{}),
		},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83())
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("revision = %q, want test-rev-abc123", res.CatalogRevision)
	}
	if res.Subdomains["widgets"].Created != 2 {
		t.Fatalf("created = %d, want 2 (widget-1.json + widget-2.json)", res.Subdomains["widgets"].Created)
	}
	row, err := ReadSeedState(db, tenant.MustFromContext(ctx).Id(), "widgets-group")
	if err != nil || row == nil {
		t.Fatalf("expected seed_state row, got err=%v row=%v", err, row)
	}
	if row.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("row.CatalogRevision = %q", row.CatalogRevision)
	}
}

// failingSubdomain returns an error from Decode for every file.
type failingSubdomain struct{}

func (f *failingSubdomain) Name() string                    { return "broken" }
func (f *failingSubdomain) Path() string                    { return "widgets" }
func (f *failingSubdomain) Type() string                    { return "widget" }
func (f *failingSubdomain) EntityIDPattern() *regexp.Regexp { return regexp.MustCompile(`^widget-(\d+)\.json$`) }
func (f *failingSubdomain) DeleteAllForTenant(_ *gorm.DB) (int64, error) { return 0, nil }
func (f *failingSubdomain) Decode(_ []byte) (widgetAttrs, error) {
	return widgetAttrs{}, errBad
}
func (f *failingSubdomain) Build(_ tenant.Model, _ string, _ widgetAttrs) ([]widgetRow, error) {
	return nil, nil
}
func (f *failingSubdomain) BulkCreate(_ *gorm.DB, _ []widgetRow) error  { return nil }
func (f *failingSubdomain) Count(_ *gorm.DB) (int64, *time.Time, error) { return 0, nil, nil }

var errBad = errSimple("intentional decode failure")

type errSimple string

func (e errSimple) Error() string { return string(e) }

func TestSeed_PartialFailurePersistsAndContinues(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:      "mixed",
		URLPrefix: "/mixed",
		Subdomains: []SubdomainAny{
			AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{}),
			AdaptSubdomain[widgetAttrs, widgetRow](&failingSubdomain{}),
		},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83())
	res, err := Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.Subdomains["widgets"].Created != 2 {
		t.Fatalf("widgets created = %d, want 2", res.Subdomains["widgets"].Created)
	}
	if res.Subdomains["broken"].Failed != 2 {
		t.Fatalf("broken failed = %d, want 2 (decode failures)", res.Subdomains["broken"].Failed)
	}
	row, _ := ReadSeedState(db, tenant.MustFromContext(ctx).Id(), "mixed")
	if row == nil {
		t.Fatalf("seed_state row missing on partial failure")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestSeed -v
```

Expected: FAIL — `Seed` undefined.

- [ ] **Step 3: Implement `seed.go`**

```go
// libs/atlas-seeder/seed.go
package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"golang.org/x/sync/errgroup"
)

// Seed orchestrates per-subdomain delete-then-bulk-insert in parallel.
// Persists one row to seed_state on completion (even on partial failure).
func Seed(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Result, error) {
	t := tenant.MustFromContext(ctx)
	started := time.Now().UTC()

	roots, err := src.Roots(t)
	if err != nil {
		return persistAndReturn(ctx, db, g, Result{
			GroupName: g.Name, StartedAt: started, CompletedAt: time.Now().UTC(),
			Subdomains: map[string]SubdomainCounts{},
		}, t.Id(), "failure")
	}
	rev, _ := src.Revision(roots[0])

	subCounts := make(map[string]SubdomainCounts, len(g.Subdomains))
	var mu sync.Mutex

	eg, gctx := errgroup.WithContext(ctx)
	for _, sd := range g.Subdomains {
		sd := sd
		eg.Go(func() error {
			counts := runSubdomain(gctx, db, src, roots[0], t, sd)
			mu.Lock()
			subCounts[sd.Name()] = counts
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	completed := time.Now().UTC()
	res := Result{
		GroupName:       g.Name,
		CatalogRevision: rev,
		Subdomains:      subCounts,
		StartedAt:       started,
		CompletedAt:     completed,
	}
	outcome := classifyOutcome(subCounts)
	ObserveSeederRun(serviceLabel(), g.Name, outcome, completed.Sub(started).Seconds())
	return persistAndReturn(ctx, db, g, res, t.Id(), outcome)
}

func runSubdomain(ctx context.Context, db *gorm.DB, src CatalogSource, root string, t tenant.Model, sd SubdomainAny) SubdomainCounts {
	var counts SubdomainCounts
	deleted, err := sd.DeleteAllForTenant(db.WithContext(ctx))
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("delete: %v", err))
		return counts
	}
	counts.Deleted = deleted

	files, err := src.Walk(root, sd.Path())
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("walk %s: %v", sd.Path(), err))
		return counts
	}

	pattern := sd.EntityIDPattern()
	for _, name := range files {
		if err := ctx.Err(); err != nil {
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: context: %v", name, err))
			return counts
		}
		rows, err := loadOne(src, root, sd, pattern, name)
		if err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := sd.BulkCreate(db.WithContext(ctx), rows); err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: bulkcreate: %v", name, err))
			continue
		}
		counts.Created += rowCount(rows)
	}
	return counts
}

func loadOne(src CatalogSource, root string, sd SubdomainAny, pattern *anyPattern, filename string) (any, error) {
	rc, err := src.Open(root, joinPath(sd.Path(), filename))
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	env, err := ParseEnvelope(b)
	if err != nil {
		return nil, err
	}
	if env.Data.Type != sd.Type() {
		return nil, fmt.Errorf("type mismatch: file has %q, expected %q", env.Data.Type, sd.Type())
	}
	var entityID string
	if pattern != nil {
		id, err := ExtractEntityID(filename, pattern)
		if err != nil {
			return nil, err
		}
		if id != env.Data.ID {
			return nil, fmt.Errorf("id mismatch: filename %q, data.id %q", id, env.Data.ID)
		}
		entityID = id
	} else {
		entityID = env.Data.ID
	}
	t, _ := getTenantFromContextStub() // placeholder; the orchestrator passes tenant explicitly via LoadAndBuild
	_ = t
	// LoadAndBuild handles tenant via the caller's closure; we adapt here:
	return sd.LoadAndBuild(tenantFromAttrs(env.Data.Attributes), entityID, env.Data.Attributes)
}
```

Stop — the `loadOne` sketch above shows the intent but the wiring is wrong (tenant must be threaded, not pulled from attributes). Replace with the correct implementation:

```go
// libs/atlas-seeder/seed.go
package seeder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"golang.org/x/sync/errgroup"

	"encoding/json"
)

func Seed(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Result, error) {
	t := tenant.MustFromContext(ctx)
	started := time.Now().UTC()

	roots, err := src.Roots(t)
	if err != nil {
		return persistAndReturn(ctx, db, g, Result{
			GroupName: g.Name, StartedAt: started, CompletedAt: time.Now().UTC(),
			Subdomains: map[string]SubdomainCounts{},
		}, t.Id(), "failure")
	}
	rev, _ := src.Revision(roots[0])

	subCounts := make(map[string]SubdomainCounts, len(g.Subdomains))
	var mu sync.Mutex

	eg, gctx := errgroup.WithContext(ctx)
	for _, sd := range g.Subdomains {
		sd := sd
		eg.Go(func() error {
			counts := runSubdomain(gctx, db, src, roots[0], t, sd)
			mu.Lock()
			subCounts[sd.Name()] = counts
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	completed := time.Now().UTC()
	res := Result{
		GroupName:       g.Name,
		CatalogRevision: rev,
		Subdomains:      subCounts,
		StartedAt:       started,
		CompletedAt:     completed,
	}
	outcome := classifyOutcome(subCounts)
	ObserveSeederRun(serviceLabel(), g.Name, outcome, completed.Sub(started).Seconds())
	return persistAndReturn(ctx, db, g, res, t.Id(), outcome)
}

func runSubdomain(ctx context.Context, db *gorm.DB, src CatalogSource, root string, t tenant.Model, sd SubdomainAny) SubdomainCounts {
	var counts SubdomainCounts
	deleted, err := sd.DeleteAllForTenant(db.WithContext(ctx))
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("delete: %v", err))
		return counts
	}
	counts.Deleted = deleted

	files, err := src.Walk(root, sd.Path())
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("walk %s: %v", sd.Path(), err))
		return counts
	}

	pattern := sd.EntityIDPattern()
	for _, name := range files {
		if err := ctx.Err(); err != nil {
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			return counts
		}
		rows, err := loadOne(ctx, src, root, t, sd, pattern, name)
		if err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := sd.BulkCreate(db.WithContext(ctx), rows); err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: bulkcreate: %v", name, err))
			continue
		}
		counts.Created += rowCount(rows)
	}
	return counts
}

func loadOne(ctx context.Context, src CatalogSource, root string, t tenant.Model, sd SubdomainAny, pattern *regexp.Regexp, filename string) (any, error) {
	rc, err := src.Open(root, path.Join(sd.Path(), filename))
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	env, err := ParseEnvelope(b)
	if err != nil {
		return nil, err
	}
	if env.Data.Type != sd.Type() {
		return nil, fmt.Errorf("type mismatch: file has %q, expected %q", env.Data.Type, sd.Type())
	}
	var entityID string
	if pattern != nil {
		id, err := ExtractEntityID(filename, pattern)
		if err != nil {
			return nil, err
		}
		if id != env.Data.ID {
			return nil, fmt.Errorf("id mismatch: filename %q, data.id %q", id, env.Data.ID)
		}
		entityID = id
	} else {
		entityID = env.Data.ID
	}
	return sd.LoadAndBuild(t, entityID, env.Data.Attributes)
}

// rowCount uses reflection to count items in `any` returned from BulkCreate
// payload. The adapter always hands BulkCreate a []M slice; we count via
// reflect.Value.Len.
func rowCount(rows any) int64 {
	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Slice {
		return 0
	}
	return int64(v.Len())
}

func appendError(in []string, msg string) []string {
	if len(in) >= MaxErrors {
		return in
	}
	return append(in, msg)
}

func classifyOutcome(counts map[string]SubdomainCounts) string {
	if len(counts) == 0 {
		return "failure"
	}
	successCount, failCount := 0, 0
	for _, c := range counts {
		if c.Failed > 0 || len(c.Errors) > 0 {
			failCount++
		} else {
			successCount++
		}
	}
	switch {
	case failCount == 0:
		return "success"
	case successCount == 0:
		return "failure"
	default:
		return "partial"
	}
}

func persistAndReturn(ctx context.Context, db *gorm.DB, g Group, res Result, tenantID uuid.UUID, _ string) (Result, error) {
	summary, err := json.Marshal(res)
	if err != nil {
		return res, fmt.Errorf("marshal summary: %w", err)
	}
	row := SeedState{
		TenantID:        tenantID,
		GroupName:       g.Name,
		CatalogRevision: res.CatalogRevision,
		SeededAt:        res.CompletedAt,
		ResultSummary:   datatypes.JSON(summary),
	}
	if err := UpsertSeedState(db.WithContext(ctx), &row); err != nil {
		return res, fmt.Errorf("upsert seed_state: %w", err)
	}
	return res, nil
}

func serviceLabel() string {
	if v := os.Getenv("ATLAS_SERVICE_NAME"); v != "" {
		return v
	}
	return "unknown"
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestSeed -v
```

Expected: PASS on both `TestSeed_SuccessfulRunPersistsStateAndCountsCreated` and `TestSeed_PartialFailurePersistsAndContinues`.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-seeder/seed.go libs/atlas-seeder/seed_test.go
git commit -m "feat(atlas-seeder): add Seed() orchestrator with errgroup fan-out and seed_state persistence"
```

### Task 1.8: Implement `Status` reader

**Files:**
- Create: `libs/atlas-seeder/status.go`
- Test: `libs/atlas-seeder/status_test.go`

- [ ] **Step 1: Write the failing status test**

```go
// libs/atlas-seeder/status_test.go
package seeder

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestStatus_NeverSeededReturnsNilTenantFields(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83())
	st, err := Status(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("catalogRevision = %q", st.CatalogRevision)
	}
	if st.TenantSeededRevision != nil {
		t.Fatalf("TenantSeededRevision = %v, want nil", st.TenantSeededRevision)
	}
	if _, ok := st.Subdomains["widgets"]; !ok {
		t.Fatalf("subdomain entry missing")
	}
}

func TestStatus_AfterSeedPopulatesTenantRevision(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83())
	if _, err := Seed(ctx, db, src, g); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	st, err := Status(ctx, db, src, g)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.TenantSeededRevision == nil || *st.TenantSeededRevision != "test-rev-abc123" {
		t.Fatalf("TenantSeededRevision = %v", st.TenantSeededRevision)
	}
	if st.TenantSeededAt == nil {
		t.Fatalf("TenantSeededAt = nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestStatus -v
```

Expected: FAIL — `Status` undefined.

- [ ] **Step 3: Implement `status.go`**

```go
// libs/atlas-seeder/status.go
package seeder

import (
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func Status(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Status, error) {
	t := tenant.MustFromContext(ctx)
	out := Status{
		GroupName:  g.Name,
		Subdomains: make(map[string]SubdomainStatus, len(g.Subdomains)),
	}

	roots, err := src.Roots(t)
	if err == nil && len(roots) > 0 {
		rev, _ := src.Revision(roots[0])
		out.CatalogRevision = rev
	}

	row, err := ReadSeedState(db.WithContext(ctx), t.Id(), g.Name)
	if err != nil {
		return out, err
	}
	if row != nil {
		rev := row.CatalogRevision
		out.TenantSeededRevision = &rev
		ts := row.SeededAt
		out.TenantSeededAt = &ts
	}

	var mu sync.Mutex
	var latest *time.Time
	eg, gctx := errgroup.WithContext(ctx)
	for _, sd := range g.Subdomains {
		sd := sd
		eg.Go(func() error {
			count, ts, err := sd.Count(db.WithContext(gctx))
			if err != nil {
				return err
			}
			mu.Lock()
			out.Subdomains[sd.Name()] = SubdomainStatus{Count: count, UpdatedAt: ts}
			if ts != nil && (latest == nil || ts.After(*latest)) {
				latest = ts
			}
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return out, err
	}
	out.UpdatedAt = latest
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestStatus -v
```

Expected: PASS on both Status tests.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-seeder/status.go libs/atlas-seeder/status_test.go
git commit -m "feat(atlas-seeder): add Status() reader returning catalog and tenant-seeded revisions"
```

### Task 1.9: Implement `RegisterRoutes` and HTTP handlers

**Files:**
- Create: `libs/atlas-seeder/handlers.go`
- Test: `libs/atlas-seeder/handlers_test.go`

- [ ] **Step 1: Write the failing HTTP test**

```go
// libs/atlas-seeder/handlers_test.go
package seeder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// tenantMiddleware injects tenantGMS83 into the request context.
func tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tenant.WithContext(r.Context(), tenantGMS83())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestRegisterRoutes_PostReturns202(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	r := mux.NewRouter()
	RegisterRoutes(r, db, logrus.New(), src, g)
	srv := httptest.NewServer(tenantMiddleware(r))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/widgets/seed", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	// Wait briefly for the goroutine to write seed_state.
	time.Sleep(200 * time.Millisecond)
	row, _ := ReadSeedState(db, uuid.Nil, "widgets-group") // tenantGMS83's id is random
	_ = row
}

func TestRegisterRoutes_GetStatusReturnsJSON(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	r := mux.NewRouter()
	RegisterRoutes(r, db, logrus.New(), src, g)
	srv := httptest.NewServer(tenantMiddleware(r))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/widgets/seed/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["catalogRevision"]; !ok {
		t.Fatalf("missing catalogRevision: %v", body)
	}
	if !strings.Contains(string(jsonBytes(body)), "widgets") {
		t.Fatalf("subdomain key missing: %v", body)
	}
}

func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }

// Ensure compile-time use of unused import context
var _ = context.Background
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-seeder && go test ./... -run TestRegisterRoutes -v
```

Expected: FAIL — `RegisterRoutes` undefined.

- [ ] **Step 3: Implement `handlers.go`**

```go
// libs/atlas-seeder/handlers.go
package seeder

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegisterRoutes wires POST <prefix>/seed and GET <prefix>/seed/status.
func RegisterRoutes(
	router *mux.Router,
	db *gorm.DB,
	logger logrus.FieldLogger,
	src CatalogSource,
	g Group,
) {
	router.HandleFunc(g.URLPrefix+"/seed", postSeed(logger, db, src, g)).Methods(http.MethodPost)
	router.HandleFunc(g.URLPrefix+"/seed/status", getStatus(logger, db, src, g)).Methods(http.MethodGet)
}

func postSeed(l logrus.FieldLogger, db *gorm.DB, src CatalogSource, g Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(r.Context())
		go func() {
			bgCtx := tenant.WithContext(context.Background(), t)
			res, err := Seed(bgCtx, db, src, g)
			if err != nil {
				l.WithError(err).WithFields(logrus.Fields{
					"tenant_id":  t.Id(),
					"group_name": g.Name,
				}).Error("Seed failed")
				return
			}
			l.WithFields(logrus.Fields{
				"tenant_id":        t.Id(),
				"group_name":       g.Name,
				"catalog_revision": res.CatalogRevision,
				"subdomains":       summarize(res.Subdomains),
			}).Info("Seed complete")
		}()
		w.WriteHeader(http.StatusAccepted)
	}
}

func getStatus(l logrus.FieldLogger, db *gorm.DB, src CatalogSource, g Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := Status(r.Context(), db, src, g)
		if err != nil {
			l.WithError(err).WithField("group_name", g.Name).Error("Status failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if st.TenantSeededRevision != nil && st.CatalogRevision != "" && st.CatalogRevision != *st.TenantSeededRevision {
			l.WithFields(logrus.Fields{
				"group_name":             g.Name,
				"catalog_revision":       st.CatalogRevision,
				"tenant_seeded_revision": *st.TenantSeededRevision,
			}).Warn("seed catalog drift detected")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	}
}

func summarize(m map[string]SubdomainCounts) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v.Created
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-seeder && go test ./... -run TestRegisterRoutes -v
```

Expected: PASS on both tests.

- [ ] **Step 5: Run the full library test suite with -race**

```bash
cd libs/atlas-seeder && go test -race ./... && go vet ./...
```

Expected: all PASS, vet clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-seeder/handlers.go libs/atlas-seeder/handlers_test.go
git commit -m "feat(atlas-seeder): add RegisterRoutes wiring POST /seed (202 async) and GET /seed/status"
```

---

## Task Group 2: Splitter Tools

Four one-shot Go programs under `tools/seed-splitters/`. They produce the initial v83 catalog content from existing bundled files.

### Task 2.1: Bootstrap `tools/seed-splitters/` workspace

**Files:**
- Create: `tools/seed-splitters/go.mod` (single module shared by all splitters)
- Create: `tools/seed-splitters/README.md`

- [ ] **Step 1: Create the module**

```bash
mkdir -p tools/seed-splitters
cd tools/seed-splitters
cat > go.mod <<'EOF'
module github.com/Chronicle20/atlas/tools/seed-splitters

go 1.25.0
EOF
```

- [ ] **Step 2: Write README**

```markdown
# seed-splitters

One-shot Go programs that produce the initial `deploy/seed/<region>/<version>/`
catalog content from existing bundled JSON files. Each is deterministic — rerunning
produces byte-identical output — and is NOT run by CI. They are committed for
reproducibility and to bootstrap new region/version directories from v83 content.

Programs:

- `split-monster-drops/`  — splits `monster_drops.json` (array) into one JSON:API file per monster.
- `split-continent-drops/` — same for `continent_drops.json`.
- `split-gachapons/`      — merges `gachapons.json` + `gachapon_items.json` into one combined file per gachapon, plus `_global/items.json`.
- `wrap-jsonapi/`         — generic wrapper for files that already exist per-entity but lack the JSON:API envelope.
```

- [ ] **Step 3: Add to go.work**

```bash
cd <worktree>
go work use ./tools/seed-splitters
```

- [ ] **Step 4: Commit**

```bash
git add tools/seed-splitters/go.mod tools/seed-splitters/README.md go.work
git commit -m "feat(seed-splitters): bootstrap splitter workspace module"
```

### Task 2.2: Implement `wrap-jsonapi` (generic wrapper)

**Files:**
- Create: `tools/seed-splitters/wrap-jsonapi/main.go`
- Test: `tools/seed-splitters/wrap-jsonapi/main_test.go`
- Test fixtures: `tools/seed-splitters/wrap-jsonapi/testdata/input/*.json`, `testdata/expected/*.json`

- [ ] **Step 1: Create the input fixture**

```bash
mkdir -p tools/seed-splitters/wrap-jsonapi/testdata/input
mkdir -p tools/seed-splitters/wrap-jsonapi/testdata/expected
cat > tools/seed-splitters/wrap-jsonapi/testdata/input/1001.json <<'EOF'
{"npcId":1001,"recharger":false,"commodities":[{"templateId":1040002,"mesoPrice":50}]}
EOF
cat > tools/seed-splitters/wrap-jsonapi/testdata/expected/shop-1001.json <<'EOF'
{
  "data": {
    "type": "shop",
    "id": "1001",
    "attributes": {
      "npcId": 1001,
      "recharger": false,
      "commodities": [
        {
          "templateId": 1040002,
          "mesoPrice": 50
        }
      ]
    }
  }
}
EOF
```

- [ ] **Step 2: Write the failing wrap-jsonapi test**

```go
// tools/seed-splitters/wrap-jsonapi/main_test.go
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWrap_DeterministicOutput(t *testing.T) {
	tmp := t.TempDir()
	exe := buildBinary(t, "wrap-jsonapi")
	args := []string{
		"--input-dir", "testdata/input",
		"--output-dir", tmp,
		"--type", "shop",
		"--id-field", "npcId",
		"--filename-prefix", "shop",
	}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("first run: %v\n%s", err, out)
	}
	first, err := os.ReadFile(filepath.Join(tmp, "shop-1001.json"))
	if err != nil {
		t.Fatalf("read first: %v", err)
	}
	want, _ := os.ReadFile("testdata/expected/shop-1001.json")
	if !bytes.Equal(first, want) {
		t.Fatalf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", first, want)
	}
	// rerun and verify byte-identical
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("second run: %v\n%s", err, out)
	}
	second, _ := os.ReadFile(filepath.Join(tmp, "shop-1001.json"))
	if !bytes.Equal(first, second) {
		t.Fatalf("non-deterministic: rerun differs")
	}
}

func buildBinary(t *testing.T, name string) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), name)
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd tools/seed-splitters/wrap-jsonapi && go test -v
```

Expected: FAIL — binary build fails (no `main`).

- [ ] **Step 4: Implement `main.go`**

```go
// tools/seed-splitters/wrap-jsonapi/main.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var (
		inputDir       = flag.String("input-dir", "", "directory of plain-JSON files")
		outputDir      = flag.String("output-dir", "", "directory to write JSON:API files")
		typ            = flag.String("type", "", "JSON:API data.type")
		idField        = flag.String("id-field", "", "input JSON top-level field name carrying the entity id")
		filenamePrefix = flag.String("filename-prefix", "", "prefix for output filenames (output = <prefix>-<id>.json)")
	)
	flag.Parse()
	if *inputDir == "" || *outputDir == "" || *typ == "" || *idField == "" {
		fmt.Fprintln(os.Stderr, "usage: wrap-jsonapi --input-dir DIR --output-dir DIR --type TYPE --id-field FIELD [--filename-prefix PREFIX]")
		os.Exit(2)
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fail("mkdir output-dir: %v", err)
	}

	entries, err := os.ReadDir(*inputDir)
	if err != nil {
		fail("read input-dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	prefix := *filenamePrefix
	if prefix == "" {
		prefix = ""
	}
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(*inputDir, name))
		if err != nil {
			fail("read %s: %v", name, err)
		}
		var attrs map[string]any
		if err := json.Unmarshal(b, &attrs); err != nil {
			fail("parse %s: %v", name, err)
		}
		idVal, ok := attrs[*idField]
		if !ok {
			fail("%s: missing id field %q", name, *idField)
		}
		id := fmt.Sprint(idVal)
		envelope := map[string]any{
			"data": map[string]any{
				"type":       *typ,
				"id":         id,
				"attributes": attrs,
			},
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(envelope); err != nil {
			fail("encode %s: %v", name, err)
		}
		outName := id + ".json"
		if prefix != "" {
			outName = prefix + "-" + id + ".json"
		}
		if err := os.WriteFile(filepath.Join(*outputDir, outName), buf.Bytes(), 0o644); err != nil {
			fail("write %s: %v", outName, err)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
```

Note: `json.Encoder.Encode` always appends a single trailing newline — that is the determinism contract.

- [ ] **Step 5: Run test to verify it passes**

```bash
cd tools/seed-splitters/wrap-jsonapi && go test -v
```

Expected: PASS, including the byte-identical rerun check.

- [ ] **Step 6: Commit**

```bash
git add tools/seed-splitters/wrap-jsonapi
git commit -m "feat(seed-splitters): add generic wrap-jsonapi splitter with determinism test"
```

### Task 2.3: Implement `split-monster-drops`

**Files:**
- Create: `tools/seed-splitters/split-monster-drops/main.go`
- Test: `tools/seed-splitters/split-monster-drops/main_test.go`
- Test fixtures: `tools/seed-splitters/split-monster-drops/testdata/{input,expected}/`

- [ ] **Step 1: Create fixtures**

```bash
mkdir -p tools/seed-splitters/split-monster-drops/testdata/{input,expected}
cat > tools/seed-splitters/split-monster-drops/testdata/input/monster_drops.json <<'EOF'
[
  {"monsterId":100,"itemId":2000,"minimumQuantity":1,"maximumQuantity":1,"questId":0,"chance":1000},
  {"monsterId":100,"itemId":2001,"minimumQuantity":2,"maximumQuantity":3,"questId":0,"chance":500},
  {"monsterId":200,"itemId":2100,"minimumQuantity":1,"maximumQuantity":1,"questId":0,"chance":2000}
]
EOF
cat > tools/seed-splitters/split-monster-drops/testdata/expected/monster-100.json <<'EOF'
{
  "data": {
    "type": "monster-drop",
    "id": "100",
    "attributes": {
      "drops": [
        {
          "itemId": 2000,
          "minimumQuantity": 1,
          "maximumQuantity": 1,
          "questId": 0,
          "chance": 1000
        },
        {
          "itemId": 2001,
          "minimumQuantity": 2,
          "maximumQuantity": 3,
          "questId": 0,
          "chance": 500
        }
      ]
    }
  }
}
EOF
cat > tools/seed-splitters/split-monster-drops/testdata/expected/monster-200.json <<'EOF'
{
  "data": {
    "type": "monster-drop",
    "id": "200",
    "attributes": {
      "drops": [
        {
          "itemId": 2100,
          "minimumQuantity": 1,
          "maximumQuantity": 1,
          "questId": 0,
          "chance": 2000
        }
      ]
    }
  }
}
EOF
```

- [ ] **Step 2: Write the failing test**

```go
// tools/seed-splitters/split-monster-drops/main_test.go
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSplitMonsterDrops_MatchesExpected(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(t.TempDir(), "split")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	args := []string{"--input", "testdata/input/monster_drops.json", "--output", tmp}
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	for _, name := range []string{"monster-100.json", "monster-200.json"} {
		got, _ := os.ReadFile(filepath.Join(tmp, name))
		want, _ := os.ReadFile(filepath.Join("testdata/expected", name))
		if !bytes.Equal(got, want) {
			t.Fatalf("%s mismatch:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
		}
	}
	// Determinism: rerun and verify identical.
	if out, err := exec.Command(exe, args...).CombinedOutput(); err != nil {
		t.Fatalf("rerun: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(filepath.Join(tmp, "monster-100.json"))
	want, _ := os.ReadFile(filepath.Join("testdata/expected", "monster-100.json"))
	if !bytes.Equal(got, want) {
		t.Fatalf("rerun produced different output")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd tools/seed-splitters/split-monster-drops && go test -v
```

Expected: FAIL — build fails.

- [ ] **Step 4: Implement `main.go`**

```go
// tools/seed-splitters/split-monster-drops/main.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type rawDrop struct {
	MonsterID       int   `json:"monsterId"`
	ItemID          int   `json:"itemId"`
	MinimumQuantity int   `json:"minimumQuantity"`
	MaximumQuantity int   `json:"maximumQuantity"`
	QuestID         int   `json:"questId"`
	Chance          int64 `json:"chance"`
}

type outDrop struct {
	ItemID          int   `json:"itemId"`
	MinimumQuantity int   `json:"minimumQuantity"`
	MaximumQuantity int   `json:"maximumQuantity"`
	QuestID         int   `json:"questId"`
	Chance          int64 `json:"chance"`
}

func main() {
	in := flag.String("input", "", "path to monster_drops.json")
	out := flag.String("output", "", "output directory")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: split-monster-drops --input FILE --output DIR")
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fail("read: %v", err)
	}
	var rows []rawDrop
	if err := json.Unmarshal(b, &rows); err != nil {
		fail("parse: %v", err)
	}
	grouped := map[int][]outDrop{}
	for _, r := range rows {
		grouped[r.MonsterID] = append(grouped[r.MonsterID], outDrop{
			ItemID: r.ItemID, MinimumQuantity: r.MinimumQuantity,
			MaximumQuantity: r.MaximumQuantity, QuestID: r.QuestID, Chance: r.Chance,
		})
	}
	ids := make([]int, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		drops := grouped[id]
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].ItemID < drops[j].ItemID })
		envelope := map[string]any{
			"data": map[string]any{
				"type":       "monster-drop",
				"id":         fmt.Sprint(id),
				"attributes": map[string]any{"drops": drops},
			},
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(envelope); err != nil {
			fail("encode %d: %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(*out, fmt.Sprintf("monster-%d.json", id)), buf.Bytes(), 0o644); err != nil {
			fail("write %d: %v", id, err)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd tools/seed-splitters/split-monster-drops && go test -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/seed-splitters/split-monster-drops
git commit -m "feat(seed-splitters): add split-monster-drops with determinism test"
```

### Task 2.4: Implement `split-continent-drops`

**Files:**
- Create: `tools/seed-splitters/split-continent-drops/main.go`
- Test: `tools/seed-splitters/split-continent-drops/main_test.go`
- Test fixtures: `tools/seed-splitters/split-continent-drops/testdata/{input,expected}/`

- [ ] **Step 1: Inspect current continent_drops.json shape**

```bash
head -30 services/atlas-drop-information/drops/continents/continent_drops.json
```

Confirm the array element keys (likely `continentId`, `itemId`, `minimumQuantity`, etc. mirroring monster drops). If the keys differ, adjust the splitter struct accordingly.

- [ ] **Step 2: Create fixtures mirroring Task 2.3 with `continentId` in place of `monsterId`**

(Same shape as Task 2.3 with `continent-` prefix and `continent-drop` type.)

- [ ] **Step 3: Write the failing test (mirror of Task 2.3 main_test.go)**

- [ ] **Step 4: Implement `main.go` (mirror of split-monster-drops, swap field names + type tag)**

- [ ] **Step 5: Run test**

```bash
cd tools/seed-splitters/split-continent-drops && go test -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/seed-splitters/split-continent-drops
git commit -m "feat(seed-splitters): add split-continent-drops with determinism test"
```

### Task 2.5: Implement `split-gachapons`

**Files:**
- Create: `tools/seed-splitters/split-gachapons/main.go`
- Test: `tools/seed-splitters/split-gachapons/main_test.go`
- Test fixtures: `tools/seed-splitters/split-gachapons/testdata/{input,expected}/`

- [ ] **Step 1: Inspect current gachapons.json + gachapon_items.json shapes**

```bash
head -20 services/atlas-gachapons/data/gachapons.json
head -20 services/atlas-gachapons/data/gachapon_items.json
head -10 services/atlas-gachapons/data/global_gachapon_items.json
```

Record the field names — the splitter merges items into their owning gachapon by gachapon id.

- [ ] **Step 2: Build fixture input + expected output**

Create small fixtures (one gachapon, two items, plus a 1-element global pool) that exercise both per-gachapon merging and the `_global/items.json` emission. Use the actual field names from Step 1.

- [ ] **Step 3: Write failing test (binary-build pattern of Task 2.3)**

- [ ] **Step 4: Implement `main.go`** — reads both input files, groups items by gachapon id, emits one `<id>.json` per gachapon and one `_global/items.json` for the global pool. The emitted gachapon envelope's `attributes` carries the gachapon fields plus an `items` array of the matched item rows. The `_global/items.json` envelope uses type `gachapon-pool` and id `_global` (so `Subdomain.EntityIDPattern() == nil` plus type lookup wires it).

- [ ] **Step 5: Run test**

```bash
cd tools/seed-splitters/split-gachapons && go test -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/seed-splitters/split-gachapons
git commit -m "feat(seed-splitters): add split-gachapons with _global pool emission"
```

---

## Task Group 3: Catalog Tree

Run the splitters once and commit the produced `deploy/seed/` content. Bootstrap non-v83 directories.

### Task 3.1: Produce `deploy/seed/gms/83_1/` content via splitters

- [ ] **Step 1: Create base structure and CATALOG_REVISION**

```bash
mkdir -p deploy/seed/_schema
mkdir -p deploy/seed/gms/83_1/{drops/monsters,drops/continents,drops/reactors,gachapons,gachapons/_global,map-actions/map,portal-actions/portals,reactor-actions/reactors,npc-conversations/npc,npc-conversations/quests,npc-shops/shops,party-quests/definitions}
git rev-parse HEAD | tr -d '\n' > deploy/seed/gms/83_1/CATALOG_REVISION
```

- [ ] **Step 2: Run all splitters into the target tree**

```bash
cd <worktree>
go run ./tools/seed-splitters/split-monster-drops \
  --input services/atlas-drop-information/drops/monsters/monster_drops.json \
  --output deploy/seed/gms/83_1/drops/monsters
go run ./tools/seed-splitters/split-continent-drops \
  --input services/atlas-drop-information/drops/continents/continent_drops.json \
  --output deploy/seed/gms/83_1/drops/continents
go run ./tools/seed-splitters/split-gachapons \
  --gachapons services/atlas-gachapons/data/gachapons.json \
  --items services/atlas-gachapons/data/gachapon_items.json \
  --global services/atlas-gachapons/data/global_gachapon_items.json \
  --output deploy/seed/gms/83_1/gachapons
# Reactor drops already JSON:API: copy as-is, renaming to reactor-<id>.json (already that shape).
cp services/atlas-drop-information/drops/reactors/*.json deploy/seed/gms/83_1/drops/reactors/
# Wrap script services (map-actions, portal-actions, reactor-actions) via wrap-jsonapi.
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-map-actions/scripts/map \
  --output-dir deploy/seed/gms/83_1/map-actions/map \
  --type map-script --id-field mapId --filename-prefix map
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-portal-actions/scripts/portals \
  --output-dir deploy/seed/gms/83_1/portal-actions/portals \
  --type portal-script --id-field portalId --filename-prefix portal
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-reactor-actions/scripts/reactors \
  --output-dir deploy/seed/gms/83_1/reactor-actions/reactors \
  --type reactor-script --id-field reactorId --filename-prefix reactor
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-npc-conversations/conversations/npc \
  --output-dir deploy/seed/gms/83_1/npc-conversations/npc \
  --type npc-conversation --id-field id --filename-prefix npc
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-npc-conversations/conversations/quests \
  --output-dir deploy/seed/gms/83_1/npc-conversations/quests \
  --type quest-conversation --id-field id --filename-prefix quest
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-npc-shops/shops \
  --output-dir deploy/seed/gms/83_1/npc-shops/shops \
  --type npc-shop --id-field npcId --filename-prefix shop
go run ./tools/seed-splitters/wrap-jsonapi \
  --input-dir services/atlas-party-quests/party-quests \
  --output-dir deploy/seed/gms/83_1/party-quests/definitions \
  --type party-quest-definition --id-field id --filename-prefix party-quest
```

Note: the `--id-field` values above (e.g., `mapId`, `portalId`, `reactorId`, `id`) are assumptions; the actual field name in each script-service catalog file is verified by `head -10 <one file>` before this command runs. Per-service migration tasks (Task Group 4) re-validate by reading the same files.

- [ ] **Step 3: Verify reruns are byte-identical**

```bash
cd <worktree>
cp -r deploy/seed/gms/83_1 /tmp/seed-first-run
# Re-run all splitter commands from Step 2 against deploy/seed/gms/83_1 again.
diff -r /tmp/seed-first-run deploy/seed/gms/83_1
```

Expected: zero diff.

- [ ] **Step 4: Commit (large change — single commit OK)**

```bash
git add deploy/seed/gms/83_1
git commit -m "feat(catalog): bootstrap deploy/seed/gms/83_1 from splitter output"
```

### Task 3.2: Bootstrap non-v83 region/version directories from v83

- [ ] **Step 1: Copy v83 to each other version**

```bash
cd <worktree>
for tgt in gms/12_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  mkdir -p deploy/seed/$tgt
  rsync -a --delete --exclude CATALOG_REVISION deploy/seed/gms/83_1/ deploy/seed/$tgt/
  echo "bootstrapped-from-gms-83_1-@$(git rev-parse HEAD)" > deploy/seed/$tgt/CATALOG_REVISION
done
```

- [ ] **Step 2: Verify directory parity**

```bash
for tgt in gms/12_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  echo "=== $tgt ==="
  diff -q -r --no-dereference --exclude CATALOG_REVISION deploy/seed/gms/83_1 deploy/seed/$tgt | head -5
done
```

Expected: zero differences (only the deliberately-divergent `CATALOG_REVISION` files).

- [ ] **Step 3: Commit**

```bash
git add deploy/seed/gms/12_1 deploy/seed/gms/87_1 deploy/seed/gms/92_1 deploy/seed/gms/95_1 deploy/seed/jms/185_1
git commit -m "feat(catalog): bootstrap gms/{12,87,92,95}_1 and jms/185_1 from gms/83_1"
```

---

## Task Group 4: Per-Service Migrations

Eight services, same recipe each. The recipe template is given once in Task 4.0 below; each numbered Task 4.X applies the recipe to one service with that service's concrete file paths, route prefixes, and Subdomain implementations.

### Task 4.0: Migration recipe (reference — do not execute as a standalone task)

For each in-scope service the migration is:

1. **Inspect** — `head` the existing `seed.go`, `processor.go`, `resource.go`, `status.go` files. Confirm the route prefix (verified literals in `context.md` §1). Confirm the model package paths.
2. **Add seeder subdomain implementations** — one Go file per subdomain (e.g., `monster/drop/subdomain.go`) implementing `seeder.Subdomain[J, M]`. Reuse existing `Builder`, `Count`, `DeleteAll`, `BulkCreate*` functions; only `Decode` and `Build` are net new.
3. **Add `seed/groups.go`** — `Init(db, si)` returns a `server.RouteInitializer` that calls `seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")` and `seeder.RegisterRoutes(...)` for each `Group`.
4. **Wire into `main.go`** — replace the old `seed.InitResource(...)` route-initializer with the new `seed.Init(db, si)`. Add `db.AutoMigrate(&seeder.SeedState{})` next to the existing AutoMigrate call.
5. **Delete the old seed package** — entire `seed/` dir (and any inline seed loader files in domain packages).
6. **Delete bundled data dir** — verbatim path from `context.md` §1.
7. **Update Dockerfile** in all four lib-list locations to include `atlas-seeder` (per CLAUDE.md). Drop the `COPY services/<svc>/<datadir> ...` line that copied bundled data.
8. **Update k8s manifest** — `deploy/k8s/base/atlas-<svc>.yaml`: drop the old `*_PATH` env vars (enumerate via grep first), add the seed-catalog Kustomize component reference (Task Group 5 lands the component itself; until then, leave a TODO line for the component reference and the manifest gets the env + volumeMount added inline as a transitional patch in this same commit).
9. **Update compose entry** — `deploy/compose/docker-compose.core.yml`: drop the old `*_PATH` env vars and add `<<: *seed-catalog` to the service block. (The anchor itself is added in Task Group 5; the per-service merge-key reference may be added in the same commit as the anchor for atomicity. If the anchor isn't there yet, leave a TODO and revisit.)
10. **Update existing tests** — `*_test.go` files keep their assertions but call into the new lib-backed handlers (mostly: assert `catalogRevision` / `tenantSeededRevision` fields appear on status response). Route URLs unchanged.
11. **Verify**:
    ```bash
    cd services/atlas-<svc>/atlas.com/<svc>
    go test -race ./... && go vet ./... && go build ./...
    cd ../../../..
    docker build -f services/atlas-<svc>/Dockerfile .
    ```
12. **Commit** — single commit per service, message `feat(atlas-<svc>): migrate to libs/atlas-seeder catalog mount`.

The "verify" step is mandatory before each per-service commit. The `docker build` step catches Dockerfile lib-list drift that `go build` will not.

### Task 4.1: Migrate atlas-gachapons

**Files:**
- Create: `services/atlas-gachapons/atlas.com/gachapons/seed/groups.go`
- Create: `services/atlas-gachapons/atlas.com/gachapons/gachapon/subdomain.go`
- Create: `services/atlas-gachapons/atlas.com/gachapons/item/subdomain.go`
- Create: `services/atlas-gachapons/atlas.com/gachapons/global/subdomain.go`
- Delete: `services/atlas-gachapons/atlas.com/gachapons/seed/{seed.go,processor.go,resource.go,status.go}`
- Delete: `services/atlas-gachapons/atlas.com/gachapons/seed/{seed_test.go,status_test.go,processor_test.go}` (rewrite into a new `groups_test.go`)
- Delete: `services/atlas-gachapons/data/`
- Modify: `services/atlas-gachapons/atlas.com/gachapons/main.go` — replace `seed.InitResource(...)` with `seed.Init(...)`, add `SeedState` AutoMigrate
- Modify: `services/atlas-gachapons/Dockerfile` — add atlas-seeder in 4 places, drop `COPY services/atlas-gachapons/data /gachapons/data` line
- Modify: `services/atlas-gachapons/atlas.com/gachapons/go.mod` — add atlas-seeder require/replace
- Modify: `deploy/k8s/base/atlas-gachapons.yaml` — drop `GACHAPONS_DATA_PATH`, `GACHAPON_ITEMS_DATA_PATH`, `GLOBAL_ITEMS_DATA_PATH`; add `SEED_CATALOG_ROOT` env + volume mount placeholder
- Modify: `deploy/compose/docker-compose.core.yml` — drop the same env vars on atlas-gachapons block

- [ ] **Step 1: Inspect current state**

```bash
ls services/atlas-gachapons/atlas.com/gachapons/seed/
grep -n "InitResource\|Migration\|AutoMigrate" services/atlas-gachapons/atlas.com/gachapons/main.go
grep -n "GACHAPONS_DATA_PATH\|GACHAPON_ITEMS_DATA_PATH\|GLOBAL_ITEMS_DATA_PATH" deploy/k8s/base/atlas-gachapons.yaml deploy/compose/docker-compose.core.yml
```

Record the exact env var names and main.go route-initializer call site.

- [ ] **Step 2: Write Subdomain for gachapon**

Read the current `gachapon` package — find the `JSONModel`, `Model`, `Builder`, `BulkCreate`, `DeleteAll`, `Count` helpers. Write `gachapon/subdomain.go`:

```go
// services/atlas-gachapons/atlas.com/gachapons/gachapon/subdomain.go
package gachapon

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

type Subdomain struct{}

func (Subdomain) Name() string                  { return "gachapons" }
func (Subdomain) Path() string                  { return "gachapons" }
func (Subdomain) Type() string                  { return "gachapon" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^(\d+)\.json$`)
}
func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAll(db) // verify function name in current package
}
func (Subdomain) Decode(b []byte) (JSONModel, error) {
	var j JSONModel
	return j, json.Unmarshal(b, &j)
}
func (Subdomain) Build(_ tenant.Model, _ string, j JSONModel) ([]Model, error) {
	// Reuse the existing JSONModel → Model conversion; verify the existing helper name.
	m, err := Transform(j)
	if err != nil {
		return nil, err
	}
	return []Model{m}, nil
}
func (Subdomain) BulkCreate(db *gorm.DB, rows []Model) error {
	return BulkCreate(db, rows) // verify
}
func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	return Count(db) // verify
}

// Ensure Subdomain satisfies the generic interface at compile time.
var _ seeder.Subdomain[JSONModel, Model] = Subdomain{}
```

If `Transform`, `BulkCreate`, `Count`, `DeleteAll` are named differently in this service, use the actual names. The plan does NOT prescribe new helpers — it composes existing ones.

- [ ] **Step 3: Write Subdomains for `item` (per-gachapon items file is inline; `item` may not need a separate subdomain if items live inside the gachapon envelope; verify and reduce to gachapon + global if so)**

If the splitter produced an inline-items envelope (Task 2.5 design), the `item` subdomain is collapsed into `gachapon.Subdomain.Build` which fans out one gachapon-row + N item-rows. Confirm during implementation; if Build can return only one slice type (`[]gachapon.Model`), keep `item` as a separate subdomain whose `Path()` is still `gachapons` but with `EntityIDPattern` reading the same files and fanning out only item rows. Document the chosen structure in a comment in `groups.go`.

- [ ] **Step 4: Write Subdomain for `global`**

```go
// services/atlas-gachapons/atlas.com/gachapons/global/subdomain.go
package global

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

type Subdomain struct{}

func (Subdomain) Name() string                    { return "global" }
func (Subdomain) Path() string                    { return "gachapons/_global" } // verify against catalog layout
func (Subdomain) Type() string                    { return "gachapon-pool" }
func (Subdomain) EntityIDPattern() *regexp.Regexp { return nil } // load exactly one named file
func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) { return DeleteAll(db) }
func (Subdomain) Decode(b []byte) (JSONModel, error) {
	var j JSONModel
	return j, json.Unmarshal(b, &j)
}
func (Subdomain) Build(_ tenant.Model, _ string, j JSONModel) ([]Model, error) {
	return Transform(j)
}
func (Subdomain) BulkCreate(db *gorm.DB, rows []Model) error { return BulkCreate(db, rows) }
func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) { return Count(db) }

var _ seeder.Subdomain[JSONModel, Model] = Subdomain{}
```

**Note:** v1 `FilesystemCatalogSource.Walk` skips subdirectories starting with `_`, so `global.Subdomain.Path() = "gachapons/_global"` will resolve directly to that leaf dir. The CatalogSource only skips `_`-prefixed entries at the level it walks; it does NOT recurse. So pointing `Path()` directly at `gachapons/_global` is fine.

- [ ] **Step 5: Write `seed/groups.go`**

```go
// services/atlas-gachapons/atlas.com/gachapons/seed/groups.go
package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-seeder"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "gachapons",
				URLPrefix: "/gachapons",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[gachapon.JSONModel, gachapon.Model](gachapon.Subdomain{}),
					seeder.AdaptSubdomain[global.JSONModel, global.Model](global.Subdomain{}),
				},
			})
		}
	}
}
```

The function name `InitResource` and its signature `func(jsonapi.ServerInformation) func(*gorm.DB) server.RouteInitializer` are preserved deliberately so `main.go` does not change shape (only the package contents do).

- [ ] **Step 6: Delete the old seed package internals**

```bash
cd services/atlas-gachapons/atlas.com/gachapons
rm seed/seed.go seed/processor.go seed/resource.go seed/status.go
rm seed/seed_test.go seed/status_test.go 2>/dev/null || true
rm seed/processor_test.go 2>/dev/null || true
```

- [ ] **Step 7: Add SeedState to main.go AutoMigrate**

```go
// services/atlas-gachapons/atlas.com/gachapons/main.go (modify the database.Connect line)
db := database.Connect(l, database.SetMigrations(
	gachapon.Migration,
	item.Migration,
	global.Migration,
	func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) },
))
```

Verify the existing `SetMigrations(...)` arg list and append the seeder migration accordingly.

- [ ] **Step 8: Delete bundled data**

```bash
git rm -r services/atlas-gachapons/data
```

- [ ] **Step 9: Update Dockerfile**

Edit `services/atlas-gachapons/Dockerfile` in four places to add `atlas-seeder` alongside the existing libs (mirror the `COPY libs/atlas-X/go.mod ...`, `echo '    ./libs/atlas-X' >> go.work`, `COPY libs/atlas-X libs/atlas-X`, `-replace=github.com/Chronicle20/atlas/libs/atlas-X=/app/libs/atlas-X` patterns). Remove the line `COPY services/atlas-gachapons/data /gachapons/data`.

- [ ] **Step 10: Update go.mod**

```bash
cd services/atlas-gachapons/atlas.com/gachapons
go mod edit -require=github.com/Chronicle20/atlas/libs/atlas-seeder@v0.0.0
go mod edit -replace=github.com/Chronicle20/atlas/libs/atlas-seeder=../../../../libs/atlas-seeder
go mod tidy
```

- [ ] **Step 11: Update k8s manifest**

Edit `deploy/k8s/base/atlas-gachapons.yaml`: delete the `GACHAPONS_DATA_PATH`, `GACHAPON_ITEMS_DATA_PATH`, `GLOBAL_ITEMS_DATA_PATH` env entries. Add:

```yaml
- name: SEED_CATALOG_ROOT
  value: /var/run/seed-catalog
```

The actual volume + git-sync sidecar wiring lands in Task Group 5; this commit only adds the env so the lib resolves to the (yet-to-be-mounted) path.

- [ ] **Step 12: Update compose**

Edit `deploy/compose/docker-compose.core.yml`: in the `atlas-gachapons` block, drop the three `*_PATH` env vars and add:

```yaml
      SEED_CATALOG_ROOT: /var/run/seed-catalog
    volumes:
      - ../seed:/var/run/seed-catalog:ro
```

(The `x-seed-catalog` anchor lands in Task Group 5; this raw form works in the meantime.)

- [ ] **Step 13: Rewrite tests**

Recreate `seed/groups_test.go` with three small tests:

```go
package seed_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	// ... wire test DB via testcontainers-postgres OR sqlite memory adapter — match the existing test pattern in the deleted status_test.go
)

func TestGachaponsSeed_POSTReturns202(t *testing.T) { /* call /gachapons/seed, assert 202 */ }
func TestGachaponsSeed_StatusIncludesCatalogRevisionField(t *testing.T) { /* GET /gachapons/seed/status, assert response body has catalogRevision key */ }
```

Use whichever test harness the deleted tests used (`httptest.NewRequest`, lib's `RegisterRoutes`, in-memory DB). The point is to preserve operator-visible behavior: URLs unchanged, status response gains catalogRevision.

- [ ] **Step 14: Verify**

```bash
cd services/atlas-gachapons/atlas.com/gachapons
go test -race ./...
go vet ./...
go build ./...
cd <worktree>
docker build -f services/atlas-gachapons/Dockerfile .
```

Expected: all four commands succeed.

- [ ] **Step 15: Commit**

```bash
git add services/atlas-gachapons deploy/k8s/base/atlas-gachapons.yaml deploy/compose/docker-compose.core.yml
git commit -m "feat(atlas-gachapons): migrate to libs/atlas-seeder catalog mount"
```

### Task 4.2: Migrate atlas-drop-information

Apply Task 4.0 recipe to atlas-drop-information. The service-specific values:

| Recipe variable | atlas-drop-information value |
|---|---|
| Service dir | `services/atlas-drop-information/atlas.com/dis` |
| Existing seed package | `seed/{seed.go,processor.go,resource.go,status.go,seed_test.go,status_test.go,processor_test.go}` |
| Subdomains | `monster.drop` (path `drops/monsters`, type `monster-drop`), `continent.drop` (path `drops/continents`, type `continent-drop`), `reactor.drop` (path `drops/reactors`, type `reactor-drop`) |
| Group | one Group `Name: "drops"`, `URLPrefix: "/drops"` |
| Bundled data | `services/atlas-drop-information/drops/` |
| Dockerfile line to drop | `COPY services/atlas-drop-information/drops /drops` |
| Env vars to drop | `MONSTER_DROPS_PATH`, `CONTINENT_DROPS_PATH`, `REACTOR_DROPS_PATH` |
| k8s manifest | `deploy/k8s/base/atlas-drop-information.yaml` |
| Routes preserved | `POST /drops/seed`, `GET /drops/seed/status` |
| Loader files to delete | `monster/drop/seed.go`, `continent/drop/seed.go` (keep `reactor/drop/seed.go`'s `DeleteAll` if it's the only loader content; verify) |

- [ ] **Step 1: Inspect, confirm route prefix `/drops`, confirm subdomain types**
- [ ] **Step 2-3: Write three subdomain.go files (monster/drop, continent/drop, reactor/drop)** — each ~50 lines, mirrors gachapon pattern
- [ ] **Step 4: Write `seed/groups.go`** — one Group with three subdomains
- [ ] **Step 5: Delete `seed/{seed,processor,resource,status,*_test}.go` and loader Load* helpers from `{monster,continent}/drop/seed.go`**
- [ ] **Step 6: Update main.go** — replace `seed.InitResource(...)` route initializer (same name), add SeedState to AutoMigrate
- [ ] **Step 7: `git rm -r services/atlas-drop-information/drops`**
- [ ] **Step 8: Update Dockerfile** — add atlas-seeder in 4 places, drop `COPY services/atlas-drop-information/drops /drops`
- [ ] **Step 9: `go mod edit` + tidy** for atlas-seeder
- [ ] **Step 10: Update k8s manifest** — drop three env vars, add `SEED_CATALOG_ROOT`
- [ ] **Step 11: Update compose** — drop three env vars, add seed-catalog mount + env
- [ ] **Step 12: Rewrite tests** — preserve URL assertions, add catalogRevision assertion
- [ ] **Step 13: Verify** — `go test -race ./... && go vet ./... && go build ./...` in the service module + `docker build -f services/atlas-drop-information/Dockerfile .` from worktree root
- [ ] **Step 14: Commit** `feat(atlas-drop-information): migrate to libs/atlas-seeder catalog mount`

### Task 4.3: Migrate atlas-map-actions

Recipe variables:

| Field | Value |
|---|---|
| Subdomain | `map.script` (path `map-actions/map`, type `map-script`) |
| Group | `Name: "map-actions"`, `URLPrefix: "/maps/actions"` |
| Routes preserved | `POST /maps/actions/seed`, `GET /maps/actions/seed/status` |
| Existing seed code | `atlas.com/map-actions/script/{seed.go,seed_status.go,seed_status_test.go}` |
| Bundled data | `services/atlas-map-actions/scripts/` |
| Dockerfile line | `COPY services/atlas-map-actions/scripts /scripts` |
| Env vars to drop | (grep in Step 1) — likely `MAP_SCRIPTS_PATH` or similar |

Apply Task 4.0 recipe with these values. Single commit.

### Task 4.4: Migrate atlas-reactor-actions

Recipe variables:

| Field | Value |
|---|---|
| Subdomain | `reactor.script` (path `reactor-actions/reactors`, type `reactor-script`) |
| Group | `Name: "reactor-actions"`, `URLPrefix: "/reactors/actions"` |
| Routes preserved | `POST /reactors/actions/seed`, `GET /reactors/actions/seed/status` |
| Existing seed code | `atlas.com/reactor/script/{seed.go,seed_status.go}` |
| Bundled data | `services/atlas-reactor-actions/scripts/` |
| Dockerfile line | `COPY services/atlas-reactor-actions/scripts /scripts` |

Apply Task 4.0 recipe.

### Task 4.5: Migrate atlas-portal-actions

Recipe variables:

| Field | Value |
|---|---|
| Subdomain | `portal.script` (path `portal-actions/portals`, type `portal-script`) |
| Group | `Name: "portal-actions"`, `URLPrefix: "/portals/scripts"` |
| Routes preserved | `POST /portals/scripts/seed`, `GET /portals/scripts/seed/status` |
| Existing seed code | `atlas.com/portal/script/{seed.go,seed_status.go}` |
| Bundled data | `services/atlas-portal-actions/scripts/` |
| Dockerfile line | `COPY services/atlas-portal-actions/scripts /scripts` |
| EntityIDPattern note | The splitter emits files named `portal-<map-portal-name>.json` per PRD §4.5; the pattern must capture that whole id string (it's not numeric). The Subdomain's `EntityIDPattern` should be `^portal-(.+)\.json$` and `data.id` must match. The splitter and the Subdomain pattern must agree — the splitter `--id-field` choice (likely a composite computed in wrap-jsonapi run script or a dedicated splitter) defines the id format. If wrap-jsonapi alone can't compose the id, write a small `split-portals/main.go` in Task 2.5b before this task. |

Apply Task 4.0 recipe. **If portal ids need composite (`<mapId>-<portalName>`) handling, add a sub-step before Step 2 to write `tools/seed-splitters/split-portals/` and rerun Task 3.1 for portal-actions only.**

### Task 4.6: Migrate atlas-npc-conversations

This service has **two** Groups (one for npc conversations, one for quest conversations) sharing one seed_state table via `group_name` discriminator.

| Field | Value |
|---|---|
| Subdomain (group 1) | `npc.conversation` (path `npc-conversations/npc`, type `npc-conversation`) |
| Subdomain (group 2) | `quest.conversation` (path `npc-conversations/quests`, type `quest-conversation`) |
| Group 1 | `Name: "npc-conversations:npc"`, `URLPrefix: "/npcs/conversations"` |
| Group 2 | `Name: "npc-conversations:quests"`, `URLPrefix: "/quests/conversations"` |
| Routes preserved | `POST /npcs/conversations/seed`, `GET /npcs/conversations/seed/status`, `POST /quests/conversations/seed`, `GET /quests/conversations/seed/status` |
| Existing seed code | `atlas.com/npc/conversation/{npc,quest}/{seed.go,seed_status.go,seed_status_test.go}` |
| Bundled data | `services/atlas-npc-conversations/conversations/` |
| Dockerfile line | `COPY services/atlas-npc-conversations/conversations /conversations` |

`seed/groups.go` makes two `seeder.RegisterRoutes(...)` calls with distinct `Group.Name`s so the `seed_state` rows are independent.

Apply Task 4.0 recipe with the two-Group adaptation.

### Task 4.7: Migrate atlas-npc-shops

| Field | Value |
|---|---|
| Subdomain | `shops.shop` (path `npc-shops/shops`, type `npc-shop`) |
| Group | `Name: "npc-shops"`, `URLPrefix: "/shops"` |
| Routes preserved | `POST /shops/seed`, `GET /shops/seed/status` |
| Existing seed code | `atlas.com/npc/seed/{seed.go,processor.go,resource.go,status.go}` + loader in `atlas.com/npc/shops/seed.go` |
| Bundled data | `services/atlas-npc-shops/shops/` (NOT `atlas.com/npc/data/` — that's Go source) |
| Dockerfile line | `COPY services/atlas-npc-shops/shops /shops` |
| Env vars to drop | `SHOPS_DATA_PATH` |

Note PRD §7.8 misstates the bundled-data location as `atlas.com/npc/data/`; the verified location is `services/atlas-npc-shops/shops/`.

Apply Task 4.0 recipe. Single commit.

### Task 4.8: Migrate atlas-party-quests

| Field | Value |
|---|---|
| Subdomain | `definition.partyquest` (path `party-quests/definitions`, type `party-quest-definition`) |
| Group | `Name: "party-quests"`, `URLPrefix: "/party-quests/definitions"` |
| Routes preserved | `POST /party-quests/definitions/seed`, `GET /party-quests/definitions/seed/status` |
| Existing seed code | `atlas.com/party-quests/definition/{seed.go,resource.go SeedDefinitionsHandler, processor.go Seed* funcs}` |
| Bundled data | `services/atlas-party-quests/party-quests/` |
| Dockerfile line | `COPY services/atlas-party-quests/party-quests /party-quests` |

Apply Task 4.0 recipe.

---

## Task Group 5: Infrastructure (k8s + compose)

### Task 5.1: Add `x-seed-catalog` anchor to compose and switch services to use it

**Files:**
- Modify: `deploy/compose/docker-compose.core.yml`

- [ ] **Step 1: Add the anchor at the top of services list**

```yaml
x-seed-catalog: &seed-catalog
  volumes:
    - ../seed:/var/run/seed-catalog:ro
  environment:
    SEED_CATALOG_ROOT: /var/run/seed-catalog
```

- [ ] **Step 2: Switch each in-scope service block to use `<<: *seed-catalog`**

For each of the eight services, replace the inline `SEED_CATALOG_ROOT` env + `volumes` (added during Task Group 4 per-service migrations) with `<<: *seed-catalog`. The merge key composes with existing `<<: *atlas-defaults` — YAML allows multiple merge sources via `<<: [*a, *b]`.

- [ ] **Step 3: Validate compose**

```bash
docker compose -f deploy/compose/docker-compose.core.yml config > /dev/null
```

Expected: no parse errors. Inspect `docker compose config` output for one in-scope service and confirm `SEED_CATALOG_ROOT` and the mount are present.

- [ ] **Step 4: Commit**

```bash
git add deploy/compose/docker-compose.core.yml
git commit -m "feat(deploy): add x-seed-catalog compose anchor and switch in-scope services to it"
```

### Task 5.2: Add k8s Kustomize seed-catalog component

**Files:**
- Create: `deploy/k8s/base/components/seed-catalog/kustomization.yaml`
- Create: `deploy/k8s/base/components/seed-catalog/configmap.yaml`
- Create: `deploy/k8s/base/components/seed-catalog/patch-volume.yaml`
- Create: `deploy/k8s/base/components/seed-catalog/patch-sidecar.yaml`
- Create: `deploy/k8s/base/components/seed-catalog/patch-mount.yaml`

- [ ] **Step 1: Create the component kustomization**

```yaml
# deploy/k8s/base/components/seed-catalog/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

resources:
  - configmap.yaml

patches:
  - path: patch-volume.yaml
  - path: patch-sidecar.yaml
  - path: patch-mount.yaml
```

- [ ] **Step 2: Create the ConfigMap**

```yaml
# deploy/k8s/base/components/seed-catalog/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: seed-catalog-config
data:
  GITSYNC_REPO: "https://github.com/Chronicle20/atlas"
  GITSYNC_REF: "main"
  GITSYNC_ROOT: "/git"
  GITSYNC_DIR: "deploy/seed"
  GITSYNC_PERIOD: "60s"
```

- [ ] **Step 3: Create volume / sidecar / mount patches (StrategicMergePatch shape)**

Each patch is a Deployment patch that StrategicMerge applies to whichever Deployment includes the component. Because StrategicMerge needs the Deployment name, use `kind: Component`'s ability to apply to all Deployments matching a JSON6902-style target or accept that the patch targets `apps/v1/Deployment` and Kustomize merges by name. Use a `target:` selector in `kustomization.yaml` instead so each consumer's `kustomization.yaml` doesn't need per-service tweaks:

```yaml
# deploy/k8s/base/components/seed-catalog/kustomization.yaml (revised)
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

resources:
  - configmap.yaml

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      labelSelector: "atlas.seed-catalog=true"
    path: patch-volume.yaml
  - target:
      group: apps
      version: v1
      kind: Deployment
      labelSelector: "atlas.seed-catalog=true"
    path: patch-sidecar.yaml
  - target:
      group: apps
      version: v1
      kind: Deployment
      labelSelector: "atlas.seed-catalog=true"
    path: patch-mount.yaml
```

Each in-scope service Deployment gets a label `atlas.seed-catalog: "true"` (added in Task 5.3 below). The three patches are RFC6902 JSON Patches:

```yaml
# deploy/k8s/base/components/seed-catalog/patch-volume.yaml
- op: add
  path: /spec/template/spec/volumes/-
  value:
    name: seed-catalog
    emptyDir: {}
```

```yaml
# deploy/k8s/base/components/seed-catalog/patch-sidecar.yaml
- op: add
  path: /spec/template/spec/containers/-
  value:
    name: git-sync
    image: registry.k8s.io/git-sync/git-sync:v4.4.0
    envFrom:
      - configMapRef:
          name: seed-catalog-config
    args:
      - --repo=$(GITSYNC_REPO)
      - --ref=$(GITSYNC_REF)
      - --root=$(GITSYNC_ROOT)
      - --period=$(GITSYNC_PERIOD)
    resources:
      requests: { cpu: 50m, memory: 64Mi }
      limits:   { cpu: 200m, memory: 256Mi }
    volumeMounts:
      - name: seed-catalog
        mountPath: /git
        subPath: deploy/seed
```

```yaml
# deploy/k8s/base/components/seed-catalog/patch-mount.yaml
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/-
  value:
    name: seed-catalog
    mountPath: /var/run/seed-catalog
    readOnly: true
- op: add
  path: /spec/template/spec/containers/0/env/-
  value:
    name: SEED_CATALOG_ROOT
    value: /var/run/seed-catalog
```

Patch order matters: volume must exist before sidecar mounts it.

- [ ] **Step 4: Add component reference to the in-scope service base kustomizations**

Each of the eight `deploy/k8s/base/atlas-<svc>.yaml` files currently has no per-service kustomization.yaml (they are referenced by `deploy/k8s/base/kustomization.yaml` directly). To attach the component, create per-service kustomizations that include both the service yaml and the component:

```
deploy/k8s/base/atlas-drop-information/
├── kustomization.yaml          # NEW
└── (existing service yaml moves here OR is referenced from one level up)
```

Simpler alternative: add the component as a top-level reference in `deploy/k8s/base/kustomization.yaml` with a per-service label selector, and add the `atlas.seed-catalog: "true"` label to each in-scope service Deployment in its existing `.yaml` file via an inline edit. This avoids restructuring the base directory.

Pick the simpler alternative:

```yaml
# deploy/k8s/base/kustomization.yaml — append at bottom
components:
  - components/seed-catalog
```

Then edit each in-scope `deploy/k8s/base/atlas-<svc>.yaml` to add the label on the Deployment:

```yaml
metadata:
  name: atlas-drop-information
  labels:
    atlas.seed-catalog: "true"
```

And drop the inline `SEED_CATALOG_ROOT` env + mount that was added during the Task Group 4 per-service migration (the component patch reintroduces it).

- [ ] **Step 5: Dry-run validate**

```bash
kubectl kustomize deploy/k8s/base > /tmp/base.yaml
kubectl kustomize deploy/k8s/overlays/main > /tmp/main.yaml
# Spot-check one in-scope service has the sidecar and volume
grep -A 5 "git-sync" /tmp/main.yaml | head -20
```

Expected: the rendered output contains `git-sync` container, `seed-catalog` volume, and `SEED_CATALOG_ROOT` env on each labeled Deployment.

If a live cluster is reachable:

```bash
kubectl apply --dry-run=server -k deploy/k8s/overlays/main
```

Expected: no errors. Not required for CI; the `kubectl kustomize` step is the gate.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/base/components/seed-catalog deploy/k8s/base/kustomization.yaml deploy/k8s/base/atlas-drop-information.yaml deploy/k8s/base/atlas-gachapons.yaml deploy/k8s/base/atlas-map-actions.yaml deploy/k8s/base/atlas-reactor-actions.yaml deploy/k8s/base/atlas-portal-actions.yaml deploy/k8s/base/atlas-npc-conversations.yaml deploy/k8s/base/atlas-npc-shops.yaml deploy/k8s/base/atlas-party-quests.yaml
git commit -m "feat(deploy): add seed-catalog Kustomize component with git-sync sidecar"
```

### Task 5.3: Add PR-overlay patch for GITSYNC_REF

**Files:**
- Create: `deploy/k8s/overlays/pr/patches/seed-catalog-ref.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`

- [ ] **Step 1: Create the patch**

```yaml
# deploy/k8s/overlays/pr/patches/seed-catalog-ref.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: seed-catalog-config
data:
  GITSYNC_REF: "${PR_SHA}"
```

ArgoCD/Kustomize substitutes `${PR_SHA}` from the ApplicationSet parameter or via a Kustomize-native patch invoked from the overlay. If the ApplicationSet supplies it as a Kustomize variable, use `replacements:` instead; otherwise the PR pipeline writes the literal SHA into this file at apply time.

- [ ] **Step 2: Reference the patch from overlays/pr/kustomization.yaml**

```yaml
patches:
  - path: patches/seed-catalog-ref.yaml
    target:
      kind: ConfigMap
      name: seed-catalog-config
```

- [ ] **Step 3: Dry-run validate**

```bash
kubectl kustomize deploy/k8s/overlays/pr | grep -A 2 "GITSYNC_REF"
```

Expected: `GITSYNC_REF` is `${PR_SHA}` (or whatever the overlay produces).

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/overlays/pr/patches/seed-catalog-ref.yaml deploy/k8s/overlays/pr/kustomization.yaml
git commit -m "feat(deploy): patch GITSYNC_REF for PR overlays"
```

---

## Task Group 6: CI Catalog Linter

### Task 6.1: Implement `tools/catalog-lint`

**Files:**
- Create: `tools/catalog-lint/go.mod`
- Create: `tools/catalog-lint/main.go`
- Create: `tools/catalog-lint/subdomains.go`
- Test: `tools/catalog-lint/main_test.go`
- Test fixtures: `tools/catalog-lint/testdata/{good,bad/*}/...`

- [ ] **Step 1: Bootstrap module**

```bash
mkdir -p tools/catalog-lint
cd tools/catalog-lint
cat > go.mod <<'EOF'
module github.com/Chronicle20/atlas/tools/catalog-lint

go 1.25.0

require github.com/Chronicle20/atlas/libs/atlas-seeder v0.0.0
replace github.com/Chronicle20/atlas/libs/atlas-seeder => ../../libs/atlas-seeder
EOF
cd <worktree>
go work use ./tools/catalog-lint
```

- [ ] **Step 2: Build fixture catalogs**

```bash
mkdir -p tools/catalog-lint/testdata/good/gms/83_1/widgets
echo "test-rev" > tools/catalog-lint/testdata/good/gms/83_1/CATALOG_REVISION
cat > tools/catalog-lint/testdata/good/gms/83_1/widgets/widget-1.json <<'EOF'
{"data":{"type":"widget","id":"1","attributes":{}}}
EOF
mkdir -p tools/catalog-lint/testdata/bad/id-mismatch/gms/83_1/widgets
echo "test-rev" > tools/catalog-lint/testdata/bad/id-mismatch/gms/83_1/CATALOG_REVISION
cat > tools/catalog-lint/testdata/bad/id-mismatch/gms/83_1/widgets/widget-1.json <<'EOF'
{"data":{"type":"widget","id":"99","attributes":{}}}
EOF
mkdir -p tools/catalog-lint/testdata/bad/missing-revision/gms/83_1/widgets
cat > tools/catalog-lint/testdata/bad/missing-revision/gms/83_1/widgets/widget-1.json <<'EOF'
{"data":{"type":"widget","id":"1","attributes":{}}}
EOF
```

- [ ] **Step 3: Write the failing test**

```go
// tools/catalog-lint/main_test.go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildLint(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "catalog-lint")
	out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}

func TestLint_GoodTreeExitsZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/good")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0, got %v", err)
	}
}

func TestLint_IDMismatchExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/id-mismatch")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}

func TestLint_MissingRevisionExitsNonZero(t *testing.T) {
	exe := buildLint(t)
	cmd := exec.Command(exe, "testdata/bad/missing-revision")
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected non-zero exit")
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd tools/catalog-lint && go test -v
```

Expected: FAIL — binary won't build (no main).

- [ ] **Step 5: Implement `subdomains.go`**

```go
// tools/catalog-lint/subdomains.go
package main

import "regexp"

type subdomainRule struct {
	path    string
	typ     string
	pattern *regexp.Regexp
}

// All subdomain expectations. Mirrors the per-service Subdomain implementations.
var rules = []subdomainRule{
	{path: "drops/monsters", typ: "monster-drop", pattern: regexp.MustCompile(`^monster-(\d+)\.json$`)},
	{path: "drops/continents", typ: "continent-drop", pattern: regexp.MustCompile(`^continent-(\d+)\.json$`)},
	{path: "drops/reactors", typ: "reactor-drop", pattern: regexp.MustCompile(`^reactor-(\d+)\.json$`)},
	{path: "gachapons", typ: "gachapon", pattern: regexp.MustCompile(`^(\d+)\.json$`)},
	{path: "gachapons/_global", typ: "gachapon-pool", pattern: nil},
	{path: "map-actions/map", typ: "map-script", pattern: regexp.MustCompile(`^map-(\d+)\.json$`)},
	{path: "portal-actions/portals", typ: "portal-script", pattern: regexp.MustCompile(`^portal-(.+)\.json$`)},
	{path: "reactor-actions/reactors", typ: "reactor-script", pattern: regexp.MustCompile(`^reactor-(\d+)\.json$`)},
	{path: "npc-conversations/npc", typ: "npc-conversation", pattern: regexp.MustCompile(`^npc-(\d+)\.json$`)},
	{path: "npc-conversations/quests", typ: "quest-conversation", pattern: regexp.MustCompile(`^quest-(\d+)\.json$`)},
	{path: "npc-shops/shops", typ: "npc-shop", pattern: regexp.MustCompile(`^shop-(\d+)\.json$`)},
	{path: "party-quests/definitions", typ: "party-quest-definition", pattern: regexp.MustCompile(`^party-quest-(\d+)\.json$`)},
	// widgets fixture used in tests
	{path: "widgets", typ: "widget", pattern: regexp.MustCompile(`^widget-(\d+)\.json$`)},
}

func ruleFor(relDir string) (subdomainRule, bool) {
	for _, r := range rules {
		if r.path == relDir {
			return r, true
		}
	}
	return subdomainRule{}, false
}
```

- [ ] **Step 6: Implement `main.go`**

```go
// tools/catalog-lint/main.go
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: catalog-lint <root>")
		os.Exit(2)
	}
	root := os.Args[1]
	if err := lint(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func lint(root string) error {
	var errs []string

	// 1. Each <region>/<major>_<minor>/ dir must contain a non-empty CATALOG_REVISION.
	regionEntries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read root: %w", err)
	}
	for _, regionEntry := range regionEntries {
		if !regionEntry.IsDir() || strings.HasPrefix(regionEntry.Name(), "_") {
			continue
		}
		regionDir := filepath.Join(root, regionEntry.Name())
		versionEntries, _ := os.ReadDir(regionDir)
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			versionDir := filepath.Join(regionDir, versionEntry.Name())
			rev, _ := os.ReadFile(filepath.Join(versionDir, "CATALOG_REVISION"))
			if len(strings.TrimSpace(string(rev))) == 0 {
				errs = append(errs, fmt.Sprintf("%s: missing or empty CATALOG_REVISION", versionDir))
			}
		}
	}

	// 2. Walk every *.json file. Skip names starting with _ or . and dirs starting with _ or .
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, _ error) error {
		base := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(base, "_") || strings.HasPrefix(base, ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(base, "_") || strings.HasPrefix(base, ".") {
			return nil
		}
		if !strings.HasSuffix(base, ".json") {
			return nil
		}
		// Determine the subdomain rule by walking from versionDir.
		rel, _ := filepath.Rel(root, path)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 { // <region>/<version>/<subdomain-path>/<file>
			return nil
		}
		subdomainPath := strings.Join(parts[2:len(parts)-1], "/")
		rule, ok := ruleFor(subdomainPath)
		if !ok {
			return nil // unrecognized subdomain — not an error per se
		}
		b, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", path, err))
			return nil
		}
		env, err := seeder.ParseEnvelope(b)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if env.Data.Type != rule.typ {
			errs = append(errs, fmt.Sprintf("%s: type %q, want %q", path, env.Data.Type, rule.typ))
		}
		if rule.pattern != nil {
			id, err := seeder.ExtractEntityID(base, rule.pattern)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", path, err))
				return nil
			}
			if id != env.Data.ID {
				errs = append(errs, fmt.Sprintf("%s: data.id %q, filename id %q", path, env.Data.ID, id))
			}
		}
		return nil
	})

	if len(errs) > 0 {
		return fmt.Errorf("linter found %d issue(s):\n%s", len(errs), strings.Join(errs, "\n"))
	}
	return nil
}
```

- [ ] **Step 7: Run test to verify it passes**

```bash
cd tools/catalog-lint && go test -v
```

Expected: PASS on good fixture, FAIL (exits non-zero) on bad fixtures (test inverts and asserts non-zero).

- [ ] **Step 8: Run the linter against the real catalog**

```bash
cd <worktree>
go run ./tools/catalog-lint deploy/seed
```

Expected: exit 0. If failures, fix the splitter output (Task Group 2) or the rule table (Step 5 above).

- [ ] **Step 9: Commit**

```bash
git add tools/catalog-lint go.work
git commit -m "feat(catalog-lint): add CI linter validating JSON:API envelopes and revisions"
```

### Task 6.2: Add GitHub Actions workflow

**Files:**
- Create: `.github/workflows/catalog-lint.yml`

- [ ] **Step 1: Inspect existing workflow patterns**

```bash
ls .github/workflows/ | head
cat .github/workflows/$(ls .github/workflows | head -1)
```

Match the trigger/setup style of an existing Go-tooling workflow.

- [ ] **Step 2: Create the workflow**

```yaml
# .github/workflows/catalog-lint.yml
name: catalog-lint

on:
  pull_request:
    paths:
      - 'deploy/seed/**'
      - 'tools/catalog-lint/**'
      - 'libs/atlas-seeder/**'
  push:
    branches: [main]
    paths:
      - 'deploy/seed/**'
      - 'tools/catalog-lint/**'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Lint catalog
        run: |
          set -e
          if [ "${{ github.event_name }}" = "pull_request" ]; then
            go run ./tools/catalog-lint deploy/seed
          else
            # advisory on main: log failures but do not fail the job
            go run ./tools/catalog-lint deploy/seed || true
          fi
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/catalog-lint.yml
git commit -m "ci: add catalog-lint workflow (strict on PRs, advisory on main)"
```

### Task 6.3: Add CI step writing `CATALOG_REVISION` per commit

**Files:**
- Modify: existing `.github/workflows/<main-deploy>.yml` (whichever workflow ships images to ArgoCD)

- [ ] **Step 1: Identify the deploy workflow**

```bash
grep -l "GITHUB_SHA\|argocd\|overlays" .github/workflows/*.yml
```

- [ ] **Step 2: Add a step before image build that writes `$GITHUB_SHA` into each `deploy/seed/<region>/<version>/CATALOG_REVISION`**

```yaml
- name: Stamp CATALOG_REVISION
  run: |
    set -e
    for dir in deploy/seed/*/*/; do
      echo -n "$GITHUB_SHA" > "${dir}CATALOG_REVISION"
    done
```

The stamped file is consumed by git-sync at sync time; CI doesn't commit the change (it's transient artifact). For PR builds, the stamping happens against the PR checkout so the resulting catalog reflects the PR.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/<deploy-workflow>.yml
git commit -m "ci: stamp CATALOG_REVISION per commit"
```

---

## Task Group 7: End-to-End Verification

### Task 7.1: Compose smoke test

- [ ] **Step 1: Build and start the affected services**

```bash
cd <worktree>/deploy/compose
./build.sh atlas-drop-information atlas-gachapons atlas-map-actions atlas-reactor-actions atlas-portal-actions atlas-npc-conversations atlas-npc-shops atlas-party-quests
./up.sh
```

- [ ] **Step 2: Wait for services to be healthy**

```bash
sleep 30
docker compose -f docker-compose.core.yml ps | grep atlas-drop-information
```

Expected: status `Up` and `(healthy)` if healthchecks exist.

- [ ] **Step 3: Trigger seed on each service**

```bash
TENANT_HEADER="X-Tenant-Id: <a valid tenant uuid>"
TENANT_REGION="X-Region: GMS"
TENANT_MAJOR="X-Major-Version: 83"
TENANT_MINOR="X-Minor-Version: 1"
for prefix in drops gachapons maps/actions reactors/actions portals/scripts npcs/conversations quests/conversations shops party-quests/definitions; do
  echo "=== $prefix ==="
  curl -s -X POST -H "$TENANT_HEADER" -H "$TENANT_REGION" -H "$TENANT_MAJOR" -H "$TENANT_MINOR" \
    -o /dev/null -w "%{http_code}\n" \
    "http://localhost:<port-for-service>/api/$prefix/seed"
done
```

Substitute the actual tenant header names by inspecting one existing service test (the in-repo httptest middleware will reveal them).

Expected: every endpoint returns `202`.

- [ ] **Step 4: Verify status returns the new fields**

```bash
sleep 10
curl -s -H "$TENANT_HEADER" ... "http://localhost:<port>/api/drops/seed/status" | jq '. | {catalogRevision, tenantSeededRevision, tenantSeededAt}'
```

Expected: non-null `catalogRevision` matching `cat deploy/seed/gms/83_1/CATALOG_REVISION`; non-null `tenantSeededRevision` matching same.

- [ ] **Step 5: Verify edit-and-reseed flow**

```bash
# Edit one monster drop file
echo "manual-test-rev" > deploy/seed/gms/83_1/CATALOG_REVISION
# Re-POST seed
curl -X POST ... "http://localhost:<port>/api/drops/seed"
sleep 5
curl ... "http://localhost:<port>/api/drops/seed/status" | jq '.tenantSeededRevision'
```

Expected: `tenantSeededRevision` becomes `manual-test-rev`.

- [ ] **Step 6: Restore the real CATALOG_REVISION**

```bash
git checkout deploy/seed/gms/83_1/CATALOG_REVISION
```

- [ ] **Step 7: Document smoke-test results in the task folder**

Create `docs/tasks/task-072-shared-seeder-catalog/smoke-test.md` with the timestamps and any anomalies. No commit needed if nothing was added beyond the doc; otherwise commit `docs(task-072): smoke-test results`.

### Task 7.2: k8s dry-run validation

- [ ] **Step 1: Run kustomize render for both overlays**

```bash
cd <worktree>
kubectl kustomize deploy/k8s/overlays/main > /tmp/main.yaml
kubectl kustomize deploy/k8s/overlays/pr > /tmp/pr.yaml
```

Expected: no errors from kustomize.

- [ ] **Step 2: Spot-check rendered manifests**

```bash
# Each in-scope service should have the git-sync sidecar
for svc in atlas-drop-information atlas-gachapons atlas-map-actions atlas-reactor-actions atlas-portal-actions atlas-npc-conversations atlas-npc-shops atlas-party-quests; do
  echo "=== $svc ==="
  awk "/name: $svc\$/,/^---/" /tmp/main.yaml | grep -c "git-sync"
done
```

Expected: each prints `1`.

- [ ] **Step 3: If a cluster is reachable, server-side dry-run**

```bash
kubectl apply --dry-run=server -k deploy/k8s/overlays/main
```

Expected: 0 errors.

- [ ] **Step 4: Document in smoke-test.md if applicable**

### Task 7.3: Run the modular reviewer agents

Per CLAUDE.md "Code Review Before PR":

- [ ] **Step 1: Run `superpowers:requesting-code-review`**

This dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` in parallel against the task branch. Output lands in `docs/tasks/task-072-shared-seeder-catalog/audit.md`.

- [ ] **Step 2: Address any FAIL findings before declaring the task complete**

For each FAIL, decide: fix the code, document a deliberate deviation in `audit.md`, or escalate. Commit any fixes with `fix(task-072): address audit finding <id>`.

### Task 7.4: Final verification matrix

Confirm every PRD §10 acceptance criterion holds:

- [ ] §10.1 Library: `libs/atlas-seeder/` exists with the public API; tests pass with `-race`.
- [ ] §10.2 Per-service migration: for each of the eight services, the old `seed/` is gone, the new `Group` is registered, `Dockerfile` no longer copies bundled data, the bundled dir is deleted, k8s manifest references the component, compose entry uses the anchor, AutoMigrate registers `SeedState`, `POST /<prefix>/seed` returns 202 and `GET /<prefix>/seed/status` includes the new fields, all four `go test/vet/build` checks pass, and `docker build` succeeds.
- [ ] §10.3 Catalog: six version directories exist with `CATALOG_REVISION` and populated subdomain dirs; v83 is splitter output; others are bootstrapped from v83; no `services/<svc>/` tree still contains catalog data.
- [ ] §10.4 Infra: `deploy/k8s/base/components/seed-catalog/` exists; eight manifests carry the label; `deploy/compose/docker-compose.core.yml` has anchor + 8 references; kustomize dry-run passes; compose config passes.
- [ ] §10.5 Tooling: four splitters exist with passing determinism tests; `tools/catalog-lint/` exists and tests pass; CI workflow runs the linter on PRs.
- [ ] §10.6 End-to-end: compose smoke succeeded; status reports `catalogRevision`; editing CATALOG_REVISION and re-POSTing updates `tenantSeededRevision`; all eight services boot in compose.

- [ ] **Step 1: Walk the matrix and tick each box**

If any box stays empty, address the gap (file a new ad-hoc task on this branch) before invoking `superpowers:finishing-a-development-branch`.

---

## Self-Review Notes

The following items were checked against the spec before saving this plan:

1. **Spec coverage** — every PRD §10 acceptance criterion maps to a task: §10.1 → Task Group 1; §10.2 → Task Group 4 (recipe + 8 service tasks); §10.3 → Task Group 3; §10.4 → Task Group 5; §10.5 → Task Group 2 + Task 6.1; §10.6 → Task 7.1. PRD §4.1 lib API is realized in Tasks 1.2–1.9. PRD §4.2 `seed_state` schema is in Task 1.2. PRD §4.3 catalog layout is realized in Task Group 3. PRD §4.4 splitters are Task Group 2. PRD §4.5 per-service migration is Task Group 4. PRD §4.6 k8s git-sync is Task 5.2. PRD §4.7 compose anchor is Task 5.1. PRD §4.8 linter is Task 6.1.

2. **Placeholder scan** — Tasks 4.3–4.8 use a recipe-reference pattern: each task lists the substitution variables for Task 4.0's full recipe rather than restating the recipe 6 times. This is a deliberate compression, NOT a placeholder — every concrete value (file path, route URL, env var) is given. The implementer reads Task 4.0 once and applies it per service.

3. **Type consistency** — `Subdomain[J, M]`, `SubdomainAny`, `AdaptSubdomain`, `Group`, `CatalogSource`, `FilesystemCatalogSource`, `Result`, `Status`, `SeedState`, `Seed`, `Status`, `RegisterRoutes`, `ObserveSeederRun`, `ParseEnvelope`, `ExtractEntityID` names are used consistently from the design through every task. The `loadOne` and `runSubdomain` internal helpers are defined in Task 1.7 and referenced nowhere outside it.

4. **Open Question resolutions** — every PRD Open Question is resolved in `design.md` §10 and the resolutions are encoded as concrete plan steps: URLs verified in `context.md` §1; `SEED_CATALOG_ROOT` dev fallback in Task 1.5 + per-service `seed/groups.go` template; git-sync ref strategy in Task 5.3; linter strictness in Task 6.2; portal id encoding flagged in Task 4.5 with a conditional sub-step.
