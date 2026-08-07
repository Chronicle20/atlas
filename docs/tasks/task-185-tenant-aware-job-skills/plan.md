# Tenant-Aware Job Skill Enumeration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the compiled-in job→skill table in `libs/atlas-constants/job` with per-tenant `JOB` documents derived from each tenant's own ingested `Skill.wz`, expose them through `GET /data/jobs` and `GET /data/jobs/{jobId}/skills`, and retire atlas-ui's hand-maintained version-floor tables in favour of the tenant's actual job set.

**Architecture:** A new `JOB` document type rides the existing generic `documents` table through `document.Storage`, written by a second registration pass inside the existing `SKILL` worker (no new worker, no new `data.Workers` entry, no DDL). The read path swaps `constJob.Jobs[id].Skills()` for `document.Storage.GetById`, inheriting tenant scoping and the canonical-tenant fallback for free. A separate `ListRestModel` type — never persisted — carries the JSON:API `skills` to-many relationship for the new list endpoint, keeping `relationships`/`included` out of the stored `content` column. atlas-ui replaces its `major: number` version-floor predicate with an `available: ReadonlySet<number>` of job ids fetched from the backend.

**Tech Stack:** Go 1.x (atlas-data, libs/atlas-constants), GORM + Postgres (sqlite in-memory for tests), `github.com/jtumidanski/api2go/jsonapi`, `gorilla/mux`, `logrus`, `testify/require`; TypeScript + React 19 + Vite + TanStack React Query 5 + Vitest (atlas-ui).

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-185-tenant-aware-job-skills/` on branch `task-185-tenant-aware-job-skills`. Never edit the main repo checkout. Shell snippets below use `cd "$(git rev-parse --show-toplevel)"` to reach the worktree root — never a literal absolute path.
- **Hard cutover, no fallback.** The constants lists are deleted in this same change. A transitional fallback to `constJob.Jobs[id].Skills()` is explicitly rejected (PRD §7.5) — do not add one, not even temporarily behind a flag.
- **`libs/atlas-constants/go.mod` must not change.** The library must not gain a dependency on `atlas-tenant` (FR-5.6).
- **`Jobs` keeps all 82 job ids and all 23 `fourthJob: true` markers** (FR-5.3). Verified counts against the current file: `grep -c "^var [A-Z][A-Za-z0-9]* = Job{" libs/atlas-constants/job/constants.go` → 82; `grep -c "fourthJob: true" …` → 23.
- **`Job` stays a struct `{id, fourthJob}`** — not `map[Id]bool` (FR-5.4). `FromSkillId` returns `(Job, bool)`; callers use `.Id()` and `.IsFourthJob()`.
- **The `Id` constant block, the `Type` constants, and `advancement.go` are untouched** (FR-5.5).
- **`GET /data/jobs/{jobId}/skills` response shape is unchanged** (PRD §5) — `type: "jobs"`, `id`, `attributes.skills`, and **no** `relationships` block.
- **Stored `content` must contain neither `relationships` nor `included`** (FR-4.4 / D2). This is enforced by a test, not by convention.
- **Skill id order is WZ document order, not sorted** (FR-1.2 / D5) — re-ingest must be byte-stable so baseline dumps do not churn.
- **Job id comes from the image name, never from dividing skill ids** (FR-2.2).
- **Repo-relative paths only in committed files.** Never write a literal home or absolute path into any file.
- **Preserve line endings.** Do not normalize CRLF→LF as an editing side effect.
- **No `// TODO`, stubs, or 501s in landed commits.**
- **Model/cost:** if dispatching review subagents, pin them to Sonnet/Haiku.

### Design decisions already settled (do not relitigate)

Design §7 left two decisions open with stated recommendations. Both recommendations are adopted here:

1. **`includeSkills` ships on the UI service** with no production caller today. `useJobSkillDefinitions` is **not** rewired through it (Task 8 note).
2. **The list projection type is named `ListRestModel`** in `job/list_rest.go`.

---

## File Structure

### `services/atlas-data/atlas.com/data`

| File | Responsibility | Change |
|---|---|---|
| `job/registry.go` | `sync.Once` singleton `document.Registry[string, RestModel]` for JOB | **new** |
| `job/reader.go` | XML → `[]RestModel` (0 or 1 model) for one `Skill.wz` per-job image; `parseJobId` | **new** |
| `job/reader_test.go` | Fixture coverage of the D5 table | **new** |
| `job/rest.go` | Persisted document + `/{jobId}/skills` response. Relationship-free. | unchanged |
| `job/rest_test.go` | FR-4.4 guard: stored `content` has no `relationships`/`included` | **new** |
| `job/list_rest.go` | `ListRestModel` — list-endpoint projection with the `skills` to-many | **new** |
| `job/processor.go` | `NewProcessor(l, ctx, db)`, `NewStorage`, storage-backed `GetSkillsForJob`, `Register`, `RegisterJob` | rewritten |
| `job/processor_test.go` | Storage-backed processor tests (seeded sqlite) | rewritten |
| `job/resource.go` | `InitResource(db)(si)`; adds `GET /data/jobs`; `/{jobId}/skills` 404s off storage | modified |
| `job/resource_test.go` | Seeded-storage endpoint tests incl. two-tenant divergence | rewritten |
| `job/mock/processor.go` | Mock gains the new interface methods | modified |
| `data/workers/skill.go` | Second registration pass + JOB-document count/warn | modified |
| `data/workers/skill_test.go` | `countingRegister` + `logJobDocCount` unit tests | **new** |
| `baseline/dump_test.go` | JOB rides the whole-table `documents` dump (D9) | modified |
| `tenantpurge/purge_test.go` | A `type='JOB'` row is purged (D9) | modified |
| `main.go:184` | `job.InitResource(db)(GetServer())` | modified |
| `services/atlas-data/docs/rest.md` | Document `GET /api/data/jobs`; update the 404 wording | modified |

### `libs/atlas-constants/job`

| File | Change |
|---|---|
| `constants.go` | 1,236 → ~180 lines: drop the 82 `Job` value vars, inline `Jobs` as a literal map |
| `model.go` | Drop the `skills` field, `Skills()`, `Buffs()` |
| `constants_test.go` | **new** — 82-entry / 23-marker / every-`*Id`-is-a-key table test |
| `go.mod`, `README.md`, `advancement.go`, `model_test.go` | untouched |

### `services/atlas-ui/src`

| File | Change |
|---|---|
| `types/api/responses.ts` | `JsonApiResource` + `ApiResponse.included?` |
| `lib/api/client.ts` | `api.getListDocument` primitive (`api.getList` untouched) |
| `services/api/jobs.service.ts` | `getJobs({ includeSkills })` with `links.next` following |
| `lib/hooks/api/useJobs.ts` | **new** — React Query hook keyed on tenant id |
| `lib/jobs/job-advancement-tree.ts` | Delete `BRANCH_FLOORS`/`NODE_FLOORS`/`floorOf` + rationale comment; `major` → `available` |
| `components/features/jobs/rail-groups.ts` | `visibleRailGroups(available)` |
| `components/features/jobs/advancement-flow.tsx` | prop `major: number` → `available: ReadonlySet<number>` |
| `components/features/jobs/branch-rail.tsx` | `isPending` skeleton state |
| `pages/JobsPage.tsx` | Source `available` from `useJobs`; gate visibility/redirect on `isSuccess` |
| `services/api/__tests__/jobs.service.test.ts` | **new** |
| `lib/jobs/__tests__/job-advancement-tree.test.ts` | rewritten set-driven |
| `pages/__tests__/JobsPage.test.tsx` | extended with the async-visibility states |

### `docs`

| File | Change |
|---|---|
| `docs/runbooks/job-document-backfill.md` | **new** — the 11-version rollout runbook |

---

## Task 1: `job` reader and registry

**Files:**
- Create: `services/atlas-data/atlas.com/data/job/registry.go`
- Create: `services/atlas-data/atlas.com/data/job/reader.go`
- Test: `services/atlas-data/atlas.com/data/job/reader_test.go`

**Interfaces:**
- Consumes: `atlas-data/xml` (`xml.Node`, `xml.FromByteArrayProvider`, `xml.FromPathProvider`), `atlas-data/document` (`Registry`, `NewRegistry`), the existing `job.RestModel{Id uint32; Skills []uint32}` from `job/rest.go`.
- Produces:
  - `func GetModelRegistry() *document.Registry[string, RestModel]`
  - `func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[[]RestModel]`
  - unexported `func parseJobId(name string) (uint32, error)`

**Background.** `xml.Node.Name` is the root `<imgdir name="…">` attribute, which for a `Skill.wz` per-job image is e.g. `"112.img"` (see `skill/reader.go:48`, which calls `parseJobId(exml.Name)`). `Node.ChildByName` returns `(*Node, error)` with `errors.New("child not found")` when absent (`xml/model.go:20-27`).

Two deliberate divergences from `skill.Read` (D5):
- a non-numeric image name yields an **empty slice and no error** (so `registerAllInDirectory` logs no warn for `MobSkill.img`, unlike the existing skill pass);
- a missing `skill` child yields **one model with an empty skill list**, not an error (FR-2.4 requires "present with zero skills" to be representable and distinguishable from absence).

`ctx` is accepted for signature parity with `skill.Read` and the `Register` plumbing; the JOB reader needs no tenant.

- [ ] **Step 1: Write the failing reader tests**

Create `services/atlas-data/atlas.com/data/job/reader_test.go`:

```go
package job

import (
	"atlas-data/xml"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

const jobImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="112.img">
  <imgdir name="info">
    <canvas name="icon" width="26" height="30"/>
  </imgdir>
  <imgdir name="skill">
    <imgdir name="1121000"/>
    <imgdir name="1121001"/>
    <imgdir name="1121002"/>
  </imgdir>
</imgdir>`

const emptySkillNodeXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="800.img">
  <imgdir name="skill"></imgdir>
</imgdir>`

const noSkillNodeXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="900.img">
  <imgdir name="info"></imgdir>
</imgdir>`

const mobSkillImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="MobSkill.img">
  <imgdir name="100">
    <imgdir name="level"></imgdir>
  </imgdir>
</imgdir>`

const nonNumericSkillChildXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="1.img">
  <imgdir name="skill">
    <imgdir name="1000"/>
    <imgdir name="notaskill"/>
    <imgdir name="1001"/>
  </imgdir>
</imgdir>`

// zeroJobImageXML covers job id 0 (Beginner) — document_id 0 is a legitimate
// key, and the reader must not confuse it with "no job id".
const zeroJobImageXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0.img">
  <imgdir name="skill">
    <imgdir name="1000"/>
  </imgdir>
</imgdir>`

func readAll(t *testing.T, data string) []RestModel {
	t.Helper()
	l, _ := test.NewNullLogger()
	ms, err := Read(l)(context.Background())(xml.FromByteArrayProvider([]byte(data)))()
	require.NoError(t, err)
	return ms
}

// writeTempImage materializes a fixture on disk for the RegisterJob tests,
// which go through xml.FromPathProvider rather than FromByteArrayProvider.
func writeTempImage(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestRead_NumericImageWithSkills_PreservesDocumentOrder(t *testing.T) {
	ms := readAll(t, jobImageXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(112), ms[0].Id)
	require.Equal(t, []uint32{1121000, 1121001, 1121002}, ms[0].Skills)
}

func TestRead_EmptySkillNode_ProducesEmptyList(t *testing.T) {
	ms := readAll(t, emptySkillNodeXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(800), ms[0].Id)
	require.NotNil(t, ms[0].Skills)
	require.Empty(t, ms[0].Skills)
}

func TestRead_MissingSkillNode_ProducesEmptyList(t *testing.T) {
	ms := readAll(t, noSkillNodeXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(900), ms[0].Id)
	require.NotNil(t, ms[0].Skills)
	require.Empty(t, ms[0].Skills)
}

func TestRead_NonNumericImage_ProducesNothingAndNoError(t *testing.T) {
	ms := readAll(t, mobSkillImageXML)
	require.Empty(t, ms)
}

func TestRead_NonNumericSkillChild_IsSkipped(t *testing.T) {
	ms := readAll(t, nonNumericSkillChildXML)
	require.Len(t, ms, 1)
	require.Equal(t, []uint32{1000, 1001}, ms[0].Skills)
}

func TestRead_JobIdZeroIsValid(t *testing.T) {
	ms := readAll(t, zeroJobImageXML)
	require.Len(t, ms, 1)
	require.Equal(t, uint32(0), ms[0].Id)
	require.Equal(t, []uint32{1000}, ms[0].Skills)
}

func TestParseJobId(t *testing.T) {
	id, err := parseJobId("112.img")
	require.NoError(t, err)
	require.Equal(t, uint32(112), id)

	_, err = parseJobId("MobSkill.img")
	require.Error(t, err)

	_, err = parseJobId("112")
	require.Error(t, err, "a name without the .img suffix is not a job image")
}

func TestGetModelRegistry_IsSingleton(t *testing.T) {
	require.Same(t, GetModelRegistry(), GetModelRegistry())
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./job/... -run 'TestRead|TestParseJobId|TestGetModelRegistry' -v
```

Expected: FAIL — `undefined: Read`, `undefined: parseJobId`, `undefined: GetModelRegistry`.

- [ ] **Step 3: Write `job/registry.go`**

```go
package job

import (
	"atlas-data/document"
	"sync"
)

var (
	mmReg  *document.Registry[string, RestModel]
	mmOnce sync.Once
)

func GetModelRegistry() *document.Registry[string, RestModel] {
	mmOnce.Do(func() {
		mmReg = document.NewRegistry[string, RestModel]()
	})
	return mmReg
}
```

- [ ] **Step 4: Write `job/reader.go`**

