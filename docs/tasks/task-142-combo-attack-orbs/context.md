# Combo Attack Orbs — Implementation Context

Task: task-142-combo-attack-orbs
Companion to: `plan.md` (implementation plan), `design.md` (approved design), `prd.md` (requirements).

## What this builds

Server-side orb truth for Crusader/Hero and Dawn Warrior Combo Attack. atlas-channel decides the change (gain 1–2 orbs / finisher reset) and emits a delta-style `UPDATE_STAT_VALUE` Kafka command; atlas-buffs mutates the COMBO stat amount on the stored buff (clamped, atomic per character via the characterId partition key) and emits `STAT_UPDATED`; atlas-channel re-announces through the existing GIVE_BUFF / GIVE_FOREIGN_BUFF writers. No new packets, REST, DB, or atlas-data changes; works on all tenant versions because only existing writers carry the value.

**Version set (post-rebase onto `main`):** GMS v12/v48/v61/v72/v79/v83/v84/v87/v92/v95 + JMS v185 (`deploy/k8s/base/versions.json`). The pre-Big-Bang legacy columns (v12/v48/v61/v72/v79) were added to `main` after the docs' first draft and are in scope — no code change absorbs them: COMBO is version-independent (bit21 of `mask.L`, read by all clients incl. v48, IDA-verified `legacyGmsMask`), and the channel-side gate ("owns Combo Attack level > 0") is a safe no-op where the class/skill doesn't exist (e.g. Dawn Warrior pre-Cygnus). **Unverified for legacy: client orb-render** (only v83 reverse-verified) — verify in-game per legacy column or record as deferred. See PRD §8.

## Key decisions (from design.md — do not relitigate)

- **Delta command, mutation inside atlas-buffs** (design Q1). The channel never reads orb state; concurrency is handled by Kafka partition ordering (`key = characterId`) + registry clamp. Value must never exceed cap or go below 1.
- **COMBO stat value = orb count + 1**; cast already applies value 1 via atlas-data (untouched). Finisher consume = `SET 1`, emitted unconditionally (no hit requirement, no orb-count requirement, attack never rejected).
- **Gain gate order** (channel side): owns Combo Attack at level > 0 → finisher? SET and stop → Shout (1111008) or zero DamageInfo entries → stop → INCREMENT by 1 (or 2 on Advanced Combo `prop` roll) with `cap = x + 1` from the governing effect (Advanced Combo's effect when learned, else Combo Attack's). Normal attacks (skillId 0) DO gain. Melee attacks only.
- **"Is the buff active" is delegated to atlas-buffs**: missing/expired buff = Debug-level no-op (the buff can lapse between attack and command processing). Combo-capable characters emit speculative commands; that's accepted.
- **New event type STAT_UPDATED (not re-emitted APPLIED)** so atlas-effective-stats / atlas-mounts / atlas-rates ignore it via their existing type guards — zero changes in those services.
- **Original createdAt/expiresAt travel in the event**; the packet layer encodes duration relative to `now` — modern path `character_temporary_stat.go:666`, legacy path `legacyDurationUnits :688`, both relative to `time.Now()` — so re-broadcast carries remaining duration and the buff never extends on every version. (Design cited `:623` at draft; the file grew on `main`.)
- **GM max-level substitution: not replicated** (design Q4).
- Out of scope: Aran combo/SHOW_COMBO, Energy Charge, server-side damage scaling, finisher rejection, accumulate-mode stat updates, `isComboReset` (Aran-only).

## Key files

### atlas-buffs (`services/atlas-buffs/atlas.com/buffs/`)
| File | Role |
|---|---|
| `kafka/message/character/kafka.go` | Wire contract owner: `UPDATE_STAT_VALUE` command + `STAT_UPDATED` event, operation constants |
| `buff/model.go` | Immutable buff model; gains `WithStatAmount` copy-mutator |
| `character/registry.go` | Redis-backed state; gains `UpdateStatValue` (get-modify-put like `Cancel`, `srcKey` only) |
| `character/processor.go` | Processor interface + impl; gains `UpdateStatValue` emitting via `message.Emit` |
| `character/producer.go` | Gains `statUpdatedStatusEventProvider` (mirror of applied, minus `FromId`) |
| `kafka/consumer/character/consumer.go` | Gains `handleUpdateStatValue` registration |

