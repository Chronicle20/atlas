# Review: fix(atlas-channel): correct Heal party XP formula magnitude

Range reviewed: `8e5df4e27..b9904f7f6` (single commit `b9904f7f6`).
Brief: `docs/tasks/task-256-zombify-healing-consequences/bug-heal-party-xp-magnitude.md`.

## Scope confirmed

`git diff --stat 8e5df4e27..b9904f7f6` touches exactly the eight files the
brief's `## Fix` table names, no more:

```
services/.../heal/formula.go          | 28 +++++---
services/.../heal/formula_test.go     | 77 +++++++++++++++++-----
services/.../heal/heal.go             | 17 +++--
services/.../heal/heal_apply_test.go  | 10 +--
services/.../heal/recipients.go       |  1 +
services/.../recipients.go            |  6 ++
services/.../recipients_map_test.go   |  5 +-
services/.../recipients_test.go       | 32 ++++++++-
```

`docs/tasks/task-256-zombify-healing-consequences/report-bug-heal-party-xp-magnitude.md`,
referenced by the task prompt as the implementer's own report, **does not
exist** in the worktree (`find … -iname "*heal-party-xp*"` returns only the
bug file). Its claims could not be checked against anything; see
`## Not evaluable`.

## Findings

### 1. Arithmetic matches the spec exactly (PASS)

`formula.go:105-118` (`HealXp`):

```go
if r.IsCaster || r.Level == 0 {
    continue
}
applied := int64(appliedPerRecipient(perTarget, r))
total += applied * int64(casterLevel) / int64(r.Level) / 15
```

This is computed **per recipient inside the loop**, not on a summed total —
each recipient's `applied * casterLevel / r.Level / 15` is truncated
independently before being added to the accumulator, satisfying "per-recipient
independence" and the exact truncation order `actualHeal * casterLevel /
targetLevel / 15` specified in `## Expected`.

Verified by hand against `formula_test.go`:
- `TestHealXp_LoggedCastMatchesExpectedFormula` (`formula_test.go:96-104`):
  `2880*70/70/15 = 192` — matches the brief's own worked example.
- `TestHealXp_TwoRecipientsSummedIndependently` (`formula_test.go:117-126`):
  recip1 `150*70/70/15=10`, recip2 `200*70/35/15=26` (floor), sum `36`. Hand
  check: `150*70=10500/70=150/15=10`; `200*70=14000/35=400/15=26.67→26`. Both
  correct; sum matches the asserted `36`.
- `TestApply_NotZombified_HealsEveryRecipient` (`heal_apply_test.go:172`,
  expects `Amount = 12`): caster level 30, `perTarget = floor(240/4) = 60`
  (partyTargets = 4, base = 240 per `HealAmount`'s doc comment), three
  non-caster recipients each contribute `60*30/30/15 = 4`, sum `12`. This is
  a real behavior change from the pre-fix value `24`, and I reconstructed why:
  the old formula summed **all four** recipients including the caster
  (`240/10*1 = 24`, skill level 1 from `castInfo`); this is direct evidence
  the test was re-derived from the new formula rather than rubber-stamped to
  whatever the code produced.

### 2. Seven `## Expected` consequences — six directly asserted, one only structural (PASS with a gap noted)

| Consequence | Test | Status |
|---|---|---|
| Self-heal awards nothing | `TestHealXp_CasterExcludedFromSum` (`formula_test.go:60-67`) | asserted |
| Non-party recipient awards nothing | none (brief says "already true", not a new assertion) | pre-existing, not re-verified here — acceptable per brief |
| Full-HP member awards nothing | `TestHealXp_FullHpMemberYieldsZero` (`formula_test.go:69-75`) | asserted |
| Overheal excluded | `TestHealXp_OverhealClampedToMissingHp` (`formula_test.go:77-85`) | asserted, and hand-checked: `100*70/70/15=6` |
| Per-recipient independence | `TestHealXp_TwoRecipientsSummedIndependently` (`formula_test.go:117-126`) | asserted |
| Lower-level target yields more | `TestHealXp_LowerLevelTargetYieldsMoreThanHigherLevel` (`formula_test.go:107-115`) | asserted |
| No skill-level dependence, no cap | not directly asserted | **gap** — see below |

`HealXp`'s signature no longer takes a skill level at all (`casterLevel byte`
replaces `skillLevel byte`, `formula.go:105`), and `heal.go:220` now passes
`c.Level()` where it used to pass `info.SkillLevel()`, so skill-level
independence is structurally guaranteed by the type signature, not merely
untested. There is no cap anywhere in `HealXp` or `appliedPerRecipient`, so
"no cap" is also structurally true. Neither is pinned by an explicit
regression test (e.g., two casts at different `info.SkillLevel()` values
with identical HP/levels asserting equal XP). This is a real gap against the
brief's literal instruction that these are "assertions for the regression
suite," but it is non-blocking: the removal of the `skillLevel` parameter
from the function signature makes the "no dependence" property much harder to
silently regress than a mere missing test would suggest.

