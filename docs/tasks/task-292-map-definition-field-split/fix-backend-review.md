# Review — fix-backend-report (task-292, pre-PR fix round)

- **Scope:** commits `041d8a784` (B1, atlas-maps/field) and `379839b3e` (B2,
  atlas-data/map/object), diffed against base `9f6679907`, restricted to `*.go`
  files. TypeScript changes from the concurrent frontend implementer are out
  of scope and were excluded from all diffs pulled.
- **Verdict:** APPROVED

## 1. Do the two blocking findings actually close?

### B1 — `atlas-maps/field` (DOM-04, DOM-05, SUB-01)

- `field/processor.go` (new) — `Processor` interface + `ProcessorImpl` +
  `NewProcessor(l, ctx)`, same shape as `character/processor.go`.
  `GetFields(t, worldId, channelId, mapId)` holds the filter loop and the
  sort — the exact logic that used to live inline in the handler — file
  `services/atlas-maps/atlas.com/maps/field/processor.go:47-78`. PASS.
- `field/rest.go:39-58` — `Transform(character.FieldOccupancy) (RestModel,
  error)` and `TransformSlice`, matching the sibling pattern at
  `services/atlas-maps/atlas.com/maps/character/location/rest.go:55`. PASS.
- `field/resource.go:91-99` — the handler now does
  `NewProcessor(...).GetFields(...)` → `TransformSlice(occ)` with the error
  checked and routed through `server.WriteErrorResponse(d.Logger())(w)(err)`
  before falling through to `paginate.Slice`/`MarshalPaginatedResponse`. No
  filter/sort/RestModel-construction logic remains in the handler. PASS
  (closes DOM-04, DOM-05, SUB-01, and the DOM-13 non-blocking note).
- `character.FieldOccupancy` is reused directly as the domain `Model`, per
  the brief's escape hatch — its fields are already exported and
  `field.Model` (constants lib) already exposes the getters `resource.go`
  was calling before. No parallel type was invented, and the package does
  not reach into `character` internals beyond the pre-existing
  `character.NewProcessor(...).GetFieldsWithCharacters(t)` call, which was
  already legitimate (DOM-14 PASS in the original audit and unaffected here).

### B2 — `atlas-data/map/object` (DOM-04)

- `map/object/model.go` (new) — immutable `Model` (unexported fields +
  getters) + `Builder`/`NewBuilder()`, `services/atlas-data/atlas.com/data/map/object/model.go:12-127`.
  `Id()` computes `"{kind}:{name}"` on demand (`model.go:28-30`) — matches
  the old inline `id := kind + ":" + name` in `reader.go`. PASS.
- `map/object/rest.go:32-63` — `Transform(Model) (RestModel, error)` and
  `TransformSlice`. PASS.
- `map/reader.go:354-397` — `getObjects` now builds `[]object.Model` via
  `object.NewBuilder()...Build()`; the production call site
  (`reader.go:104-109`) transforms via `object.TransformSlice(getObjects(t,
  exml))` with the error checked (`if err != nil { return
  model.ErrorProvider[RestModel](err) }`) before assigning `m.Objects`.
  PASS. `err` is a plain re-declared-alongside-a-new-var `:=` off the
  outer-scope `i, err := exml.ChildByName("info")` at `reader.go:57` — no
  shadowing bug, confirmed by reading the enclosing function body.

Both original blocking findings are closed with a genuine processor +
Transform layering, not a relabeling.

## 2. HTTP/JSON output byte-identical?

- **Field sort order** — `field/processor.go:64-75`: world, then channel,
  then map, then `Instance().String()` — identical comparator to the deleted
  `resource.go:114-126` code (confirmed via `git diff`, the old block was
  removed verbatim and re-emitted with `models[i].Field.X()` in place of
  `models[i].X` — same field-by-field logic, same tie-break order).
- **Pagination** — `field/resource.go:101,103` still calls
  `paginate.Slice(models, page)` and
  `server.MarshalPaginatedResponse[[]RestModel](...)` unchanged; neither
  line moved or changed signature.
