# Review: bug-disease-command-emits-nonexistent-zombify-stat

Range reviewed: `b2653c07c..HEAD` (fix commit `49465ebac`; `60c478d78` is docs-only,
adding the bug diagnosis file itself).

Requirement doc:
`docs/tasks/task-256-zombify-healing-consequences/bug-disease-command-emits-nonexistent-zombify-stat.md`

## Scope

```
git diff --stat b2653c07c..HEAD
 .../bug-disease-command-emits-nonexistent-zombify-stat.md | 119 ++++
 .../messages/command/disease/commands.go                  |  33 +++---
 .../messages/command/disease/commands_test.go             |  70 +++++++
```

Matches the documented fix exactly: `validDiseases` alias table plus a new
regression test in `atlas-messages`, and the bug-report doc itself. No file
outside `services/atlas-messages` is touched, consistent with the "Not yet
answered"/"Fix" sections' claim that atlas-buffs, atlas-channel,
atlas-consumables, and `libs/atlas-constants` need no change.

## Requirement-by-requirement

1. **`validDiseases` becomes an alias table sourced from
   `libs/atlas-constants/character` constants, not bare string literals.**
   `services/atlas-messages/atlas.com/messages/command/disease/commands.go:19,28-42`
   — `map[string]statconstant.TemporaryStatType`, values are
   `statconstant.TemporaryStatType{Seal,Darkness,Weaken,Stun,Curse,Poison,Slow,
   Seduce,Undead,Confuse,StopPortion}`. PASS.

2. **`ZOMBIFY` → `TemporaryStatTypeUndead`, `WEAKNESS` → `TemporaryStatTypeWeaken`,
   remaining nine map to matching constants.** `commands.go:29-41` — verified
   each of the 11 disease words against
   `libs/atlas-constants/character/temporary_stat.go` (grep, full constant
   block read): `TemporaryStatTypeSeal="SEAL"`, `...Darkness="DARKNESS"`,
   `...Weaken="WEAKEN"`, `...Stun="STUN"`, `...Curse="CURSE"`,
   `...Poison="POISON"`, `...Slow="SLOW"`, `...Seduce="SEDUCE"`,
   `...Undead="UNDEAD"`, `...Confuse="CONFUSE"`, `...StopPortion="STOP_PORTION"`
   — all eleven constants genuinely exist and the string values match what the
   bug report cites. PASS.

3. **Typed word kept as key; canonical names accepted as additional keys
   (`UNDEAD`, `WEAKEN`) so both spellings work.** `commands.go:32,39` — `"WEAKEN"`
   and `"UNDEAD"` added as separate map entries pointing at the same constants.
   `diseaseType := strings.ToUpper(match[2])` at `commands.go:61` normalizes
   case before lookup, so both `@disease me zombify` and `@disease me undead`
   resolve. The "Valid: …" help line (`commands.go:81-90`) now lists 13 words
   instead of 11 — an intentional, documented side effect, not a defect. PASS.

4. **New table-driven regression test asserting every `validDiseases` value
   resolves to the canonical constant; a case pinning `ZOMBIFY` → `UNDEAD` on
   the emitted command.** `commands_test.go:16-70` —
   `TestValidDiseasesResolveToCanonicalTemporaryStatTypes` iterates all 13 map
   entries (including both spelling pairs) against expected
   `character.TemporaryStatType` constants, and additionally asserts
   `len(validDiseases) == len(tests)` so a stray/missing entry fails loudly.
   `TestZombifyEmitsUndeadStatChange` builds a `buff.StatChange` from
   `validDiseases["ZOMBIFY"]` and asserts `Type == "UNDEAD"`, pinning the exact
   symptom from the bug report. Ran locally:
   `go test -count=1 -v ./command/disease/...` → all 13 subtests + both top
   level tests PASS (fresh run, not cached). Test honesty: before the fix
   `validDiseases["ZOMBIFY"]` was the literal `"ZOMBIFY"` (as
   `TemporaryStatType`), which fails `want == TemporaryStatTypeUndead` — this
   is not a test that would pass either way. PASS.

