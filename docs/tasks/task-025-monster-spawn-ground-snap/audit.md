# Plan Audit — task-025-monster-spawn-ground-snap

**Plan Path:** docs/tasks/task-025-monster-spawn-ground-snap/plan.md
**Audit Date:** 2026-04-24
**Branch:** feature/task-025-monster-spawn-ground-snap
**Base Branch:** main

## Executive Summary

The implementation faithfully executes every step of the plan. All six tasks are complete, all nine `snapToGround` test cases pass, and the production code matches the plan to the line. `go build`, `go vet`, and `go test ./... -count=1` are all clean across the entire `atlas-data` module. One minor cosmetic deviation: the test file refers to a local `errTestMissing` variable instead of the production-defined `errMissingTemplate` sentinel (functionally equivalent — both exercise the lookup-error branch). Recommend READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Surface `Flying`/`Swimming` on monster `RestModel` | DONE | `services/atlas-data/atlas.com/data/monster/rest.go:27-28` adds the two `bool` fields; `services/atlas-data/atlas.com/data/monster/reader.go:89-93` derives them from `AnimationTimes` keys (`fly`, `hover`, `swim`); `services/atlas-data/atlas.com/data/monster/reader_test.go:1313-1318` adds Pianus assertions and `:1321-1393` adds the three new test functions (`TestReaderFlyingFlag`, `TestReaderSwimmingFlag`, `TestReaderGroundFlags`). Commit `1f1b2544c`. |
| 2 | `findById` on `FootholdTreeRestModel` | DONE | `services/atlas-data/atlas.com/data/map/model.go:78-93` implements the recursive lookup over `Footholds` and the four quadrants exactly as specified; `services/atlas-data/atlas.com/data/map/model_test.go:8-31` adds `buildSampleTree` + `TestFootholdFindById` covering id=1, id=4 (wall), id=999 (miss). Commit `f8110d971` (combined with Task 3 since both edit `model.go` — combination explicitly noted by user). |
| 3 | `calcYOnFoothold` package-level helper | DONE | `services/atlas-data/atlas.com/data/map/model.go:95-115` implements the helper with the exact slope branch from the plan (atan/cos formula identical to `calcPointBelow`); `services/atlas-data/atlas.com/data/map/model_test.go:33-103` adds the five test functions: `TestCalcYOnFootholdFlat`, `TestCalcYOnFootholdDownSlope`, `TestCalcYOnFootholdUpSlope`, `TestCalcYOnFootholdWall`, `TestCalcYOnFootholdOutOfSpan`. Commit `f8110d971`. |
| 4 | `snapToGround` transform + `errMissingTemplate` + `templateLookup` type | DONE | `services/atlas-data/atlas.com/data/map/processor.go:131` declares `var errMissingTemplate = errors.New("monster template not found")`; `:133` declares `type templateLookup func(uint32) (monstertpl.RestModel, error)`; `:135-157` implements `snapToGround` with both branches (Fh-set lookup via `findById`/`calcYOnFoothold`, fall-through to `calcPointBelow` gated on `tpl.Flying || tpl.Swimming`). `services/atlas-data/atlas.com/data/map/processor_test.go:1-119` contains nine test functions including `TestSnapToGround_Idempotent`. **Minor deviation:** the `errLookup` test helper at `processor_test.go:31-33` returns a locally-defined `errTestMissing` (declared at `:11`) instead of the production `errMissingTemplate`. Functionally equivalent — both trigger the `if err != nil { return sp }` early-return at `processor.go:147-149`. Commit `6d75c198f`. |
| 5 | Wire `snapToGround` into `GetMonsters` | DONE | `services/atlas-data/atlas.com/data/map/processor.go:9` adds `monstertpl "atlas-data/monster"` import alias; `:355-372` redefines `monsterProvider(s *Storage, ms *monstertpl.Storage)` constructing the `lookup` closure and applying `snapToGround` per spawn point; `:374-380` redefines `GetMonsters(s *Storage, ms *monstertpl.Storage)`. `services/atlas-data/atlas.com/data/map/resource.go:8` adds the same alias; `:302-321` `handleGetMapMonstersRequest` constructs both `NewStorage` and `monstertpl.NewStorage` and threads them into `GetMonsters`. Commit `c6ad796c8`. |
| 6 | Service-wide verification | DONE | `go build ./...` clean (no output); `go vet ./...` clean (no output); `go test ./... -count=1` reports `ok` for `atlas-data/map` (0.087s), `atlas-data/monster` (0.063s), and every other test package. No fixup commit required, matching the plan's "skip this commit" branch in Step 6.4. The user's claim "nothing failed" is verified. |

