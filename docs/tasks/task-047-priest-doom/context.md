# Priest Doom (Skill 2311005) — Quick Context

Version: v2 (revised after wrong-channel discovery)
Reference card for executing agents picking up `plan.md`. Read alongside
`prd.md`, `design.md`, and (for the wrong-channel postmortem) `postmortem.md`.

## What this task does (one paragraph)

Closes the gaps so a Priest casting Doom (skill 2311005) reliably
polymorphs every legal target into snails for the skill's duration on
the v83 client, while bosses stay immune and one Magic Rock is consumed
per cast. The cast flows through the buff (SPECIAL_MOVE) opcode →
`handler.UseSkill` → a new per-skill Doom handler that does
server-authoritative bounding-box mob selection (caster pos + facing +
skill `lt`/`rb`), capped at `mobCount`, gated per-mob by the existing
magic-reflect mirror and a `prop` RNG. The atlas-monsters elemental and
boss immunity gates (already in place from v1 work) handle the
per-target reject/accept. `itemConsume` is plumbed generically into
`UseSkill`'s cost block so it covers Doom, Mystic Door, summons, mists,
and any future `itemConsume` skill — not just Doom.

## What changed from v1

- v1 assumed Doom flowed through the magic-attack opcode; the v83
  client uses the buff opcode. See `postmortem.md`.
- v1 placed `itemConsume` in `processAttack`'s cost gate; v2 moves it
  to `handler.UseSkill`'s cost block where every `itemConsume` skill
  actually flows.
- v1 added a Doom-gated reflect probe inside `processDamageInfoEntry`'s
  empty-damage branch; v2 reverts that and re-implements the reflect
  check inside the new Doom handler instead.
- atlas-monsters' DOOM short-circuit, boss-immunity reject, refresh
  semantics, `testInformationLookup` extension, and the three Doom
  apply tests are unchanged from v1 and engage on the corrected path.
- atlas-data's reader test for the Doom effect mapping is unchanged
  from v1.

## Key files (touched, v2)

| File | Why it matters |
|---|---|
| `services/atlas-monsters/atlas.com/monsters/monster/registry.go` | New `GetMonstersInFieldRect` walk powering Doom's server-side target selection. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | New `GetInFieldRect` processor wrapper; existing DOOM short-circuit in `isElementallyImmune` stays. |
| `services/atlas-monsters/atlas.com/monsters/monster/resource.go` (or equivalent route file) | New REST endpoint exposing the rect query. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | New client wrapper that issues the rect-query GET. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | Generic `itemConsume` charge added to the existing cost block. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/` (new package) | New per-skill Doom handler + pure `calculateBoundingBox` helper. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | Revert of the v1 Doom-gated reflect probe and the `itemConsume` cost-gate addition. The `processDamageInfoEntry` extraction stays. |

## Key files (read-only references)

| File | What to look for |
|---|---|
| `libs/atlas-constants/skill/constants.go:3067` | `PriestDoomId = Id(2311005)` |
| `libs/atlas-constants/monster/status.go:16` | `StatusDoom = "DOOM"` |
| `libs/atlas-constants/monster/skill.go` | `ReflectKindMagical` for the magic-reflect probe. |
| `libs/atlas-packet/model/monster.go:108` | `TemporaryStatTypeDoom` mask bit. |
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go` | Reference shape for a per-skill handler (`init` registration, `Apply` curried signature, summary log line). |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go:19-54` | The `UseSkill` dispatcher — where `itemConsume` is plumbed, where `Lookup` finds the per-skill handler. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1107-1111` | Existing DOOM short-circuit in `isElementallyImmune`. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1117-1131` | `isBossAllowedStatus` — DOOM is **not** in the allow list, so boss reject is automatic. |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` | `LT`, `RB`, `MobCount`, `Prop`, `Duration`, `ItemConsume` accessors. |
| Cosmic `server/StatEffect.java:1180-1204` | Reference implementation: server-authoritative `applyMonsterBuff` with bounding-box selection, mob-count cap, prop chance gate. |
| Cosmic `server/StatEffect.java:1206-1218` | Reference for `calculateBoundingBox` semantics (left-facing mirror about caster X). |
| Cosmic `net/server/channel/handlers/SpecialMoveHandler.java:138` | Confirms Doom flows through the buff opcode (`effect.applyTo`). |