```go
package job

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// parseJobId derives the job id from a Skill.wz image name. The name reaches
// this helper as the root <imgdir name="…"> attribute (e.g. "112.img"), which
// is the mapping FR-2.2 requires: the job id is read off the image, never
// derived by dividing skill ids. Duplicated rather than exported from `skill`
// so that `job` and `skill` stay dependency-free in this direction (D1 — the
// list resource makes `job` import `skill`, so `skill` must not import `job`).
func parseJobId(name string) (uint32, error) {
	baseName := filepath.Base(name)
	if !strings.HasSuffix(baseName, ".img") {
		return 0, fmt.Errorf("file does not match expected format: %s", name)
	}
	idStr := strings.TrimSuffix(baseName, ".img")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// Read produces the JOB document for one Skill.wz image: exactly one model for
// a per-job (numeric) image, none for a non-numeric one such as MobSkill.img
// or BFSkill.img.
//
// Two deliberate divergences from skill.Read (design D5):
//   - A non-numeric image yields an empty slice and NO error, so the new
//     registration pass adds no `register MobSkill.img.xml` warn noise to the
//     SKILL worker (see data/workers/walk.go:45-47).
//   - A missing or empty `skill` child yields a model with an empty skill
//     list. FR-2.4 requires "the job exists with zero skills" to be
//     representable and distinguishable from "the job is absent".
//
// Skill ids are emitted in WZ document order, not sorted (FR-1.2): the order is
// deterministic per archive, so re-ingest is byte-stable and baseline dumps do
// not churn.
//
// ctx is accepted for signature parity with skill.Read and the shared
// Register plumbing; the JOB reader itself needs no tenant.
func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			jobId, err := parseJobId(exml.Name)
			if err != nil {
				// Not a per-job image (MobSkill.img, BFSkill.img, ...). FR-2.3.
				return model.FixedProvider([]RestModel{})
			}

			skills := make([]uint32, 0)
			if ssxml, err := exml.ChildByName("skill"); err == nil {
				for _, sxml := range ssxml.ChildNodes {
					skillId, err := strconv.ParseUint(sxml.Name, 10, 32)
					if err != nil {
						continue
					}
					skills = append(skills, uint32(skillId))
				}
			}
			l.Debugf("Read [%d] skills for job [%d].", len(skills), jobId)

			return model.FixedProvider([]RestModel{{Id: jobId, Skills: skills}})
		}
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./job/... -run 'TestRead|TestParseJobId|TestGetModelRegistry' -v
```

Expected: PASS (all 8 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/job/registry.go \
        services/atlas-data/atlas.com/data/job/reader.go \
        services/atlas-data/atlas.com/data/job/reader_test.go
git commit -m "feat(task-185): add JOB document reader and model registry"
```

---

## Task 2: Storage-backed job processor and resource wiring

**Files:**
- Modify: `services/atlas-data/atlas.com/data/job/processor.go` (full rewrite)
- Modify: `services/atlas-data/atlas.com/data/job/resource.go`
- Modify: `services/atlas-data/atlas.com/data/job/mock/processor.go`
- Modify: `services/atlas-data/atlas.com/data/main.go:184`
- Test: `services/atlas-data/atlas.com/data/job/processor_test.go` (full rewrite)
- Test: `services/atlas-data/atlas.com/data/job/resource_test.go` (full rewrite)

**Interfaces:**
- Consumes: `job.Read` and `job.GetModelRegistry` (Task 1); `document.NewStorage`, `document.Storage[string, RestModel]` (`GetById`, `Add`, `AllPagedProvider`); `database.ExecuteTransaction`.
- Produces:
  - `func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel]`
  - `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`
  - `type Processor interface { GetSkillsForJob(jobId uint32) (RestModel, bool); Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error; RegisterJob(path string) (int, error) }`
  - `func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer`
  - Test helpers in package `job`: `setupResourceTestDB(t) *gorm.DB`, `setupTestRouter(db) *mux.Router`, `seedJob(t, db, tenantId, region, major, minor, m)`, `setTenantHeaders(req, tenantId, region, major, minor)`, `type testDocumentEntity`, `type testServerInfo`.

**Background.** `RegisterJob` returns `(int, error)` — the count of documents written — so the SKILL worker (Task 4) can emit the D12 observability line without the processor holding mutable state. It therefore does **not** match `workers.RegisterFunc` (`func(string) error`) directly; Task 4 adapts it.

`Storage.ByIdProvider` (`document/storage.go:28-64`) already gives registry cache → tenant rows → canonical-tenant fallback keyed on `canonical.TenantId(region, major, minor)`, satisfying FR-1.3. Any error — including `gorm.ErrRecordNotFound` — collapses to `ok=false` → HTTP 404 (FR-3.2). `rest.ParseJobId` (`rest/`, `ParseJobId` at line 58) still 400s a non-numeric path segment.

The list endpoint (Task 3) reads storage directly in the handler, matching `commodity/resource.go:33-55` and `skill/resource.go:49`; it is deliberately not added to `Processor`.

**Test-isolation warning.** `GetModelRegistry()` is a package-level `sync.Once` singleton keyed by `tenant.Model` (`document/registry.go:10-13`), and it persists across tests in the package. Every test **must** use a fresh random tenant UUID so a previous test's cached entry cannot satisfy a lookup.

- [ ] **Step 1: Write the failing processor tests**

Replace `services/atlas-data/atlas.com/data/job/processor_test.go` entirely:

```go
package job

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T, tenantId uuid.UUID, region string, major, minor uint16) context.Context {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

func TestGetSkillsForJob_ReadsSeededDocument(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	ctx := testCtx(t, tenantId, "GMS", 83, 1)

	_, err := NewStorage(l, db).Add(ctx)(RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})()
	require.NoError(t, err)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(112)
	require.True(t, ok)
	require.Equal(t, uint32(112), got.Id)
	require.Equal(t, []uint32{1121000, 1121001}, got.Skills)
}

func TestGetSkillsForJob_AbsentJobIsNotOk(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(99999)
	require.False(t, ok)
	require.Equal(t, uint32(99999), got.Id)
	require.Empty(t, got.Skills)
}

// Job id 0 (Beginner) is a legitimate document_id — DbStorage.Add derives it by
// strconv.Atoi on GetID() (document/db_storage.go:133), so 0 must round-trip.
func TestGetSkillsForJob_JobIdZeroRoundTrips(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	_, err := NewStorage(l, db).Add(ctx)(RestModel{Id: 0, Skills: []uint32{1000, 1001}})()
	require.NoError(t, err)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(0)
	require.True(t, ok)
	require.Equal(t, uint32(0), got.Id)
	require.Equal(t, []uint32{1000, 1001}, got.Skills)
}

// The headline PRD acceptance criterion: two tenants on different versions get
// different skill lists for the same job id.
func TestGetSkillsForJob_DivergesByTenantVersion(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()

	oldCtx := testCtx(t, uuid.New(), "GMS", 61, 1)
	newCtx := testCtx(t, uuid.New(), "GMS", 95, 1)

	_, err := NewStorage(l, db).Add(oldCtx)(RestModel{Id: 510, Skills: []uint32{5101000, 5101001}})()
	require.NoError(t, err)
	_, err = NewStorage(l, db).Add(newCtx)(RestModel{Id: 510, Skills: []uint32{5101000}})()
	require.NoError(t, err)

	oldGot, ok := NewProcessor(l, oldCtx, db).GetSkillsForJob(510)
	require.True(t, ok)
	newGot, ok := NewProcessor(l, newCtx, db).GetSkillsForJob(510)
	require.True(t, ok)

	require.Equal(t, []uint32{5101000, 5101001}, oldGot.Skills)
	require.Equal(t, []uint32{5101000}, newGot.Skills)
	require.NotEqual(t, oldGot.Skills, newGot.Skills)
}

func TestRegisterJob_WritesOneDocumentAndReturnsCount(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	path := writeTempImage(t, "112.img.xml", jobImageXML)
	n, err := NewProcessor(l, ctx, db).RegisterJob(path)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(112)
	require.True(t, ok)
	require.Equal(t, []uint32{1121000, 1121001, 1121002}, got.Skills)
}

func TestRegisterJob_NonNumericImageWritesNothing(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	path := writeTempImage(t, "MobSkill.img.xml", mobSkillImageXML)
	n, err := NewProcessor(l, ctx, db).RegisterJob(path)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
```

- [ ] **Step 2: Write the failing resource tests**

Replace `services/atlas-data/atlas.com/data/job/resource_test.go` entirely:

```go
package job

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInfo struct{}

func (t testServerInfo) GetVersion() string { return "1.0.0" }
func (t testServerInfo) GetURI() string     { return "/api/data/" }
func (t testServerInfo) GetPrefix() string  { return "/api/data/" }
func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }

// testDocumentEntity mirrors document.Entity without the PostgreSQL-specific
// column defaults, so it can be AutoMigrated onto sqlite. Copied from
// skill/resource_test.go:28.
type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e testDocumentEntity) TableName() string { return "documents" }

func setupResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(logrus.StandardLogger(), logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		}),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testDocumentEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func setupTestRouter(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(db)(testServerInfo{})(router, l)
	return router
}

// seedJob writes a JOB document through the real storage path, so the stored
// `content` is exactly what production writes.
func seedJob(t *testing.T, db *gorm.DB, tenantId uuid.UUID, region string, major, minor uint16, m RestModel) {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	_, err = NewStorage(l, db).Add(ctx)(m)()
	require.NoError(t, err)
}

func setTenantHeaders(req *http.Request, tenantId uuid.UUID, region string, major, minor uint16) {
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", region)
	req.Header.Set("MAJOR_VERSION", strconv.Itoa(int(major)))
	req.Header.Set("MINOR_VERSION", strconv.Itoa(int(minor)))
}

type singleJobResponse struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Skills []uint32 `json:"skills"`
		} `json:"attributes"`
		Relationships json.RawMessage `json:"relationships"`
	} `json:"data"`
}

func getJobSkills(t *testing.T, db *gorm.DB, path string, tenantId uuid.UUID, region string, major, minor uint16) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	setTenantHeaders(req, tenantId, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter(db).ServeHTTP(rr, req)
	return rr
}

func TestGetJobSkills_Found(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})

	rr := getJobSkills(t, db, "/data/jobs/112/skills", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	var body singleJobResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "jobs", body.Data.Type)
	require.Equal(t, "112", body.Data.Id)
	require.Equal(t, []uint32{1121000, 1121001}, body.Data.Attributes.Skills)
	// PRD §5 pins this response shape as unchanged: no relationships block.
	require.Nil(t, body.Data.Relationships)
}

func TestGetJobSkills_NotFoundWhenAbsentForTenant(t *testing.T) {
	db := setupResourceTestDB(t)
	rr := getJobSkills(t, db, "/data/jobs/99999/skills", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// FR-3.2: "unknown job id" and "job absent from this tenant's version" are
// deliberately the same 404. Job 112 exists for the v95 tenant, not the v61 one.
func TestGetJobSkills_NotFoundForVersionWithoutTheJob(t *testing.T) {
	db := setupResourceTestDB(t)
	newTenant := uuid.New()
	oldTenant := uuid.New()
	seedJob(t, db, newTenant, "GMS", 95, 1, RestModel{Id: 112, Skills: []uint32{1121000}})

	require.Equal(t, http.StatusOK, getJobSkills(t, db, "/data/jobs/112/skills", newTenant, "GMS", 95, 1).Code)
	require.Equal(t, http.StatusNotFound, getJobSkills(t, db, "/data/jobs/112/skills", oldTenant, "GMS", 61, 1).Code)
}

func TestGetJobSkills_TwoTenantsDifferentVersionsDifferentSkills(t *testing.T) {
	db := setupResourceTestDB(t)
	oldTenant := uuid.New()
	newTenant := uuid.New()
	seedJob(t, db, oldTenant, "GMS", 61, 1, RestModel{Id: 510, Skills: []uint32{5101000, 5101001}})
	seedJob(t, db, newTenant, "GMS", 95, 1, RestModel{Id: 510, Skills: []uint32{5101000}})

	var oldBody, newBody singleJobResponse
	rr := getJobSkills(t, db, "/data/jobs/510/skills", oldTenant, "GMS", 61, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &oldBody))

	rr = getJobSkills(t, db, "/data/jobs/510/skills", newTenant, "GMS", 95, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &newBody))

	require.Equal(t, []uint32{5101000, 5101001}, oldBody.Data.Attributes.Skills)
	require.Equal(t, []uint32{5101000}, newBody.Data.Attributes.Skills)
}

