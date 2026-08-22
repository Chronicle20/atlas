# Review — Task 25: rewire the submit handler (A7 class/SP gate + `al` mapping)

Commit: `6bd233780` (range `6880ad6c2..6bd233780`)
Brief: `.superpowers/sdd/plan/task-25-brief.md`
Report: `.superpowers/sdd/plan/task-25-report.md`

## Scope

`git diff --stat 6880ad6c2..6bd233780`:

```
.../character_cash_item_use_maple_life_test.go     |  10 +-
 .../channel/socket/handler/maple_life_create.go    | 118 ++++---
 .../socket/handler/maple_life_create_test.go       | 348 ++++++++++++++++++---
 3 files changed, 391 insertions(+), 85 deletions(-)
```

Matches the brief's file list exactly, plus one rename-only file
(`character_cash_item_use_maple_life_test.go`) the brief did not list but
which was required to keep the package building after the `seedCharacterFunc`
→ `createMapleLifeFunc` rename — same package, no behavioural change, and the
report discloses it plainly. Scope confirmed.

## 1. The `al` mapping arithmetic

Cross-checked `maple_life_create.go:59-73` against
`docs/tasks/task-246-maple-life-character-creation/open-selected-al-mapping-round2.md`
("What is now settled (wire side)" and the `al2` CORRECTION section) and the
brief's own table:

| wire | brief says | implementation (`maple_life_create.go:65-72`) | verdict |
|---|---|---|---|
| `al0` → face | `uint32(al0)` unchanged | `uint32(sub.AL0())` | matches |
| `al1` → hair | `uint32(al1/10)*10` | `uint32(sub.AL1()/10)*10` | matches |
| `al2` → hairColor | `uint32(al2 % 10)` | `uint32(sub.AL2()%10)` | matches |
| `al3` → skinColor | `byte(al3)` | `byte(sub.AL3())` | matches |
| `gender` | — | `byte(sub.Gender())` | matches `CreateMapleLife`'s `gender byte` param |
| `currentClass` | `classOrdinal` | `uint32(sub.CurrentClass())` | matches — passed as the class-family ordinal, not `jobIndex` (design §11 Decision 1, correctly *not* repeated here) |
| `sp` | — | `byte(sub.SP())` | matches |

Call-site argument order (`maple_life_create.go:66-73`) was checked
byte-for-byte against `factory.Processor.CreateMapleLife`'s actual signature
(`character/factory/processor.go:25`):

```go
CreateMapleLife(accountId uint32, worldId world.Id, name string, classOrdinal uint32, gender byte, face uint32, hair uint32, hairColor uint32, skinColor byte, sp byte) (string, error)
```

— `accountId, worldId, sub.Name(), uint32(sub.CurrentClass()), byte(sub.Gender()), uint32(sub.AL0()), uint32(sub.AL1()/10)*10, uint32(sub.AL2()%10), byte(sub.AL3()), byte(sub.SP())`.
Positional order matches exactly; no field is silently transposed.

**Mutation-tested independently, on a field the implementer's own check did
not cover** (report only mutated `al1`; I mutated `al2`):

```
sed -i 's/uint32(sub.AL2()%10)/uint32(sub.AL2())/' maple_life_create.go
go test ./socket/handler/... -run TestMapleLifeCreateMapsTheWireToTheFactoryRequest -v
```

Result: exactly and only the `colour_needs_normalising` subtest failed
(`hairColor = 12, want 2`); `plain_values` and `hair_needs_normalising`
passed. Reverted (`git checkout -- maple_life_create.go`), confirmed
`git status --short` clean, confirmed `go build ./... && go test ./...`
green afterward. This is genuine coverage of the arithmetic, not a seam
that's swapped around it.

`al3`/skinColor is left as a raw pass-through per the design doc's own
finding that its domain is unconfirmed and *not* closable by the `al2`
argument — correctly not normalised, matching the brief.

**Verdict: PASS.** Every mapped field matches the settled derivation exactly, with no off-by-one, no swapped index, and the finding independently reproduced against production code (not eyeballed).

## 2. The A7 class/SP gate

`maple_life_create.go:186-197` (Gate 5, placed after gate 4's name re-check
and immediately before `createMapleLifeFunc`):

```go
if sub.CurrentClass() < 0 || sub.CurrentClass() > 4 {
    fail(...)
    return
}
if sub.SP() < 0 || sub.SP() > 10 {
    fail(...)
    return
}
```

- Boundary values match the brief's table exactly: class ordinal `0..4`
  inclusive accepted, `5` and `-1` rejected; SP `0..10` inclusive accepted,
  `11` and `-1` rejected. Confirmed against
  `TestMapleLifeCreateRejectsAnUnconfiguredClassOrdinal` and
  `TestMapleLifeCreateRejectsAnOutOfRangeSP` (`maple_life_create_test.go:566-660`),
  which cover exactly those four cases per gate.
- **Rejected, never clamped**: both branches `return` before reaching
  `createMapleLifeFunc` — a rejected submit cannot fall through into a
  factory call. `fail()` writes `MapleLifeErrorUnknownError` and does not
  touch the maplelife registry or the cash item.
- Gate ordering is correct: gate 5 sits strictly before the factory call and
  strictly after gate 4 (name re-check), matching the brief's placement
  instruction and the updated doc-comment gate list.

**Mutation-tested independently** (inverted the class-ordinal condition from
`< 0 || > 4` to `>= 0 && <= 4`, i.e. accept exactly what should be rejected):

```
go test ./socket/handler/... -run TestMapleLifeCreateRejectsAnUnconfiguredClassOrdinal -v
```

Result: all four subtests failed (`min_in_range`/`max_in_range` — expected a
factory call but got a reject; `above_range`/`negative` — expected a reject
but got a factory call). Reverted, confirmed clean, confirmed
`go build ./... && go test ./...` green afterward.

