# Review: Task 5 — atlas-pets spawnPoint plumbing

Commit range: c027e20d3..2dc3d66a5
Brief: .superpowers/sdd/plan/task-5-brief.md
Report: .superpowers/sdd/plan/task-5-report.md

## Scope

`git diff --stat c027e20d3..2dc3d66a5` touches exactly the three files named in the
brief:

- `services/atlas-pets/atlas.com/pets/character/model.go`
- `services/atlas-pets/atlas.com/pets/character/rest.go`
- `services/atlas-pets/atlas.com/pets/character/rest_test.go`

No other files, services, or libs touched. Scope matches the brief.

## Findings

### PASS — accessor un-stubbed correctly

`model.go:207-209`:
```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```
Previously `func (m Model) SpawnPoint() byte { return 0 }`. Now returns the
backing field `spawnPoint uint32` (declared at `model.go:38`), matching the
`SpawnPoint() uint32` accessor contract established by earlier tasks in this
plan. No `byte` narrowing needed here because `RestModel.SpawnPoint` is also
`uint32` (`rest.go:38`) — there is no byte-wide wire field on this leg.

### PASS — both legs plumbed, scope held to spawnPoint only

`rest.go` `Extract` gained `spawnPoint: m.SpawnPoint`; `Transform` gained
`SpawnPoint: m.spawnPoint`. Diff shows only this one field added per struct
literal — the ~26 other RestModel fields (AccountId, WorldId, Name, Level,
etc.) remain absent from both `Extract` and `Transform`, unchanged. Matches
brief instruction not to "finish" the struct literals.

### PASS — tests are honest, not just coverage

`rest_test.go`: `TestTransformRoundTrip` fixture gained `spawnPoint: 55` plus
an explicit `got.SpawnPoint() != 55` assertion (not just relying on
`DeepEqual`, which would pass trivially if both legs dropped the field
symmetrically). New `TestExtract_SpawnPoint` asserts the inbound leg directly
against `RestModel{Id: 11, SpawnPoint: 55}`.

Verified by inspection and re-run: reverting `model.go` accessor to
`return 0` reproduces the exact predicted RED failure
(`SpawnPoint() = 0, want 55`) in both tests; with the fix applied both pass:

```
$ cd services/atlas-pets/atlas.com/pets && go test ./character/... -v -run 'SpawnPoint|TransformRoundTrip'
--- PASS: TestTransformRoundTrip (0.00s)
--- PASS: TestExtract_SpawnPoint (0.00s)
ok  	atlas-pets/character	0.007s
```

### PASS — no live payload change (as claimed)

`grep -rln "character\.Transform\|character\.Extract" . | grep -v _test.go`
from the module root returns no matches — confirms the brief's claim that
`character.Transform`/`Extract` have zero callers outside `rest_test.go` in
this service, so this change is inert on the wire until a future task wires
a real caller.

### Not applicable — Task 5 context note

Per the task instructions, atlas-pets' Transform dropping/adding fields
outside the PRD's stated non-goal is pre-authorized by context.md decision 2
for `spawnPoint` specifically. Not raised as a finding.

## Build/test

```
cd services/atlas-pets/atlas.com/pets && go build ./...   # succeeds
go test ./character/...                                    # PASS (all)
```

## Verdict

APPROVED. Change matches the brief exactly, scope is held tightly to
`spawnPoint`, tests are honest and demonstrably fail without the fix, no
cross-service seam risk (zero live callers).