func TestGetJobSkills_BadRequest(t *testing.T) {
	db := setupResourceTestDB(t)
	rr := getJobSkills(t, db, "/data/jobs/notanumber/skills", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
```

- [ ] **Step 3: Run both test files to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./job/... 2>&1 | head -30
```

Expected: FAIL to compile — `NewProcessor` takes 2 args not 3, `NewStorage` undefined, `RegisterJob` undefined, `InitResource` takes 1 arg not 2.

- [ ] **Step 4: Rewrite `job/processor.go`**

```go
package job

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"strconv"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	GetSkillsForJob(jobId uint32) (RestModel, bool)
	Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error
	// RegisterJob reads one Skill.wz per-job image and returns the number of
	// JOB documents written (0 for a non-numeric image such as MobSkill.img).
	// The count feeds the SKILL worker's ingest observability (design D12).
	RegisterJob(path string) (int, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "JOB")
}

// GetSkillsForJob resolves the tenant's JOB document. Storage.ByIdProvider
// supplies the registry cache, the tenant rows, and the canonical-tenant
// fallback keyed on canonical.TenantId(region, major, minor) (FR-1.3). Any
// error — including gorm.ErrRecordNotFound — collapses to ok=false, so
// "unknown job id" and "job absent from this tenant's version" are the same
// 404 (FR-3.2).
func (p *ProcessorImpl) GetSkillsForJob(jobId uint32) (RestModel, bool) {
	m, err := NewStorage(p.l, p.db).GetById(p.ctx)(strconv.Itoa(int(jobId)))
	if err != nil {
		p.l.WithError(err).Debugf("Unable to locate JOB document [%d].", jobId)
		return RestModel{Id: jobId, Skills: []uint32{}}, false
	}
	return m, true
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error {
	ms, err := r()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if _, err = s.Add(p.ctx)(m)(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) RegisterJob(path string) (int, error) {
	written := 0
	err := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		ms, err := Read(p.l)(p.ctx)(xml.FromPathProvider(path))()
		if err != nil {
			return err
		}
		if err = p.Register(NewStorage(p.l, tx), model.FixedProvider(ms)); err != nil {
			return err
		}
		written = len(ms)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return written, nil
}
```

- [ ] **Step 5: Update `job/resource.go` to take the db handle**

Replace the whole file (the `GET /data/jobs` route arrives in Task 3):

```go
package job

import (
	"atlas-data/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/jobs").Subrouter()
			r.HandleFunc("/{jobId}/skills",
				registerGet("get_job_skills", handleGetJobSkills(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetJobSkills(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseJobId(d.Logger(), func(jobId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, ok := NewProcessor(d.Logger(), d.Context(), db).GetSkillsForJob(jobId)
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(m)
			}
		})
	}
}
```

- [ ] **Step 6: Update `job/mock/processor.go`**

```go
package mock

import (
	"atlas-data/document"
	"atlas-data/job"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetSkillsForJobFunc func(jobId uint32) (job.RestModel, bool)
	RegisterFunc        func(s *document.Storage[string, job.RestModel], r model.Provider[[]job.RestModel]) error
	RegisterJobFunc     func(path string) (int, error)
}

var _ job.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetSkillsForJob(jobId uint32) (job.RestModel, bool) {
	if m.GetSkillsForJobFunc != nil {
		return m.GetSkillsForJobFunc(jobId)
	}
	return job.RestModel{}, false
}

func (m *ProcessorMock) Register(s *document.Storage[string, job.RestModel], r model.Provider[[]job.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterJob(path string) (int, error) {
	if m.RegisterJobFunc != nil {
		return m.RegisterJobFunc(path)
	}
	return 0, nil
}
```

- [ ] **Step 7: Update `main.go` route wiring**

In `services/atlas-data/atlas.com/data/main.go`, change line 184 from:

```go
		AddRouteInitializer(job.InitResource(GetServer())).
```

to:

```go
		AddRouteInitializer(job.InitResource(db)(GetServer())).
```

- [ ] **Step 8: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go build ./... && go test ./job/... -v
```

Expected: PASS. `go build ./...` must also be clean (it catches the `main.go` wiring).

- [ ] **Step 9: Commit**

```bash
git add services/atlas-data/atlas.com/data/job/ services/atlas-data/atlas.com/data/main.go
git commit -m "feat(task-185): read job skills from tenant JOB documents"
```

---

## Task 3: `GET /data/jobs` compound list endpoint

**Files:**
- Create: `services/atlas-data/atlas.com/data/job/list_rest.go`
- Modify: `services/atlas-data/atlas.com/data/job/resource.go`
- Test: `services/atlas-data/atlas.com/data/job/rest_test.go` (new)
- Test: `services/atlas-data/atlas.com/data/job/resource_test.go` (append)

**Interfaces:**
- Consumes: `job.NewStorage` (Task 2), `skill.NewStorage` + `skill.RestModel` (`atlas-data/skill`), `paginate.ParseParams` / `paginate.EnvelopeFor` / `paginate.DefaultPageSize` (50) / `paginate.MaxPageSize` (250), `server.MarshalPaginatedResponse`.
- Produces:
  - `type ListRestModel struct { Id uint32; Skills []uint32; resolved []skill.RestModel }`
  - `func ListFrom(m RestModel) ListRestModel`
  - `func ListFromAll(ms []RestModel) []ListRestModel`
  - `func WithResolvedSkills(items []ListRestModel, byId map[uint32]skill.RestModel) []ListRestModel`
  - `func includesSkills(query url.Values) bool`

**Background — verified api2go behaviour (v1.0.4).** `marshalSlice` calls `GetReferencedStructs()` unconditionally on any element implementing `MarshalIncludedRelations`, then `filterDuplicates` dedupes by (type, id), and finally `if len(includedElements) > 0 { result.Included = … }` (`jsonapi/marshal.go:186-208`). So:
- an empty `resolved` slice produces **no** `included` key — which is exactly D3's "linkage always, `included` only when requested";
- dedup across jobs is api2go's job, not ours.

`MarshalPaginatedResponse` (`libs/atlas-rest/server/paginated_response.go:24-31`) calls `jsonapi.MarshalToStruct` and then attaches `Meta`/`Links`, so `Included` survives.

This is why `RestModel` must stay relationship-free: `DbStorage.Add` marshals the **same** model type into the stored `content` column (`document/db_storage.go:123-130`), and there is no "only when asked" gate at that layer.

**Design divergence from the `shops` reference (FR-4.3).** In `shops/resource.go:75-84` the linkage itself is a function of the `include` decorator and disappears without it. Here `GetReferencedIDs` reads the stored `Skills` id list, so linkage is unconditional; only `GetReferencedStructs` is gated.

- [ ] **Step 1: Write the failing FR-4.4 stored-content guard**

Create `services/atlas-data/atlas.com/data/job/rest_test.go`:

```go
package job

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestStoredContentCarriesNoRelationships is the FR-4.4 / D2 regression guard.
// document.DbStorage.Add persists json.Marshal(jsonapi.MarshalToStruct(m, …))
// (document/db_storage.go:123-130), and api2go populates Document.Included for
// ANY model implementing MarshalIncludedRelations — there is no "only when
// asked" gate at that layer. If someone later merges RestModel and
// ListRestModel into one type, relationship/included data starts leaking into
// the stored `content` column and this test fails.
func TestStoredContentCarriesNoRelationships(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	_, err = NewStorage(l, db).Add(ctx)(RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})()
	require.NoError(t, err)

	var raw string
	require.NoError(t, db.Raw(
		`SELECT content FROM documents WHERE tenant_id = ? AND type = 'JOB' AND document_id = 112`,
		tenantId.String(),
	).Scan(&raw).Error)

	require.NotEmpty(t, raw)
	require.Contains(t, raw, `"skills"`)
	require.False(t, strings.Contains(raw, `"relationships"`), "stored content leaked relationships: %s", raw)
	require.False(t, strings.Contains(raw, `"included"`), "stored content leaked included: %s", raw)
}
```

- [ ] **Step 2: Write the failing list-endpoint tests**

Append to `services/atlas-data/atlas.com/data/job/resource_test.go`, and add `"sort"` plus `dataskill "atlas-data/skill"` to that file's import block:

```go
// seedSkill writes a SKILL document through the real skill storage path, so
// include=skills resolves against production-shaped rows.
func seedSkill(t *testing.T, db *gorm.DB, tenantId uuid.UUID, region string, major, minor uint16, id uint32, name string) {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	_, err = dataskill.NewStorage(l, db).Add(ctx)(dataskill.RestModel{Id: id, Name: name})()
	require.NoError(t, err)
}

type jobsListResponse struct {
	Data []struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Skills []uint32 `json:"skills"`
		} `json:"attributes"`
		Relationships struct {
			Skills struct {
				Data []struct {
					Type string `json:"type"`
					Id   string `json:"id"`
				} `json:"data"`
			} `json:"skills"`
		} `json:"relationships"`
	} `json:"data"`
	Included []struct {
		Type string `json:"type"`
		Id   string `json:"id"`
	} `json:"included"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func getJobs(t *testing.T, db *gorm.DB, path string, tenantId uuid.UUID, region string, major, minor uint16) (*httptest.ResponseRecorder, jobsListResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	setTenantHeaders(req, tenantId, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter(db).ServeHTTP(rr, req)
	var body jobsListResponse
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body), "body: %s", rr.Body.String())
	}
	return rr, body
}

func TestGetJobs_ListsTenantJobsWithLinkageAndNoIncluded(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 100, Skills: []uint32{1001000}})
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})

	rr, body := getJobs(t, db, "/data/jobs", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	require.Len(t, body.Data, 2)
	require.Equal(t, 2, body.Meta.Total)
	require.Equal(t, "jobs", body.Data[0].Type)

	// FR-4.3: linkage is always present, derived from the stored id list.
	byId := map[string][]string{}
	for _, d := range body.Data {
		for _, ref := range d.Relationships.Skills.Data {
			require.Equal(t, "skills", ref.Type)
			byId[d.Id] = append(byId[d.Id], ref.Id)
		}
	}
	require.Equal(t, []string{"1001000"}, byId["100"])
	require.Equal(t, []string{"1121000", "1121001"}, byId["112"])

	// NFR §8: no include => no skill drain => no `included` key.
	require.Empty(t, body.Included)
	require.NotContains(t, rr.Body.String(), `"included"`)
}

func TestGetJobs_IncludeSkillsPopulatesIncludedDeduped(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	// 1001000 is shared by both jobs; it must appear once in `included`.
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 100, Skills: []uint32{1001000}})
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 110, Skills: []uint32{1001000, 1101000}})
	seedSkill(t, db, tenantId, "GMS", 83, 1, 1001000, "Power Strike")
	seedSkill(t, db, tenantId, "GMS", 83, 1, 1101000, "Sword Mastery")

	rr, body := getJobs(t, db, "/data/jobs?include=skills", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	ids := make([]string, 0, len(body.Included))
	for _, inc := range body.Included {
		require.Equal(t, "skills", inc.Type)
		ids = append(ids, inc.Id)
	}
	sort.Strings(ids)
	require.Equal(t, []string{"1001000", "1101000"}, ids)
}

func TestGetJobs_EmptyForTenantWithNoJobs(t *testing.T) {
	db := setupResourceTestDB(t)
	rr, body := getJobs(t, db, "/data/jobs", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Empty(t, body.Data)
	require.Equal(t, 0, body.Meta.Total)
}

func TestGetJobs_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	for _, id := range []uint32{100, 110, 111, 112} {
		seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: id, Skills: []uint32{id * 10000}})
	}

	rr, body := getJobs(t, db, "/data/jobs?page[number]=1&page[size]=2", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, body.Data, 2)
	require.Equal(t, 4, body.Meta.Total)
	require.Equal(t, "100", body.Data[0].Id, "AllPaged orders by document_id")
	require.Equal(t, "110", body.Data[1].Id)
}

func TestGetJobs_RejectsBadPageParams(t *testing.T) {
	db := setupResourceTestDB(t)
	rr, _ := getJobs(t, db, "/data/jobs?page[size]=notanumber", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./job/... 2>&1 | head -30
```

Expected: FAIL — `/data/jobs` is not routed, so every `TestGetJobs_*` case sees 404 instead of its expected status.

- [ ] **Step 4: Write `job/list_rest.go`**

```go
package job

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"

	"atlas-data/skill"
)

// ListRestModel is the GET /data/jobs projection of RestModel. It is NEVER
// persisted: document.DbStorage.Add marshals the model type it is handed
// (document/db_storage.go:123), and api2go writes `relationships` — and
// `included`, once any referenced struct is attached — for any model
// implementing the relationship interfaces (jsonapi/marshal.go:186-208).
// Keeping the persisted type (RestModel) relationship-free is what stops that
// leaking into the stored `content` column (FR-4.4, design D2), and what keeps
// GET /data/jobs/{jobId}/skills at the shape PRD §5 pins as unchanged.
//
// It deliberately implements only the marshal side. The Unmarshal* /
// SetToManyReferenceIDs / SetReferencedStructs counterparts that
// shops.RestModel carries are omitted: nothing ever unmarshals this type, and
// adding them would imply it is persistable or inbound, which it is not.
type ListRestModel struct {
	Id     uint32   `json:"-"`
	Skills []uint32 `json:"skills"`
	// resolved holds the full skill resources emitted into `included`. It is
	// unexported, so encoding/json never sees it; it is populated only when the
	// request asked for include=skills (design D3).
	resolved []skill.RestModel
}

func (r ListRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }
func (r ListRestModel) GetName() string { return "jobs" }

// GetReferences to satisfy jsonapi.MarshalReferences.
func (r ListRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "skills",
			Name: "skills",
		},
	}
}

// GetReferencedIDs to satisfy jsonapi.MarshalLinkedRelations. Derived from the
// stored id list, so linkage is ALWAYS present (FR-4.3) — a deliberate
// divergence from the `shops` reference implementation, where linkage itself is
// a function of the include decorator (shops/resource.go:75-84).
func (r ListRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := make([]jsonapi.ReferenceID, 0, len(r.Skills))
	for _, id := range r.Skills {
		result = append(result, jsonapi.ReferenceID{
			ID:   strconv.Itoa(int(id)),
			Type: "skills",
			Name: "skills",
		})
	}
	return result
}

// GetReferencedStructs to satisfy jsonapi.MarshalIncludedRelations. Empty
// unless the handler resolved skills, and api2go omits the `included` key
// entirely when nothing is returned (jsonapi/marshal.go:203).
func (r ListRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := make([]jsonapi.MarshalIdentifier, 0, len(r.resolved))
	for _, s := range r.resolved {
		result = append(result, s)
	}
	return result
}

// ListFrom is the only bridge from the persisted type to the list projection.
func ListFrom(m RestModel) ListRestModel {
	return ListRestModel{Id: m.Id, Skills: m.Skills}
}

func ListFromAll(ms []RestModel) []ListRestModel {
	out := make([]ListRestModel, 0, len(ms))
	for _, m := range ms {
		out = append(out, ListFrom(m))
	}
	return out
}

// WithResolvedSkills attaches the full skill resources for each job's id list.
// Ids with no matching skill document are skipped — the linkage still names
// them, which is the JSON:API-correct way to express "referenced but not
// included".
func WithResolvedSkills(items []ListRestModel, byId map[uint32]skill.RestModel) []ListRestModel {
	out := make([]ListRestModel, 0, len(items))
	for _, it := range items {
		resolved := make([]skill.RestModel, 0, len(it.Skills))
		for _, id := range it.Skills {
			if s, ok := byId[id]; ok {
				resolved = append(resolved, s)
			}
		}
		it.resolved = resolved
		out = append(out, it)
	}
	return out
}
```