**Verdict: PASS.**

## 3. The `newFactoryProcessorFunc` seam — judged on its merits

The implementer added a second-level package var,
`newFactoryProcessorFunc` (`maple_life_create.go:46-52`), between
`createMapleLifeFunc` and `factory.NewProcessor`, so that
`TestMapleLifeCreateMapsTheWireToTheFactoryRequest` can restore the *real*
`createMapleLifeFunc` (via `originalCreateMapleLifeFunc`, captured at
package-init time) and intercept one level lower, letting the al-mapping
arithmetic execute for real while still avoiding a live REST round trip.

**The alternative it avoids is real and would be worse.** The brief's Step 1
text literally describes capturing `createMapleLifeFunc`'s own arguments —
if the seam stayed there, the mapping arithmetic lives *inside* the swapped
function and never runs under test, reproducing exactly the A8 gap this task
exists to close (a wrong `al` mapping shipping with a green suite). This
reviewer's own mutation test above required the deeper seam to be
meaningful — verified directly, not taken on the report's word.

**Precedent check, not just self-declared.** The report and the doc comment
both cite `cashItemInSlotFunc` (`character_cash_item_use.go:1034`) as the
established idiom. That check holds up: `maple_life_create.go` alone already
has four package-var seams of this exact shape
(`accountSlotsFunc`, `charactersInWorldFunc`, `mapleLifeNameValidityFunc`
pre-existing, plus `createMapleLifeFunc`/`newFactoryProcessorFunc` added
here), and a repo-wide grep confirms this "one `var …Func = func(...)` per
swappable external call" pattern recurs across most of `socket/handler/*.go`
(`character_cash_item_use.go` alone has 5; `rps_action.go` has 5). This is
not a novel pattern invented for this task — it is the file's (and the
package's) standing idiom applied one call deeper than usual.

**On unifying with Task 23's `mapleLifeProcessor` smell.** Task 23's flagged
seam (`services/atlas-character-factory/atlas.com/character-factory/factory/resource.go`)
lives in a **different service, different Go module, different package** —
`atlas-character-factory`, not `atlas-channel`. The two cannot be literally
unified (they don't share a compilation unit, and nothing routes through
both). More importantly, they are not the same *kind* of finding: Task 23's
review (`review-task-23.md:73`) found `mapleLifeProcessor` was the **first
of its kind** in `resource.go` — its two sibling handlers both construct
`NewProcessor(...)` inline with no seam, so it was flagged as an
inconsistency worth a follow-up. `newFactoryProcessorFunc` here is the
**opposite case** — it is the Nth instance of an idiom already established
four times over in the same file. There is nothing to unify; if a queued
cleanup exists, it should track Task 23's `mapleLifeProcessor` on its own
merits (bring it in line with its two siblings in `resource.go`), and should
not fold this task's seam into that ticket — this one is not a smell, it is
idiom-compliant.

**Verdict: justified, no cleanup owed by this task.**

## 4. Hollowness re-check

- Independent mutation of the `al2` formula (not the implementer's own
  `al1` check) — confirmed above, exactly the expected subtest failed.
- Independent gate inversion of the class-ordinal branch — confirmed above,
  all four subtests failed as expected.
- Both mutations reverted; `git status --short` on the module clean; fresh
  `go build ./... && go test ./...` green after revert (not relying on a
  cached result from before the mutation).

## 5. `logCreateFailure` / conflict-to-name-taken mapping

`maple_life_create.go:239-251`: `requests.ErrConflict` → `mlcb.MapleLifeErrorNameTakenAtSubmit`, everything else → `mlcb.MapleLifeErrorUnknownError`, matching the brief's three-arm constraint exactly. `TestMapleLifeCreateMapsConflictToNameTaken` asserts the specific arm and that no registry entry survives. **PASS.**

## 6. Task 24 build-break follow-up

Task 24's review noted a transient `undefined: logCreateFailure` break from
this task's then-uncommitted work. In the committed state, `logCreateFailure`
is defined at `maple_life_create.go:239` and both call sites
(`maple_life_create.go:200` and none elsewhere) resolve; `go build ./...`
exits 0. Resolved, no residue.

## Pre-existing tests

`TestMapleLifeCreateNeverConsumesTheItem` and `TestMapleLifeCreatePreCheckOrder`
present unchanged in behaviour (field renames only:
`seedCalls`→`mapleLifeCalls` etc.); full `go test ./...` on the module: exit
0, `ok atlas-channel/socket/handler`.

## Not evaluable

- Nothing found within the diff's surface that could not be evaluated. The
  factory-side `CreateMapleLife` implementation itself (Task 24) is
  read-only per the brief and out of this task's scope; this review checked
  only that the call site matches its published signature, not the
  correctness of what the factory does with the values.

## Non-blocking notes

- `maple_life_create.go:199-205`: on a factory error, `logCreateFailure`
  already logs a `Warn`/`Error` line internally, and the call site then
  *also* emits `l.Warnf("Maple Life character creation for account [%d] was
  rejected by the factory.", accountId)` — two log lines for one failure.
  Cosmetic only; no behavioural or test impact.

## Verdict

APPROVED

---

```text
verdict: APPROVED
artifact: docs/tasks/task-246-maple-life-character-creation/review-task-25.md
scope_confirmed: commit 6bd233780 — maple_life_create.go (al mapping + A7 gate + logCreateFailure rename), maple_life_create_test.go (new/renamed tests), character_cash_item_use_maple_life_test.go (rename-only, same package)
blocking: 0
non_blocking: 1
  - services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go:199-205 — logCreateFailure logs internally and the call site logs again on the same failure path; cosmetic double log line, no behavioural impact.
not_evaluable: 0
```
