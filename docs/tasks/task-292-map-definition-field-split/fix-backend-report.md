# Fix round — backend blocking findings (DOM-04, DOM-05, SUB-01)

## Summary

Both blocking findings from `backend-guidelines-reviewer` fixed, each in its
own commit, per the brief.

## B1 — atlas-maps/field Transform layering (DOM-04, DOM-05, SUB-01)

Commit `041d8a784` — fix(atlas-maps): add field.Processor Transform layering.

- `field/processor.go` (new): `Processor` interface + `ProcessorImpl` +
  `NewProcessor(l, ctx)`, matching the shape of
  `character/processor.go`. `GetFields(t, worldId, channelId, mapId)` calls
  `character.NewProcessor(...).GetFieldsWithCharacters(t)`, applies the
  optional world/channel/map filters, and sorts by world, channel, map, then
  instance-id string — identical logic to the old inline code in
  `resource.go`, just relocated.
- Reused `character.FieldOccupancy` directly as the domain `Model` (per the
  brief's escape hatch): its fields (`Field field.Model`, `CharacterCount
  uint32`) are already exported, and `field.Model` (from
  `atlas-constants/field`) already exposes `WorldId()/ChannelId()/MapId()/
  Id()/Instance()` getters — the same accessors `resource.go` was already
  calling before this change. No reach into `character` package internals was
  needed, so I did not introduce a parallel `field.Model` type.
- `field/rest.go`: added `Transform(character.FieldOccupancy) (RestModel,
  error)` and `TransformSlice([]character.FieldOccupancy) ([]RestModel,
  error)`, matching the pattern in `character/location/rest.go`.
- `field/resource.go`: `handleGetFields` now does parse → `NewProcessor(...).
  GetFields(...)` → `TransformSlice(...)` (with the error checked and routed
  through `server.WriteErrorResponse`) → `paginate.Slice` →
  `MarshalPaginatedResponse`. No filter/sort logic remains in the handler.
- `field/processor_test.go` (new): `TestProcessorGetFieldsFilters` (no
  filter/world/channel/map/all-three/no-match) and
  `TestProcessorGetFieldsSortOrder` (shuffled map insertion, asserts sorted
  output), calling `Processor.GetFields` directly, independent of HTTP.
- `field/resource_test.go`: unchanged, no signature forced a change (the
  handler's HTTP contract — query params, JSON:API shape, status codes,
  pagination — is identical before/after).

### Tests

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...
```
All packages `ok` (or `no test files`), including `atlas-maps/field`
(TestGetFieldsTenantIsolation, TestGetFieldsDrainedFieldExcluded,
TestGetFieldsFilters, TestGetFieldsMalformedFilter,
TestGetFieldsPaginationDeterminism, TestGetFieldsAttributes, plus the two new
processor tests) and `atlas-maps/character` unaffected.

## B2 — atlas-data/map/object Transform (DOM-04)

Commit `379839b3e` — fix(atlas-data): add object.Transform in map/object
sub-package.

- `map/object/model.go` (new): immutable `Model` (unexported fields:
  `kind, name, objectSource, l0, l1, l2, x, y, z, layer`) with getters, plus
  `Builder`/`NewBuilder()` per the `skill/effect/model.go` +
  `skill/effect/builder.go` convention (combined into one file here since the
  sub-package is small and the brief named only `model.go`). `Model.Id()`
  computes `"{kind}:{name}"` on demand rather than storing it, so there is no
  risk of the stored id ever diverging from Kind()/Name() — the exact
  computation `getObjects` used to do inline.
- `map/object/rest.go`: added `Transform(Model) (RestModel, error)` and
  `TransformSlice([]Model) ([]RestModel, error)`.
- `map/reader.go`:
  - `getObjects` now builds and returns `[]object.Model` (via
    `object.NewBuilder()...Build()`) instead of `[]object.RestModel`. Dedup
    by `{kind}:{name}` (keep-first) is unchanged; the closing sort now calls
    `Kind()`/`Name()` getters instead of struct-literal field access but is
    the identical kind-then-name comparator.
  - The single production call site (`m.Objects = getObjects(t, exml)`) is
    now `objs, err := object.TransformSlice(getObjects(t, exml)); if err !=
    nil { return model.ErrorProvider[RestModel](err) }; m.Objects = objs`.
    `err` is reused from the enclosing scope (already declared a few lines up
    via `i, err := exml.ChildByName("info")`), so this is a plain `:=`
    re-declaration of `objs` only — no shadowing bug.
  - `map/processor.go` (`GetObjects`), `map/resource.go` (objects endpoint),
    and `map/rest.go` (`Objects []object.RestModel` field, `Storage`) are
    untouched, per the brief — the HTTP surface is unchanged.
- `map/reader_test.go`: `TestGetObjectsOnlyNamedEntries`,
  `TestGetObjectsResolvesObstacle`, `TestGetObjectsDuplicateIdKeepsFirst` now
  call `.Id()/.Kind()/.Name()/.ObjectSource()/.L0()/.L1()/.L2()/.X()/.Y()/
  .Z()/.Layer()` getters instead of struct fields (their assertions are
  unchanged in substance — same values checked, same failure messages
  reworded only where they referenced `.Id`/`.X` as a field).
  `TestGetObjectsEmptyLayers` only checked `len(objs)`/`objs == nil`, no
  change needed.
- `map/object/rest_test.go` (new): `TestTransformRoundTripsEveryField` (every
  field set via the builder, every field asserted on the resulting
  `RestModel`, including the derived `Id`), `TestTransformSlice`, and
  `TestTransformSliceEmpty` (nil input yields a non-nil empty slice, matching
  the existing `make([]RestModel, 0, ...)` allocation convention used
  throughout the codebase).

### Tests

```
cd services/atlas-data/atlas.com/data && go build ./... && go test ./...
```
All packages `ok` (or `no test files`), including `atlas-data/map` (existing
`reader_test.go`, `rest_test.go`, `resource_test.go` — objects-JSON
byte-identity confirmed by these passing unchanged) and the new
`atlas-data/map/object`.

## Files changed

- `services/atlas-maps/atlas.com/maps/field/processor.go` — new
- `services/atlas-maps/atlas.com/maps/field/processor_test.go` — new
- `services/atlas-maps/atlas.com/maps/field/rest.go` — added Transform/TransformSlice
- `services/atlas-maps/atlas.com/maps/field/resource.go` — handler shrunk to parse → processor → transform → marshal
- `services/atlas-data/atlas.com/data/map/object/model.go` — new
- `services/atlas-data/atlas.com/data/map/object/rest.go` — added Transform/TransformSlice
- `services/atlas-data/atlas.com/data/map/object/rest_test.go` — new
- `services/atlas-data/atlas.com/data/map/reader.go` — getObjects returns []object.Model; call site transforms
- `services/atlas-data/atlas.com/data/map/reader_test.go` — call sites updated to getter methods

## Self-review

- Re-read both diffs (`git diff` on the reader.go hunk especially, since it's
  the riskiest one for byte-identity) — confirmed the id derivation, dedup
  key, and sort comparator are semantically identical to the pre-fix code.
- Confirmed neither commit touched any file under `services/atlas-ui`
  (concurrent frontend work) and staged only the named files, never `-A`/`.`.
- Did not touch `portal`/`reactor`/`npc`/`monster` siblings in either
  service, per the brief's explicit scope fence.
- Did not introduce a domain layer across the whole atlas-data `map`
  package — `map/rest.go`, `map/processor.go`, `map/resource.go` are
  untouched; the fix is confined to the `object` sub-package as instructed.
- Both module-local `go build ./... && go test ./...` runs are clean with no
  stray output.

## Issues or concerns

None. Both findings are narrowly scoped and the module-local test suites
(including the pre-existing tenant-isolation and pagination-determinism
tests explicitly named in the brief) pass unchanged.