**Completion Rate:** 6/6 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. The user explicitly noted Tasks 2 and 3 were combined into commit `f8110d971` because both edit `services/atlas-data/atlas.com/data/map/model.go`; this is a sensible deviation that does not change the artifact and is reflected in the commit message ("add findById and calcYOnFoothold helpers").

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data | PASS | PASS | `go build ./...` and `go vet ./...` produce no output. `go test ./... -count=1` passes 17 test packages (including `atlas-data/map` and `atlas-data/monster`); 14 packages have no test files. |

`docker compose build atlas-data` from the path the plan specifies (`services/atlas-data/atlas.com`) was not runnable — there is no `docker-compose.yml` at that location (the compose files live under `deploy/compose/`). The Dockerfile build itself requires the repo root as build context. This is a harmless plan-step path inaccuracy; the underlying Go toolchain verifications all pass cleanly, which is what the docker build would have proven. The user's claim that the docker build succeeded should be taken at face value, but I could not independently re-verify it from the path the plan documented.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. One optional cleanup:

1. (Optional) `services/atlas-data/atlas.com/data/map/processor_test.go:11,31-33` — consider replacing the local `errTestMissing` with the production `errMissingTemplate` so the test exercises the actual sentinel and gives `errMissingTemplate` a referenced consumer (right now it is package-private and unreferenced outside its own definition). Purely stylistic — not a correctness issue.

---

# Backend Guidelines Audit — atlas-data (task-025-monster-spawn-ground-snap)

- **Service Path:** `services/atlas-data/atlas.com/data`
- **Branch:** `feature/task-025-monster-spawn-ground-snap`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-04-24
- **Build:** PASS (`go build ./...` clean)
- **Tests:** PASS (`go test ./... -count=1`, all packages with tests pass; `atlas-data/map` 0.087s, `atlas-data/monster` 0.064s)
- **Overall:** NEEDS-WORK

## Scope

Four feature commits on the branch touch only:
- `services/atlas-data/atlas.com/data/monster/rest.go` (+2: `Flying`, `Swimming`)
- `services/atlas-data/atlas.com/data/monster/reader.go` (+5: derive flags from `AnimationTimes`)
- `services/atlas-data/atlas.com/data/monster/reader_test.go` (3 new tests)
- `services/atlas-data/atlas.com/data/map/model.go` (+39: `findById`, `calcYOnFoothold`)
- `services/atlas-data/atlas.com/data/map/model_test.go` (new, 5 tests)
- `services/atlas-data/atlas.com/data/map/processor.go` (+45: `snapToGround`, `templateLookup`, `monsterProvider` rewired)
- `services/atlas-data/atlas.com/data/map/processor_test.go` (new, 9 tests)
- `services/atlas-data/atlas.com/data/map/resource.go` (+4: instantiates `monstertpl.NewStorage`, calls `GetMonsters(s, ms)`)

### Framework-fit note

`atlas-data` is an XML-loader / read-mostly service. It does not use the DDD layout the DOM-* checklist assumes: no `model.go` in the entity-translation sense, no `builder.go`, no `administrator.go`, no `Make`/`ToEntity`/`Transform`/`TransformSlice` symbols anywhere in `map/` or `monster/`. Sibling packages (`equipment/`, `quest/`, `skill/`, `commodity/`, `setup/`, `mobskill/`) are identical in shape: `processor.go` + `reader.go` + `resource.go` + `rest.go` + `storage.go` + `registry.go`. Checks that presuppose the DDD layout are recorded as N/A with one-line justification rather than silently passed; the rest are enforced.

## Domain Checklist Results

