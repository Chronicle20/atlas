# task-292 — Planning Context

Companion to `plan.md`. Records the key files, the decisions made at plan time,
the dependency order, and the corrections planning made to `design.md`.

## Scope decision made at plan time

`[278]` (`task-278-map-environment-object-state`) is **not on `main`** —
verified: `git ls-tree -r --name-only origin/main -- services/atlas-maps/atlas.com/maps/map/environment`
returns nothing, and `field.ObjectKind` is absent from `libs/atlas-constants`
on `main`. Design §10 item 1 flagged this as needing a human decision.

**The user chose: plan now, with the runtime half as trailing gated tasks.**
Tasks 1–21 build on `main` today. Task 22 (FR-32 tracked half, FR-34–FR-38 Set
/ Reset writes) carries an explicit BLOCKED marker and a `git ls-tree` check as
its first step; execution must stop there until `[278]` lands and this branch
rebases onto it. The definition half of the Map Objects tab (Task 20) and its
whole layout still ship, so nothing about that panel is stubbed — a field with
no tracked state renders exactly what Task 20 produces.

The gateway `/environment` location block (Task 10) is the one deliberate
exception. It is inert and harmless before `[278]` lands (nginx proxies to
`atlas-maps`, which returns its own 404), and the PRD requires it regardless.
It is routing config for an endpoint another branch already implements — not an
unimplemented handler.

## Corrections planning made to `design.md`

These are recorded in `plan.md`'s Global Constraints as C1–C9 and **override the
design**. Each was verified against code in this worktree.

| # | Impact |
|---|---|
| C1 | `getObjects` must follow the numeric-layer idiom of `getBackgroundTypes` (`reader.go:323-338`), not `getReactors` (`reader.go:341`). Reactors sit at one top-level node; objects sit per layer under an `obj` child. Cloning `getReactors` would have produced an empty result on every map. |
| C2 | The two ingest `InitString` neighbours have **different** failure policies — `data/workers/mapw.go:39-41` warns and continues, `data/data/processor.go:117-121` returns the error. Design §10 item 3 said "follow whichever the neighbour does" assuming they agreed. They do not; each `InitObj` call matches its own site. |
| C3 | `paginate.ParseParams` defaults differ per service: atlas-maps `(MaxPageSize, MaxPageSize)`, atlas-data `(DefaultPageSize, MaxPageSize)`. Copying one into the other silently changes the default page size. |
| C4 | The `^/api/worlds(/.*)?$` catch-all at `routes.conf:691` proxies to **`atlas-world`**, not `atlas-maps`. An unrouted `/environment` path would reach the wrong service. The explicit block is mandatory, not cosmetic. |
| C5 | The sort-then-paginate block is `map/resource.go:56-59`, not `:57-60`. |
| C6 | Design §10 item 2 (Characters-tab columns) is **resolved**: `atlas-character`'s REST model emits `name`, `level`, `jobId`, `x`, `y`, `fh`, `stance`, and explicitly **not** `mapId` (`rest.go:42-44` — atlas-maps owns character location). Columns are Name, Character ID, Level, Job, Position. **No State column** — `stance` is an animation frame, not a state. |
| C7 | `useGridRefresh` exists but is **not** currently used by `MapDetailPage.tsx`; it is new to this feature area, not pre-wired. |
| C8 | `[278]`'s JSON:API type is `environment-objects`, not `environment`. |
| C9 | `610030300` yields **seven** rows, not six: six `OBSTACLE` (`menhir0`–`menhir5`) plus one `ENVIRONMENT` (`3pt`, no `obstacle` flag). Do not write a test asserting a row count of six. |

## Key files by service

### `atlas-maps` (module root `services/atlas-maps/atlas.com/maps`)

| File | Role |
|---|---|
| `map/character/registry.go` | `Registry` = `map[MapKey][]uint32` under `sync.RWMutex`. `GetMapsWithCharacters` (line 89) is cross-tenant and stays untouched — its only caller, `tasks/respawn.go:39`, wants that. `GetInMapAllInstances` (line 74) is the only existing `mk.Tenant == t` filter and is the pattern to copy. |
| `map/character/model.go:8-11` | `MapKey{Tenant tenant.Model; Field field.Model}` |
| `map/character/processor.go:16-23` | `Processor` interface |
| `map/character/mock/processor.go` | **Handwritten** mock — no `//go:generate`. Every interface method needs a `XxxFunc` field plus a method. |
| `map/resource.go:27-74` | The resource template: registration closure, `paginate.ParseParams`, `server.WriteBadRequest`, sort-then-`paginate.Slice`, `MarshalPaginatedResponse`. |
| `map/resource_paginate_test.go:26-77` | The test harness: server-information stub, `setupMapRouter`, `mapRequestWithTenant`, tenant + field + registry setup. |
| `main.go:147-155` | The `AddRouteInitializer` chain. |