### 3. `PartyRecipient.Level` — all three build sites populated, no silent zero (PASS)

`services/.../skill/handler/recipients.go`:
- `SelectDeadInRangeMapPlayers` — `SetLevel(mc.Level())` before `Build()`
  (diff hunk, `:174`).
- `SelectAllCharactersInMap` — same (`:222`).
- `selectPartyMembers` — same (`:303`).

Confirmed via `grep -n "PartyRecipientBuilder\|NewPartyRecipientBuilder"` that
these are the only three call sites constructing `PartyRecipient`, all three
now call `SetLevel`. `PartyRecipient.Level()` (new accessor,
`recipients.go:39`) is additive-only: `grep -rl PartyRecipient
skill/handler/` shows `dispel`, `echoofhero`, `healdispel`, `resurrection`,
`timeleap` reference the type, but none of them call `.Level()` — so no
existing consumer's behavior changes, matching the brief's "additive"
claim. `heal/recipients.go:21` (`selectRecipients`) is the only new
consumer, and it carries `Level: p.Level()` through correctly.

Tests added: `TestSelectPartyMembersInMap_PopulatesLevel`
(`recipients_test.go`), plus `Level()` assertions folded into
`TestSelectAllCharactersInMap` (`recipients_map_test.go:60-62`) and
`TestSelectDeadInRangeMapPlayers_AllDeadRegardlessOfParty`
(`recipients_test.go:373-375`). All three selector functions are covered.

### 4. OQ-1 gate deletion — genuinely dead, `!zombified` skip intact (PASS)

Old gate (`heal.go`, pre-diff): `if !zombified && (len(recipients) != 1 ||
len(info.AffectedMobIds()) != 0) { xp := HealXp(perTarget, recipients,
info.SkillLevel()) ... }`.

New: `if !zombified { xp := HealXp(perTarget, recipients, c.Level()) ... }`
(`heal.go:220-221`).

`grep -n AffectedMobIds` across the `heal` package after the change returns
only the test-helper `castInfo(mobIds []uint32)` constructor
(`heal_apply_test.go:39`) — `info.AffectedMobIds()` is not read anywhere in
production code post-diff. The gate's second disjunct
(`len(info.AffectedMobIds()) != 0`) previously let XP through for a
**solo** cast (`len(recipients) == 1`) when mobs were affected — a branch
with no corresponding logic anywhere in `HealXp`/`healDelta` even before this
diff (no mob-count or damage-formula path exists in this handler). I checked
`git diff --stat` for `heal_apply_test.go` in this commit: it only changes 10
lines (level additions + the `24→12` amount), no test for that mob-affected
solo-cast branch existed before or after, so its removal cannot have broken
a pinned assertion — confirmed by `go test ./skill/...` passing (see below).

The `!zombified` skip survives unchanged and is still asserted: `go test`
`TestApply_ZombifiedCaster_DamagesEveryRecipient`
(`heal_apply_test.go:184-215`) asserts `len(*xpCalls) != 0 → error, want 0`
under `casterZombifiedFunc` returning `true`, unchanged by this commit.

## Build/test verification

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./skill/...
```
All packages `ok`, including `atlas-channel/skill/handler/heal` (module-local,
not the full `tools/verify.sh` gate — this review does not claim that gate
was run).

## Not evaluable

- `docs/tasks/task-256-zombify-healing-consequences/report-bug-heal-party-xp-magnitude.md`
  does not exist in the worktree, so the implementer's own report and its
  claims could not be cross-checked; this review is based solely on the diff,
  the brief, and hand-verified arithmetic.

## Summary

The commit implements the specified formula exactly, per-recipient, with the
correct truncation order, and the worked example from the bug report
(`192` for the logged 2880/70/70 cast) is directly pinned by a test. The
`PartyRecipient.Level` plumbing is complete across all three build sites and
additive to other consumers. The OQ-1 gate removal is verified dead — no
existing test pinned its mob-affected-solo-cast branch, and the
`!zombified` skip it was adjacent to is preserved and still tested. The one
gap is non-blocking: "no skill-level dependence" and "no cap" are true by
construction (parameter rename, absence of any capping code) but not backed
by an explicit regression test as the brief's `## Fix` table implies they
should be.