### atlas-channel (`services/atlas-channel/atlas.com/channel/`)
| File | Role |
|---|---|
| `kafka/message/buff/kafka.go` | Channel-side contract mirror (byte-identical JSON, pinned by canonical tests both sides) |
| `character/buff/producer.go` / `processor.go` | `UpdateStatValueCommandProvider` + `Processor.UpdateStatValue` (key = characterId) |
| `socket/handler/character_attack_combo.go` (new) | `comboLine`, `comboSkillIds`, `isComboFinisher`, `comboGainAmount`, `comboOrbDeps`, `comboOrbProductionDeps`, `comboOrbTryUpdate` |
| `socket/handler/character_attack_common.go` | TODO at line ~500 (was ~404 at draft; find by marker `// TODO apply combo orbs (add or consume)`) replaced with melee-gated `comboOrbTryUpdate` call (fire-and-forget, post-broadcast) |
| `kafka/consumer/buff/consumer.go` | Shared `announceBuffGive` helper (extracted from APPLIED handler) + `handleStatusEventStatUpdated` |

## Verified facts (planning-time, cite-checked)

- Skill constants all exist in `libs/atlas-constants/skill/constants.go` (`:2942/2948/2952` adventurer, `:3308/3312` Cygnus on current `main`; find by name — line refs drift). Exact names in plan Global Constraints.
- `character.TemporaryStatTypeCombo = "COMBO"` at `libs/atlas-constants/character/temporary_stat.go:27`.
- `effect.Model.X() int16` / `Prop() float64`; atlas-data normalizes `prop` to `[0,1]` at parse time, so the roll is `roll < prop` (same as `mpEaterShouldProc`).
- `data/skill` `GetEffect(uniqueId uint32, level byte) (effect.Model, error)`.
- `processAttack` already fetches the character with `SkillModelDecorator` (`character_attack_common.go:272`) — skill levels are free.
- **No mocks implement either Processor interface** (searched: no `buff.Processor` fakes in channel tests; no atlas-buffs character Processor mock). Design §4.6's "update mocks" is a no-op; interface additions only touch the two `ProcessorImpl`s.
- No identifier collisions for any planned name (grepped both services).
- Test construction patterns: `skill.Extract(skill.RestModel{...})`, `effect.Extract(effect.RestModel{...})` (see `newDoomEffect` in `skill/handler/common_apply_to_mobs_test.go`), `character.NewModelBuilder().SetId(...).SetSkills(...).MustBuild()` (Build requires non-zero id), `packetmodel.NewAttackInfo(type).SetSkillId(...).AddDamageInfo(*packetmodel.NewDamageInfo(hits))`, miniredis via existing `setupTestRegistry`/`setupProcessorTest` helpers.
- atlas-buffs processor tests call emitting methods with `_ =` (no broker in tests) and assert against the registry — follow that convention.
- kafka_test.go convention: canonical-JSON literal shared verbatim across services keeps re-declared contracts byte-identical (existing APPLY/atlas-summons precedent).

## Dependencies / order

Tasks 1→5 (atlas-buffs, bottom-up), 6→10 (atlas-channel), 11 (verification). Task 6 depends on Task 1 only for the canonical literal string; the services compile independently. Task 9 needs Tasks 7+8. Task 10 needs Task 6.

## Gotchas for implementers

- Run go commands from the module roots (`services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel`), not the repo root.
- `registry.go`, `processor.go`, `producer.go`, and their tests are all package `character` — the `character2` alias for `atlas-buffs/kafka/message/character` must be added per-file where used.
- `capValue` (not `cap`) as the parameter name — avoids shadowing the builtin.
- Registry addresses `srcKey(sourceId)` only; accumulate-mode (`sourceId:statType`) buffs are intentionally NOT addressable.
- INCREMENT no-ops when `current >= capValue` (never decreases toward a smaller cap); SET no-ops when `amount < 1` or unchanged.
- `tools/redis-key-guard.sh` runs from the worktree root WITHOUT `GOWORK=off` (false-FAIL otherwise).
- `docker buildx bake atlas-channel atlas-buffs` from the worktree root is mandatory before calling the branch done.
- Code review (`superpowers:requesting-code-review`) before any PR.
- In-game v83 acceptance (orb gain / double proc / finisher consume / foreign visibility / no duration extension) is user-assisted, post-deploy.
