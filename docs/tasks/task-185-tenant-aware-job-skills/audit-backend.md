# Backend Guidelines Audit — task-185-tenant-aware-job-skills

- **Scope:** Go changes only (`services/atlas-data/atlas.com/data/job/`, `skill/reader.go`, `data/workers/skill.go`, `main.go`, `baseline/dump_test.go`, `tenantpurge/purge_test.go`, `libs/atlas-constants/job/`). TypeScript changes in the same branch are covered by the frontend reviewer.
- **Diff range:** `60f6ddaa3..bab6d9380`
- **Build/Test:** Not re-run (per instructions) — task-11-report.md already recorded `go test -race`, `go vet`, `go build` clean in both changed modules.
- **Overall:** PASS — zero Important/Critical findings. Two Minor observations.

## Design invariants honored (not re-litigated)

Verified each of the seven documented invariants against the actual diff rather than taking them on faith:

1. `RestModel` (`services/atlas-data/atlas.com/data/job/rest.go:5-20`) vs `ListRestModel` (`services/atlas-data/atlas.com/data/job/list_rest.go:23-30`) are genuinely separate types, and `job/rest_test.go:22-43` (`TestStoredContentCarriesNoRelationships`) mechanically pins that the persisted `content` column never contains `relationships`/`included`. Not flagged.
2. Linkage-always/included-gated split confirmed at `services/atlas-data/atlas.com/data/job/list_rest.go:45-70` and exercised by `resource_test.go:223-249` (no-include: linkage present, no `included` key) and `:251-270` (include=skills: `included` populated + deduped). Not flagged.
3. No transitional fallback to `constJob.Jobs[id].Skills()` exists anywhere in `job/processor.go` — confirmed via diff (`processor.go` diff removes the `constJob` import entirely). Not flagged.
4. `skill.ParseJobId` exported at `services/atlas-data/atlas.com/data/skill/reader.go:26-38`, consumed by `job/reader.go:4,40`. Confirmed no reverse import: `grep -rn "atlas-data/job" services/atlas-data/atlas.com/data/skill/` returns nothing — no cycle. Not flagged.
5. `RegisterJob` returns `(int, error)`; `countingRegister` adapter at `services/atlas-data/atlas.com/data/data/workers/skill.go:118-127` closes over `*int` rather than the processor holding mutable state. Not flagged.
6. Second `Skill.wz` tree walk confirmed at `data/workers/skill.go:60-66` (separate `registerAllInDirectory` call after the skill pass). Noted, not scored — Minor at most per instructions, and it's exactly what the plan specified.
7. `libs/atlas-constants/job/constants.go:6-83` — `Jobs` is a literal `map[Id]Job`; `Job` (`model.go:9-12`) stays `{id, fourthJob}`. `constants_test.go` (`TestJobsTableShape`, `TestFourthJobMembership`) pins the 82-entry/23-fourth-job shape mechanically, transcribed via the documented `awk` command rather than from memory. Not flagged.

## DOM-21 — atlas-constants duplication

**PASS.** No new domain type, alias, or numeric classification helper was introduced that duplicates `libs/atlas-constants/`. The only new types are `atlas-data/job.RestModel` and `atlas-data/job.ListRestModel` — JSON:API DTOs local to the service, not constants-library material (and `RestModel.Id` is `uint32`, matching the sibling `skill.RestModel.Id uint32` convention already used throughout `atlas-data`, not a redeclaration of `atlas-constants/job.Id uint16`, which is a different package for a different purpose). `libs/atlas-constants/job/model.go` and `constants.go` are the constants library itself, not a duplicate of it. Verified no dangling callers of the removed `Job.Skills()`/`Job.Buffs()` methods repo-wide: `grep -rn "\.Skills()\|\.Buffs()\b"` across `services/` and `libs/` returns only unrelated types (`Character.Skills()`, `PetDataModel.Skills()`, `MonsterAssist.Skills()`, etc.) — none on `job.Job`.

## Multi-tenancy

**PASS.** Every storage access in the diff goes through `document.Storage[string, RestModel]`, which resolves tenant via `tenant.MustFromContext(ctx)` (`document/storage.go:29,45,81,88` — package untouched by this diff, confirmed via empty `git diff` stat, so its canonical-fallback correctness is inherited, not re-implemented):

