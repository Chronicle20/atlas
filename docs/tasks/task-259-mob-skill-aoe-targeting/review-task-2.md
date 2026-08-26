# Review: task-259 Task 2 — the pure target selector

Range reviewed: `0ff80b1e1..08f85678e` (single commit `08f85678e`).

## Scope confirmation

`git diff --stat 0ff80b1..08f8567` shows exactly three files, matching the brief's
"Files" list one-for-one:

- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets.go` (new, 44 lines)
- `services/atlas-monsters/atlas.com/monsters/monster/disease_targets_test.go` (new, 188 lines)
- `services/atlas-monsters/atlas.com/monsters/monster/mobskill/builder.go` (+10 lines)

No `processor.go` change, no go.mod/go.sum change. Scope matches Task 2's brief exactly.
(Note: the worktree currently has *uncommitted* local modifications to `processor.go` and
`disease_targets.go` — these are Task 3's in-progress work, outside `0ff80b1..08f8567`, and
were excluded from this review; I stashed/restored them only to get a clean vet/gofmt run
against the true commit boundary, then restored the tree to its original state. `git status`
after the review matches what it was before.)

## Requirement-by-requirement (against `task-2-brief.md`)

1. **`SetCount` on `mobskill.ModelBuilder`** — PASS. `count uint32` field added after
   `duration` (`mobskill/builder.go`), setter added with the exact doc comment from the
   brief, `count: b.count` wired into `Build()`. Verified field order and wiring by reading
   `builder.go:11-19` and `:94-109`.

2. **`positionedCharacter` struct** — PASS. `disease_targets.go:15-19`, fields `id uint32`,
   `x int16`, `y int16`, exact match to the brief's interface contract, package `monster`.

3. **`selectDiseaseTargets` signature and behavior** — PASS.
   `disease_targets.go:35` — `func selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32` matches the "Produces, for Task 3" interface exactly.

4. **Bounding-box comparison form** — PASS, byte-for-byte identical to the mandated
   `executeHeal` reference. Compared directly:
   - `processor.go:1204-1206` (`executeHeal`): `dx := int32(other.X()) - int32(m.X())` / `dy := int32(other.Y()) - int32(m.Y())` / `if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY()`
   - `disease_targets.go:36-39`: `dx := int32(c.x) - int32(mobX)` / `dy := int32(c.y) - int32(mobY)` / `if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY()`
   Same operand order, same casts, same inclusive four-sided comparison.

5. **No literal `128` for seduce** — PASS. `disease_targets.go:42` reads
   `uint16(skillId) == monster2.SkillTypeSeduce`, imported from
   `github.com/Chronicle20/atlas/libs/atlas-constants/monster` as `monster2`. Confirmed
   `SkillTypeSeduce = 128` lives only in the constants lib (`libs/atlas-constants/monster/skill.go:40`), never hardcoded in the diff. `grep -n "128" disease_targets.go` — no match.

6. **Cap applies to SEDUCE only** — PASS. The cap block at `disease_targets.go:41-44` is
   gated on `uint16(skillId) == monster2.SkillTypeSeduce`; every other skill id falls
   through with `ids` untouched. Test case `non-seduce ignores count` (skill=slow,
   count=2, 4 in-box candidates) asserts all 4 are returned, directly pinning this.

7. **Deterministic ordering, no `rand`** — PASS. `grep -n "rand" disease_targets.go` —
   no match. The selector appends in candidate-slice order and slices
   `ids[:sd.Count()]` — no map iteration, no randomness. `TestSelectDiseaseTargets_IsDeterministic` (`disease_targets_test.go:158-188`) calls the function twice with identical
   arguments and asserts both results equal `[]uint32{1, 2}` and each other — this is a
   real FR-4.2 assertion, not a tautology (it would fail if map iteration or `rand.Shuffle`
   were introduced).

8. **No facing-direction mirroring** — PASS, confirmed absent by reading the full 44-line
   file; only `dx`/`dy` translation and box comparison, no facing/direction parameter
   exists in the signature at all.

9. **No dead-character (`hp`) filtering** — PASS, confirmed absent; `positionedCharacter`
   carries no `hp` field and `selectDiseaseTargets` never inspects one.

10. **`selectDiseaseTargets` purity (zero I/O, no context, no network)** — PASS. Function
    signature takes only value types (`int16`, `mobskill.Model`, `byte`,
    `[]positionedCharacter`); no `context.Context` parameter, no import beyond
    `mobskill` and the constants package, no goroutines, no channel, no external call.

11. **Test setup uses Builder pattern, no `*_testhelpers.go`** — PASS. Test file uses
    `mobskill.NewModelBuilder().SetBoundingBox(...).SetCount(...).Build()` directly
    (`disease_targets_test.go:132-135`, `:159-162`); no new test-only constructor file
    created. `git diff --stat` confirms only the two new files plus `builder.go`.

12. **No new module dependency; `golang.org/x/sync` stays indirect** — PASS.
    `git diff 0ff80b1..08f8567 -- '**/go.mod' '**/go.sum'` is empty. `go.mod:40` still
    reads `golang.org/x/sync v0.22.0 // indirect`.

