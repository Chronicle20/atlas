# Review: Task 3 — `atlas-maps` projects `state` onto REST surface

**Range reviewed:** `c99525e21..a56c5b15d` (single commit `a56c5b15d`)
**Files touched:** `services/atlas-maps/atlas.com/maps/character/location/rest.go`,
`services/atlas-maps/atlas.com/maps/character/location/resource_test.go`

## Scope confirmation

Diff stat matches the brief exactly:

```
.../maps/character/location/resource_test.go | 68 ++++++++++++++++++++++
.../maps/character/location/rest.go           | 13 +++--
2 files changed, 76 insertions(+), 5 deletions(-)
```

`resource.go` is untouched (`git diff --stat` for that path against the range is empty),
matching the brief's explicit "read-only" scoping. No new HTTP harness was added — the
appended tests are `TestTransform_CarriesState` and `TestRestModel_StateJSONKey`, both
calling `Transform`/`json.Marshal` directly, no router/handler invocation. This matches
the brief's explicit instruction not to build one.

## Findings

### PASS — `RestModel.State` field and JSON tag

`rest.go:142-147` (post-diff):
```go
type RestModel struct {
	Id        uint32                       `json:"-"`
	WorldId   world.Id                     `json:"worldId"`
	ChannelId channel.Id                   `json:"channelId"`
	MapId     _map.Id                      `json:"mapId"`
	Instance  uuid.UUID                    `json:"instance"`
	State     characterconst.PresenceState `json:"state"`
}
```
JSON key is exactly `state`, lowercase — matches the interface contract Task 6 depends on.
Confirmed via `TestRestModel_StateJSONKey`, which marshals a `RestModel` and asserts the
decoded map has key `"state"` with value `"IN_FIELD"`, plus asserts `worldId`, `channelId`,
`mapId`, `instance` are still present (additive-change guard).

### PASS — `Transform` carries state through

`rest.go:169`: `State: m.State()` added to the `Transform` function's return literal. This
is the only place `RestModel` is constructed from a domain `Model` on this path (`TransformSlice`
delegates to `Transform`), so all callers get the new field.

### PASS — wire-value correctness against the global constraint

Verified independently against `libs/atlas-constants/character/presence.go`:
```go
PresenceStateOffline    PresenceState = "OFFLINE"
PresenceStateInField    PresenceState = "IN_FIELD"
PresenceStateInCashShop PresenceState = "IN_CASH_SHOP"
```
`PresenceStateOffline` is the zero value of the underlying `string` type (declared first in
the `const` block with no `iota`, and `presence_test.go` pins `ParsePresenceState("")` ==
`PresenceStateOffline`). `RestModel.State` has no `omitempty` and no custom marshaler, so an
`OFFLINE` value serializes as `"state":"OFFLINE"` rather than being dropped — correct per the
"absent state resolves to OFFLINE" constraint, since the field is never actually absent from
the JSON; it just carries the zero-value string. This is consistent with Task 2's `location.Model`
having `state characterconst.PresenceState` as a plain field (`model.go:21`), so a `Model` built
without an explicit state carries `PresenceStateOffline` through `Transform` unchanged.

### PASS — `location.Model.State()` contract matches what Task 3 consumes

`character/location/model.go:29`: `func (m Model) State() characterconst.PresenceState { return m.state }`
and `builder.go:61`: `func (b *Builder) SetState(v characterconst.PresenceState) *Builder`. Both
match the brief's described Task 2 interface and what `rest.go`/`resource_test.go` call.

### PASS — Builder pattern used in tests, no test-only constructor file

`TestTransform_CarriesState` builds its fixture via `NewBuilder(1234).SetWorldId(...).
SetChannelId(...).SetMapId(...).SetInstance(...).SetState(...).Build()` — the existing
project Builder, not a new helper. No `*_testhelpers.go` file was added; diff stat confirms
only `rest.go` and `resource_test.go` changed.

### PASS — build and module test evidence

`go build ./...` from `services/atlas-maps/atlas.com/maps` exits clean (independently
re-run for this review, no test suite re-run). Report's captured `go test` output for the
two new tests and the full-module suite (`ok atlas-maps/character/location 0.017s`, etc.)
is consistent with the diff — not independently re-executed per review instructions.

### Non-blocking — RED evidence gap acknowledged by implementer

The report explicitly flags that it did not capture a separate RED (pre-change failing
test) run, applying the test file and the `rest.go` change together and reasoning the
failure mode (`rm.State undefined`, `unknown field State`) would be trivially reproducible.
This is disclosed honestly in the report's "TDD Evidence" section rather than fabricated,
and the two tests as written are structurally guaranteed to fail without the `rest.go`
change (compile error, not merely an assertion failure) — a strong form of "the test fails
without the change" even without a captured transcript. Not a blocking defect, but noting
it as the one place the brief's Step 2 ("run test to verify it fails") wasn't executed
as specified.

## Not evaluable

- Task 6's actual client decode of the `state` key (out of this task's diff; brief says
  Task 6 depends on this contract, but Task 6's code is not part of this range).
- Whether `atlas-channel`, `atlas-character`, `atlas-login`'s existing decoders genuinely
  ignore unknown JSON fields (asserted by the brief as the compatibility rationale, but
  those consumers' structs are outside this task's file list and this diff).

## Verdict rationale

All in-scope requirements are met: the field is added with the exact wire key and value
set required, `Transform` carries it, the brief's explicit "no resource.go change, no HTTP
harness" scoping was honored (no over-building), the Builder pattern was used correctly,
and the zero-value/OFFLINE behavior is structurally correct. The only gap (missing captured
RED output) is process-level, honestly disclosed, and does not affect correctness of the
shipped code.