5. **"Not yet answered": constants-only form vs. `CharacterTemporaryStatTypeByName`.**
   The implementer chose the constants-only form (asserting against
   `character.TemporaryStatType` directly, no `libs/atlas-packet/model`
   dependency added to `go.mod`). Plan explicitly said "either satisfies the
   regression guard; prefer the constants-only form if it avoids adding a
   module dependency" — this resolves the open question correctly. Checked
   `go.mod`/imports: no new `atlas-packet` dependency was introduced. PASS.

## Consumer seam trace (hand-traced, not just grep)

- **atlas-buffs** immunity/disease classification —
  `services/atlas-buffs/atlas.com/buffs/character/immunity.go:7-10` —
  `diseaseStatTypes` keys on `"UNDEAD"`, `"WEAKEN"` (and the other 9 canonical
  names), matching what the fixed producer now emits. No storage or periodic
  handling in this package keys on stat-type strings beyond this set (checked
  the whole file, 29 lines). PASS — consumer already correct, unaffected by
  this diff, and now actually reachable.
- **atlas-channel** temporary-stat mask encoding —
  `libs/atlas-packet/model/character_temporary_stat.go:258` —
  `newAndIncDiseased(character.TemporaryStatTypeUndead)` registers the
  `UNDEAD` mask bit; `CharacterTemporaryStatTypeByName` (`:264-271`) looks up
  by the `character.TemporaryStatType` value, which will now resolve for the
  emitted `"UNDEAD"` string instead of failing with "character temporary stat
  type not found" as the bug report's live evidence showed. PASS.
- **atlas-channel** `character/buff.IsZombified` —
  `services/atlas-channel/atlas.com/channel/character/buff/model.go:22-36` —
  predicate tests `c.Type() == string(charconst.TemporaryStatTypeUndead)`.
  Matches the new emitted value. PASS.
- **atlas-consumables** `character/buff.IsZombified` + HP-restoration halving —
  `services/atlas-consumables/atlas.com/consumables/character/buff/model.go:63-77`
  — predicate tests `c.Type == charconst.TemporaryStatTypeUndead`; the halving
  logic itself lives outside this diff's file set (task-256, not this fix) and
  is out of scope for this review per the bug doc's own statement ("Nothing in
  the task-256 diff is implicated"). PASS for the seam this fix controls
  (emission now matches the predicate's expectation); halving logic not
  re-verified here (out of scope, correctly so — verified in existing test
  `services/atlas-consumables/atlas.com/consumables/character/buff/processor_test.go`
  which predates this fix and is untouched by it).
- **No other producer emits the old literals.** `grep -rn '"ZOMBIFY"'` across
  the worktree returns zero Go source matches (only doc/comment/test
  references to the word `ZOMBIFY` as an English disease name, all of which
  resolve to `TemporaryStatTypeUndead`). `grep -rn '"WEAKNESS"'` finds one
  unrelated hit: `libs/atlas-constants/monster/temporary_stat.go:40` and
  `monster/skill.go:158`, which is the *monster* skill-type domain (mob
  zombify skill → its own `SkillTypeWeakness`/`SkillTypeUndead` constants,
  already correct per the bug doc, and not a producer of the character-buff
  APPLY command this fix touches). PASS.

## Build/test verification

```
cd services/atlas-messages/atlas.com/messages
go build ./...                              # clean
go test -count=1 -v ./command/disease/...   # 13+2 tests, all PASS, fresh run
```

## Findings

None blocking. No non-blocking findings beyond the documented, intentional
"Valid: …" help-line expansion noted in item 3 above (not a defect — it is the
directly-specified consequence of accepting both spellings).

## Not evaluable

- HP-restoration halving arithmetic in atlas-consumables (FR-5/FR-6) — owned
  by the task-256 diff, not this fix; the bug doc itself states "Nothing in
  the task-256 diff is implicated," and this fix's diff does not touch that
  code, so it is outside this unit's review surface.
- Live re-verification in the `pr-1449` environment (the bug was originally
  reproduced live) — not re-run here; only source-level and unit-test
  verification performed, per this review's scope (diff + directly-relied-on
  consumer contracts).

## Verdict rationale

Every requirement in the bug/fix doc is met, verified against the actual
constant definitions (not just diff review), all four consumer seams named in
the review brief were hand-traced and match the new emitted values, no
lingering producer emits the old literals, and the new tests are honest
regression guards (verified to assert against real constants and to cover the
exact reported symptom). Build and tests pass on a fresh run.
