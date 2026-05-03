# Priest Doom (Skill 2311005) — Quick Context

Reference card for executing agents picking up `plan.md`. Read alongside `prd.md` and `design.md`.

## What this task does (one paragraph)

Closes the last gaps so a Priest casting Doom (skill 2311005) reliably polymorphs every legal target — including element-resistant mobs — into snails for the skill's duration, while bosses stay immune. All the wiring (skill grant, atlas-data effect mapping, magic-attack handler's empty-damage status branch, monster-status `DOOM` mask bit, `STATUS_APPLIED` Kafka event, channel-side `MonsterStatSet` broadcast) already exists. We add (1) an explicit DOOM short-circuit in atlas-monsters' elemental-immunity gate, (2) a Doom-gated magic-reflect probe in atlas-channel's empty-damage attack branch (via a fresh per-`DamageInfo` helper), (3) a Doom-specific Debugf in atlas-channel's monster wrapper, and (4) unit tests across atlas-data, atlas-monsters, and atlas-channel.

## Key files (touched)

| File | Why it matters |
|---|---|
| `services/atlas-data/atlas.com/data/skill/reader.go:351-352` | Maps Priest Doom (`PriestDoomId`) to `MonsterStatus[StatusDoom]=1`. Plan adds a reader test (no production change). |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1083-1131` | `ApplyStatusEffect` calls `isElementallyImmune` and `isBossAllowedStatus`. Plan adds a DOOM short-circuit at the top of `isElementallyImmune` and routes the `information.GetById` lookup through the existing `testInformationLookup` hook so tests can drive the boss/immunity branches. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` | Today exposes only skill/attack/recovery setters. Plan adds `SetBoss(bool)` and `SetResistances(map[string]string)` so tests can construct an `information.Model` with realistic boss/resistance state. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:151-216` | Per-`DamageInfo` body of `processAttack`. Plan extracts it into a `processDamageInfoEntry` helper with explicit closure deps, then adds a Doom-gated magic-reflect probe inside the helper's empty-damage branch. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go:70` | `Processor.ApplyStatus`. Plan adds a Doom-specific Debugf alongside the existing generic one. |

## Key files (read-only references)

| File | What to look for |
|---|---|
| `libs/atlas-constants/skill/constants.go:3067` | `PriestDoomId = Id(2311005)` |
| `libs/atlas-constants/monster/status.go:16` | `StatusDoom = "DOOM"` |
| `libs/atlas-packet/model/monster.go:108` | `TemporaryStatTypeDoom` mask bit; already on the wire. |
| `services/atlas-monsters/atlas.com/monsters/monster/builder.go:140-163` | `AddStatusEffect` semantics: non-VENOM statuses **replace** any existing same-type entry. This refutes the PRD's "no-op while already active" assumption — see "Realized behavior" below. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go:62-64,715` | The existing `testInformationLookup` package var is currently consulted only inside `UseBasicAttack`. Plan extends the same hook to `ApplyStatusEffect`. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1133-1148` | `isBossAllowedStatus` — DOOM is **not** in the allow list, so the boss-immunity branch already rejects it. Plan pins this with a test rather than adding code. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:153-167` | The existing empty-damage branch — runs `ApplyStatus` for any monster status, with no reflect probe. The Doom-gated probe lands here. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` | Existing pattern uses pure-helper tests (`computeReflect`, `attackKindFromAttackType`, `snapshotVenomDamagePerTick`) plus orchestration tests against the real `StatusMirror`. The new helper tests in the plan follow the same shape — closure fakes, no real session. |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:2905-2954` | `TestReader_LT_RB_Present` is the model the plan's new Doom test mirrors — small inline XML, `Read` provider, `model.CollectToMap`, then assert on `rm.Effects[0]`. |

## Architectural decisions (mirrors design.md §2)

1. **Elemental-immunity bypass:** explicit DOOM short-circuit at the top of `isElementallyImmune`. Pins intent next to overridden cases; defends against future maintainers adding `case "DOOM":` for symmetry.
2. **Reflect for empty-damage Doom:** narrow probe inside the empty-damage branch of the extracted helper, gated on the inbound status set containing `"DOOM"`. Defense-in-depth against a hostile/buggy client target list. No reflect damage emitted (Doom does no damage).
3. **Doom Debugf placement:** inline in `Processor.ApplyStatus` (atlas-channel monster wrapper), not in the handler. Single funnel every Doom apply must pass through.
4. **atlas-data test:** uses the same `Read(...)` + small inline XML pattern as `TestReader_LT_RB_Present` rather than the more invasive "call `getEffect` directly" path the design originally proposed. Same coverage, less coupling to the unexported helper's signature.
5. **Helper extraction:** function-typed deps wrapped in a small `damageInfoEntryDeps` struct (sanctioned by design §3.3 as the readable alternative to a 12-positional-parameter signature).

## Realized behavior — PRD/design departures

**Re-apply semantics for DOOM.** The PRD §4.7 expects "re-applying DOOM while DOOM is already active is a no-op." This is wrong. `ModelBuilder.AddStatusEffect` (`services/atlas-monsters/atlas.com/monsters/monster/builder.go:140-163`) *removes* the existing same-type entry and appends the new one. So a re-apply emits a second `STATUS_APPLIED` event and replaces the prior effect's `EffectId`/`ExpiresAt`. The plan's third atlas-monsters test (`TestApplyStatusEffect_Doom_ReapplyReplacesExisting`) asserts the realized refresh behavior. The design.md §6 risk note explicitly flagged this — implementer should also amend PRD §4.7 row 3 in the same commit so the documented expectation matches the test.

## Sequencing rationale

Order in `plan.md` (Tasks 1–9) is calibrated for blast radius:

1. Atlas-data test first — pins the upstream effect mapping every later step depends on.
2. Atlas-monsters builder + hook + short-circuit + tests next — independent of channel-side work.
3. Atlas-channel handler refactor (extract helper, no behavior change) before the probe — gives the build/test gate a chance to catch a bad refactor before the probe layers on top.
4. Doom probe inside the extracted helper.
5. Helper tests (cast, reflect block, multi-target spread, non-Doom guard).
6. Doom Debugf in monster wrapper (cheap; could move earlier or later without consequence).
7. Cross-service build/test + manual verification handoff.

## Build/test cadence

Per CLAUDE.md, every multi-service refactor expects a fix-and-rebuild cycle. None of this task's changes touch shared types, so a single pass of:

```
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
( cd services/atlas-data/atlas.com/data && go build ./... && go test ./... )
```

…should suffice at task end. Plan also bakes per-task build/test gates after each refactor or change.

## Out of scope (PRD §2 non-goals — do not expand)

- Other Priest skills (Mystic Door, Holy Symbol, Summon Dragon, Dispel).
- Refactoring the magic-attack handler beyond what Doom needs (the per-`DamageInfo` extract is the maximum).
- Server-side polymorph entity swap; v83 client handles snail rendering off the mask bit.
- Server-side elemental damage recomputation while a mob is Doomed; client computes damage and sends it.
- XP attribution changes; Doom does no damage.
- New Kafka topic, event type, HTTP route, or `libs/atlas-constants` type.
- Solution test framework (task-042). Use the per-package unit-test pattern that `services/atlas-channel/.../skill/handler/heal/` follows.

## After implementation

When `plan.md` is fully checked off, hand back to the user with:

- A summary of which commits landed.
- A note that the manual verification checklist (Task 9 step 3) is the only acceptance criterion that cannot be discharged by the implementer alone.
- The reminder that PRD §4.7 row 3 likely needs amending to match the realized refresh semantics, if the implementer did not already update it during Task 4.