### `atlas-data` (module root `services/atlas-data/atlas.com/data`)

| File | Role |
|---|---|
| `map/string_registry.go:11-60` | The complete registry template: identifier struct with `GetID() string`, `sync.Once` accessor, `Init*` walking XML. |
| `document/registry.go:10-79` | `Registry[I string, M Identifier[I]]`. `M` must implement `GetID() string`, so the obstacle index needs a struct, not a bare enum. `Get` returns `(M, error)`; a miss is `errors.New("not found")`. |
| `map/reader.go:39-41` | `Read` — `t := tenant.MustFromContext(ctx)` is already in scope, so `getObjects(t, exml)` is free. |
| `map/reader.go:104` | `m.Reactors = getReactors(exml)` — the insertion point for `m.Objects`. |
| `map/reader.go:323-338` | `getBackgroundTypes` — the numeric-layer iteration idiom (`strconv.Atoi(node.Name)` with `continue` on error). **The correct template.** |
| `map/reactor/rest.go:1-30` | The `map/object/rest.go` template (string id, so no `strconv`). |
| `map/rest.go` | The five relationship hooks; the `reactors` blocks are the template. Line numbers drift — locate with `grep -n 'reactors'`. |
| `map/resource.go:38, :234-259` | Route registration and `handleGetMapReactorsRequest`. |
| `map/reader_test.go:19` | Fixtures are **inline backtick XML strings** (`testXML`), not testdata files. |
| `xml/model.go:12-21, :111-201` | `Node.Name`, `Node.ChildNodes`, `ChildByName`, `GetString`, `GetIntegerWithDefault`. There is **no** `GetInteger`, `GetInt`, or `GetFloat`. |
| `data/workers/mapw.go:32-51` | Ingest site 1 — warn-and-continue. |
| `data/data/processor.go:113-129` | Ingest site 2 — hard error. |

### Gateway

`deploy/shared/routes.conf` — atlas-maps instance blocks at 490-503
(`characters`, `weather`, `jukebox`); `atlas-monsters` block at 505; the
`^/api/worlds(/.*)?$` → `atlas-world` catch-all at **691**. nginx matches regex
locations in file order, so the new blocks go at ~503.

### `atlas-ui`

| File | Role |
|---|---|
| `src/services/api/map-entities.service.ts:34-45, :80-82` | Interface + getter template; `api.getList<T>` unwraps the JSON:API `data` array and errors propagate as rejected promises (no per-call try/catch). |
| `src/lib/hooks/api/useMapEntities.ts:11-53` | Key factory + `useQuery` shape; the definition cache profile. |
| `src/lib/api/client.ts:357-395` | `getList`, `getListDocument`, `getOne`, `post`, `patch`, `delete`. |
| `src/context/tenant-context.tsx:68` | `queryClient.clear()` on tenant change — the reason the tenant is deliberately absent from query keys (D9). |
| `src/lib/breadcrumbs/routes.ts:378-386` | A `nonNavigable: true` grouping node (the `/templates/[id]/character` entry) — the D11 template. `ROUTES.MAP_DETAIL` is at `:682`. |
| `src/App.tsx:113-118, :370-371` | The lazy-import and `<Route>` idioms. |
| `src/components/app-sidebar-items.ts:46-47` | `Maps` then `Reactors` — `Fields` goes between them (FR-10). `src/components/__tests__/app-sidebar.test.tsx` asserts sidebar sync and may need updating. |
| `src/pages/MapDetailPage.tsx` (89 lines) | The page to extend; its `HoverHighlightProvider` / `MapImagePanel` / overlay wiring is untouched (FR-4). |
| `src/components/features/maps/MapDetailTabs.tsx:85-96` | The `TabsList` block; the monster trigger is at 90-92. |
| `src/lib/hooks/useGridRefresh.ts:16-37` | `(queries, options?) → { isRefreshing, onRefresh, lastUpdatedAt }` — FR-40. |
| Test framework | **Vitest** (`npm test` = `vitest run`) + `@testing-library/react`. Component tests live in sibling `__tests__/` dirs. Closest full-page patterns: `src/pages/__tests__/TransportsPage.test.tsx` (273 lines, filtered list) and `TransportRouteDetailPage.test.tsx` (504 lines, detail page). **There is no existing `MapDetailPage` / `MapDetailTabs` test** — the maps feature area is untested today. |

