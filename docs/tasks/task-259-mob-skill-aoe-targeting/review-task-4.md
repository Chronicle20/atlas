# Review: Task 4 of 4 — task-259 (mob skill AoE targeting)

Commit under review: `9b9c0b9` (`test(atlas-monsters): assert banish and dispel
inherit box-scoped targeting`), range `51c5242..9b9c0b9`.

Brief: `.superpowers/sdd/plan/task-4-brief.md`
Report: `.superpowers/sdd/plan/task-4-report.md`

## Scope

```
$ git show --stat 9b9c0b9
 monster/disease_callers_test.go   | 149 ++++++++++++++++++++++
 monster/information/builder.go    |   9 ++
 monster/processor.go              |  14 ++++++++----
 3 files changed, 168 insertions(+), 4 deletions(-)
```

Matches the brief's file list exactly (processor.go, information/builder.go,
new test file). No other files touched. Scope confirmed.

## Findings

### 1. `p.emit` routing — topic/payload identity (PASS)

`git show 9b9c0b9 -- monster/processor.go` diff hunks:

- `executeDebuff` (processor.go:1245): `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicCharacterBuff)(applyDiseaseCommandProvider(...))` → `p.emit(EnvCommandTopicCharacterBuff, applyDiseaseCommandProvider(...))`. Provider arguments unchanged, argument order unchanged.
- `executeBanish` (processor.go:1274): `producer.ProviderImpl(...)(EnvCommandTopicPortal)(warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId)))` → `p.emit(EnvCommandTopicPortal, warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId)))`. Identical.
- `executeDispel` (processor.go:1285): `producer.ProviderImpl(...)(EnvCommandTopicCharacterBuff)(cancelAllBuffsCommandProvider(m.Field(), characterId))` → `p.emit(EnvCommandTopicCharacterBuff, cancelAllBuffsCommandProvider(m.Field(), characterId))`. Identical.

`NewProcessor` (processor.go:107-112) binds `emit: func(topic string, provider ...) error { return producer.ProviderImpl(p.l)(p.ctx)(topic)(provider) }` (verified by reading processor.go:99-118) — so production behavior is byte-identical to the pre-change direct call. No topic constant changed, no provider argument dropped or reordered.

### 2. `testInformationLookup` guard in `executeBanish` (PASS)

processor.go:1251-1262 reproduces the exact shape used elsewhere in the same
file (verified at processor.go:985-986, 1369-1370, 1677-1678, 1749-1750 via
`grep -n testInformationLookup monster/processor.go`): declare `var ma
information.Model; var err error`, branch on `testInformationLookup != nil`,
fall back to `information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId())`.
Matches the brief's Step 5 snippet verbatim.

### 3. Uncapped banish/dispel (PASS)

`disease_targets.go:41` — the only cap check — reads:
```go
if uint16(skillId) == monster2.SkillTypeSeduce && sd.Count() > 0 && uint32(len(ids)) > sd.Count() {
```
`executeBanish` and `executeDispel` (processor.go:1251-1289) call
`p.getDiseaseTargets` unconditionally with no additional filtering before or
after; no cap logic was added to either executor in this commit. Confirmed
by diff — the only change to `executeBanish`/`executeDispel` bodies is the
`p.emit` substitution and the lookup guard; the targeting/emission loop
itself is untouched.

### 4. No literal seduce id (PASS)

```
$ grep -rn 'SkillTypeSeduce' monster/*.go
disease_targets.go:41: ... monster2.SkillTypeSeduce ...
disease_targets_shell_test.go:183, disease_targets_test.go:12,170, disease_value_test.go:24
```
No literal `128` anywhere in `monster/*.go` associated with seduce.

### 5. `math/rand` import intact (PASS)

`monster/processor.go:13` still imports `"math/rand"`; only use is
`rand.Intn(base) / 10` at processor.go:714 (damage formula). No
`rand.Shuffle`.

### 6. `SetBanish` builder addition (PASS)