## Architectural decisions (mirrors design.md §2)

1. **Server-authoritative target selection**: caster's position +
   facing + skill `lt`/`rb` define the rectangle; atlas-monsters
   returns up to `mobCount` mobs in that rectangle. The v83 client
   sends no mob list for Doom (the buff packet only carries cast
   position), so the client cannot be trusted for target selection.
   Matches Cosmic.
2. **`itemConsume` placed in `UseSkill` cost block**: every
   `itemConsume` skill flows through `UseSkill` (none through
   `processAttack`), so the cost block is the single source of
   truth. Generic across Doom, Mystic Door, summons, mists.
3. **Per-mob `prop` RNG via injectable closure**: the v83 Doom data
   carries `prop = 0.52`. Mirrors Cosmic's per-mob `makeChanceResult()`
   gate. Tests inject deterministic behavior via a package-private
   `propRollFunc` variable.
4. **Per-mob magic-reflect probe in the new handler**: Doom does no
   damage, so on reflect we skip the apply (no reflect damage to
   emit). Logs `Doom: monster [%d] has MAGICAL reflect; status apply skipped.`
   for production diagnoses.
5. **Pure `calculateBoundingBox` extracted to `bbox.go`**: easy to
   unit-test in isolation against the Cosmic reference; left/right
   facing branches and asymmetric rects covered.

## Realized behavior — PRD departures

**Re-apply semantics for DOOM**: same as v1. Per
`ModelBuilder.AddStatusEffect` in atlas-monsters, a re-apply *replaces*
the existing same-type entry (refresh, not no-op). The atlas-monsters
test `TestApplyStatusEffect_Doom_ReapplyReplacesExisting` pins the
realized behavior; the PRD describes it.

**Boss reject is per-target, not per-cast**: per Cosmic, the cast itself
succeeds (charging HP / MP / `itemConsume`), and only the per-target
status apply is rejected. PRD §10 AC 5 reflects this.

**Mob ordering**: atlas-monsters' rect query returns the N-closest mobs
to the rect center (when `limit > 0`). Cosmic returns the first N in
registry iteration order. Functionally equivalent; ours is more
deterministic.

## Sequencing rationale

Order in `plan.md` (Tasks R, A, B, C, E, D, F) is calibrated for blast
radius and dependencies:

1. Task R reverts the wrong-path code first so subsequent additions
   layer onto a clean baseline.
2. Task A is leaf-most (no atlas-channel callers yet).
3. Task B wraps A.
4. Task C is independent of A/B; sequenced for review simplicity.
5. Task E is a small dependency for D (effect.Model accessors).
6. Task D depends on B and E.
7. Task F is the final cross-service gate + manual handoff.

## Build/test cadence

Same as v1: per-service `go build ./... && go test ./...` after each
task, plus a final cross-service pass in Task F. The slow atlas-monsters
suite (~190s for `monster/`) is mostly recovery/aggro tests, not Doom.

## Out of scope (PRD §2 non-goals — do not expand)

- Other Priest skills (Mystic Door, Holy Symbol, Summon Dragon, Dispel).
- Server-side polymorph entity swap.
- Server-side elemental damage recomputation while a mob is Doomed.
- XP attribution changes; Doom does no damage.
- New Kafka topic / event type / `libs/atlas-constants` type.
- Any change to `libs/atlas-packet` (the buff packet for Doom carries
  no mob list and no cast-position decoder is needed because the
  bounding box uses caster position, mirroring Cosmic).
- Solution test framework (task-042). Use the per-package unit-test
  pattern that `services/atlas-channel/.../skill/handler/heal/`
  follows.

## After implementation

When `plan.md` is fully checked off, hand back to the user with:

- A summary of which commits landed (Task R revert + Tasks A through D
  + the Task F manual checklist).
- A note that the manual verification checklist (Task F step 3) is the
  only acceptance criterion that cannot be discharged by the
  implementer alone.
- The PR description for #377 needs amending to reflect the v2
  approach. The original Task 10 generic-`itemConsume` claim should be
  replaced with the corrected v2 placement (`UseSkill` cost block), and
  the wrong-channel discovery should be linked to `postmortem.md`.