## Table test — brief table vs. implementation

Compared all 10 rows of the brief's table against `disease_targets_test.go` line by line
(`in box and out of box`, `boundary is inclusive`, `y outside box`,
`non-seduce ignores count`, `seduce caps at count`, `seduce count zero does not cap`,
`seduce count above candidate count`, `seduce cap applies after box filter`,
`no candidates`, `all candidates out of box`). Every candidate id/x/y, skill, count, and
`want` matches exactly. Both empty-result rows (`no candidates`, `all candidates out of
box`) use the `len(got) == 0` assertion form the brief specifies; the other 8 use exact
slice equality.

The brief's narrative sentence ("Box is lt(-50,-30), rb(50,30) in every case except `no
bounding box`") references a row that does not exist in the brief's own table, and the
implementer flagged this explicitly in the report rather than silently guessing. Per this
review's stated scope boundary, `!sd.HasBoundingBox()` is Task 3's deliverable (it lives
in the `ProcessorImpl.getDiseaseTargets`/`resolvePositions` boxless-skill path, not in
`selectDiseaseTargets`, which unconditionally applies the box comparison). Omitting a "no
bounding box" row from `TestSelectDiseaseTargets` is therefore correct, not a gap — the
brief's own table (10 rows, no such case) is what Step 1 says to transcribe, and the
narrative sentence is describing the box literal, not prescribing an 11th row. Non-blocking
note only: the brief's own prose is internally inconsistent (references a table row it
never defines); this is a brief-authoring nit, not an implementation defect, and worth a
one-line fix if the brief is touched again.

## Verification run in this review

```
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./monster/... -run TestSelectDiseaseTargets -v
```
All 10 subtests plus `TestSelectDiseaseTargets_IsDeterministic` PASS.

At the clean `08f85678e` commit (verified by stashing then restoring unrelated
uncommitted Task-3-in-progress changes in the worktree):
```
go vet ./monster/...     # exit 0
gofmt -l disease_targets.go disease_targets_test.go mobskill/builder.go   # no output
```
Both clean.

## Not evaluable

None — everything in Task 2's stated scope (`positionedCharacter`, `selectDiseaseTargets`,
`mobskill.ModelBuilder.SetCount`, and the table test) was directly readable and testable
from this diff alone.

## Verdict

APPROVED. Every requirement in the brief is met with `file:line` evidence, the mandated
box-comparison form is copied byte-for-byte from `executeHeal`, the SEDUCE-only cap uses
the constants-lib symbol with no literal `128`, ordering is deterministic with a real
pinning test, purity holds, the Builder pattern is used correctly for test setup, and no
new dependency was introduced. The only observation is a non-blocking note about an
inconsistency in the brief's own prose (not the implementation).