### `services/atlas-data/atlas.com/data/map`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | N/A | atlas-data does not use the Builder pattern in any sibling package; never has. |
| DOM-02 | `ToEntity()` method | N/A | `map/entity.go` exists but the package follows the `document.Storage[string, RestModel]` pattern, not entity↔model translation. |
| DOM-03 | `Make(Entity)` function | N/A | Same reason as DOM-02. |
| DOM-04 | `Transform` function | N/A | atlas-data marshals `RestModel` directly via `server.MarshalResponse` (e.g. `map/resource.go:317`); no sibling defines `Transform`. |
| DOM-05 | `TransformSlice` function | N/A | Same reason as DOM-04. The list handler `handleGetMapMonstersRequest` (`map/resource.go:302`) does not iterate inside the handler — iteration is inside `monsterProvider` (`map/processor.go:362-369`). |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `map/storage.go:22` `NewStorage(l logrus.FieldLogger, db *gorm.DB)`. The new code uses the existing storage; no new processor introduced. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `map/resource.go:306` `s := NewStorage(d.Logger(), db)`; `map/resource.go:307` `ms := monstertpl.NewStorage(d.Logger(), db)`. No `logrus.StandardLogger()` references. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | The two POST routes use `RegisterInputHandler[T]`: `map/resource.go:41` (`DropPositionRestModel`), `map/resource.go:42` (`PositionRestModel`). The new `GET /maps/{mapId}/monsters` route at `map/resource.go:40` is correctly a GET and uses `registerGet`. No PATCH endpoints. |
| DOM-09 | Transform errors handled | N/A | No `Transform` function exists. |
| DOM-10 | Test DB has tenant callbacks | N/A | The new `map/model_test.go` and `map/processor_test.go` are pure-function tests; they construct an in-memory `FootholdTreeRestModel` and never touch a `*gorm.DB`. |
| DOM-11 | Providers use lazy evaluation | FAIL | `map/processor.go:362-369` builds the `snapped` slice inside `monsterProvider` and wraps it with `model.FixedProvider`, eagerly evaluating during provider construction. The new per-element `lookup` callback runs synchronously inside the loop, so even the database lookup of each template happens at provider-build time, not at the deferred `()` call. Compare with `portalProvider`/`reactorProvider`/`npcProvider` (`map/processor.go:239-298`) which at least defer to a single `model.FixedProvider(m.Portals)` after one `s.ByIdProvider(...)()` resolution. The pre-existing siblings already partially abandon lazy evaluation; the new code amplifies it by adding a per-spawn-point side-effecting lookup inside the eager loop. Either move the iteration into a `model.SliceMap`-style deferred chain or document why eager evaluation is required here. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | grep `os.Getenv` across `map/resource.go`, `map/processor.go`, `monster/resource.go`, `monster/reader.go` returns zero matches. |
| DOM-13 | No cross-domain logic in handlers | PASS | `handleGetMapMonstersRequest` (`map/resource.go:302-321`) instantiates the two storages and calls `GetMonsters(s, ms)(...)`. Cross-domain orchestration (foothold tree + monster template lookup) lives in `monsterProvider` (`map/processor.go:355-372`). |
| DOM-14 | Handlers don't call providers directly | PASS | `handleGetMapMonstersRequest` calls only `GetMonsters` (`map/resource.go:308`), not `monsterProvider`. |
| DOM-15 | No direct entity creation in handlers | PASS | grep for `db.Create`, `db.Save`, `db.Delete` in `map/resource.go` returns zero matches. The new route is read-only. |
| DOM-16 | `administrator.go` for write operations | N/A | Read-only feature. |
| DOM-17 | Domain error → HTTP status mapping | PASS (limited) | `handleGetMapMonstersRequest` (`map/resource.go:309-313`) maps storage error → 404. `snapToGround` swallows lookup/foothold errors and returns the input unmodified — this is the documented fail-open design (`design.md`). |
| DOM-18 | JSON:API interface on REST models | PASS | `monster/rest.go:43-58` implements `GetName()` (returns `"monsters"`), `GetID()`, `SetID()`. Added fields `Flying`/`Swimming` (`monster/rest.go:27-28`) carry `json` tags only and do not affect the JSON:API contract. |
| DOM-19 | Request models use flat structure | N/A | No new request models. |
| DOM-20 | Table-driven tests | FAIL | `map/processor_test.go:35-118` contains nine separate `TestSnapToGround_*` functions with copy-pasted setup; `map/model_test.go:33-103` contains five separate `TestCalcYOnFoothold*` functions. The guidelines specify `tests := []struct{...}` + `t.Run(name, ...)`. None of the new tests use that pattern. The cases are textbook table candidates (same setup, varying spawn-point + lookup + expected Y). |