- [ ] **Step 5: Add the list route to `job/resource.go`**

Register the route inside `InitResource`'s inner func, immediately before the existing `/{jobId}/skills` line:

```go
			r.HandleFunc("", registerGet("get_jobs", handleGetJobsRequest(db))).Methods(http.MethodGet)
```

Append the handler and its `include` helper to the same file:

```go
// includesSkills reports whether the request asked for the skills relationship
// to be materialized. Neither server.MarshalResponse nor
// server.MarshalPaginatedResponse knows about `include` —
// jsonapi.FilterSparseFields handles `fields[type]` only — so the handler
// parses it itself, exactly as shops/resource.go:77 does.
func includesSkills(query url.Values) bool {
	for _, include := range query["include"] {
		for _, part := range strings.Split(include, ",") {
			if strings.TrimSpace(part) == "skills" {
				return true
			}
		}
	}
	return false
}

func handleGetJobsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewStorage(d.Logger(), db).AllPagedProvider(d.Context())(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve jobs.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			items := ListFromAll(paged.Items)
			if includesSkills(query) {
				// One drain, never N per-id lookups: a 50-job page can reference
				// ~3,000 skills, and the registry cache only helps after the
				// first miss (design D4). When include is absent the skill
				// storage is not touched at all (NFR §8).
				all, err := skill.NewStorage(d.Logger(), db).DrainAllProvider(d.Context())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve skills for include=skills.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				byId := make(map[uint32]skill.RestModel, len(all))
				for _, s := range all {
					byId[s.Id] = s
				}
				items = WithResolvedSkills(items, byId)
			}

			envelope := paginate.EnvelopeFor(model.Paged[ListRestModel]{
				Items: items,
				Total: paged.Total,
				Page:  paged.Page,
			})
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]ListRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(items, envelope, r)
		}
	}
}
```

Extend `job/resource.go`'s import block to:

```go
import (
	"atlas-data/rest"
	"atlas-data/skill"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go build ./... && go test ./job/... -v
```

Expected: PASS. If `paginate.EnvelopeFor` produces wrong links, check that `paged.Page` was carried over verbatim — the envelope's link builder needs the served page's number/size, which `DbStorage.AllPaged` sets (`document/db_storage.go:76`).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/job/
git commit -m "feat(task-185): add GET /data/jobs compound list endpoint"
```

---

## Task 4: Ingest — write JOB documents from the SKILL worker

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/workers/skill.go`
- Test: `services/atlas-data/atlas.com/data/data/workers/skill_test.go` (new)

**Interfaces:**
- Consumes: `job.NewProcessor(l, ctx, db).RegisterJob(path) (int, error)` (Task 2), `workers.registerAllInDirectory`, `workers.RegisterFunc`.
- Produces: unexported `func countingRegister(total *int, rf func(path string) (int, error)) RegisterFunc`, unexported `func logJobDocCount(l logrus.FieldLogger, written int)`.