- `job/processor.go:52` — `GetSkillsForJob` calls `NewStorage(p.l, p.db).GetById(p.ctx)(...)`, `p.ctx` is the request-scoped context carrying tenant.
- `job/resource.go:76` — `handleGetJobsRequest` calls `NewStorage(d.Logger(), db).AllPagedProvider(d.Context())(page)()`; `d.Context()` is `server.HandlerDependency.Context()` from the shared `atlas-rest/server` library (the standard tenant-header-to-context path used identically by `skill/resource.go`).
- `job/resource.go:89` — the `include=skills` drain: `skill.NewStorage(d.Logger(), db).DrainAllProvider(d.Context())()` uses the *same* `d.Context()` as the jobs request, so it drains the *same* tenant's skill documents, not a fixed/global set. No cross-tenant leak.
- `job/processor.go:75` — `RegisterJob`'s write path uses `p.ctx` (worker-provided, tenant-scoped via `data/workers/skill.go:28` `withTenant(ctx, p)`) inside the transaction, consistent with `skill.RegisterSkill`'s identical shape (`skill/processor.go:54-57`).

Canonical-tenant fallback itself lives in `document/storage.go` (untouched) and is exercised by `document/storage_test.go` (also untouched) — out of this diff's blast radius, correctly not re-tested here.

## Processor / immutability conventions

