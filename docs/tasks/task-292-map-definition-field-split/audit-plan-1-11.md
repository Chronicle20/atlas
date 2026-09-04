# Plan Audit — task-292-map-definition-field-split (Tasks 1–11)

**Plan Path:** docs/tasks/task-292-map-definition-field-split/plan.md
**Audit Date:** 2026-09-03
**Branch:** task-292-map-definition-field-split
**Base Branch:** main (merge base `9613e7259`)
**Scope:** Tasks 1–11 only (backend `atlas-maps`/`atlas-data` field/object work,
gateway routing, and the first `atlas-ui` terminology/badge task). Tasks 12–21
are covered by a separate shard.

## Executive Summary

All 11 tasks in this range have a corresponding commit on the branch, and in
every case the code inspected matches the plan's specified behavior, not just
the presence of a file. Test suites specified by the plan are present with
matching test names and table-driven cases. No stubs, placeholder handlers, or
TODO-marked incomplete work were found in the files touched by these tasks.
`tools/verify.sh` has already been confirmed green by the dispatcher, so build
and test status is not re-verified here. Completion rate for this range: 11/11
(100%).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `atlas-maps` tenant-scoped field occupancy read model | DONE | `services/atlas-maps/atlas.com/maps/map/character/registry.go:102-129` — `FieldOccupancy` + `GetFieldsWithCharacters` filters `mk.Tenant != t` and `len(mc)==0` inside `RLock`, matching plan verbatim. `processor.go:20,54-56` and `mock/processor.go:19,46-49` wired. `registry_test.go` has all 6 planned subtests (empty registry, single occupied field, drained field excluded, drained plus live, tenant isolation, two instances same map), commit `42c47c9fa`. |
| 2 | `atlas-maps` `GET /api/fields` resource | DONE | `field/rest.go` RestModel matches plan's shape exactly. `field/resource.go` implements `parseFilters`, tenant scoping via `tenant.MustFromContext`, filter-then-sort-then-paginate pipeline, `MarshalPaginatedResponse`. Uses `o.Field.Id()` (shared `field.IdFormat = "%d:%d:%d:%s"`, verified at `libs/atlas-constants/field/constants.go:4`) instead of manual `fmt.Sprintf` — functionally identical to plan spec, a reasonable equivalent (see commit `0a1d71a12`). All 6 planned test functions present in `field/resource_test.go` (`TestGetFieldsDrainedFieldExcluded`, `TestGetFieldsTenantIsolation`, `TestGetFieldsFilters`, `TestGetFieldsMalformedFilter`, `TestGetFieldsPaginationDeterminism`, `TestGetFieldsAttributes`). Route wired in `main.go:149` immediately after `_map.InitResource`, matching plan Step 5. Commits `ec27e22d8`, `0a1d71a12`. |
| 3 | `atlas-maps` document the field endpoint | DONE | `services/atlas-maps/docs/rest.md:377-435` documents `GET /api/fields` with liveness rule, tenant scoping, filter/pagination params, ordering, example JSON:API document, and error table. No `/home/` literal paths found. Commits `dc6bbd4c9`, `fe8177449` (doc-only fix removing an erroneous 500 from the docs). |
| 4 | `atlas-data` obstacle definition registry | DONE | `map/object_registry.go` matches plan's `InitObj`/`ResolveObjKind`/`MapObjectDefinition` design, including the plan's Step-3 correction returning `(int, error)` rather than swallowing the count. `object_registry_test.go` has all 3 planned test functions (`TestInitObjIndexesObstacles`, `TestInitObjMissingDirectory`, `TestInitObjTenantIsolation`). Commit `3d0d50fcb`. |
| 5 | `atlas-data` wire `InitObj` into both ingest entry points | DONE | `data/workers/mapw.go:51-56` — warn-and-continue policy (`l.WithError(err).Warnf(...)`) plus `defer Clear(t)`, correctly differentiated from `data/processor.go:126-131` — hard-error policy (`p.l.WithError(err).Errorf(...); return err`) plus `Clear(t)` at line 135 alongside the existing string-registry clear. This is exactly correction C2's intent: each site keeps its own pre-existing failure policy. Commit `d8640cf9f`. |
| 6 | `atlas-data` map object REST model and WZ reader | DONE | `map/object/rest.go` matches plan's `RestModel` field-for-field. `map/reader.go:350-390` `getObjects` iterates numeric layers via `strconv.Atoi(layer.Name)` then descends into the `obj` child (the `getBackgroundTypes` idiom per correction C1, not `getReactors`), skips unnamed entries, dedupes on `{kind}:{name}` keeping first, sorts by `(Kind, Name)`. Called at `reader.go:107` (`m.Objects = getObjects(t, exml)`), immediately after `m.Reactors = getReactors(exml)` per plan. `reader_test.go` has all 4 planned tests (`TestGetObjectsOnlyNamedEntries`, `TestGetObjectsResolvesObstacle`, `TestGetObjectsDuplicateIdKeepsFirst`, `TestGetObjectsEmptyLayers`). Commit `69227f6ba`. |
| 7 | `atlas-data` persist map objects as JSON:API relationship | DONE | `map/rest.go` — all four hooks present: `GetReferences` (line 81, `Type: "map-objects", Name: "objects"`), `GetReferencedIDs` (lines 114-120), `GetReferencedStructs` (lines 139-141), `SetToManyReferenceIDs` (lines 197-207, `name == "objects"` branch). `rest_test.go::TestMapObjectsSurviveJSONAPIRoundTrip` asserts field-by-field equality (Kind, ObjectSource, etc., lines 208-215) and checks the `included` array contains 2 `map-objects` entries with the expected ids — the load-bearing assertion the plan calls out. Commit `bf3ffb772`. |
| 8 | `atlas-data` `GET /api/data/maps/{mapId}/objects` | DONE | `map/processor.go:36,306-316` — `objectProvider`/`GetObjects` clone the `reactorProvider`/`GetReactors` shape exactly. `map/resource.go:40` registers the route; `resource.go:263-287` `handleGetMapObjectsRequest` clones `handleGetMapReactorsRequest`, uses `paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)` per atlas-data's own convention (correction C3), returns 404 on unknown map. `resource_test.go` has all 3 planned tests (`TestGetMapObjectsEmpty`, `TestGetMapObjectsReturnsRows`, `TestGetMapObjectsUnknownMap`). Commits `1360c785b`, `98549c072` (errcheck fix on `resp.Body.Close`). |
| 9 | `atlas-data` document map objects | DONE | `docs/rest.md:660-696` documents the endpoint with attributes, sort order, example, and error table matching plan spec. `docs/domain.md:56-67` documents the Object sub-model including the required migration note ("this sub-model was added to the stored Map document after existing tenants were already ingested... No migration is possible or needed"). No `/home/` literal paths found. Commit `642602613`. |
| 10 | Gateway routes | DONE | `deploy/shared/routes.conf:505-512` — both the `.../environment(/.*)?$` and `^/api/fields(/.*)?$` blocks inserted immediately after the `jukebox` block and before the `^/api/worlds(/.*)?$` catch-all at line 701 (confirmed via `grep -n`), matching correction C4's ordering requirement exactly. Directive style (`set $u`; `proxy_pass http://$u$request_uri;`) matches the surrounding blocks. The pre-existing `^/api/data(/.*)?$` catch-all (line 460) already covers `GET /api/data/maps/{mapId}/objects` from Task 8, so no additional atlas-data block was needed — consistent with the plan not requiring one. Commit `709bd51f4`. |
| 11 | `atlas-ui` Definition/Runtime terminology and kind badge | DONE | `SurfaceKindBadge.tsx` implements the exact component from the plan (word always rendered, `outline`/`default` variant by kind, not colour-only). `__tests__/SurfaceKindBadge.test.tsx` has all 3 planned test cases including the FR-3 "never relies on colour alone" test. `MapHeader.tsx:41-43` renders `<SurfaceKindBadge kind="definition" />` as the first child of the badge row, before `streetName`. `MapDetailTabs.tsx:106` renamed to `Monster Spawns {monsters && ...}`; `MapEntitySummary.tsx:117,128,154` renamed to `Configured Monster Spawns`. No hook/service files touched in this task's diff (FR-5 compliance). Commit `936316a26`. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task in this range was skipped, stubbed, or deferred.

## Build & Test Results

Per dispatcher instruction, the flagless `tools/verify.sh` has already been
confirmed green on this HEAD (`9f6679907`), including both Go modules under
`-race`, and was not re-run for this audit. Targeted evidence collected above
(test function names, exact source matching plan-specified logic) corroborates
that the Task 1–11 code compiles and its test suites exist as specified.

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-maps | PASS (per prior `verify.sh`) | PASS (per prior `verify.sh`) | Confirmed by dispatcher; not re-run. |
| atlas-data | PASS (per prior `verify.sh`) | PASS (per prior `verify.sh`) | Confirmed by dispatcher; not re-run. |
| atlas-ui (Task 11 slice only) | PASS (per prior `verify.sh`) | PASS (per prior `verify.sh`) | Confirmed by dispatcher; not re-run. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the Tasks 12–21 shard's own verdict, since this audit covers only Tasks 1–11)

## Action Items

None. No fixes required for Tasks 1–11.