**Background.** FR-2.1 requires no new entry in `data.Workers`: the JOB pass folds into the existing `SKILL` worker exactly as `mobskill` already does (`data/workers/skill.go:52`). FR-2.5 (GMS v12's monolithic `Data.wz`) is inherited for free — the pass reads the same serialized `root/Skill.wz` tree the skill pass reads, and `runtime.go` has already resolved the monolith sub-view by then.

`registerAllInDirectory` (`data/workers/walk.go:34-50`) logs and swallows per-file errors, so a single bad image cannot abort the walk; only the directory walk itself can fail.

D12: a `Skill.wz` ingest that yields zero `JOB` documents must log at warn or above (NFR §8) — silent success is the exact failure mode the rejected transitional fallback would have hidden.

- [ ] **Step 1: Write the failing worker-helper tests**

Create `services/atlas-data/atlas.com/data/data/workers/skill_test.go`:

```go
package workers

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestCountingRegister_SumsWrittenDocuments(t *testing.T) {
	total := 0
	rf := countingRegister(&total, func(path string) (int, error) {
		if path == "MobSkill.img.xml" {
			return 0, nil
		}
		return 1, nil
	})

	require.NoError(t, rf("112.img.xml"))
	require.NoError(t, rf("MobSkill.img.xml"))
	require.NoError(t, rf("100.img.xml"))
	require.Equal(t, 2, total)
}

func TestCountingRegister_PropagatesErrorAndAddsNothing(t *testing.T) {
	total := 0
	rf := countingRegister(&total, func(path string) (int, error) {
		return 0, errors.New("boom")
	})

	require.Error(t, rf("112.img.xml"))
	require.Equal(t, 0, total)
}

func TestLogJobDocCount_WarnsOnZero(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	logJobDocCount(l, 0)

	require.Len(t, hook.Entries, 2)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Equal(t, logrus.WarnLevel, hook.Entries[1].Level)
}

func TestLogJobDocCount_NoWarnWhenDocumentsWritten(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	logJobDocCount(l, 82)

	require.Len(t, hook.Entries, 1)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Contains(t, hook.Entries[0].Message, "written=82")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./data/workers/... -run 'TestCountingRegister|TestLogJobDocCount' -v
```

Expected: FAIL — `undefined: countingRegister`, `undefined: logJobDocCount`.

- [ ] **Step 3: Add the helpers and the registration pass to `data/workers/skill.go`**

Add `"atlas-data/job"` to the import block, then append these helpers to the file:

```go
// countingRegister adapts job.Processor.RegisterJob — which returns the number
// of documents it wrote — to the RegisterFunc shape registerAllInDirectory
// expects, accumulating the total into *total. A failing register contributes
// nothing to the count.
func countingRegister(total *int, rf func(path string) (int, error)) RegisterFunc {
	return func(path string) error {
		n, err := rf(path)
		if err != nil {
			return err
		}
		*total += n
		return nil
	}
}

// logJobDocCount emits the JOB-document ingest summary. A Skill.wz pass that
// produced no JOB documents leaves /data/jobs empty for the tenant, so it
// escalates to warn: silent success here is exactly the failure mode the
// rejected transitional fallback would have hidden (PRD §8 Observability).
func logJobDocCount(l logrus.FieldLogger, written int) {
	l.Infof("job documents: written=%d", written)
	if written == 0 {
		l.Warnf("Skill.wz ingest produced no JOB documents; /data/jobs will be empty for this tenant")
	}
}
```

Then, in `Skill.Run`, insert the JOB pass immediately after the existing `mobskill` registration (currently `skill.go:52-54`) and before the icon loop:

```go
	// FR-2.1: the JOB pass folds into the SKILL worker rather than adding a
	// data.Workers entry, exactly as mobskill does. It re-reads the same
	// serialized Skill.wz tree, so monolithic-archive tenants (GMS v12's
	// all-in-one Data.wz) are handled by the runtime's sub-view with no
	// monolith-specific code (FR-2.5).
	jobDocs := 0
	jobRegister := countingRegister(&jobDocs, job.NewProcessor(l, ctx, db).RegisterJob)
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), jobRegister); err != nil {
		return err
	}
	logJobDocCount(l, jobDocs)
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go build ./... && go test ./data/workers/... -v
```

Expected: PASS, including the pre-existing `TestRegisteredSize` / `TestRegisteredUniqueNames` structural tests — the `data.Workers` list must be unchanged (FR-2.1). If `TestRegisteredSize` fails, a new worker entry was added by mistake; remove it.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/workers/
git commit -m "feat(task-185): write JOB documents during Skill.wz ingest"
```

---

## Task 5: Verify JOB documents ride baseline publish and tenant purge

**Files:**
- Modify: `services/atlas-data/atlas.com/data/baseline/dump_test.go`
- Modify: `services/atlas-data/atlas.com/data/tenantpurge/purge_test.go`

**Interfaces:**
- Consumes: `baseline.DumpTables`, `baseline.copyOutSQL`, `tenantpurge.Purge`, `tenantpurge.PurgeTables`.
- Produces: nothing — this task verifies FR-1.4 rather than asserting it (design D9). **No production code changes.**

**Background — what was already read.** `baseline/dump.go:20-27` lists `"documents"` as a whole table; there is no per-`type` filter anywhere in `dump.go`, `publish.go`, or `restore.go`. `copyOutSQL` (`baseline/publish.go:186-190`) emits `COPY (SELECT <cols> FROM <table> WHERE tenant_id = '<uuid>' ORDER BY <key>) TO STDOUT (FORMAT binary)` — filtered on tenant only. `tenantpurge/purge.go:21` likewise lists `"documents"` whole-table. So the claim is true; these tests pin it so a future per-type filter cannot silently drop `JOB`.

- [ ] **Step 1: Write the baseline test**

Append to `services/atlas-data/atlas.com/data/baseline/dump_test.go`, and add `"strings"` to its import block (it currently imports only `"testing"`):

```go
// TestCopyOutSQLDocumentsHasNoTypeFilter pins design D9 / FR-1.4: the documents
// dump is whole-table, filtered on tenant only. A future per-`type` filter
// would silently drop JOB rows (and every other document type added after this
// test was written) from every published baseline.
func TestCopyOutSQLDocumentsHasNoTypeFilter(t *testing.T) {
	sql := copyOutSQL("documents", []string{"tenant_id", "type", "document_id", "content"}, "GMS", 83, 1)
	if strings.Contains(sql, "type =") || strings.Contains(sql, `"type" =`) {
		t.Fatalf("documents dump gained a type filter; JOB rows would be dropped: %s", sql)
	}
	if !strings.Contains(sql, "FROM documents WHERE tenant_id = ") {
		t.Fatalf("documents dump is no longer tenant-filtered whole-table: %s", sql)
	}
}
```

- [ ] **Step 2: Write the purge test**

In `services/atlas-data/atlas.com/data/tenantpurge/purge_test.go`, inside `TestPurgeDeletesRows`, add a `JOB` row alongside the existing `'item'` row. Immediately after this existing block:

```go
	if err := db.Exec(`INSERT INTO documents (id, tenant_id, type) VALUES ('a', ?, 'item')`, id.String()).Error; err != nil {
		t.Fatal(err)
	}
```

insert:

```go
	// FR-1.4 / design D9: PurgeTables lists `documents` whole-table, so the JOB
	// type added by task-185 is purged with no per-type registration.
	if err := db.Exec(`INSERT INTO documents (id, tenant_id, type) VALUES ('b', ?, 'JOB')`, id.String()).Error; err != nil {
		t.Fatal(err)
	}
```

- [ ] **Step 3: Run the tests**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-data/atlas.com/data" && go test ./baseline/... ./tenantpurge/... -run 'TestCopyOutSQLDocumentsHasNoTypeFilter|TestPurgeDeletesRows' -v
```

Expected: PASS. These tests are green on first run — they pin behaviour that already holds. That is the point: D9 says "verified, not assumed." If either fails, the design's D9 claim is wrong; stop and report rather than relaxing the assertion.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-data/atlas.com/data/baseline/dump_test.go \
        services/atlas-data/atlas.com/data/tenantpurge/purge_test.go
git commit -m "test(task-185): pin JOB documents in baseline dump and tenant purge"
```

---

## Task 6: Collapse `libs/atlas-constants/job`

**Files:**
- Modify: `libs/atlas-constants/job/constants.go` (1,236 → ~180 lines)
- Modify: `libs/atlas-constants/job/model.go`
- Test: `libs/atlas-constants/job/constants_test.go` (new)

**Interfaces:**
- Consumes: nothing new.
- Produces: `var Jobs map[Id]Job` (unchanged name and type, rewritten as a literal); `type Job struct { id Id; fourthJob bool }`.
- Removes: the 82 exported `Job` value vars (`Beginner`, `Warrior`, …), `Job.skills`, `Job.Skills()`, `Job.Buffs()`.

**Background — consumer audit, re-verified against the current tree.** The only caller of `Job.Skills()` repo-wide is `services/atlas-data/atlas.com/data/job/processor.go:34-35`, which Task 2 already rewrote. Every other `.Skills()` hit belongs to a different service's own model type (atlas-channel's `character.Model`, atlas-pets, atlas-consumables, atlas-messages, atlas-monsters). `Job.Buffs()` has zero callers — the `.Buffs()` hits are all atlas-buffs' own `character.Model`. `Jobs[id]` is read outside `libs/atlas-constants` by exactly two call sites, both preserved:

- `services/atlas-configurations/atlas.com/configurations/templates/characters/preset/validator.go:76`
- `services/atlas-configurations/atlas.com/configurations/tenants/characters/preset/validator.go:76`

Both use it purely as a "does this job exist" key-set check, which the literal map preserves.

**Do this task after Task 2**, or atlas-data will not compile.

**Mechanical warning.** The 82 `Job` literals span `constants.go:7-1058`, and `Jobs` at `constants.go:1060-1143` maps `<XId>: <X>`. The rewrite is: delete lines 7-1058, and rewrite each `Jobs` entry from `HeroId: Hero,` to `HeroId: {id: HeroId, fourthJob: true},`. The `fourthJob: true` marker must be carried over for exactly the 23 jobs whose `Job` literal had it. Derive that list mechanically — do not retype it from memory:

```bash
cd "$(git rev-parse --show-toplevel)"
awk '/^var [A-Z][A-Za-z0-9]* = Job\{/{name=$2} /fourthJob: true/{print name}' \
  libs/atlas-constants/job/constants.go | sort
```

That prints the 23 var names; the corresponding `Jobs` entries are the ones that get `fourthJob: true`.

- [ ] **Step 1: Write the table test**

Create `libs/atlas-constants/job/constants_test.go`:

```go
package job

import "testing"

// TestJobsTableShape guards the hand-rewritten Jobs literal (task-185 collapsed
// the 82 package-level Job value vars into inline map values). A dropped job or
// a dropped fourthJob marker is silent data loss: atlas-configurations' two
// preset validators use the key set as their "does this job exist" check, and
// atlas-character reads IsFourthJob off it.
func TestJobsTableShape(t *testing.T) {
	if got := len(Jobs); got != 82 {
		t.Fatalf("len(Jobs) = %d; want 82", got)
	}
	fourth := 0
	for id, j := range Jobs {
		if j.Id() != id {
			t.Errorf("Jobs[%d].Id() = %d; every entry must be self-keyed", id, j.Id())
		}
		if j.IsFourthJob() {
			fourth++
		}
	}
	if fourth != 23 {
		t.Fatalf("fourthJob markers = %d; want 23", fourth)
	}
}

// TestFourthJobMembership pins the exact ids, so a marker cannot migrate from
// one job to another while keeping the count correct. Populate `want` from the
// awk command in the plan's Task 6 preamble — constants.go is the authority.
func TestFourthJobMembership(t *testing.T) {
	want := fourthJobIdsUnderTest()
	if len(want) != 23 {
		t.Fatalf("want list has %d entries; the fourthJob count is 23", len(want))
	}
	for _, id := range want {
		j, ok := Jobs[id]
		if !ok {
			t.Errorf("Jobs is missing id %d", id)
			continue
		}
		if !j.IsFourthJob() {
			t.Errorf("Jobs[%d].IsFourthJob() = false; want true", id)
		}
	}
	for id, j := range Jobs {
		if !j.IsFourthJob() {
			continue
		}
		found := false
		for _, w := range want {
			if w == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Jobs[%d] is marked fourthJob but is not in the expected set", id)
		}
	}
}

// TestFromSkillIdStillResolves covers the two accessors external services use:
// atlas-character/character/processor.go:1045 calls .Id(), and
// atlas-character/skill/model.go:34 calls .IsFourthJob() (FR-5.4 — this is why
// Job stays a struct rather than collapsing to map[Id]bool).
func TestFromSkillIdStillResolves(t *testing.T) {
	j, ok := FromSkillId(1121000)
	if !ok {
		t.Fatal("FromSkillId(1121000) not ok")
	}
	if j.Id() != HeroId {
		t.Fatalf("FromSkillId(1121000).Id() = %d; want %d", j.Id(), HeroId)
	}
	if !j.IsFourthJob() {
		t.Fatal("Hero must be a fourth job")
	}
}
```

Then add the id list at the bottom of the same file, filled in from the `awk` output — map each printed var name to its `*Id` constant (e.g. `Hero` → `HeroId`):

```go
// fourthJobIdsUnderTest is the transcription of every job whose literal carried
// `fourthJob: true` before the task-185 collapse. Derived mechanically from
// constants.go, not from memory.
func fourthJobIdsUnderTest() []Id {
	return []Id{
		// … 23 entries, one per awk-reported var name …
	}
}
```

- [ ] **Step 2: Run the test against the pre-collapse file**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-constants" && go test ./job/... -run 'TestJobsTableShape|TestFourthJobMembership|TestFromSkillIdStillResolves' -v
```

Expected: PASS against the *current* (pre-collapse) file — all three assertions describe behaviour that must survive the rewrite. This is the safety net; write it before touching `constants.go`, not after. If `TestFourthJobMembership` fails now, the transcribed id list is wrong — fix it from the `awk` output before proceeding.

- [ ] **Step 3: Collapse `constants.go`**

Delete the 82 `var <Name> = Job{…}` blocks (lines 7-1058) and rewrite `Jobs` as a literal map. The resulting file is:

```go
package job

type Id uint16

// Jobs is the registry of every job id the server knows about. It carries no
// skill data: the job→skill mapping is version-varying and now lives in
// per-tenant JOB documents served by atlas-data (task-185). The key set is the
// "does this job exist" check used by atlas-configurations' preset validators.
var Jobs = map[Id]Job{
	BeginnerId: {id: BeginnerId},
	WarriorId:  {id: WarriorId},
	// … one entry per id, preserving the existing key order …
	HeroId:     {id: HeroId, fourthJob: true},
	// …
}
```

followed by the untouched `Id` constant block and `Type` constants that already live in the file. The `skill` import leaves `constants.go`; `model.go` still needs it (`mpEaterSkillIds`, `FromSkillId`, `IdFromSkillId`), so `go.mod` is unchanged (FR-5.6).

- [ ] **Step 4: Drop `skills` from `model.go`**

In `libs/atlas-constants/job/model.go`, change the struct and delete the two accessors:

```go
type Job struct {
	id        Id
	fourthJob bool
}

func (j Job) Id() Id {
	return j.id
}

func (j Job) IsFourthJob() bool {
	return j.fourthJob
}
```

Everything else in `model.go` — `IsA`, `Is`, `FromSkillId`, `IdFromSkillId`, `IsFourthJob`, `IsBeginner`, `GetType`, `IsCygnus`, `GetSkillBook`, `mpEaterSkillIds`, `MpEaterSkillId`, `FromIndex` — is unchanged, including the `skill` import.

- [ ] **Step 5: Run the module's tests**

```bash
cd "$(git rev-parse --show-toplevel)/libs/atlas-constants" && go build ./... && go vet ./... && go test -race ./...
wc -l job/constants.go
```

Expected: PASS, and a line count under 200 (acceptance criterion).

- [ ] **Step 6: Verify every dependent module still compiles**

```bash
cd "$(git rev-parse --show-toplevel)"
for m in services/atlas-data/atlas.com/data \
         services/atlas-character/atlas.com/character \
         services/atlas-skills/atlas.com/skills \
         services/atlas-configurations/atlas.com/configurations; do
  echo "== $m"; (cd "$m" && go build ./... && go test ./... >/dev/null) || echo "FAILED $m"
done
```

Expected: no `FAILED` lines. Per the acceptance criteria, atlas-skills, atlas-character, and atlas-configurations must compile and pass **with no source changes** — if one of them needs an edit, the deletion went too far; re-check against the consumer audit above before changing any consumer.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/job/
git commit -m "refactor(task-185): drop job skill lists from atlas-constants"
```

---

## Task 7: Documentation — REST reference and rollout runbook

**Files:**
- Modify: `services/atlas-data/docs/rest.md`
- Create: `docs/runbooks/job-document-backfill.md`

**Interfaces:**
- Consumes: the endpoint behaviour from Tasks 2-3.
- Produces: documentation only. No code.

**Background.** `services/atlas-data/docs/rest.md:1073` already documents `GET /api/data/jobs/{jobId}/skills`. There is **no** route conflict with the new `GET /api/data/jobs`: the ingest-job listing lives at `GET /api/data/process` (`rest.md:79`), not `/api/data/jobs`.

The runbook lands next to `docs/runbooks/canonical-version-migration.md` and `docs/runbooks/ephemeral-pr-deployments.md`.

- [ ] **Step 1: Update the REST reference**

In `services/atlas-data/docs/rest.md`, add a `### GET /api/data/jobs` section immediately **before** the existing `### GET /api/data/jobs/{jobId}/skills` section at line 1073:

````markdown
### GET /api/data/jobs

Tenant-scoped list of jobs present in the tenant's ingested data. Sourced from
`JOB` documents written by the `SKILL` ingest worker from the tenant's
`Skill.wz`, so the set reflects the tenant's client version.

#### Query Parameters

- `page[number]`, `page[size]` — pagination, consistent with `GET /api/data/skills`
  (default size 50, max 250).
- `include=skills` — materialize the referenced skill resources into `included`.

#### Response

- 200: a JSON:API compound document. Each `jobs` resource carries
  `attributes.skills` (the ordered skill id list) and a `skills` to-many
  relationship. Relationship linkage is always present; `included` appears only
  when `include=skills` is requested.
- 400 Bad Request: malformed pagination parameters.

```json
{
  "data": [
    {
      "type": "jobs",
      "id": "112",
      "attributes": { "skills": [1121000, 1121001] },
      "relationships": {
        "skills": {
          "data": [
            { "type": "skills", "id": "1121000" },
            { "type": "skills", "id": "1121001" }
          ]
        }
      }
    }
  ],
  "meta": { "total": 82 }
}
```

---
````

Then update the `GET /api/data/jobs/{jobId}/skills` section's 404 wording so it reads:

```markdown
- 404 Not Found: the job id is unknown **or** absent from this tenant's client
  version. The two cases are deliberately indistinguishable — a tenant whose
  `Skill.wz` has no image for the job has no `JOB` document for it.
```

- [ ] **Step 2: Write the rollout runbook**

Create `docs/runbooks/job-document-backfill.md`:

````markdown
# Runbook — JOB document backfill (task-185)

**Audience:** operator, at deploy time.
**Status:** required before task-185 ships to a tenant.

## Why

task-185 made the job→skill mapping tenant data instead of a compiled-in table.
`JOB` documents are written only by a `Skill.wz` ingest, so **until a tenant is
re-ingested or restored from a re-published baseline, both job endpoints 404**:

- `GET /api/data/jobs` returns an empty list
- `GET /api/data/jobs/{jobId}/skills` returns 404 for every job
- the atlas-ui Jobs page renders an empty branch rail

This is a deliberate hard cutover. A transitional fallback to the old constants
list was considered and rejected: it would have kept the deleted table alive for
another release and would have masked a failed ingest as a success.

## Scope

All 11 versions registered in `deploy/k8s/base/versions.json`:

GMS 12.1, 48.1, 61.1, 72.1, 79.1, 83.1, 84.1, 87.1, 92.1, 95.1, JMS 185.1.

## Per-version procedure

Run these in order, one version at a time. Do not batch — step 3 is the gate.

1. **Re-ingest the version's canonical dataset** from its already-uploaded WZ
   archives:

   ```
   POST /api/data/process?scope=shared
   X-Atlas-Operator: 1
   ```

2. **Poll until the job reports `succeeded`:**

   ```
   GET /api/data/process
   ```

3. **Verify — this is the gate.** For a tenant on that version:

   ```
   GET /api/data/jobs
   ```

   - `meta.total` must be `> 0`.
   - Spot-check the returned id set against the expectation table below.

   `GET /api/data/status` is **not** sufficient: it reports only an aggregate
   `documentCount` with no per-type breakdown, so it cannot distinguish "JOB
   documents were written" from "skills were written and JOB was not."

   If `meta.total` is 0, check the ingest logs for
   `Skill.wz ingest produced no JOB documents` — the worker warns explicitly on
   this case. Do not proceed to step 4.

4. **Publish the baseline** so ephemeral PR environments (baseline-only, and
   they fail fast without one) pick up the `JOB` documents:

   ```
   POST /api/data/baseline/publish
   X-Atlas-Operator: 1
   ```

## Expected job-set changes

Derived from probing `GET /api/data/skills/{id}` across the live tenants for one
representative skill per job image (task-185 design §3). The Jobs page changes
visibly on five of the ten currently-provisioned versions:

| Version | Change vs the retired floor table |
|---|---|
| GMS 48 | GM (900) + Super GM (910) **disappear** — no `9xx` skills exist at this version |
| GMS 61 | no change — GM/Super GM stay |
| GMS 72 | Maple Leaf Brigadier (800) + the whole Cygnus branch (1000) **appear** |
| GMS 79 | Maple Leaf Brigadier + Cygnus **appear** (same cause) |
| GMS 83/84/87/92/95 | no change |
| JMS 185 | Super GM (910) **disappears**; GM (900) stays |

These are correct outcomes, not ingest failures. Every one of them moves the UI
toward the tenant's actual data.

**Caveat on the table's provenance:** it probes `SKILL` documents as a *proxy*
for the presence of a per-job image, because no `JOB` document existed anywhere
when it was produced. Two ways the proxy can be wrong — a job image can exist
with zero skills (the job then appears with an empty skill list, which the Jobs
page renders as its "empty" state), and a representative skill can be absent
while its image exists. Step 3's `GET /api/data/jobs` check is the authoritative
verification; this table only sets the expectation it is checked against.

## GMS 12.1 — extra step

GMS 12.1 is registered in `versions.json` but has **no provisioned tenant and no
ingested data** in the current cluster: the live tenant list holds exactly ten
tenants (GMS 48/61/72/79/83/84/87/92/95 + JMS 185), and probing
`/api/data/skills/1000` for GMS 12.1 returns 404, as do maps, monsters,
consumables, equipment, and npcs.

It must be **provisioned and ingested**, not merely re-published, before its
step-3 verification can pass.

This is a data-state gap, not an archive-capability gap: v12's monolithic
`Data.wz` root does contain a `Skill/` directory, and task-172's live v12 ingest
ran the SKILL worker successfully (skill `1001003` "Iron Body" with full effects,
plus 175 skill icons). The categories v12 genuinely lacks are Quest, Morph,
TamingMob, and `Item/Cash`.

## Rollback

There is no data rollback: `JOB` documents are additive and no existing document
type changes. Rolling back the *code* restores the old compiled-in list and
leaves the `JOB` rows in place, harmlessly.
````

- [ ] **Step 3: Verify no absolute paths leaked into either file**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -n "/home/\|/Users/" docs/runbooks/job-document-backfill.md services/atlas-data/docs/rest.md
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-data/docs/rest.md docs/runbooks/job-document-backfill.md
git commit -m "docs(task-185): document GET /data/jobs and the JOB backfill runbook"
```

---

## Task 8: atlas-ui compound-document plumbing

**Files:**
- Modify: `services/atlas-ui/src/types/api/responses.ts`
- Modify: `services/atlas-ui/src/lib/api/client.ts`
- Modify: `services/atlas-ui/src/services/api/jobs.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useJobs.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/jobs.service.test.ts` (new)

**Interfaces:**
- Consumes: `GET /api/data/jobs` from Task 3; the existing `apiClient.get`, `ApiPagedResponse`.
- Produces:
  - `export interface JsonApiResource { type: string; id: string; attributes?: Record<string, unknown> }`
  - `ApiResponse<T>` gains `included?: JsonApiResource[]`
  - `api.getListDocument: <T>(url: string, options?: ApiRequestOptions) => Promise<ApiPagedResponse<T> & { included?: JsonApiResource[] }>`
  - `export interface JobResource { id: string; type: string; attributes: { skills: number[] } }`
  - `export interface JobsResult { jobs: JobResource[]; skillsById: Map<number, JsonApiResource> }`
  - `jobsService.getJobs(opts?: { includeSkills?: boolean }): Promise<JobsResult>`
  - `export const jobsKeys = { all, list }`; `export function useJobs(tenant): UseQueryResult<JobsResult, Error>`

**Background.** `api.getList` (`lib/api/client.ts:352`) returns `r.data` only and discards the rest of the envelope, so it cannot surface `included`. It is left **exactly as is** — a new primitive is added beside it so no existing caller changes (FR-7.1).

`jobsService.getSkillsByJobId` and `useJobSkills` are untouched: they hit `/jobs/{id}/skills`, whose shape is unchanged, through a query key already scoped on `activeTenant.id` (FR-7.3). `JobSkillsAddButton` therefore keeps working with no edit.

**`includeSkills` has no production caller** (design §7 decision 1, adopted). It ships because FR-7.2 asks for the capability and the backend implements it either way; a service test exercises it. Do **not** rewire `useJobSkillDefinitions` through it: that hook issues one React Query per skill id keyed `["skill-definition", tenantId, skillId]`, so definitions cache per skill *across* jobs. Routing it through `include=skills` would fetch every skill of every job — thousands of full effect tables — on first paint and lose the per-skill cache.

- [ ] **Step 1: Write the failing service test**

Create `services/atlas-ui/src/services/api/__tests__/jobs.service.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";

const getOneMock = vi.fn();
const getListDocumentMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    getOne: (...args: unknown[]) => getOneMock(...args),
    getListDocument: (...args: unknown[]) => getListDocumentMock(...args),
  },
}));

