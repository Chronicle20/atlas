# Tenant-Aware Job Skill Enumeration — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read from the tree at
plan time; file:line references are to the state of the
`task-185-tenant-aware-job-skills` branch before any implementation commit.

---

## 1. The problem in one paragraph

`GET /data/jobs/{jobId}/skills` answers from `constJob.Jobs[id].Skills()` — a
hardcoded, v83-shaped table of 82 `Job` literals carrying ~540 skill references,
with no tenant or version input. Job skill sets drift across every version the
project supports (live probe: skill `5101001` resolves on GMS v48/v61 and 404s
from v72 onward). Each tenant's own `Skill.wz` already *is* the authoritative
per-version job→skill mapping, and atlas-data already walks it — it just
discards the job id after using it. This task persists that mapping as a `JOB`
document type and deletes the hardcoded lists.

---

## 2. Key files, by role

### The read path (what changes)

| File | Line refs | Why it matters |
|---|---|---|
| `services/atlas-data/atlas.com/data/job/processor.go` | 29-39 | `GetSkillsForJob` reads `constJob.Jobs[…].Skills()`. The whole task pivots on this function. |
| `services/atlas-data/atlas.com/data/job/resource.go` | 14-39 | `InitResource(si)` — the only `InitResource` in atlas-data that does **not** take `*gorm.DB`. |
| `services/atlas-data/atlas.com/data/job/rest.go` | 5-20 | `RestModel{Id uint32; Skills []uint32}` with `GetName()`/`GetID()`/`SetID()`. Already exactly the persisted shape. Do not add relationship interfaces to it. |
| `services/atlas-data/atlas.com/data/main.go` | 184 | `job.InitResource(GetServer())` — the odd one out; every neighbour on lines 179-197 is `X.InitResource(db)(GetServer())`. |

### The patterns to copy

| Pattern | Reference | Note |
|---|---|---|
| Document storage wiring | `skill/processor.go:37-59` | `NewStorage(l, db)` → `document.NewStorage(l, db, GetModelRegistry(), "SKILL")`; `Register` loops `s.Add(ctx)(m)()`; `RegisterX(path)` wraps in `database.ExecuteTransaction`. |
| Model registry singleton | `skill/registry.go` | 18 lines, `sync.Once`. Copy verbatim with the `job` package name. |
| Reader | `skill/reader.go:26-76` | `parseJobId(exml.Name)` then iterate `ChildByName("skill").ChildNodes`. The `job` reader diverges on two error cases — see §4. |
| Paged list handler | `commodity/resource.go:33-55` | DB-side paging via `AllPagedProvider`. This is the model for `GET /data/jobs`, **not** `skill`'s drain-then-`paginate.Slice` (`skill/resource.go:49-95`). |
| JSON:API to-many relationship | `services/atlas-npc-shops/atlas.com/npc/shops/rest.go:35-101` | `GetReferences` / `GetReferencedIDs` / `GetReferencedStructs`. Note this task implements only the marshal half. |
| `include=` parsing in a handler | `shops/resource.go:75-84` | The marshal layer knows nothing about `include`; the handler parses `r.URL.Query()["include"]`. |
| sqlite test harness | `skill/resource_test.go:27-107` | `testDocumentEntity` (no PG-specific defaults) + `setupResourceTestDB` + `database.RegisterTenantCallbacks`. |

### The storage layer (read, unchanged)