`monster/information/builder.go`: adds `banish Banish` field to
`ModelBuilder`, `SetBanish(banish Banish) *ModelBuilder` method (matches the
existing fluent-setter shape of `SetBoss`/`SetResistances` in the same
file), and `banish: b.banish` added to the `Model{}` literal in `Build()`.
No other builder fields disturbed.

### 7. Module dependencies unchanged (PASS)

`git diff 51c5242..9b9c0b9 -- go.mod go.sum` (repo root and module root) —
empty output. No new dependency introduced.

### 8. Test builder-pattern / no test-helper file convention (PASS)

New file `monster/disease_callers_test.go` is a `_test.go` file in package
`monster`; no `*_testhelpers.go` created. Uses `newRecordingProcessor` (an
existing helper in `processor_test.go`) and `mobskill.NewModelBuilder()` /
`information.NewModelBuilder()` — both existing Builder-pattern
constructors.

### 9. Tests genuinely assert box-scoped inheritance, not vacuously (PASS)

Read `disease_targets.go`'s `selectDiseaseTargets`/`getDiseaseTargets` (the
Task 3 selector both executors share). With the pre-Task-3 whole-field
selector, all three field characters (`1,2,3`) would have been targeted by
`TestExecuteDispel_TargetsOnlyInBoxCharacters` and
`TestExecuteBanish_TargetsOnlyInBoxCharacters` — character `2` at `(400,
205)` is outside the box `(-50,-30)-(50,30)` translated from mob position
`(100,200)` → box in world coords `x:[50,150], y:[170,230]`; `400` falls
well outside. The test's expected count of `2` (not `3`, the full candidate
set) is therefore a genuine assertion that would fail against the old
selector. `TestExecuteDispel_NoCapForNonSeduce` uses `SetCount(2)` with 4
in-box characters and asserts 4 events — this is a genuine assertion against
a hypothetical (wrong) implementation that applied the SEDUCE-only cap
universally; expected set (4) is the full candidate set here, but that is
correct given the test's actual claim (no cap applies) — not a vacuous
"expect everything" pattern, since the alternative implementation under test
(cap applied) would yield exactly 2, not 4.
`TestExecuteBanish_NoBanishMapEmitsNothing` asserts 0 events with
`MapId: 0`, matching the unchanged early-return branch — this is a
regression guard, not a targeting proof, and is correctly framed as such in
the report.

All four tests were also confirmed to build-fail before Step 3
(`SetBanish` undefined) and confirmed passing after — I additionally ran
them directly:

```
$ go test ./monster/... -run 'TestExecuteDispel|TestExecuteBanish' -v
--- PASS: TestExecuteDispel_TargetsOnlyInBoxCharacters (0.00s)
--- PASS: TestExecuteDispel_NoCapForNonSeduce (0.00s)
--- PASS: TestExecuteBanish_TargetsOnlyInBoxCharacters (0.00s)
--- PASS: TestExecuteBanish_NoBanishMapEmitsNothing (0.00s)
PASS
ok  	atlas-monsters/monster	0.061s
```

### 10. Build/vet (PASS)

```
$ go build ./...   # exit 0, no output
$ go vet ./monster/...   # exit 0, no output
```

## Not evaluable

- Full `go test ./... -race` for the entire module was not run in this
  review, because a concurrent Task-3 fix-round implementer is actively
  editing `monster/disease_targets_shell_test.go` in this same worktree per
  the task instructions (do not disturb in-flight work; avoid overlapping
  the shared test binary while it may be mid-edit). The targeted run above
  (`-run 'TestExecuteDispel|TestExecuteBanish'`) and `go build ./...`
  substitute for this. The full `tools/verify.sh` gate is explicitly the
  controller's responsibility per the brief and was not attempted here.

## Verdict rationale

All binding constraints for this task hold: emit routing is byte-identical
in topic and payload, the `testInformationLookup` guard copies the existing
form exactly, banish/dispel remain uncapped, no literal 128, `math/rand`
import intact, no new dependencies, Builder-pattern test conventions
followed, and the four new tests are demonstrably non-vacuous (they encode
an expected count strictly smaller than the full candidate set, or reason
about a specific counter-implementation that would produce a different
count). Scope matches the brief's file list exactly.

No blocking or non-blocking findings.