## Dependency order

```
Tasks 1-3   atlas-maps GET /api/fields        ─┐
Tasks 4-9   atlas-data map objects            ─┼─ independent, parallelisable
Task  10    gateway                           ─┘

Task 11     terminology + kind badge          (independent)
Task 12     Map Objects tab                    needs 8 (atlas-data endpoint)
Task 13     fields/worlds services + hooks     needs 2 (GET /api/fields)
Task 14     Live Fields section                needs 13
Task 15     routes, sidebar, breadcrumbs       needs 11 (SurfaceKindBadge), 13
Task 16     Fields locator                     needs 13, 15
Task 17     Field detail shell                 needs 13, 15
Task 18     Characters tab                     needs 17
Task 19     Monsters tab                       needs 14 (live-monsters service), 17
Task 20     Map Objects tab, declared half     needs 12 (useMapObjects), 17
Task 21     docs + flagless verify.sh          needs everything above

Task 22     Map Objects runtime half           BLOCKED on [278] reaching main
```

## Task sizing

No task is deliberately oversized. The largest by file count are Task 12
(6 files, one service) and Task 15 (6 files, one service) — both at the ~6-file
guideline, and both a single coherent deliverable (one tab end to end; one
routing surface end to end) that a reviewer would reject or accept as a unit.
Splitting either would leave a half-registered route or a service method with
no consumer.

Tasks 6 and 7 are deliberately split even though both edit
`atlas-data/map/rest.go`: Task 6 adds the `Objects` field and the reader,
Task 7 adds the four relationship hooks and the storage round-trip test. The
split exists because the round-trip test is the one non-obvious failure mode of
design D5 — objects would appear in the reader and silently vanish through
storage — and it deserves its own review gate.

No codemod is warranted. There is no repeated templated transformation here;
each task is a distinct seam.

## Rebase hazard

`[278]` modifies `services/atlas-maps/atlas.com/maps/map/character/registry.go`
(changing `RemoveCharacterFromAllMaps` to return `[]MapKey`). Task 1 adds
`GetFieldsWithCharacters` to that same file. The two edits do not overlap
textually, but the rebase before Task 22 will touch it — resolve by keeping
both.

## What `plan-lint.sh` caught on the first pass

Six errors, all presentational rather than evidential — no invented symbol and
no unresolved path survived into the plan (F5 was clean on the first run, which
is what the `--symbols` survey was for).

- **F1 x5** — three were `path:lines` citations inside "Patterns to copy" /
  reference blocks, which the linter reads as file paths; rewritten as
  `path` + prose line ranges. Two were `FieldsPage.tsx` / `FieldDetailPage.tsx`
  referenced in Tasks 16/17 without the `new file` marker they carry in Task 15;
  now marked at every mention.
- **F2 x1** — the plan's own *prohibition* against landing a `return null` or a
  `// TODO` matched the stub detector. Reworded to describe the requirement
  positively.
- **F4 x1 (warning)** — Task 18 appeared to span three services because it cited
  `atlas-maps` and `atlas-character` Go files as read-only references. Those are
  established facts, not edit targets; restating them as prose removed the
  false positive. Task 18 touches only `atlas-ui`.

Final run: `plan-lint: clean`, exit 0.

## Verification

Per-task: module-local `go build` / `go vet` / `go test` for Go tasks;
`npm test -- <path>`, `npx tsc --noEmit`, `npm run lint` for TS tasks.

Repo-wide: **flagless** `tools/verify.sh` must exit 0 before the branch is
called done (Task 21, re-run in Task 22). `--quick` and `--no-docker` skip the
bake and `-race` and do not satisfy the gate. Code review is a separate gate
and runs before the PR regardless of a green `verify.sh` — this change crosses
three service boundaries plus the gateway.