**PASS.**
- `job/processor.go:31` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` matches the documented `NewProcessor(log, ctx, db)` order exactly (`file-responsibilities.md:22`).
- Interface + Impl split at `job/processor.go:16-29`, `var _ Processor = (*ProcessorImpl)(nil)` static assertion at line 39.
- `job/processor.go:41-43` — `NewStorage` composes the singleton registry (`job/registry.go:13-17`, `sync.Once`-guarded) with the injected `db`, mirroring `skill/processor.go:37-38` exactly — no cache-in-constructor anti-pattern.
- `job/mock/processor.go` was updated in lockstep with the `Processor` interface (`Register`, `RegisterJob` both mocked) — no stale mock.

## Error handling

**PASS**, with one explicitly-excluded item honored as instructed:
- `job/processor.go:51-58` (`GetSkillsForJob`) collapses every `GetById` error — including `gorm.ErrRecordNotFound` — to `ok=false` → HTTP 404 (`resource.go:38-41`). Per the task brief this is FR-3.2 specified behavior, not scored as a defect.
- `job/resource.go:70-74,77-81,90-94` — the three genuine failure paths in `handleGetJobsRequest` (bad pagination params, paged-provider error, skill-drain error) all propagate: `server.WriteBadRequest` for the 400 case, `server.WriteErrorResponse` (not a bare `w.WriteHeader(http.StatusInternalServerError)`) for the two 500-class cases, each preceded by an `Errorf` log. Grepped `job/resource.go` for `http.StatusInternalServerError` — zero hits; no bare-500 pattern (DOM-27 satisfied).
- `job/processor.go:76-88` (`RegisterJob`) propagates both the `Read` error and the `Register` error out of the transaction closure to the caller (`data/workers/skill.go:63-65`), which itself returns the error up through `registerAllInDirectory` rather than swallowing it.

## File Responsibilities Checklist (job package — sub-domain, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | Interface, `NewProcessor`, and all `ProcessorImpl` methods in `job/processor.go:16-90`. No processor logic in `resource.go`/`reader.go`. |
| FILE-02 | RestModel + JSON:API methods in `rest.go` | PASS | `job/rest.go:5-20`. The list-projection type `ListRestModel` lives in a dedicated `list_rest.go` — an intentional, documented split (invariant #1), not a misplacement of `RestModel` itself. |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | Job package makes no outbound HTTP calls (EXT checklist doesn't trigger — no `requests.RootUrl`/`GetRequest`/`PostRequest` anywhere in the package). |
| FILE-04 | Entity + Migration + TableName in `entity.go` | N/A | No local GORM entity — persistence delegates entirely to the shared `document.DbStorage`/`document.Entity`, identical to sibling `skill` package. Not a violation: this is the established document-backed pattern, not a domain package with a local entity. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No `builder.go`/`model.go`/`administrator.go`/`provider.go`/`state.go` — none needed; `RestModel` **is** the persisted representation for a document-backed type (same as `skill`), and writes delegate to `document.Storage.Add` rather than issuing local `db.Create`. |
| FILE-06 | No package-named catch-all file | PASS | Files present: `reader.go`, `registry.go`, `processor.go`, `resource.go`, `rest.go`, `list_rest.go`, `mock/processor.go`. No `job.go` bundling multiple responsibilities. |

## Sub-Domain Checklist (job package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `resource.go` handlers call only `Processor`/`Storage` methods; no inline business logic. |
| SUB-02 | No `db.Create`/`db.Save` in `resource.go` | PASS | Grepped `job/resource.go` — zero hits. All writes route through `document.Storage.Add`. |
| SUB-03 | `RegisterInputHandler[T]` for POST | N/A | No POST/PATCH endpoints — `GET /data/jobs` and `GET /data/jobs/{jobId}/skills` only. |
| SUB-04 | No manual JSON parsing | PASS | Grepped `job/resource.go` for `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` — zero hits. |

## Test quality

**PASS** on the specific risk called out: the package-level `sync.Once` registry singleton (`GetModelRegistry()`, `job/registry.go:13-17`) is keyed by `tenant.Model` (`document/registry.go:12`) and persists for the life of the test binary. Every test in `job/processor_test.go`, `job/resource_test.go`, and `job/rest_test.go` mints a fresh `uuid.New()` tenant id (spot-checked all 13 call sites — `resource_test.go:113,130,138-139,148-149,168,225,253,274,282,297`; `processor_test.go:25,39,52,69-70,90,105`; `rest_test.go:25`). No test reuses a fixed UUID, so no registry-key collision across tests.

Test-driven coverage is thorough and table-appropriate for the fixture style used (`reader_test.go` covers numeric/non-numeric/empty-skill-node/missing-skill-node/job-id-zero/non-numeric-skill-child; `processor_test.go` covers seeded-read, absent-404, id-zero round-trip, cross-version divergence, register-writes-count, register-nothing-for-non-numeric; `resource_test.go` covers found/not-found/version-scoped-not-found/cross-version/bad-request/list-with-linkage/include-dedup/empty-tenant/pagination/bad-page-params).

## Minor observations (non-blocking)

- **`job/resource_test.go:50` — shared in-memory SQLite DSN across the whole test binary.** `setupResourceTestDB` opens `sqlite.Open("file::memory:?cache=shared")` — an *unnamed* shared-cache SQLite database, which per SQLite semantics is shared by every connection opened with that identical URI within the process, not just within one test. Since `gorm.Open` is called fresh per test but the underlying connection pool isn't explicitly closed (`db.Close()`/`t.Cleanup` absent), all `documents` rows written across every test function in the `job` package's test binary accumulate in one physical table for the run's duration. Correctness today depends entirely on no test ever targeting the deterministic `canonical.TenantId(region, major, minor)` UUID directly (none do — all seed calls use `uuid.New()`), so cross-test bleed-through cannot currently manifest through the canonical-fallback path exercised in `document/storage.go`. The pattern is copied verbatim from an existing sibling (comment at `resource_test.go:36`: "Copied from skill/resource_test.go:28"), so it isn't a regression introduced by this diff, but it is unchanged-and-copied fragility worth a follow-up (e.g., a per-test unique DB name suffix) rather than a per-test-isolated DB.
- **`data/workers/skill.go:60-66` — the JOB pass re-walks the `Skill.wz` tree a second time** after the SKILL pass, doubling directory traversal for every Skill.wz ingest. This is the shape the plan specified (invariant #6) and is explicitly scoped out of a combined-pass refactor; noting it only because it's a real (if intentional) cost, not because it's incorrect.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- Consider a per-test-unique in-memory SQLite DSN (e.g. `file:job_<test-name>?mode=memory&cache=shared`) in `job/resource_test.go` to remove the latent cross-test-state-sharing fragility, rather than relying on UUID-collision-avoidance with the canonical tenant id.