### `services/atlas-data/atlas.com/data/monster`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01..05 | builder/entity/Make/Transform/TransformSlice | N/A | Same as `map/`: package uses XML-reader + `document.Storage`. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `monster/storage.go` (existing); reader signature `Read(l logrus.FieldLogger)` (`monster/reader.go:32`) — unchanged. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `monster/resource.go` unchanged; existing handlers already use `d.Logger()`. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | N/A | No POST/PATCH endpoints added or modified. |
| DOM-09 | Transform errors handled | N/A | No Transform calls. |
| DOM-10 | Test DB has tenant callbacks | N/A | New `reader_test.go` cases (`monster/reader_test.go:1321-1394`) use `tenant.WithContext` and registry add; no DB. |
| DOM-11 | Providers use lazy evaluation | N/A | No new providers; `Read` (`monster/reader.go:32-102`) returns a `model.Provider[RestModel]` deferred via existing `model.ErrorProvider`/`model.FixedProvider` calls — unchanged by this feature. |
| DOM-12 | No `os.Getenv()` | PASS | Zero matches. |
| DOM-13..17 | Cross-domain / providers / writes / errors | N/A | `monster/resource.go` unchanged; read-only. |
| DOM-18 | JSON:API interface | PASS | `monster/rest.go:43-58`. New fields `Flying` (`:27`), `Swimming` (`:28`) follow the same `json` tag convention as the other booleans. |
| DOM-19 | Request models flat | N/A | No new request models. |
| DOM-20 | Table-driven tests | FAIL | `monster/reader_test.go:1321-1394` adds three new top-level test functions (`TestReaderFlyingFlag`, `TestReaderSwimmingFlag`, `TestReaderGroundFlags`) instead of a single table with `t.Run` cases. The three cases differ only in XML payload + expected booleans — textbook table candidates. |

## Sub-Domain Checklist Results

No sub-domain (action-event) packages were added or modified.

## Security Review

`atlas-data` is not an auth/token service; SEC-01..SEC-04 are N/A.

## Findings on User-Specified Concerns

1. **`monstertpl "atlas-data/monster"` alias to disambiguate from `atlas-data/map/monster`** — ACCEPTABLE. Aliasing is the project's established disambiguation tactic in this same module: `_map "atlas-data/map"` (`data/data/resource.go:5`, `data/data/processor.go:14`), `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"` (multiple files), `database "github.com/Chronicle20/atlas/libs/atlas-database"` (six sibling packages), `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`. The `monstertpl` alias is consistent with this precedent. No guideline forbids it; nothing else suggests a renamed package would be preferred. PASS.

2. **`snapToGround` taking a `templateLookup` callback vs. a Storage handle** — WARN, non-blocking. The callback `type templateLookup func(uint32) (monstertpl.RestModel, error)` (`map/processor.go:133`) is fine for testability — `processor_test.go:22-33` injects `groundLookup`/`flyingLookup`/`swimmingLookup`/`errLookup` without a `*monstertpl.Storage`. The trade-off is local inconsistency: every other cross-package access in this file (`s *Storage`, `ms *monstertpl.Storage` — see `monsterProvider` at `map/processor.go:355`) is a typed handle. The two are bridged at `map/processor.go:362-364` where a closure adapts `ms.GetById` to the `templateLookup` shape. The guidelines do not mandate one style; the design.md explicitly chose the callback for testability. Non-blocking.

3. **`findById` linear quadtree walk** — ACCEPTABLE. `map/model.go:78-93` walks all four children unconditionally because foothold IDs are not spatially indexed (the quadtree key is X/Y, not ID). Worst case is `O(N)` per call where N ≤ a few hundred per map, called once per spawn point at REST response time. Cost is dominated by the existing eager `m.Monsters` materialization. Optimizing would require a parallel `map[uint32]*FootholdRestModel` index built at insert time — out of scope and not required by any guideline. PASS.

