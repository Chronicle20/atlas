# Review: Task 1 — atlas-channel spawnPoint plumbing (accessor, Extract, wire cast)

Commit range reviewed: `8a86867a7..c4b1fd3be`
Single commit: `c4b1fd3be` — `fix(atlas-channel): decode and return spawnPoint, narrow at the wire`

## Scope

`git diff --stat 8a86867a7..c4b1fd3be` shows exactly the five files named in
the brief:

- `services/atlas-channel/atlas.com/channel/character/model.go` (+2/-2)
- `services/atlas-channel/atlas.com/channel/character/rest.go` (+1/-0)
- `services/atlas-channel/atlas.com/channel/character/rest_test.go` (+7/-0)
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` (+1/-1)
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go` (+31/-0)

No other files changed. Matches the brief's file list and the implementer's
report exactly.

## Findings

### PASS — `Model.SpawnPoint()` returns `uint32` / `m.spawnPoint`

`character/model.go:240-242`:
```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```
No `byte` return type remains anywhere in the diff. Confirmed via
`grep -rn '\.SpawnPoint()'` across `services/atlas-channel/atlas.com/channel`
— the only two occurrences are the definition (model.go) and the one call
site (character_data.go:47), both `uint32`-typed on the accessor side.

### PASS — narrowing to `byte` happens only at the wire call site

`socket/writer/character_data.go:47`:
```go
SpawnPoint: byte(c.SpawnPoint()),
```
This is the only wire call site touched (task scope is atlas-channel only;
the second call site named in the plan's global constraints belongs to a
different task/service, not this diff). No other `byte(...)` casts of
`SpawnPoint()` were introduced.

### PASS — `services/atlas-character/**` and `libs/atlas-packet/**` absent from diff

`git diff --stat 8a86867a7..c4b1fd3be -- services/atlas-character libs/atlas-packet`
returns empty.

### PASS — `Extract` decodes `spawnPoint`, field order mirrors `Transform`

`character/rest.go:152` inserts `spawnPoint: m.SpawnPoint,` between `sp:`
(line 151) and `gm:` (line 153), matching the position of `SpawnPoint:` in
`Transform` at line 116 (`Sp: m.sp,` then `SpawnPoint: m.spawnPoint,` then
`Gm: m.gm,`). Field order is consistent between the two functions.

### PASS — test fixtures use non-zero `spawnPoint`

- `rest_test.go`: `RestModel{..., SpawnPoint: 11, ...}` (line 43), asserted
  via `m.SpawnPoint() != 11` at line 59 — non-zero, and compared against the
  literal input value, not round-trip-derived.
- `character_data_test.go`: new `TestBuildCharacterData_SpawnPoint` table
  test has cases `{set: 7, want: 7}` and `{set: 256, want: 0}`. The first
  case is non-zero and load-bearing; the second (`256 -> 0`) is explicitly
  documented in the test's doc comment and in the implementer's report as
  "passing vacuously against the stub" and included to pin the truncation
  semantics of the one-byte wire field, not as fix-proof. This is disclosed
  honestly rather than presented as coverage.

### PASS — assertions compare against literal input, not `Extract∘Transform` idempotence

`rest_test.go:56-60` explicitly calls out in its comment why the pre-existing
`DeepEqual` round-trip is insufficient (a dropped field is invisible to it)
and adds a direct comparison of `m.SpawnPoint()` against the literal `11`
from the `RestModel` fixture, before the `Transform`/second-`Extract`
round-trip runs. This satisfies the brief's explicit prohibition on
idempotence-only proof.

### PASS — no `UpdateSpawnPoint` producer added; no dedup of model copies; Rank/RankMove/JobRank/JobRankMove untouched

`grep -rn "UpdateSpawnPoint" services/atlas-channel/atlas.com/channel` is
empty. The diff touches only the one `Model` copy in atlas-channel (no
cross-service dedup attempted, consistent with the constraint that model
copies not be deduplicated). No Rank/RankMove/JobRank/JobRankMove code
appears in the diff.

### PASS — TDD evidence is credible

The report's RED-phase failures (`SpawnPoint() = 0, want 11` and
`Stats.SpawnPoint = 0, want 7`) are consistent with what the stubbed
`SpawnPoint() byte { return 0 }` and unset `Extract` field would produce.
The GREEN-phase full-suite run (`2677 passed in 321 packages`) is reported
per the task's instruction not to re-run the suite in review; taken as
implementer-supplied evidence per plan instructions.

### PASS — test-setup convention (Builder pattern)

Both new/amended test call sites use `character.NewBuilder()...MustBuild()`
(`character_data_test.go:86-90`) and a `RestModel{...}` struct literal
(`rest_test.go:19-46`, the sanctioned `Extract` input shape per the plan's
constraints, not a test-only constructor). No new `*_testhelpers.go`-style
helper introduced.

### PASS — observable wire output unchanged

`byte(c.SpawnPoint())` where `c.SpawnPoint()` now returns `m.spawnPoint`
(`uint32`). Since the persisted `spawnPoint` column is stated to be always 0
today (per the global constraint), `byte(0)` is byte-identical to the prior
hardcoded `byte(0)` stub output for all current production data. The change
only alters behavior once `spawnPoint` becomes non-zero (a downstream
concern of later tasks that would need to actually populate the field).

## Not evaluable

- Full module `go build ./... && go test ./...` was not independently
  re-run in this review, per the task instruction ("Do not re-run the
  module test suite — the implementer's report carries that evidence").
  Build/test success is taken on the implementer's report.
- Whether the second wire call site referenced by the plan's global
  constraints (outside atlas-channel) is handled correctly is out of scope
  for this task/diff and is not evaluated here.

## Verdict

All in-scope requirements from the brief and the plan's global constraints
are satisfied by this diff, with correct evidence at each site. No blocking
or non-blocking defects found.
