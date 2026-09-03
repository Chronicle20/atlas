# Review — fix-round-2 (task-278)

Commit under review: `6c9acd433` (range `5efddd01b..HEAD`)
Brief: `docs/tasks/task-278-map-environment-object-state/fix-round-2-brief.md`
Report: `docs/tasks/task-278-map-environment-object-state/fix-round-2-report.md`

## Scope

`git show 6c9acd433 --stat` touches exactly:

- `docs/tasks/task-278-map-environment-object-state/fix-round-2-report.md` (new)
- `services/atlas-data/atlas.com/data/map/reader_object_test.go`
- `services/atlas-maps/atlas.com/maps/data/map/object/builder.go` (new)
- `services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` (new)
- `services/atlas-maps/atlas.com/maps/data/map/object/rest.go`

No other files touched. `services/atlas-data/atlas.com/data/map/object/rest.go`
(explicitly ruled out of scope by the user for DOM-04) is untouched — confirmed
by the stat above and by `git diff 5efddd01b..6c9acd433 --stat` returning the
identical file list. Scope matches the brief exactly.

## Fix 1 — DOM-01, `builder.go`

`services/atlas-maps/atlas.com/maps/data/map/object/builder.go`:

```go
type Builder struct{ m Model }
func NewBuilder() *Builder { return &Builder{} }
func (b *Builder) SetName(v string) *Builder  { b.m.name = v; return b }
func (b *Builder) SetState(v uint32) *Builder { b.m.state = v; return b }
func (b *Builder) Build() Model               { return b.m }
```

Matches the sibling `data/map/info/builder.go` shape exactly (`type Builder
struct{ m Model }`, `NewBuilder`, fluent setters, bare `Build() Model`, no
invented validation). `Extract` (`rest.go:32-37`) now constructs via
`NewBuilder().SetName(rm.Name).SetState(rm.State).Build()` instead of the
former struct literal. Field-for-field this is behaviourally identical for
every input including zero values (`SetName("")`/`SetState(0)` produce the
same zero-value `Model{name:"", state:0}` the struct literal did — the builder
only ever writes the two fields the literal wrote, no defaulting logic was
introduced). **PASS.**

## Fix 2 — DOM-04, `Transform`

