# Review: Task 6 — atlas-npc-shops accessor, positional fields, builder setters

**Commit reviewed:** `64344f5d81c6600c20e809bdfc56dc0d8ce9ce85` — `fix(atlas-npc-shops): return spawnPoint and decode x/y/stance`
(reviewed in isolation via `git show 64344f5`, not a range, per instructions — tasks 7/8 landed concurrently on the same branch)

**Brief:** `.superpowers/sdd/plan/task-6-brief.md`
**Report:** `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git show --stat 64344f5`:

```
services/atlas-npc-shops/atlas.com/npc/character/builder.go   |  3 ++
services/atlas-npc-shops/atlas.com/npc/character/model.go     |  4 +-
services/atlas-npc-shops/atlas.com/npc/character/rest.go      |  3 ++
services/atlas-npc-shops/atlas.com/npc/character/rest_test.go | 50 ++++++++++++++++++++++
4 files changed, 58 insertions(+), 2 deletions(-)
```

Exactly the four files the brief's "Files" section names. No scope creep.

## Findings

### PASS — `SpawnPoint()` un-stubbed correctly

`character/model.go:208-210`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

Replaces the stub (`func (m Model) SpawnPoint() byte { return 0 }`). Return type is `uint32`, matching `Model.spawnPoint uint32` (`builder.go:73`), `RestModel.SpawnPoint uint32` (`rest.go:38`), and the brief's interface spec. Per the task's resolved-ambiguity note, this widening (not narrowing to the stub's stale `byte`) is correct and not a defect. Confirmed no caller elsewhere in the module depended on the old `byte` signature: `grep -rn "SpawnPoint()" .` (excluding tests) returns only the definition itself — no cross-caller breakage from the signature change.

### PASS — `Extract` now decodes `x`/`y`/`stance`

`character/rest.go:163-165`:

```go
x:                  rm.X,
y:                  rm.Y,
stance:             rm.Stance,
```

Added after the existing `meso: rm.Meso,` line, exactly as specified. The pre-existing `spawnPoint: rm.SpawnPoint,` line (`rest.go:160`) is untouched, as instructed. `RestModel.X`/`Y` are `int16`, `Stance` is `byte`, matching `Model.x`/`y`/`stance` and `Model.X()`/`Y()`/`Stance()` accessors (`model.go:224-234`) — no type mismatch, no cast needed.

### PASS — Three builder setters added

`character/builder.go:116-118`:

```go
func (b *Builder) SetX(v int16) *Builder                   { b.x = v; return b }
func (b *Builder) SetY(v int16) *Builder                   { b.y = v; return b }
func (b *Builder) SetStance(v byte) *Builder                { b.stance = v; return b }
```

Placed after `SetGm`, matches the one-line setter form used throughout the file. Types match the `Builder` struct fields (`x int16`, `y int16`, `stance byte`, `builder.go:75-77`) and `Build()` already copies them into `Model` (`builder.go:151-153`, pre-existing, untouched). `gofmt -l character/` reports no files — formatting is clean. Per context.md decision 1 (cited in the task instructions), these setters having no production caller yet is expected, not a defect.

### PASS — Tests are honest and exercise the new behavior

`rest_test.go` additions:
- Four assertions added to `TestTransformRoundTrip` (`SpawnPoint()`, `X()`, `Y()`, `Stance()` each checked against distinct non-zero fixture values 11/10/12/14) — these are not redundant with the existing `reflect.DeepEqual` round-trip check, which (per the report) is blind to a field that's zero on both sides when dropped.
- New `TestTransform_PositionalFieldsFromBuilder` builds a `Model` purely through `NewBuilder()...SetX/SetY/SetStance().Build()`, transforms it, and asserts the outbound `RestModel` fields — this is the only path (per the brief) that can originate non-zero positional values through the sanctioned Builder, and it would fail to compile before this commit (`SetX undefined`).

Ran both tests directly against the worktree:

```
=== RUN   TestTransformRoundTrip
--- PASS: TestTransformRoundTrip (0.00s)
=== RUN   TestTransform_PositionalFieldsFromBuilder
--- PASS: TestTransform_PositionalFieldsFromBuilder (0.00s)
PASS
ok  	atlas-npc/character	0.011s
```

Full module build/test also green:

```
go build ./... && go test ./character/...
ok  	atlas-npc/character	0.011s
ok  	atlas-npc/character/skill	0.008s
```

(Only ran the touched package's tests plus a full `go build ./...`, per instructions not to run `tools/verify.sh`.)

## Not evaluable

None — the full scope of this unit (accessor, decode wiring, builder setters, tests) is inside a single small, self-contained module and was fully inspected.

## Verdict rationale

All requirements from the brief are implemented verbatim: the accessor is un-stubbed with the correct type, `Extract` no longer drops `x`/`y`/`stance`, the three builder setters exist with correct signatures and are gofmt-clean, and the test additions genuinely pin the new behavior (would fail/not-compile without the change, per the report's documented RED run, which is plausible and consistent with the code before this commit). No cross-service seam: this commit only touches internal `Model`/`RestModel`/`Builder` plumbing within `atlas-npc-shops`, with no producer/consumer boundary crossed.

No defects found.