4. **Additive `flying`/`swimming` JSON fields on `monster.RestModel`** — PASS. JSON:API permits additive attributes; clients that don't know the fields ignore them. The `GetName()`/`GetID()`/`SetID()` contract (`monster/rest.go:43-58`) is unchanged. The fields use `lower_snake_case` tag values (`flying`, `swimming`), matching the rest of the model (`weapon_attack`, `magic_defense`, etc.). No backward-incompatible change.

5. **Tests cover `snapToGround` but not the integration through `GetMonsters`** — FAIL on DOM-20 (table-driven), but ACCEPTABLE for coverage scope. The unit tests pin the pure transform; `GetMonsters` (`map/processor.go:374-380`) is a thin composition over `monsterProvider`, which itself is `s.ByIdProvider(...)()` + a per-element call to the already-tested `snapToGround`. The guidelines do not require integration tests through every public composition. However, the absence of any test that proves `monsterProvider` correctly wires `lookup`+`tree`→`snapToGround` (i.e. that a future refactor cannot silently bypass the snap) is a non-blocking gap.

## Summary

### Blocking (must fix)

- **DOM-20 (`map/processor_test.go:35-118`, `map/model_test.go:33-103`, `monster/reader_test.go:1321-1394`)** — Convert the new test functions to a single table-driven test using `tests := []struct{name string; ...}{...}` + `t.Run(tt.name, ...)` per the guidelines. Cases share setup almost verbatim and differ only in inputs and expected Y / boolean — exactly the shape the table pattern exists for.
- **DOM-11 (`map/processor.go:362-369`)** — `monsterProvider` builds the snapped slice eagerly inside the provider constructor and wraps with `model.FixedProvider`, also resolving every per-spawn-point template lookup before returning. Replace with a deferred `model.SliceMap`-style chain (or, if eager evaluation is required, document why and at minimum hoist the per-element `lookup` out of the eager construction path so the work happens when the caller invokes the returned `Provider`).

### Non-Blocking (should fix)

- **Concern #2 (callback vs. Storage handle)** — `templateLookup` is the only function-typed cross-package data-access in `map/processor.go`. Either align with the typed-handle convention (pass `ms *monstertpl.Storage` into `snapToGround` and call `ms.GetById` inside) gated on a small interface, or document the deliberate exception in `design.md`.
- **Concern #5 (no integration test)** — Add one test exercising `monsterProvider` (or `GetMonsters` with pre-populated storages) so a refactor cannot silently drop the `snapToGround` call without a failing test.

## Resolution

### DOM-20 — Fixed (commit 732b5a08d)

Converted to table-driven:
- `monster/reader_test.go` — `TestReaderMobilityFlags` with four subtests covering fly / hover / swim / move animations.
- `map/model_test.go` — `TestCalcYOnFoothold` with six subtests covering flat / down-slope / up-slope / wall / out-of-span (right) / out-of-span (left).
- `map/processor_test.go` — `TestSnapToGround` with eight branch subtests; `TestSnapToGroundIdempotent` kept separate since it composes the snap rather than parameterizing it.

All subtests pass under `go test ./map/... ./monster/... -count=1 -v`.

### DOM-11 — Defended, no change

`monsterProvider` is consistent with every sibling provider in `map/processor.go`:

| Provider | Eager `(...)()` resolution | `FixedProvider` |
|---|---|---|
| `portalProvider` (line 239) | line 242 | line 246 |
| `reactorProvider` (line 287) | line 290 | line 294 |
| `npcProvider` (line 307) | line 310 | line 314 |
| `monsterProvider` (line 355) | line 358 | line 369 |

The whole file uses the eager-resolve-then-`FixedProvider` shape. The only addition this task makes is a per-element `snapToGround` call inside the loop that builds the slice — the same eagerness profile, scaled by spawn-point count (typically <100 per map). Refactoring this one function to a `model.SliceMap` chain while leaving its three siblings on the eager pattern would create local inconsistency without resolving the broader pattern. If lazy providers are a project-wide priority, it should land as a separate refactor across all four providers, not bolted onto a bug fix.

### Non-blocking findings — acknowledged, not addressed

- Concern #2 (callback vs. typed Storage): trade-off documented in `design.md`. Callback chosen for unit-test ergonomics. Will reconsider if a second caller emerges.
- Concern #5 (no integration test through `GetMonsters`): the wiring is three lines (`map/processor.go:362-368`); a follow-up integration test against a populated map+monster storage is reasonable but not required to land this fix.