`rest.go:39-41`:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{Id: m.Name(), Name: m.Name(), State: m.State()}, nil
}
```

`RestModel.Id` is documented (`rest.go:3-4`) as the resource id, which is the
object name — the brief's snippet was used as written, and the doc comment
above `RestModel` corroborates `Id: m.Name()` is correct, not a blind copy.
This is a genuine inverse of `Extract`: `Extract(Transform(m)) == m` and
`Transform(Extract(rm)).Name/State == rm.Name/State` (Id is not round-tripped
through Extract, which is expected — Extract never reads `rm.Id`). **PASS.**

## Fix 3 — EXT-02, httptest coverage

`services/atlas-maps/atlas.com/maps/data/map/object/processor_test.go` (new,
119 lines), shaped after `data/map/monster/processor_drain_test.go` per the
brief.

- `TestGetDefaultStateReturnsDeclaredState` — serves a real JSON:API fixture
  (`{"data":[{"id":"gate","type":"objects","attributes":{"name":"gate","state":1}}], ...}`)
  through `httptest.NewServer`, calls `object.NewProcessor(...).GetDefaultState(1,
  "gate")`, and asserts `state == 1` — a populated field value, not merely a
  non-empty/no-error check. PASS.
- `TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName` — same fixture,
  requests a name (`"barricade"`) the fixture does not declare, asserts
  `errors.Is(err, object.ErrUnknownObject)` (real `errors.Is`, not string
  matching). PASS.
- `TestGetDefaultStateDrainsBeyondOnePage` — serves 260 objects across two
  pages of 250 (`objectDoc(1,250,260,1,250,2)` on page 1,
  `objectDoc(251,260,260,2,250,2)` on page 2, matching the `monster` fixture's
  `meta.page` shape), then asks for `object-260`, which exists only on page 2,
  and asserts both `err == nil` and `state == 260`. I confirmed by reading
  `processor.go:43-54` and `requests.go` that `GetDefaultState` calls
  `inMapProvider` → `requests.DrainProvider[RestModel, Model](...)(url, 250,
  Extract, ...)`, i.e. drain is the only page-following mechanism in the path.
  A single-fetch implementation (one call to page 1 only) would never see
  `object-260` and this test would return `ErrUnknownObject`, failing the
  `t.Fatalf("expected object-260 ... got error: %v", err)` branch — this is a
  genuine RED-without-the-fix assertion, not a fixture arranged so page 1
  already contains everything (page 1 explicitly stops at id 250). PASS.

Ran the package test suite directly:

```
go test ./data/map/object/... -v -count=1
--- PASS: TestGetDefaultStateReturnsDeclaredState (0.02s)
--- PASS: TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName (0.00s)
--- PASS: TestGetDefaultStateDrainsBeyondOnePage (0.01s)
PASS
```

**PASS** overall for fix 3.

## Fix 4 — DOM-20, table-driven `TestGetObjects`

Diffed the rewritten `services/atlas-data/atlas.com/data/map/reader_object_test.go`
against `git show 5efddd01b:...reader_object_test.go` (the pre-commit form).

Pre-commit had three sequential `if` blocks inside `TestGetObjects` asserting,
in order:
1. `os[0]` (`gate`, `l2=1`) → `Name == "gate"`, `State == 1`
2. `os[1]` (`barricade`, `l2="on"`, non-numeric) → `Name == "barricade"`,
   `State == 0` (non-numeric `l2` falls back to 0)
3. `os[2]` (`lever`, absent `l2`) → `Name == "lever"`, `State == 0` (absent
   `l2` falls back to 0)

plus a separate `len(os) != 3` precondition check, and a wholly separate test
function `TestGetObjectsWithoutObjNode` covering the missing/empty `obj`-node
case (`getObjects` on `reactorTestXML`, asserting `len(os) == 0`).

The rewritten version (`reader_object_test.go:32-80`):
- keeps the `len(os) != 3` precondition check outside the table (correctly —
  it gates indexing into `os`, not a parallel scenario),
- converts the three per-object scenarios into a `tests := []struct{name,
  index, wantName, wantState}` table run via `t.Run(tt.name, ...)`, preserving
  every name/state assertion for `gate`/`barricade`/`lever` verbatim,
  including the `l2="1"→1`, non-numeric `l2`→0, and absent `l2`→0 semantics,
  and preserving the explanatory comments (moved into table entry names:
  `"non-numeric l2 falls back to state 0"`, `"absent l2 falls back to state
  0"`),
- leaves `TestGetObjectsWithoutObjNode` (the missing/empty `obj`-node case)
  completely untouched as its own test function — this was the case flagged
  as highest-risk for silent loss, and it survived unmodified.

No scenario or assertion was dropped. Ran the tests directly:

```
go test ./map/... -run TestGetObjects -v -count=1
=== RUN   TestGetObjects
=== RUN   TestGetObjects/declared_l2_is_parsed_as_the_default_state
=== RUN   TestGetObjects/non-numeric_l2_falls_back_to_state_0
=== RUN   TestGetObjects/absent_l2_falls_back_to_state_0
--- PASS: TestGetObjects (0.00s)
=== RUN   TestGetObjectsWithoutObjNode
--- PASS: TestGetObjectsWithoutObjNode (0.00s)
PASS
```

**PASS.**

## Ground truth check

The `gate`/`l2=1` fixture and assertion (`state == 1`, declared default) match
the brief's ground truth: map 103000800's `gate` has visible in-client state
`0`, and the map declares default `l2=1`; a test asserting `gate -> 1` as the
declared default is correct and was not "fixed" by this commit (it was already
correct pre-commit and remains unchanged by the table-driven rewrite).

## Build/test verification (module-local, per brief)

```
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... -count=1
cd services/atlas-data/atlas.com/data && go build ./... && go test ./... -count=1
```

Both ran clean for the packages touched by this commit (`data/map/object` in
atlas-maps, `map` in atlas-data); no `verify.sh`/lint/docker was run, per
instruction.

## Not evaluable

None — the full diff surface (5 files) was read and exercised; no file the
diff calls has a contract this review could not check within the stated
surface.

## Findings

No blocking findings. No non-blocking findings.
