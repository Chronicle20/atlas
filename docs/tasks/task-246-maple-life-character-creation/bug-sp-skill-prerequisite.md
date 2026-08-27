# bug: the opt-in Maple Life SP skill is granted without its level-5 prerequisite

Task: task-246-maple-life-character-creation
PR: atlas-pr-1466
Worktree: `.worktrees/task-246-maple-life-character-creation`

## Reproduced

Not a live defect report — a **requirement the PRD missed**, stated by the user
after live testing:

> For Warrior and Magician which have the opt-in Improved (HP/MP) Increase
> skill option. If the user chooses that, non-negotiably, they need to be
> awarded level 5 of the corresponding Improving (HP/MP) Recovery skill. With
> their SP further reduced by 5. Level 5 of each of those skills is a
> prerequisite of the 'increase' skill.

## Observed

`toPreset`
(`services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go:64-66,86`)
grants exactly one skill and spends exactly `in.SP`:

```go
if e.SpSkillId != 0 && in.SP > 0 {
    skills = []preset.SkillEntry{{SkillId: e.SpSkillId, Level: in.SP}}
}
...
remainingSP := spendSPPool(e.SP, in.SP)
```

A Warrior who picks 4 into Improved MaxHP Increase (1000001) is created with
that skill at level 4, **no** Improved HP Recovery (1000000), and `61 − 4 = 57`
SP left in slot 0. The character therefore holds a skill whose prerequisite it
does not meet.

## Expected

When `e.SpSkillId != 0 && in.SP > 0`, additionally grant **level 5** of the
prerequisite recovery skill and spend **5 more** SP:

| class | SP skill (`e.SpSkillId`) | prerequisite to grant at level 5 |
|---|---|---|
| Warrior (ordinal 0) | `skill.WarriorImprovedMaxHpIncreaseId` = 1000001 | `skill.WarriorImprovedHpRecoveryId` = 1000000 |
| Magician (ordinal 1) | `skill.MagicianImprovedMaxMpIncreaseId` = 2000001 | `skill.MagicianImprovedMpRecoveryId` = 2000000 |

So a Warrior picking 4 ends with `skills = [{1000000, 5}, {1000001, 4}]` and
`sp = "52,0,…"` (61 − 4 − 5). Picking 0 grants neither skill and spends
nothing — the flat cost applies **only** when the player opts in.

Ordinals 2/3/4 have `SpSkillId == 0` and are unaffected.

## Root cause

The design modelled the SP step as a single skill at a player-chosen level and
never modelled the client's skill-prerequisite rule. Nothing in the repo is
wrong per its own spec; the spec was incomplete.

## Ruling — where the prerequisite mapping lives

**In code, not in seed data.** `toPreset` already carries a
`switch e.SpSkillId` over exactly these two ids (`maple_life.go:78-83`) to
route the HP vs. MP stat contribution, so a second mapping keyed the same way
is consistent with the existing shape. The alternative — a
`spPrereqSkillId` sibling field on `ClassEntry` — would require editing all
four templates plus the model/projection chain, and would then need a template
reseed **and** a tenant PATCH in every live environment to take effect. That
PATCH is a known landmine (see
`bug-maple-life-seed-never-reaches-live-db.md`, "Follow-on incident"). Keeping
it in code makes the fix take effect on deploy alone.

If the mapping is ever needed for a third class, revisit.

## Fix

### Files

- `services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go`
  - `toPreset` (line 64-66): build a two-entry `skills` slice — prerequisite at
    level 5 first, then `e.SpSkillId` at `in.SP` — via a small
    `prerequisiteFor(spSkillId uint32) (uint32, bool)` helper keyed on the two
    constants above. A class whose `SpSkillId` has no known prerequisite grants
    only the SP skill (do not invent one, do not error).
  - line 86: spend `in.SP + 5` from slot 0 when a prerequisite was granted;
    spend `in.SP` otherwise. `spendSPPool` takes a `byte` today — widen its
    parameter or pass an int, whichever keeps it total; it must not overflow.
  - The `stats.Hp`/`stats.Mp` contribution at lines 76-84 is **unchanged**: the
    recovery skills affect regeneration, not max HP/MP. Do not add an
    `effectX` lookup for the prerequisite.
- `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:431`
  - the pool-sufficiency guard is `int(in.SP) > pool[0]`; it must now account
    for the extra 5 when a prerequisite applies. With the seeded pool of 61 and
    `in.SP ≤ 10` this can never trip, but the guard must stay honest.
- Tests in `factory/maple_life_test.go` (and `processor` tests if the guard
  changes shape): a Warrior with `SP=4` yields both skills and `"52,0,…"`;
  a Warrior with `SP=0` yields no skills and `"61,0,…"`; a Magician with
  `SP=10` yields `[{2000000,5},{2000001,10}]` and `"46,0,…"`; a Bowman
  (`SpSkillId == 0`) is untouched. Use the project's Builder pattern.

No seed-data, template, packet, or Kafka-contract change. No client-side
change: the client's own preview is not driven by this value.

### Verification

Module-local `go build ./...` and `go test ./...` in
`services/atlas-character-factory/atlas.com/character-factory`. The repo-wide
gate runs separately.

## Not yet answered

- The client's pre-submit dialog shows the SP it *thinks* remains. Whether it
  subtracts the prerequisite's 5 is unknown — the preview strings were never
  decoded (see `bug-ap-sp-and-starting-equipment.md`, "Not yet answered"). If
  the dialog turns out to promise `61 − nSP`, the server will now be 5 lower
  than the promise and the user must rule on which wins. Flagged, not blocking.

## Resolution

(pending)