| File | Line refs | What it gives you free |
|---|---|---|
| `document/storage.go` | 28-64 | `ByIdProvider`: registry cache → tenant rows → **canonical fallback** on `canonical.TenantId(region, major, minor)`. This is FR-1.3, already implemented. |
| `document/storage.go` | 80-106 | `AllPagedProvider` carries the same canonical fallback (added by PR #759 to close the batch-GetAll asymmetry). |
| `document/storage.go` | 111-126 | `DrainAllProvider` — pages internally at 1000/page. Used for `include=skills`. |
| `document/db_storage.go` | 120-157 | `Add` marshals via `jsonapi.MarshalToStruct` and derives `document_id` by `strconv.Atoi(GetID())` at line 133. **This is why the persisted type must stay relationship-free.** |
| `document/db_storage.go` | 60-79 | `AllPaged` filters `type = ?` and orders by `document_id`; tenant scoping comes from `database.RegisterTenantCallbacks`. |
| `document/registry.go` | 10-13, 24-38 | Registry is keyed by `tenant.Model` (a comparable value). **Package-level `sync.Once` singleton → shared across tests in a package.** |

### The ingest path

| File | Line refs | Note |
|---|---|---|
| `data/workers/skill.go` | 26-98 | The `SKILL` worker. Line 49 registers skills, line 52 registers `mobskill` — the JOB pass goes between line 54 and the icon loop. |
| `data/workers/walk.go` | 34-50 | `registerAllInDirectory` logs and swallows per-file errors; only the walk itself can fail. It filters on `.img.xml`. |
| `data/workers/walk.go` | 58-65 | `imgID` — the WZ-object-model twin of `parseJobId`, tolerant of a present or absent `.img` suffix. |
| `xml/reader.go` | 28-48 | `FromPathProvider` reads eagerly into a `FixedProvider`. Calling it twice on the same path is a genuine second parse. |
| `xml/model.go` | 9-27 | `Node.Name` is the root `<imgdir name="…">` attribute; `ChildByName` returns `errors.New("child not found")` when absent. |

### The constants library

| File | Line refs | Note |
|---|---|---|
| `libs/atlas-constants/job/constants.go` | 7-1058 | The 82 `Job` value vars to delete. |
| `libs/atlas-constants/job/constants.go` | 1060-1143 | `Jobs` maps `<XId>: <X>` — the block to rewrite as inline literals. |
| `libs/atlas-constants/job/model.go` | 9-27 | `Job` struct + `Skills()` + `Buffs()`. Drop the `skills` field and both accessors; keep `Id()` and `IsFourthJob()`. |
| `libs/atlas-constants/job/model.go` | 82-92 | `mpEaterSkillIds` — the reason `model.go` keeps the `skill` import and `go.mod` is untouched. |
| `libs/atlas-constants/job/advancement.go` | all | Untouched. |

### atlas-ui

| File | Line refs | Note |
|---|---|---|
| `src/lib/jobs/job-advancement-tree.ts` | 110-145 | `BRANCH_FLOORS` + `NODE_FLOORS` + the rationale comment. Delete. |
| `src/lib/jobs/job-advancement-tree.ts` | 176-193 | `floorOf`. Delete. |
| `src/lib/jobs/job-advancement-tree.ts` | 196, 205, 226, 261 | The four `major` consumers. |
| `src/components/features/jobs/rail-groups.ts` | 78-89 | `visibleRailGroups(major)`. |
| `src/components/features/jobs/advancement-flow.tsx` | 13, 68, 78-79 | The `major` prop and its single use. |
| `src/pages/JobsPage.tsx` | 36-54, 134 | `major` derivation, `jobIdValid`, the normalize effect, the `AdvancementFlow` prop. |
| `src/components/features/jobs/branch-rail.tsx` | all | **No skeleton exists today** — one is added (corrects design D10). |
| `src/lib/api/client.ts` | 351-355 | `api.getList` returns `r.data` only. Left untouched; a sibling primitive is added. |
| `src/types/api/responses.ts` | 7-10, 99-118 | `ApiResponse` has no `included`; `ApiPagedResponse` already carries `meta`/`links`. |
| `src/services/api/jobs.service.ts` | all | 16 lines today; gains `getJobs`. |
| `src/lib/hooks/api/useJobSkills.ts` | all | Untouched — query key already scoped on `tenant?.id`. |

---

## 3. Decisions carried in from the design

| # | Decision | Consequence for the implementer |
|---|---|---|
| D1 | The JOB writer lives in package `job`, driven by a second pass in the SKILL worker | `job` imports `skill` (for `included`), so `skill` must **not** import `job`. `parseJobId` is duplicated rather than exported. |
| D2 | **Two REST types**: `RestModel` (persisted, relationship-free) and `ListRestModel` (list projection, never persisted) | Resolves PRD §9.3 / FR-4.4. Guarded by a raw-`content` test. |
| D3 | Linkage always; `included` only when requested | `GetReferencedIDs` reads `Skills`; `GetReferencedStructs` reads `resolved`. Divergence from `shops`, where linkage itself is include-gated. |
| D4 | `include=skills` resolves via **one** `DrainAllProvider`, never N `GetById`s | A 50-job page can reference ~3,000 skills. |
| D5 | Reader returns 0 or 1 model; non-numeric image → empty **and no error**; missing `skill` node → empty list | Two deliberate divergences from `skill.Read`, both required by FR-2.3/FR-2.4. |
| D6 | Any storage error → `ok=false` → 404 | "Unknown job" and "absent from this version" collapse deliberately. |
| D7 | `GET /data/jobs` follows `commodity`'s DB-side paging | Not `skill`'s drain-then-slice. |
| D8 | `Jobs` becomes a literal map; `Job` stays a struct | `map[Id]bool` is not a drop-in — `FromSkillId` returns `(Job, bool)`. |
| D9 | FR-1.4 is **verified**, not assumed | Baseline and purge both list `documents` whole-table; tests pin it. |
| D10 | UI: `major: number` → `available: ReadonlySet<number>`; visibility becomes async | The `isSuccess` gate is the entire regression mitigation. |
| D11 | UI gets one new client primitive and one new service method | `api.getList` unchanged; `useJobSkillDefinitions` deliberately untouched. |
| D12 | Zero JOB documents from a `Skill.wz` pass logs at warn | The failure mode a fallback would have hidden. |

**Design §7's two open questions are closed** (recommendations adopted): ship
`includeSkills` on the UI service with no production caller; name the projection
type `ListRestModel`.

---

## 4. Verified facts worth not re-deriving

**api2go v1.0.4 marshal behaviour** (`jsonapi/marshal.go`, read from the module
cache):
- `marshalSlice:186-208` calls `GetReferencedStructs()` unconditionally on any
  element implementing `MarshalIncludedRelations`.
- `filterDuplicates:208+` dedupes `included` by (type, id) — the endpoint does
  not need to dedupe itself.
- `if len(includedElements) > 0 { result.Included = … }` at line 203 — an empty
  `resolved` slice produces **no** `included` key. This is what makes D3 work
  without an explicit gate at the model layer.
- `MarshalPaginatedResponse` (`libs/atlas-rest/server/paginated_response.go:24-31`)
  marshals first and then attaches `Meta`/`Links`, so `Included` survives.

**Route conflict: none.** The ingest-job listing is `GET /api/data/process`
(`services/atlas-data/docs/rest.md:79`), not `/api/data/jobs`. The
`/data/jobs` subrouter currently registers only `/{jobId}/skills`, so adding
`""` is safe.

**Consumer audit** (re-run at plan time, not taken from the design):
- `Job.Skills()` — exactly one caller repo-wide: `atlas-data/job/processor.go:34-35`.
  Every other `.Skills()` hit is a different service's own model type
  (atlas-channel `character.Model`, atlas-pets, atlas-consumables,
  atlas-messages, atlas-monsters).
- `Job.Buffs()` — zero callers. The `.Buffs()` hits are all atlas-buffs' own
  `character.Model`.
- `Jobs[id]` outside `libs/atlas-constants` — exactly two, both existence checks:
  `atlas-configurations/.../templates/characters/preset/validator.go:76` and
  `.../tenants/characters/preset/validator.go:76`.
- The 82 exported `Job` value vars — zero external references.

**Counts** (`libs/atlas-constants/job/constants.go`): 82 `Job` value vars, 23
`fourthJob: true` markers, 1,236 lines.

**`job/mock/processor.go` has no consumer** in the tree today. It is still kept
in sync — an out-of-date mock breaks the `var _ job.Processor` assertion and
fails the build.

---

## 5. Traps

1. **Registry singleton bleeds across tests.** `GetModelRegistry()` is a
   package-level `sync.Once` keyed by `tenant.Model`. Every test must use a
   fresh `uuid.New()` tenant, or a prior test's cached entry silently satisfies
   a lookup that should 404.

2. **Canonical fallback fires on every miss.** A per-id lookup that misses the
   tenant falls back to `canonical.TenantId(region, major, minor)` — a
   *deterministic* UUID. Tests seeding under a random tenant are safe; a test
   that ever seeds under a canonical tenant would leak into every other test at
   the same version.

3. **`document_id` 0 is legitimate.** Job 0 is Beginner. `DbStorage.Add` derives
   the id by `strconv.Atoi(GetID())` — a "0 means unset" assumption anywhere in
   the chain silently drops Beginner.

4. **Skill order must not be sorted.** FR-1.2 says WZ document order. Sorting
   makes re-ingest produce different bytes and churns every baseline dump.

5. **Two REST types must stay two.** Merging them leaks `relationships` (and
   eventually `included`) into the stored `content` column *and* changes the
   `/{jobId}/skills` response shape PRD §5 pins as unchanged. The raw-`content`
   test in `job/rest_test.go` is the only thing standing between a future
   refactor and that bug.

6. **`registerAllInDirectory` swallows per-file errors.** A JOB pass that
   silently writes nothing looks identical to one that succeeded. That is
   exactly why D12's zero-document warn exists.

7. **UI visibility is now async.** `TenantProvider` calls
   `queryClient.clear()` on every tenant switch, so "empty set" is the normal
   state right after a switch. Any predicate that treats empty as "this version
   has no jobs" will bounce a valid `/jobs/112` to `/jobs` on every switch.

8. **`npm run build`, not `vitest`.** Vitest does not type-check; `tsc -b` runs
   only in the build. A signature change that compiles nowhere still passes
   `npm run test`.

9. **`tools/lint.sh --check` is a separate gate from `go vet`.** Both are
   required (CLAUDE.md items 2 and 8); the guard's govet is diff-gated while the
   standalone one runs full-module.

10. **Existing `JobsPage` tests need the new mock.** `JobsPage` will call
    `useJobs` unconditionally, so every pre-existing case in
    `JobsPage.test.tsx` needs `useJobsMock.mockReturnValue(...)` added to its
    arrange block or it will throw on `jobsQuery.isSuccess`.

---

## 6. Verification gates

Per CLAUDE.md §Build & Verification and design §9:

| Gate | Scope |
|---|---|
| `go test -race ./...` | `libs/atlas-constants`, `services/atlas-data/atlas.com/data` |
| `go vet ./...` | same two modules |
| `go build ./...` | same two modules |
| `docker buildx bake atlas-data` | **only if** a `go.mod` changed — verify with `git diff --stat`, don't assume |
| `tools/redis-key-guard.sh` | repo root |
| `tools/goroutine-guard.sh` | repo root |
| `tools/lint.sh --check` | repo root (fix mode: `tools/lint.sh`) |
| `npm run test` + `npm run build` | `services/atlas-ui` |
| `superpowers:requesting-code-review` | before opening the PR, reviewers pinned to Sonnet/Haiku |

Not applicable, but confirm rather than assume:
`tools/service-registration-guard.sh` (no service added) and
`tools/template-opcode-order-guard.sh` (no socket-config template touched).

---

## 7. Deliverables beyond code

- `docs/runbooks/job-document-backfill.md` — the 11-version rollout runbook.
  Executing it is the operator's job at deploy time; **producing it is this
  task's**.
- `services/atlas-data/docs/rest.md` — `GET /api/data/jobs`, plus the reworded
  404 on `/{jobId}/skills`.
- A PR description that names the visible legacy-tenant behaviour change (GM/
  Super GM vanish on GMS 48; Brigadier + Cygnus appear on GMS 72/79; Super GM
  vanishes on JMS 185) and links the runbook.