import { jobsService } from "@/services/api/jobs.service";

describe("jobsService.getSkillsByJobId", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the skills attribute of the jobs resource", async () => {
    getOneMock.mockResolvedValue({
      id: "112",
      type: "jobs",
      attributes: { skills: [1121000, 1121001] },
    });
    await expect(jobsService.getSkillsByJobId(112)).resolves.toEqual([
      1121000, 1121001,
    ]);
    expect(getOneMock).toHaveBeenCalledWith("/api/data/jobs/112/skills");
  });
});

describe("jobsService.getJobs", () => {
  beforeEach(() => vi.clearAllMocks());

  it("requests a full page and returns the jobs without include by default", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [
        { id: "100", type: "jobs", attributes: { skills: [1001000] } },
        { id: "112", type: "jobs", attributes: { skills: [1121000] } },
      ],
    });

    const result = await jobsService.getJobs();

    expect(getListDocumentMock).toHaveBeenCalledTimes(1);
    const url = getListDocumentMock.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/data/jobs");
    expect(url).toContain("page%5Bsize%5D=250");
    expect(url).not.toContain("include");
    expect(result.jobs.map((j) => j.id)).toEqual(["100", "112"]);
    expect(result.skillsById.size).toBe(0);
  });

  it("follows links.next until exhausted", async () => {
    getListDocumentMock
      .mockResolvedValueOnce({
        data: [{ id: "100", type: "jobs", attributes: { skills: [] } }],
        links: { next: "/api/data/jobs?page%5Bnumber%5D=2&page%5Bsize%5D=250" },
      })
      .mockResolvedValueOnce({
        data: [{ id: "112", type: "jobs", attributes: { skills: [] } }],
        links: {},
      });

    const result = await jobsService.getJobs();

    expect(getListDocumentMock).toHaveBeenCalledTimes(2);
    expect(getListDocumentMock.mock.calls[1]?.[0]).toBe(
      "/api/data/jobs?page%5Bnumber%5D=2&page%5Bsize%5D=250",
    );
    expect(result.jobs.map((j) => j.id)).toEqual(["100", "112"]);
  });

  it("indexes included skills by numeric id when includeSkills is set", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [{ id: "100", type: "jobs", attributes: { skills: [1001000] } }],
      included: [
        { id: "1001000", type: "skills", attributes: { name: "Power Strike" } },
      ],
    });

    const result = await jobsService.getJobs({ includeSkills: true });

    expect(getListDocumentMock.mock.calls[0]?.[0]).toContain("include=skills");
    expect(result.skillsById.get(1001000)?.attributes?.name).toBe(
      "Power Strike",
    );
  });

  it("ignores non-skills members of included", async () => {
    getListDocumentMock.mockResolvedValue({
      data: [{ id: "100", type: "jobs", attributes: { skills: [] } }],
      included: [{ id: "9", type: "somethingelse", attributes: {} }],
    });

    const result = await jobsService.getJobs({ includeSkills: true });
    expect(result.skillsById.size).toBe(0);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test -- src/services/api/__tests__/jobs.service.test.ts
```

Expected: FAIL — `api.getListDocument is not a function` / `jobsService.getJobs is not a function`.

- [ ] **Step 3: Add the compound-document types**

In `services/atlas-ui/src/types/api/responses.ts`, add above `ApiResponse` and extend it:

```ts
/**
 * A JSON:API resource object as it appears in a compound document's `included`
 * array. Attributes are left untyped here — each consumer narrows them.
 */
export interface JsonApiResource {
  type: string;
  id: string;
  attributes?: Record<string, unknown>;
}

export interface ApiResponse<T = unknown> {
  /** The response data - can be a single object or array depending on endpoint */
  data: T;
  /**
   * Compound-document side-loaded resources (JSON:API §5). Present only when
   * the request asked for them via `include=`.
   */
  included?: JsonApiResource[];
}
```

- [ ] **Step 4: Add the `getListDocument` client primitive**

In `services/atlas-ui/src/lib/api/client.ts`, add to the `api` object immediately after `getList` (leave `getList` itself unchanged — every existing caller depends on it returning `r.data`):

```ts
  /**
   * Like `getList`, but returns the whole JSON:API envelope instead of just
   * `data`. Use this when the endpoint is a compound document and you need
   * `included` or the pagination `links`.
   */
  getListDocument: <T>(
    url: string,
    options?: ApiRequestOptions,
  ): Promise<ApiPagedResponse<T> & { included?: JsonApiResource[] }> =>
    apiClient.get<ApiPagedResponse<T> & { included?: JsonApiResource[] }>(
      url,
      options,
    ),
```

Add `ApiPagedResponse` and `JsonApiResource` to that file's existing import from `@/types/api/responses`.

- [ ] **Step 5: Rewrite `jobs.service.ts`**

```ts
import { api } from "@/lib/api/client";
import type { JsonApiResource } from "@/types/api/responses";

export interface JobResource {
  id: string;
  type: string;
  attributes: { skills: number[] };
}

export interface JobsResult {
  jobs: JobResource[];
  /** Populated only when getJobs was called with includeSkills. */
  skillsById: Map<number, JsonApiResource>;
}

const BASE_PATH = "/api/data/jobs";

// The backend caps page[size] at 250 and there are ~82 jobs, so one request
// normally suffices — but links.next is followed anyway so the ceiling is not
// load-bearing.
const PAGE_SIZE = 250;

export const jobsService = {
  async getSkillsByJobId(jobId: number): Promise<number[]> {
    const job = await api.getOne<JobResource>(`${BASE_PATH}/${jobId}/skills`);
    return job.attributes.skills;
  },

  /**
   * Every job present for the active tenant, as ingested from that tenant's
   * Skill.wz. `includeSkills` side-loads the full skill resources into
   * `skillsById`; it defaults to false and has no production caller today —
   * per-skill definitions are fetched through useJobSkillDefinitions, which
   * caches per skill id across jobs.
   */
  async getJobs(opts?: { includeSkills?: boolean }): Promise<JobsResult> {
    const params = new URLSearchParams({ "page[size]": String(PAGE_SIZE) });
    if (opts?.includeSkills) params.set("include", "skills");

    let url: string | undefined = `${BASE_PATH}?${params.toString()}`;
    const jobs: JobResource[] = [];
    const skillsById = new Map<number, JsonApiResource>();

    while (url) {
      const doc = await api.getListDocument<JobResource>(url);
      jobs.push(...(doc.data ?? []));
      for (const inc of doc.included ?? []) {
        if (inc.type !== "skills") continue;
        const id = Number(inc.id);
        if (Number.isInteger(id)) skillsById.set(id, inc);
      }
      url = doc.links?.next;
    }

    return { jobs, skillsById };
  },
};
```

- [ ] **Step 6: Add the React Query hook**

Create `services/atlas-ui/src/lib/hooks/api/useJobs.ts`:

```ts
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { jobsService, type JobsResult } from "@/services/api/jobs.service";

export const jobsKeys = {
  all: ["jobs"] as const,
  list: (tenantId: string | undefined) => ["jobs", tenantId] as const,
};

/**
 * The tenant's job set, as ingested from its Skill.wz. This is the backend
 * replacement for the retired BRANCH_FLOORS / NODE_FLOORS version-floor tables:
 * a job is visible if and only if the tenant has a JOB document for it.
 *
 * TenantProvider calls queryClient.clear() on every tenant switch, so callers
 * MUST treat the pending state as "unknown", not "empty" — see JobsPage.
 */
export function useJobs(
  tenant: Tenant | null | undefined,
): UseQueryResult<JobsResult, Error> {
  return useQuery({
    queryKey: jobsKeys.list(tenant?.id),
    queryFn: () => jobsService.getJobs(),
    enabled: !!tenant?.id,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });
}
```

- [ ] **Step 7: Run the tests and the type check**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test -- src/services/api/__tests__/jobs.service.test.ts && npm run build
```

Expected: tests PASS and `npm run build` (`tsc -b` + `vite build`) succeeds. `npm run build` is the real type gate — `vitest` alone does not type-check.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-ui/src/types/api/responses.ts \
        services/atlas-ui/src/lib/api/client.ts \
        services/atlas-ui/src/services/api/jobs.service.ts \
        services/atlas-ui/src/lib/hooks/api/useJobs.ts \
        services/atlas-ui/src/services/api/__tests__/jobs.service.test.ts
git commit -m "feat(task-185): fetch the tenant job set from GET /data/jobs"
```

---

## Task 9: Retire the version floors from the job tree

**Files:**
- Modify: `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`
- Modify: `services/atlas-ui/src/components/features/jobs/rail-groups.ts`
- Modify: `services/atlas-ui/src/components/features/jobs/advancement-flow.tsx`
- Test: `services/atlas-ui/src/lib/jobs/__tests__/job-advancement-tree.test.ts` (rewritten)

**Interfaces:**
- Consumes: nothing from Task 8 (this task is a pure signature change; `JobsPage` is wired in Task 10).
- Produces:
  - `visibleRoots(available: ReadonlySet<number>): number[]`
  - `visibleChildrenOf(id: number, available: ReadonlySet<number>): number[]`
  - `advancementChains(entryId: number, available: ReadonlySet<number>): number[][]`
  - `subtreeCount(entryId: number, available: ReadonlySet<number>): number`
  - `visibleRailGroups(available: ReadonlySet<number>): VisibleRailGroup[]`
  - `AdvancementFlow` prop `available: ReadonlySet<number>` (replacing `major: number`)
- Removes: `BRANCH_FLOORS`, `NODE_FLOORS`, `floorOf`, and the rationale comment block at `job-advancement-tree.ts:110-145`.

**Background.** `JOB_GRAPH`, `JOB_ROOTS`, `childrenOf`, `rootOf`, `jobTreePath`, and `tierLabel` are untouched — they encode topology, not version data (FR-6.4). The graph stays the display authority for *structure*; the tenant data becomes the authority for *existence*. Ids present in the tenant data but absent from `JOB_GRAPH` are simply not rendered.

`advancementChains` currently drops a whole chain if any node is below floor (`.filter((chain) => chain.every((id) => floorOf(id) <= major))`). That semantic is preserved verbatim with `available.has`.

**This task leaves `JobsPage.tsx` broken** (it still passes `major`). Task 10 fixes it. To keep each task independently verifiable, Step 6 runs the targeted vitest suites rather than the full build; the full `npm run build` gate is Task 10's.

- [ ] **Step 1: Rewrite the tree test set-driven**

Replace `services/atlas-ui/src/lib/jobs/__tests__/job-advancement-tree.test.ts` entirely:

```ts
import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH,
  JOB_ROOTS,
  childrenOf,
  rootOf,
  visibleRoots,
  visibleChildrenOf,
  jobTreePath,
  advancementChains,
  tierLabel,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";

/** Every id in the graph — the "modern tenant has everything" case. */
const ALL: ReadonlySet<number> = new Set(
  Object.values(JOB_GRAPH).map((e) => e.id),
);

/** A legacy tenant: explorers only, no Pirate, no GM line, no Cygnus/Legends. */
const LEGACY: ReadonlySet<number> = new Set([
  0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 210, 211, 212, 220,
  221, 222, 230, 231, 232, 300, 310, 311, 312, 320, 321, 322, 400, 410, 411,
  412, 420, 421, 422,
]);

describe("job-advancement-tree", () => {
  it("exposes the five branch roots ascending (GM line is not a root)", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500, 900]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(900)).toEqual([910]);
    expect(childrenOf(112)).toEqual([]);
  });

  it("walks to the branch root", () => {
    expect(rootOf(112)).toBe(0);
    expect(rootOf(1112)).toBe(1000);
    expect(rootOf(2112)).toBe(2000);
    expect(rootOf(2218)).toBe(2001);
    expect(rootOf(910)).toBe(0);
    expect(rootOf(99999)).toBe(99999);
  });

  it("shows every root when the tenant has every job", () => {
    expect(visibleRoots(ALL)).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("hides roots the tenant has no job document for", () => {
    const roots = visibleRoots(LEGACY);
    expect(roots).toEqual([0]);
    expect(roots).not.toContain(1000);
    expect(roots).not.toContain(2000);
    expect(roots).not.toContain(2001);
    expect(roots).not.toContain(800);
  });

  it("hides an absent subtree while keeping its present siblings", () => {
    // LEGACY has no Pirate (500) and no GM (900).
    expect(visibleChildrenOf(0, LEGACY)).toEqual([100, 200, 300, 400]);
    expect(visibleChildrenOf(0, ALL)).toEqual([100, 200, 300, 400, 500, 900]);
  });

  it("returns nothing when the tenant set is empty", () => {
    const none: ReadonlySet<number> = new Set();
    expect(visibleRoots(none)).toEqual([]);
    expect(visibleChildrenOf(0, none)).toEqual([]);
    expect(subtreeCount(0, none)).toBe(0);
    expect(advancementChains(0, none)).toEqual([]);
  });

  it("drops an advancement chain containing any absent node", () => {
    // Warrior's chains are [110,111,112], [120,121,122], [130,131,132].
    const noHero: ReadonlySet<number> = new Set(
      [...LEGACY].filter((id) => id !== 112),
    );
    const chains = advancementChains(100, noHero);
    expect(chains).not.toContainEqual([110, 111, 112]);
    expect(chains).toContainEqual([120, 121, 122]);
    expect(chains).toContainEqual([130, 131, 132]);
  });

  it("counts only the jobs the tenant actually has", () => {
    // Warrior subtree: 100 + 110,111,112 + 120,121,122 + 130,131,132 = 10.
    expect(subtreeCount(100, LEGACY)).toBe(10);
    expect(subtreeCount(100, new Set([100, 110]))).toBe(2);
    expect(subtreeCount(100, new Set([110]))).toBe(0);
    expect(subtreeCount(99999, ALL)).toBe(0);
  });

  it("keeps topology helpers version-independent", () => {
    expect(jobTreePath(112).map((e) => e.id)).toEqual([0, 100, 110, 111, 112]);
    expect(tierLabel(0)).toBe("Base");
    expect(tierLabel(112)).toBe("4th");
    expect(tierLabel(99999)).toBe("");
  });
});
```

Before running, confirm the Warrior-branch expectations — the `noHero` chains and `subtreeCount(100, LEGACY) === 10` — against the actual `JOB_GRAPH` entries. The graph is the authority; adjust the expected values if it differs, not the implementation.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test -- src/lib/jobs
```

Expected: FAIL — `visibleRoots(ALL)` passes a `Set` where a `number` is expected, so every visibility assertion returns the wrong result.

- [ ] **Step 3: Rewrite the visibility predicates**

In `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`:

**(a)** Delete the entire comment block and both tables — the block beginning `// Version-introduction floor per branch ROOT id.` through the closing `};` of `NODE_FLOORS` (currently lines 110-145). Its rationale is obsolete: it asserts "the atlas-data /jobs/{id}/skills endpoint is NOT version-gated", which task-185 makes false (FR-6.5).

**(b)** Delete `floorOf` entirely, including its docblock.

**(c)** Replace the four visibility functions:

```ts
/**
 * Branch root ids the tenant actually has, ascending.
 *
 * `available` is the set of job ids returned by GET /api/data/jobs — the
 * tenant's ingested job set. It replaces the retired BRANCH_FLOORS/NODE_FLOORS
 * version-floor tables: existence is data, not a hand-maintained table. Callers
 * must not pass an empty set to mean "unknown"; see JobsPage's isSuccess gate.
 */
export function visibleRoots(available: ReadonlySet<number>): number[] {
  return JOB_ROOTS.filter((r) => available.has(r));
}

/**
 * Direct children of a node that the tenant has. Lets the tree hide a subtree
 * the tenant's version never shipped (e.g. Pirate on a v48 tenant) while
 * showing its siblings.
 */
export function visibleChildrenOf(
  id: number,
  available: ReadonlySet<number>,
): number[] {
  return childrenOf(id).filter((c) => available.has(c));
}
```

and, further down:

```ts
/**
 * Every advancement chain below entryId: one array per root-to-leaf path of
 * the subtree, EXCLUDING entryId itself, DFS in ascending child order. A chain
 * containing any job the tenant does not have is dropped entirely (matches
 * visibleChildrenOf semantics). A leaf entry yields [].
 */
export function advancementChains(
  entryId: number,
  available: ReadonlySet<number>,
): number[][] {
  const walk = (id: number): number[][] => {
    const kids = childrenOf(id);
    if (kids.length === 0) return [[]];
    const out: number[][] = [];
    for (const k of kids) {
      for (const rest of walk(k)) out.push([k, ...rest]);
    }
    return out;
  };
  return walk(entryId)
    .filter((chain) => chain.length > 0)
    .filter((chain) => chain.every((id) => available.has(id)));
}
```

```ts
/** Count of tenant-available nodes in entryId's subtree, entry included (0 if the entry itself is absent). */
export function subtreeCount(
  entryId: number,
  available: ReadonlySet<number>,
): number {
  if (JOB_GRAPH[entryId] === undefined || !available.has(entryId)) return 0;
  return (
    1 +
    visibleChildrenOf(entryId, available).reduce(
      (n, k) => n + subtreeCount(k, available),
      0,
    )
  );
}
```

- [ ] **Step 4: Update `rail-groups.ts`**

Drop `floorOf` from the import, and rewrite `visibleRailGroups`:

```ts
import {
  JOB_GRAPH,
  jobTreePath,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";
```

```ts
/** Rail groups for the tenant's job set, with display name + subtree count; empty groups dropped. */
export function visibleRailGroups(
  available: ReadonlySet<number>,
): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries
      .filter((e) => available.has(e.id))
      .map((e) => ({
        ...e,
        name: JOB_GRAPH[e.id]?.name ?? `Job ${e.id}`,
        count: subtreeCount(e.id, available),
      })),
  })).filter((g) => g.entries.length > 0);
}
```

- [ ] **Step 5: Update `advancement-flow.tsx`**

Change the prop and the two use sites:

```ts
interface AdvancementFlowProps {
  entryId: number;
  available: ReadonlySet<number>;
  selectedJobId: number;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
  onSelect: (id: number) => void;
}
```

```ts
export function AdvancementFlow({
  entryId,
  available,
  selectedJobId,
  accent,
  onSelect,
}: AdvancementFlowProps) {
```

```ts
  const chains = useMemo(
    () => advancementChains(entryId, available),
    [entryId, available],
  );
```

- [ ] **Step 6: Run the tree tests to verify they pass**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test -- src/lib/jobs src/components/features/jobs
```

Expected: PASS. `npm run build` still fails at this point because `JobsPage.tsx` passes `major` — that is expected and is fixed in Task 10.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/lib/jobs/ \
        services/atlas-ui/src/components/features/jobs/rail-groups.ts \
        services/atlas-ui/src/components/features/jobs/advancement-flow.tsx
git commit -m "refactor(task-185): drive job-tree visibility from the tenant job set"
```

---

## Task 10: Wire `JobsPage` to the tenant job set

**Files:**
- Modify: `services/atlas-ui/src/pages/JobsPage.tsx`
- Modify: `services/atlas-ui/src/components/features/jobs/branch-rail.tsx`
- Test: `services/atlas-ui/src/pages/__tests__/JobsPage.test.tsx` (extended)

**Interfaces:**
- Consumes: `useJobs` (Task 8), `visibleRailGroups(available)` / `AdvancementFlow available` (Task 9).
- Produces: `BranchRail` gains an optional `isPending?: boolean` prop.

**Background — the one real regression risk (design D10).** `floorOf` was synchronous, so `JobsPage` could compute `jobIdValid` on first render. `available` now arrives from a query, and `TenantProvider` calls `queryClient.clear()` on every tenant switch, so `available` is empty **during load and immediately after a tenant change**. Without a guard, the normalize effect at `JobsPage.tsx:50-54` would redirect a perfectly valid `/jobs/112` to `/jobs`, and the rail would render empty.

The mitigation is part of the design, not an afterthought:
- `jobIdValid` and the normalize effect are both gated on `jobsQuery.isSuccess` — while pending, no redirect fires and the current `jobId` is retained;
- `BranchRail` renders a skeleton while pending (it has none today — one is added here);
- `defaultJobId` is only consulted once `available` is non-empty;
- `jobsQuery.isError` renders an error card rather than an empty tree: "the backend is down" must not look like "this version has no jobs".

- [ ] **Step 1: Write the failing page tests**

In `services/atlas-ui/src/pages/__tests__/JobsPage.test.tsx`, add the `useJobs` mock beside the existing mocks at the top of the file — it must be declared with the others, before the `import { JobsPage }` line:

```ts
const useJobsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobs", () => ({
  useJobs: (...args: unknown[]) => useJobsMock(...args),
}));
```

Then append the helper and the new cases at the end of the file:

```ts
const ALL_JOBS = [
  0, 100, 110, 111, 112, 120, 121, 122, 130, 131, 132, 200, 300, 400, 500, 900,
  910, 800, 1000, 2000, 2001,
];

function jobsQuery(
  state: "pending" | "success" | "error",
  ids: number[] = ALL_JOBS,
) {
  return {
    data:
      state === "success"
        ? {
            jobs: ids.map((id) => ({
              id: String(id),
              type: "jobs",
              attributes: { skills: [] },
            })),
            skillsById: new Map(),
          }
        : undefined,
    isPending: state === "pending",
    isSuccess: state === "success",
    isError: state === "error",
  };
}

describe("JobsPage — tenant job set", () => {
  it("does not redirect a valid jobId while the job set is still loading", async () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobsMock.mockReturnValue(jobsQuery("pending"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: true,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");

    await waitFor(() =>
      expect(screen.getByTestId("location").textContent).toBe("/jobs/112"),
    );
  });

  it("renders the rail skeleton while the job set is loading", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobsMock.mockReturnValue(jobsQuery("pending"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: true,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");
    expect(screen.getByTestId("branch-rail-skeleton")).toBeInTheDocument();
  });

  it("renders an error card, not an empty tree, when the job set fails to load", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    useJobsMock.mockReturnValue(jobsQuery("error"));
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");
    expect(screen.getByTestId("jobs-load-error")).toBeInTheDocument();
    expect(screen.queryByTestId("branch-rail-skeleton")).not.toBeInTheDocument();
  });

  it("shows only the branches present in the tenant job set", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(48) });
    // A GMS 48-shaped set: explorers only, no Pirate/GM/Cygnus/Legends/Brigadier.
    useJobsMock.mockReturnValue(
      jobsQuery("success", [0, 100, 110, 111, 112, 200, 300, 400]),
    );
    useJobSkillsMock.mockReturnValue({
      data: [1121000],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [def(1121000, "Brandish")],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/112");

    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.queryByText("Pirate")).not.toBeInTheDocument();
    expect(screen.queryByText("Noblesse")).not.toBeInTheDocument();
  });

  it("redirects a jobId absent from the tenant job set once the query succeeds", async () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(48) });
    useJobsMock.mockReturnValue(
      jobsQuery("success", [0, 100, 110, 111, 112, 200, 300, 400]),
    );
    useJobSkillsMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    useMediaQueryMock.mockReturnValue(true);

    renderAt("/jobs/1000"); // Noblesse — not in this tenant's set

    await waitFor(() =>
      expect(screen.getByTestId("location").textContent).toBe("/jobs"),
    );
  });
});
```

Confirm the rail label strings (`"Warrior"`, `"Pirate"`, `"Noblesse"`) against `JOB_GRAPH`'s `name` values before running; the graph is the authority. The pre-existing cases in this file drive `useTenantMock`/`useJobSkillsMock` only — they now also need `useJobsMock.mockReturnValue(jobsQuery("success"))` in their setup, since `JobsPage` calls `useJobs` unconditionally. Add that line to each existing case's arrange block.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test -- src/pages/__tests__/JobsPage.test.tsx
```

Expected: FAIL — `JobsPage` does not call `useJobs`, and `branch-rail-skeleton` / `jobs-load-error` do not exist.

- [ ] **Step 3: Add the pending skeleton to `BranchRail`**

In `services/atlas-ui/src/components/features/jobs/branch-rail.tsx`, add the prop and an early return:

```ts
interface BranchRailProps {
  groups: VisibleRailGroup[];
  selectedEntryId: number;
  onSelect: (id: number) => void;
  /** True while the tenant job set is still loading — renders a skeleton. */
  isPending?: boolean;
}

export function BranchRail({
  groups,
  selectedEntryId,
  onSelect,
  isPending = false,
}: BranchRailProps) {
  if (isPending) {
    return (
      <Card className="flex min-h-0 flex-col">
        <CardHeader className="pb-1">
          <CardTitle className="text-[15px]">Branches</CardTitle>
        </CardHeader>
        <CardContent
          data-testid="branch-rail-skeleton"
          className="min-h-0 flex-1 space-y-2 px-2 pb-3 pt-2"
        >
          {Array.from({ length: 8 }, (_, i) => (
            <div
              key={i}
              className="h-7 w-full animate-pulse rounded-md bg-muted"
            />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    // …existing body unchanged…
  );
}
```

- [ ] **Step 4: Rewire `JobsPage.tsx`**

Change the `job-advancement-tree` import to drop `floorOf`, and add `useJobs`:

```ts
import { useJobs } from "@/lib/hooks/api/useJobs";
import { JOB_GRAPH } from "@/lib/jobs/job-advancement-tree";
```

Replace lines 36-54 of the current file with:

```ts
  const jobsQuery = useJobs(activeTenant);
  // The tenant's actual job set, from GET /api/data/jobs. Empty while the query
  // is pending — which is why every consumer below is gated on isSuccess rather
  // than on the set being non-empty. TenantProvider calls queryClient.clear()
  // on every tenant switch, so "pending with an empty set" is the state right
  // after a switch, not an error.
  const available = useMemo<ReadonlySet<number>>(
    () => new Set((jobsQuery.data?.jobs ?? []).map((j) => Number(j.id))),
    [jobsQuery.data],
  );
  const groups = useMemo(
    () => (jobsQuery.isSuccess ? visibleRailGroups(available) : []),
    [jobsQuery.isSuccess, available],
  );
  const defaultJobId = groups[0]?.entries[0]?.id ?? 100;

  const parsedJobId = jobIdParam !== undefined ? Number(jobIdParam) : null;
  const jobIdValid =
    parsedJobId !== null &&
    Number.isInteger(parsedJobId) &&
    JOB_GRAPH[parsedJobId] !== undefined &&
    jobsQuery.isSuccess &&
    available.has(parsedJobId);
  // While the job set is pending the current jobId is retained: redirecting on
  // an unknown set would bounce a valid /jobs/112 to /jobs on every tenant
  // switch (design D10).
  const jobId = jobIdValid
    ? parsedJobId
    : jobsQuery.isSuccess
      ? defaultJobId
      : (parsedJobId ?? defaultJobId);

  // FR-1.2 / FR-7.3: an unknown or tenant-absent jobId normalizes to /jobs with
  // replace, so Back doesn't bounce. Gated on isSuccess so a pending or failed
  // job-set query never triggers it.
  useEffect(() => {
    if (
      activeTenant &&
      jobsQuery.isSuccess &&
      parsedJobId !== null &&
      !jobIdValid
    ) {
      navigate("/jobs", { replace: true });
    }
  }, [activeTenant, jobsQuery.isSuccess, parsedJobId, jobIdValid, navigate]);
```

Add the error branch to the render. The `!activeTenant ? … : ( …grid… )` ternary becomes `!activeTenant ? … : jobsQuery.isError ? … : ( …grid… )` — insert this immediately after the `!activeTenant` branch's closing `) : `:

```tsx
      jobsQuery.isError ? (
        <Card>
          <CardContent
            data-testid="jobs-load-error"
            className="py-10 text-center text-muted-foreground"
          >
            Could not load this tenant&apos;s job list. Check that atlas-data is
            reachable and that this version has been ingested.
          </CardContent>
        </Card>
      ) : (
```

Pass the new props through:

```tsx
          <BranchRail
            groups={groups}
            selectedEntryId={entry.id}
            onSelect={selectJob}
            isPending={jobsQuery.isPending}
          />
```

```tsx
              <AdvancementFlow
                entryId={entry.id}
                available={available}
                selectedJobId={jobId}
                accent={entry.accent}
                onSelect={selectJob}
              />
```

Finally, fold the job-set query into the skill-list loading state so the list shows its skeleton rather than "empty" while jobs are still resolving:

```ts
  const loading =
    jobsQuery.isPending ||
    skillsQuery.isLoading ||
    (skillIds.length > 0 && defsLoading);
```

- [ ] **Step 5: Run the full UI test suite and build**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui" && npm run test && npm run build
```

Expected: both PASS. `npm run build` is the gate that proves no `major` parameter survives anywhere in the job-tree path (FR-6.2).

- [ ] **Step 6: Verify no version-floor remnants remain**

```bash
cd "$(git rev-parse --show-toplevel)"
grep -rn "BRANCH_FLOORS\|NODE_FLOORS\|floorOf" services/atlas-ui/src/
```

Expected: no output (FR-6.1, FR-6.5, FR-6.6).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/pages/ services/atlas-ui/src/components/features/jobs/branch-rail.tsx
git commit -m "feat(task-185): render the jobs page from the tenant job set"
```

---

## Task 11: Full verification gates

**Files:** none created or modified unless a gate fails.

**Interfaces:**
- Consumes: everything from Tasks 1-10.
- Produces: a verified branch, ready for the pre-PR code review.

**Background.** Per CLAUDE.md §Build & Verification and design §9. `docker buildx bake atlas-data` is required **only if** a `go.mod` was touched — this change is not expected to touch one, but the check is not optional: confirm with `git diff --stat` rather than assuming.

`tools/service-registration-guard.sh` and `tools/template-opcode-order-guard.sh` are **not** expected to apply: no service was added and no socket-config template changed. Confirm that with Step 1's diffs rather than skipping on assumption.

- [ ] **Step 1: Confirm the change surface**

```bash
cd "$(git rev-parse --show-toplevel)"
git status --short
git diff --stat main -- '**/go.mod' '**/go.sum'
git diff --stat main -- .github/config/services.json deploy/k8s docker-bake.hcl go.work tools/db-bootstrap.sh
git diff --stat main -- services/atlas-configurations/seed-data/templates
```

Expected: a clean working tree, and **empty output** from all three `git diff --stat` commands. If the first is non-empty, Step 6's bake is mandatory. If either of the last two is non-empty, run the corresponding guard from CLAUDE.md items 7 and 9.

- [ ] **Step 2: Go tests, race-enabled, in both changed modules**

```bash
cd "$(git rev-parse --show-toplevel)"
(cd libs/atlas-constants && go test -race ./...)
(cd services/atlas-data/atlas.com/data && go test -race ./...)
```

Expected: `ok` for every package, no failures.

- [ ] **Step 3: `go vet` and `go build` in both modules**

```bash
cd "$(git rev-parse --show-toplevel)"
(cd libs/atlas-constants && go vet ./... && go build ./...)
(cd services/atlas-data/atlas.com/data && go vet ./... && go build ./...)
```

Expected: no output from either.

- [ ] **Step 4: Confirm the unaffected consumers still build and pass untouched**

```bash
cd "$(git rev-parse --show-toplevel)"
for m in services/atlas-character/atlas.com/character \
         services/atlas-skills/atlas.com/skills \
         services/atlas-configurations/atlas.com/configurations; do
  echo "== $m"; (cd "$m" && go build ./... && go test -race ./... >/dev/null) || echo "FAILED $m"
done
git diff --stat main -- services/atlas-character services/atlas-skills services/atlas-configurations
```

Expected: no `FAILED` lines, and **empty** `git diff --stat` output — the acceptance criterion is that these three compile and pass with *no source changes*.

- [ ] **Step 5: Repo-root guards**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all three exit 0. If `tools/lint.sh --check` reports formatting drift, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check` and amend the affected commit.

- [ ] **Step 6: Docker bake — only if a `go.mod` changed**

If and only if Step 1's `go.mod`/`go.sum` diff was non-empty:

```bash
cd "$(git rev-parse --show-toplevel)"
docker buildx bake atlas-data
```

Expected: a successful build. `go build`/`go test` against the workspace `go.work` will **not** catch a missing `COPY libs/…` line in the shared Dockerfile — only the bake will.

- [ ] **Step 7: atlas-ui build and tests**

```bash
cd "$(git rev-parse --show-toplevel)/services/atlas-ui"
npm run test
npm run build
```

Expected: both PASS. `npm run build` (`tsc -b` + `vite build`) is the type gate; `vitest` alone is not sufficient. If `npm` fails to start, the project's Node version is nvm 22.

- [ ] **Step 8: Re-check the acceptance criteria mechanically**

```bash
cd "$(git rev-parse --show-toplevel)"
wc -l libs/atlas-constants/job/constants.go                              # expect < 200
grep -c "^var [A-Z][A-Za-z0-9]* = Job{" libs/atlas-constants/job/constants.go || echo "0 Job vars (expected)"
grep -rn "Skills()\|Buffs()" libs/atlas-constants/job/                   # expect no output
grep -rn "BRANCH_FLOORS\|NODE_FLOORS\|floorOf" services/atlas-ui/src/    # expect no output
grep -rn "constJob" services/atlas-data/atlas.com/data/job/              # expect no output
git diff main -- libs/atlas-constants/go.mod                             # expect empty
```

Expected, in order: a line count under 200; zero `Job` value vars; no `Skills()`/`Buffs()` in the job package; no floor identifiers in atlas-ui; no `constJob` in atlas-data's job package; an unchanged `go.mod`.

- [ ] **Step 9: Run the pre-PR code review**

Per CLAUDE.md, code review runs **before** opening the PR — do not skip it. Invoke `superpowers:requesting-code-review`; it dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed), and `frontend-guidelines-reviewer` (atlas-ui TS changed) in parallel. Pin the reviewer subagents to Sonnet/Haiku. Findings land in `docs/tasks/task-185-tenant-aware-job-skills/audit.md`.

Verify the worktree is clean after the reviewers run — they must not write into the main repo checkout.

- [ ] **Step 10: Note the user-visible behaviour change for the PR description**

The PR body must state that the Jobs page changes visibly on legacy tenants — GM/Super GM disappear on GMS 48, Maple Leaf Brigadier and the Cygnus branch appear on GMS 72 and 79, Super GM disappears on JMS 185 — and that these are intended moves toward the tenant's data, not regressions. It must also link `docs/runbooks/job-document-backfill.md` and state that the backfill is a required operator step at deploy time, because both job endpoints 404 for any tenant whose data predates this change.

---

## Self-Review

**Spec coverage.** Every FR maps to a task:

| Requirement | Task |
|---|---|
| FR-1.1 / FR-1.2 / FR-1.3 (`JOB` type, shape, tenant scoping) | 1, 2 |
| FR-1.4 (baseline + purge, verified not assumed) | 5 |
| FR-2.1 / FR-2.5 (SKILL worker pass, monolith inheritance) | 4 |
| FR-2.2 (job id from the image name) | 1 |
| FR-2.3 (non-numeric images produce nothing) | 1, 2 |
| FR-2.4 (zero-skill image → empty list) | 1 |
| FR-3.1 / FR-3.2 (storage read, 404 semantics) | 2 |
| FR-3.3 (`InitResource` takes `*gorm.DB`) | 2 |
| FR-4.1 … FR-4.5 (`GET /data/jobs`, linkage, `included`, persisted-vs-response split, pagination) | 3 |
| FR-5.1 … FR-5.7 (constants collapse; §7.5 records that README needs no edit) | 6 |
| FR-6.1 … FR-6.6 (floor retirement) | 9, 10 |
| FR-7.1 / FR-7.2 / FR-7.3 (UI compound-document support) | 8 |
| PRD §7.5 / design §6 (rollout runbook, 11 versions) | 7 |
| NFR multi-tenancy / performance / observability / testing / gates | 2, 3, 4, 11 |
| Design §3 (behaviour change belongs in the PR description) | 7, 11 |

Design §7's two open decisions are resolved in Global Constraints (ship `includeSkills`; name the type `ListRestModel`). FR-5.7 needs no edit — design D8 verified that `libs/atlas-constants/README.md:31` already reads "`Id`, `Type` | Job / class IDs and type-codes", implying no skill data; recorded here so no task goes looking for one.

**Placeholder scan.** No "TBD", "implement later", "add appropriate error handling", or "similar to Task N". Every code step carries the actual code. Four steps deliberately instruct verification against the source rather than asserting a value — the 23 `fourthJob` ids (Task 6 Steps 1-2), the Warrior-branch chain/count expectations (Task 9 Step 1), the rail label strings (Task 10 Step 1), and the `Jobs` map's per-id entries (Task 6 Step 3). Each names the file as the authority and gives the command to derive the answer; that is grounding per CLAUDE.md's "Verification Over Memory", not a placeholder.

**Type consistency.** `RegisterJob` returns `(int, error)` in the interface (Task 2), the mock (Task 2), and the worker adapter (Task 4). `NewProcessor(l, ctx, db)` is three-arg everywhere after Task 2, including `job/resource.go`, the tests, and `data/workers/skill.go`. `InitResource(db)(si)` is curried identically in `main.go`, `setupTestRouter`, and the design. `setupResourceTestDB` / `setupTestRouter` / `seedJob` / `setTenantHeaders` / `testServerInfo` / `testDocumentEntity` are defined once, in `job/resource_test.go` (Task 2), and consumed by `job/processor_test.go` and `job/rest_test.go`; `writeTempImage` and the XML fixture constants are defined once, in `job/reader_test.go` (Task 1), and consumed by `job/processor_test.go`. `ListRestModel`'s `resolved` field is written only by `WithResolvedSkills` and read only by `GetReferencedStructs`, both in `job/list_rest.go`. On the UI side, `available: ReadonlySet<number>` is the parameter type in `visibleRoots`, `visibleChildrenOf`, `advancementChains`, `subtreeCount`, `visibleRailGroups`, and the `AdvancementFlow` prop; `JobsResult` is produced by `jobsService.getJobs` and consumed by `useJobs` and `JobsPage`.

**One design inaccuracy corrected.** Design D10 says "`BranchRail` renders its existing skeleton while `jobsQuery.isPending`." `branch-rail.tsx` has **no** skeleton today — `grep -n "skeleton\|Skeleton\|isPending\|loading"` over that file returns nothing. Task 10 Step 3 therefore adds one rather than reusing a non-existent state.
