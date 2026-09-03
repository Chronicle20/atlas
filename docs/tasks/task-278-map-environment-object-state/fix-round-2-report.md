# Fix round 2 report — post-merge backend-guidelines findings

## Summary

Implemented all four in-scope fixes from `fix-round-2-brief.md`. Fix 5
(DOM-04 against `services/atlas-data/atlas.com/data/map/object/rest.go`) was
left untouched per explicit user ruling.

## Fix 1 — DOM-01, builder for `data/map/object`

Added `services/atlas-maps/atlas.com/maps/data/map/object/builder.go`,
mirroring `data/map/info/builder.go`:

```go
type Builder struct{ m Model }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetName(v string) *Builder  { b.m.name = v; return b }
func (b *Builder) SetState(v uint32) *Builder { b.m.state = v; return b }
func (b *Builder) Build() Model               { return b.m }
```

`Build()` returns a bare `Model` (no invariant to validate), same as `info`.

`Extract` in `rest.go` now builds through the builder instead of a struct
literal:

```go
func Extract(rm RestModel) (Model, error) {
	return NewBuilder().
		SetName(rm.Name).
		SetState(rm.State).
		Build(), nil
}
```

## Fix 2 — DOM-04, `Transform` in `data/map/object/rest.go`

Added the genuine inverse of `Extract`:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{Id: m.Name(), Name: m.Name(), State: m.State()}, nil
}
```

Confirmed against the current `RestModel` (read before writing): `Id` is
`json:"-"` and is the resource id, which the doc comment on `RestModel`
already states is the object name — so `Id: m.Name()` and `Name: m.Name()`
both hold.

## Fix 3 — EXT-02, httptest coverage for the atlas-data client

Added `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go`,
shaped after `data/map/monster/processor_drain_test.go`
(`httptest.NewServer` + `t.Setenv("DATA_SERVICE_URL", ...)` +
`tenant.Create`/`tenant.WithContext` + `test.NewNullLogger()`), covering:

- `TestGetDefaultStateReturnsDeclaredState` — a fixture with one declared
  object (`gate`, state 1) asserts `GetDefaultState` returns the declared
  state.
- `TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName` — the same
  fixture, requesting an undeclared name (`barricade`), asserts
  `errors.Is(err, object.ErrUnknownObject)`.
- `TestGetDefaultStateDrainsBeyondOnePage` — 260 objects served across two
  pages of 250 (`meta.page` shape copied from the monster fixture); the test
  requests `object-260`, which lives only on page 2, and fails if
  `inMapProvider` stops after page 1.

All three ran GREEN on first pass since `inMapProvider` already used
`requests.DrainProvider` (pre-existing production code, unmodified). To
confirm the drain assertion is load-bearing rather than vacuous, I reasoned
from the `monster` drain test's identical shape (page-size 250 objects served
one page at a time, response driven entirely by `page[number]`) — a
single-fetch `GetDefaultState` implementation would receive only objects 1-250
from the fixture server and return `ErrUnknownObject` for `object-260`,
failing the test with the message `"expected object-260 (page 2) to be found
via drain, got error: ..."`. I did not hand-roll a broken implementation to
force a literal RED run since fix 3 requires no production code change — the
production code (`inMapProvider`, unmodified) is what the test exercises.

## Fix 4 — DOM-20, table-driven `TestGetObjects`

Rewrote `services/atlas-data/atlas.com/data/map/reader_object_test.go`'s
`TestGetObjects` from three sequential `if` blocks to
`tests := []struct{...}` + `t.Run(tt.name, ...)`, following the local idiom
in `map/reader_test.go`'s `TestReactorDelayMillis`. Preserved every scenario:
declared `l2` parses as default state (`gate` -> 1), non-numeric `l2` falls
back to 0 (`barricade`), and absent `l2` falls back to 0 (`lever`). The
unrelated `TestGetObjectsWithoutObjNode` test was untouched (it was already a
single scenario, not a 3-case table).

Per the brief's ground-truth note, `gate -> state 1` (declared default) was
left as-is; I did not re-read the WZ tree.

## Verification (module-local only)

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... -count=1
```
Result: all packages `ok`, including `atlas-maps/data/map/object` (0.028s,
3 new tests passing) and no regressions elsewhere.

```
cd services/atlas-data/atlas.com/data && go build ./... && go test ./... -count=1
```
Result: all packages `ok`, including `atlas-data/map` (0.162s), no
regressions.

`gofmt -l` on all touched files: no output (all clean).

## Files changed

- `services/atlas-maps/atlas.com/maps/data/map/object/builder.go` — NEW
- `services/atlas-maps/atlas.com/maps/data/map/object/rest.go` — added
  `Transform`, rewrote `Extract` to build via the builder
- `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` —
  NEW
- `services/atlas-data/atlas.com/data/map/reader_object_test.go` — rewrote
  `TestGetObjects` table-driven

## Self-review

- Diff scoped to exactly the four fixes; no changes to monster/npc/portal/
  reactor packages, atlas-channel, or the out-of-scope `atlas-data`
  `map/object/rest.go`.
- No production behaviour changed except the internal `Extract` rewrite
  (still returns the same `Model` values, now constructed via the builder).
- `Transform` genuinely round-trips: `Extract(Transform(m))` and
  `Transform(Extract(rm))` both preserve `name`/`state`/`id` (id derives from
  name in both directions).
- New test file uses the `object_test` external package (black-box), matching
  the monster drain test's convention.
- No `*_testhelpers.go` files added; test setup uses the package's own
  `NewProcessor`/`tenant.Create` constructors, consistent with Builder-pattern
  guidance (no test-only constructors needed here — `Extract`/`Builder` are
  production code already).

## Concerns

None. All four fixes are additive or test-only as scoped by the brief.
