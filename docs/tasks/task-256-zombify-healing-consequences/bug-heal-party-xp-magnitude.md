# bug-heal-party-xp-magnitude

Task: task-256-zombify-healing-consequences
Environment: `atlas-pr-1449` (PR #1449), tenant `5ae4fd69-6971-4c88-aa5c-d55a8c861cd2`, GMS **83.1**
Reported: 2026-08-26 — "healing party member exp seems WAY high"

## Reproduced

Yes, from live `atlas-channel` logs in `atlas-pr-1449`. Cleric/Priest character 3
spamming Heal in map 240011000 with one in-range party member:

```
2026-08-26T18:31:04.667Z  Heal: caster=[3] level=[30] recipients=[2] perTarget=[2880] zombified=[false].
2026-08-26T18:31:07.498Z  Heal: caster=[3] level=[30] recipients=[2] perTarget=[2809] zombified=[false].
2026-08-26T18:31:34.562Z  Heal: caster=[3] level=[30] recipients=[2] perTarget=[2906] zombified=[false].
```

`level=[30]` in that line is `info.SkillLevel()` (Heal's master level is 30), and
`recipients=[2]` is caster + one party member.

## Observed

`HealXp` (`services/atlas-channel/atlas.com/channel/skill/handler/heal/formula.go:98-110`):

```go
xp := total / 10 * int64(skillLevel)
```

where `total` is the sum of `appliedPerRecipient` across all recipients.

With the logged cast (`perTarget = 2880`, 2 recipients, both missing at least
that much HP, `skillLevel = 30`):

```
total = 2880 + 2880 = 5760
xp    = 5760 / 10 * 30 = 17_280
```

Casts land roughly every 1.4 s in the log above, so a Priest standing next to one
damaged party member earns on the order of 10^4 EXP per second. There is no cap,
no cooldown on the award, and no scaling against caster level or monster EXP.

## Expected

Per recipient **other than the caster**, and only for HP that was actually
restored:

```
actualHeal = min(calculatedHeal, target.MaxHp - target.Hp)
healExp    = floor(actualHeal * healer.Level / target.Level / 15)
totalExp   = Σ healExp over the non-caster recipients
```

Consequences this formula must produce, all of which are assertions for the
regression suite:

- Self-heal awards nothing — the caster never contributes to the sum.
- A recipient outside the caster's party awards nothing (already true: the
  recipient list is caster + bitmap-selected party members only).
- A full-HP party member awards nothing (`actualHeal == 0`).
- Overheal does not count; only HP actually restored does.
- Each recipient is computed independently, because their levels differ.
- A lower-level target yields more EXP per HP; a higher-level target less.
- No dependence on the skill level, and no cap.

Applied to the logged cast (`perTarget = 2880`, one party member, both level 70):
`2880 * 70 / 70 / 15 = 192` EXP, against the current `17_280`.

## Root cause

Established, but it is a **specification** defect, not a coding defect.

1. The `total/10*skillLevel` formula was introduced in `e48d63b88`
   ("feat(atlas-channel): heal formula + xp math (pure)") on the
   `task-045-cleric-heal-mechanics` branch (PRs #368/#369). That task's
   `docs/tasks/task-045-cleric-heal-mechanics/{prd,design,plan}.md` **never landed** —
   `git log --all --diff-filter=A -- 'docs/tasks/task-045*'` returns only the
   unrelated `task-045-pr-teardown-leak-fixes` folder, whose number collides. The
   PR body for #368 refers to an "XP grant gated per OQ-1" in a design doc that no
   longer exists in the repository. **The formula has no recorded derivation.**
2. The repository's reference tree (`Cosmic`, `~/source/Cosmic`) grants **no EXP at
   all** for Cleric Heal. `StatEffect.applyTo` (`src/main/java/server/StatEffect.java:930`)
   and `StatEffect.applyBuff` (`:1147`) contain no `gainExp` call, and
   `AbstractDealDamageHandler`'s two `Cleric.HEAL` branches (`:181`, `:663`) are
   mob-count and damage-formula special cases only. A repo-wide grep for a heal
   EXP award in the reference returns nothing.

So the multiplier by `skillLevel` — which triples the award between level 10 and
level 30 on top of an already-linear dependence on healed HP — is an unsourced
invention. That is what makes the number "WAY high".

**Not caused by task-256.** `e48d63b88` is an ancestor of this branch's base
`1461bfc96` (`git merge-base --is-ancestor` exits 0). task-256 only added the
`!zombified &&` guard in front of the existing XP block
(`services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go:216`).

## Fix

The formula needs the **caster's character level** and **each recipient's
character level**, neither of which reaches `HealXp` today. Both are already
loaded — no new RPC is required:

- The caster's `character.Model` is loaded at `heal.go:150` and its `Level()` is
  already used for the cast broadcast (`heal.go:229`).
- `selectPartyMembers` (`skill/handler/recipients.go:236`) loads each member's
  full `character.Model` as `mc` and then discards everything except
  id/x/y/hp/maxHp when it builds the `PartyRecipient`. `mc.Level()` is there for
  free.

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` | Add a `level byte` field to `PartyRecipient`, a `Level() byte` accessor, and `SetLevel` on `PartyRecipientBuilder`. Populate it from `mc.Level()` at all three build sites: `selectPartyMembers` (`:290`), `SelectAllCharactersInMap` (`:212`), `SelectDeadInRangeMapPlayers` (`:160`). |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/formula.go` | Add `Level byte` to the `recipient` struct. Rewrite `HealXp` to take the caster level instead of the skill level and sum `appliedPerRecipient(perTarget, r) * casterLevel / r.Level / 15` over recipients with `!r.IsCaster`. Skip any recipient whose `Level` is 0 (guards the division; a level-0 character is not a real state, but the field is a `byte` and an unpopulated one would panic). Accumulate in `int64`; the existing `xp < 0 → 0` guard stays. |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/recipients.go` | Carry `Level: p.Level()` through `selectRecipients`. The caster's `recipient` literal in `heal.go` gets `Level: c.Level()` for symmetry even though it is excluded from the sum. |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go` | Pass `c.Level()` to `HealXp` in place of `info.SkillLevel()` (`:217`). Drop the now-dead OQ-1 gate `len(recipients) != 1 \|\| len(info.AffectedMobIds()) != 0` — a solo cast now yields 0 from the formula itself, so the gate is redundant. Keep the `!zombified` skip: a negated cast heals nobody. |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/formula_test.go` | Re-pin `HealXp`: caster excluded; full-HP member yields 0; overheal clamped to missing HP; lower-level target yields more than a higher-level one for the same HP; two members summed independently; level-0 recipient skipped rather than panicking. |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal_apply_test.go` | Update the award-amount assertions and any stub `PartyRecipient` construction to set a level. |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients_test.go` | Assert `Level()` is populated by the selectors. |

No other service is involved: `AwardExperience` is a fire-and-forget Kafka command
and its payload shape does not change. `PartyRecipient` is consumed by the other
party-skill handlers (buff, resurrection, healdispel) but only additively — no
existing accessor changes.

## Not yet answered

Nothing blocking. The formula above is the maintainer's decision, recorded
2026-08-26.

## Resolution

Fixed by `b9904f7f6` — "fix(atlas-channel): correct Heal party XP formula
magnitude". `HealXp` now takes the caster's character level in place of the skill
level and sums `appliedPerRecipient(...) * casterLevel / r.Level / 15` over
recipients with `!r.IsCaster && r.Level != 0`, accumulating in `int64`.
`PartyRecipient` gained a `level byte` field populated from the `character.Model`
the selectors already load, so no new RPC was added. The OQ-1 gate in `heal.go`
was removed as dead — with the caster excluded, a solo cast yields 0 from the
formula itself. The `!zombified` skip is unchanged.

Gate: `tools/verify.sh --quick --base 8e5df4e27` — every applicable check passes
(`go build/vet`, go analyzer guards, skill/job id, scope, producer seam, env
domain) **except** the lint & format guard, which aborts on the same
**pre-existing toolchain mismatch** recorded in
`bug-disease-command-emits-nonexistent-zombify-stat.md`: the pinned
`GOLANGCI_LINT_VERSION=v2.12.2` (`tools/lint.versions`) is built with go1.26 and
panics with `file requires newer Go version go1.27 (application built with
go1.26)` against the local go1.27.0 toolchain. Re-confirmed environment-wide, not
branch-specific: `tools/lint.sh --check --go services/atlas-buffs` — a module this
branch never touched — fails identically.

Per CLAUDE.md "Done means verified", a `--quick` run does not satisfy the gate on
its own and this run did not exit 0. This fix is **not** cleared for PR until the
golangci-lint pin is resolved and a flagless `tools/verify.sh` exits 0.

**Live re-test: not yet performed.** Re-testing needs a Cleric/Priest and a
damaged party member in range; confirm the awarded EXP against
`actualHeal * casterLevel / targetLevel / 15` per member.