- **Object dedup** — `reader.go:373-378`: `seen[id]` check with `id :=
  kind + ":" + name`, keep-first (the `continue` on duplicate), unchanged
  logic, only the append target changed from a `RestModel` literal to
  `object.NewBuilder()...Build()`.
- **Object sort** — `reader.go:396-401`: `results[i].Kind() !=
  results[j].Kind()` then `Kind() < Kind()`, else `Name() < Name()` —
  same two-level comparator as before, only method calls in place of field
  access.
- `map/rest.go`, `map/processor.go`, `map/resource.go` are untouched by
  either commit (confirmed by `git diff --stat` against those three paths
  returning nothing) — the brief's explicit HTTP-surface-freeze constraint
  held.
- Existing `map/rest_test.go`/`map/resource_test.go`/`field/resource_test.go`
  were not modified by either commit (`git diff 9f6679907 379839b3e --
  services/atlas-maps/.../field/resource_test.go` is empty) and pass
  unchanged — see verification runs below.

## 3. Tenant isolation untouched

- `character/registry.go:99-131` (`GetFieldsWithCharacters` and its
  `mk.Tenant != t` filter) is unmodified by either of the two fix commits —
  it was already present in the base commit `9f6679907`, and the diff
  against that base shows no further change to `registry.go` (verified with
  `git diff 9613e7259..9f6679907` for the base-lift, and `git diff
  9f6679907 379839b3e -- .../character/registry.go` returns nothing).
- `TestGetFieldsTenantIsolation` runs and passes against the fixed code
  (see test output below).

## 4. Scope discipline

- `git diff 9f6679907 379839b3e --stat -- '*.go'` shows exactly the 9 files
  named in both the brief and the implementer's report — no sibling
  `portal`/`reactor`/`npc`/`monster` package under either service was
  touched, in either commit.
- `atlas-data/map` package still stores `[]object.RestModel` as its
  document field (`map/rest.go` untouched) and `GetObjects` /
  `handleGetMapObjectsRequest` still serve `RestModel` — no wider domain
  layer was introduced across the `map` package, per the brief's explicit
  fence.

## Verification (module-local, as authorized by the brief)

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test -count=1 ./field/...
ok  atlas-maps/field  0.027s (build clean, fresh run not cached)

go test -count=1 -v ./field/... -run 'TestGetFieldsTenantIsolation|TestProcessorGetFieldsFilters|TestProcessorGetFieldsSortOrder'
--- PASS: TestProcessorGetFieldsFilters (all 6 subtests)
--- PASS: TestProcessorGetFieldsSortOrder
--- PASS: TestGetFieldsTenantIsolation
PASS

cd services/atlas-data/atlas.com/data && go build ./... && go test -count=1 ./map/...
ok  atlas-data/map         0.098s
ok  atlas-data/map/object  0.002s
```

No `tools/verify.sh` run, per instructions.

## Non-blocking notes

- The implementer's report says the `field/rest.go` `Transform` pattern
  "matches the pattern in `character/location/rest.go`" without a full
  path; the file actually lives at
  `services/atlas-maps/atlas.com/maps/character/location/rest.go` (top-level
  `character/location`, not under `map/`). Confirmed the referenced pattern
  is real and matches (`Transform(m Model) (RestModel, error)` at line 55);
  this is a minor report-wording imprecision, not a code defect.
- Both commits leave the original audit's DOM-20 (table-driven test shape)
  non-blocking note unaddressed for the new test files (`processor_test.go`
  uses `t.Run` with ad hoc setup, not a `tests := []struct{...}` table; same
  for `object/rest_test.go`). That finding was explicitly non-blocking in
  the original audit and out of scope for this fix round's brief (B1/B2
  only), so it is noted here for completeness, not as a new blocking item.

## Not evaluable

None — both fixes are fully contained within the reviewed diff and the
sibling files their correctness depends on (`character/registry.go`,
`character/location/rest.go`, `map/rest.go`/`processor.go`/`resource.go`)
were read to confirm the contract, not merely assumed.
