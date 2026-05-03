# Task-049 MP Eater — Execution Context

This file is a quick-reference companion to `plan.md`. It points executing
agents at the precise files, types, and conventions they need.

Source documents:
- PRD — `docs/tasks/task-049-mp-eater/prd.md`
- Design — `docs/tasks/task-049-mp-eater/design.md`
- Plan — `docs/tasks/task-049-mp-eater/plan.md`

---

## High-level shape

End-to-end flow (decided in design §2.1):

1. atlas-channel `processAttack` → per-damage-entry call to `mpEaterTryProc`.
2. `mpEaterTryProc` resolves the MP Eater skill id from `c.JobId()`, checks the
   caster owns the skill and has level > 0, fetches the skill effect (`Prop`, `X`),
   takes a monster snapshot via `monster.Processor.GetById`, gates on
   `MaxMp > 0` / `Mp > 0`, rolls `rand.Float64()` against `Prop`, computes
   `amount = MaxMp * X / 100`, and emits a `DRAIN_MP` command on
   `COMMAND_TOPIC_MONSTER`.
3. atlas-monsters consumes `DRAIN_MP`, re-checks guards (boss flag from
   `monster.information`, `MaxMp > 0`, `Mp > 0`), `DeductMp(t, uniqueId, amount)`,
   computes the actual drained delta, and emits `MP_CHANGED` on
   `EVENT_TOPIC_MONSTER_STATUS` with `Reason = "MP_EATER"`.
4. atlas-channel consumes `MP_CHANGED`, refunds caster MP via
   `character.Processor.ChangeMP(field, characterId, +int16(amount))`, and
   broadcasts `CharacterSkillSpecialEffectBody(skillId)` to the caster +
   `CharacterSkillSpecialEffectForeignBody(characterId, skillId)` to the rest of
   the map.

Authority split: atlas-monsters is the source of truth for `Boss`, current
`Mp`, and the clamp. atlas-channel pre-screens for cheap rejects and computes
the drain amount from its `MaxMp` snapshot.

---

## Key files

### atlas-channel

| Path | What is there now | What changes |
|---|---|---|
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` | `Model` struct holds `prop float64` and `x int16` (private fields, no accessors) | Add `Prop() float64` and `X() int16` accessors. |
| `services/atlas-channel/atlas.com/channel/monster/model.go` | `Model` carries `maxHp`, `hp`, `mp` but **not** `maxMp`. `MaxHp()` / `Mp()` accessors exist. | Add `maxMp uint32` field + `MaxMp() uint32` accessor. |
| `services/atlas-channel/atlas.com/channel/monster/builder.go` | `modelBuilder` mirrors the model's fields; `SetMaxHp`, `SetMp` exist. | Add `maxMp` field, `SetMaxMp(uint32)`, propagate through `Build()` and `CloneModel()`. |
| `services/atlas-channel/atlas.com/channel/monster/rest.go` | `RestModel.MaxMp uint32` already present (line 30). `Extract` populates `maxHp`, `mp` but **not** `maxMp`. | Wire `maxMp: m.MaxMp` in `Extract`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` | `AnnounceSkillUse`, `AnnounceForeignSkillUse` curried helpers using `charcb.CharacterEffectWriter` / `charcb.CharacterEffectForeignWriter`. | Add parallel `AnnounceSkillSpecial` / `AnnounceForeignSkillSpecial` using the same writer names but `charpkt.CharacterSkillSpecialEffectBody` / `CharacterSkillSpecialEffectForeignBody` payloads. |
| `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` | `EnvCommandTopic`, `CommandTypeDamage`, `EventStatusDamageReflected`, etc. plus `Command[E]` and `StatusEvent[E]` envelopes. | Append `CommandTypeDrainMp = "DRAIN_MP"`, `EventStatusMpChanged = "MP_CHANGED"`, `MpChangeReasonMpEater = "MP_EATER"`, plus `DrainMpCommandBody` and `StatusEventMpChangedBody` types. |
| `services/atlas-channel/atlas.com/channel/monster/producer.go` | `DamageCommandProvider` keyed by `monsterId`, status-event providers like `DamageReflectedStatusEventProvider`. | Append `DrainMpCommandProvider`. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | `Processor.Damage`, `Processor.EmitDamageReflected`, all going through `producer.ProviderImpl`. | Append `Processor.DrainMp(f, monsterId, characterId, skillId, amount)`. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` | Consumer for `EVENT_TOPIC_MONSTER_STATUS` with handlers like `handleDamageReflected` (line 492). `InitHandlers` registers each. | Add `handleStatusEventMpChanged` and register it in `InitHandlers`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | `processAttack` orchestrates magic attack flow. Per-monster loop is `for _, di := range ai.DamageInfo()` ending around line 215. `// TODO Apply MPEater` is at line 278 (outside the per-target loop). | Add three top-level helpers + `mpEaterTryProc` orchestrator. Call orchestrator inside the per-damage-entry loop after the `ApplyStatus` block. Remove the `// TODO Apply MPEater` comment. |

### atlas-monsters

| Path | What is there now | What changes |
|---|---|---|
| `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` | Wire `command[E]` envelope (note: lowercase `command`, not exported). Constants like `CommandTypeDamage`. | Append `CommandTypeDrainMp = "DRAIN_MP"`, `drainMpCommandBody` struct. |
| `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` | `InitHandlers` registers each command handler via `rf(t, message.AdaptHandler(message.PersistentConfig(...)))`. Existing handlers: `handleDamageCommand`, `handleDamageFriendlyCommand`, etc. | Append `handleDrainMpCommand` and register it in `InitHandlers`. |
| `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` | Status-event constants: `EnvEventTopicMonsterStatus`, `EventMonsterStatusDamaged`, `EventMonsterStatusDamageReflected`, etc. | Append `EventMonsterStatusMpChanged = "MP_CHANGED"`, `MpChangeReasonMpEater = "MP_EATER"`, plus `statusEventMpChangedBody` struct. |
| `services/atlas-monsters/atlas.com/monsters/monster/producer.go` | Status-event providers like `damagedStatusEventProvider`, `damageReflectedEventProvider` using `statusEventProvider[E]`. | Append `mpChangedStatusEventProvider(m Model, characterId, skillId uint32, reason string, amount uint32)`. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | `Processor` interface, `ProcessorImpl` with `Damage`, `UseSkill`. `GetMonsterRegistry().DeductMp(p.t, uniqueId, amount)` exists (registry.go:676). `information.GetById(p.l)(p.ctx)(monsterId)` returns `Model` whose `Boss()` accessor reads `m.boss`. | Add `DrainMp` to `Processor` interface and implement on `ProcessorImpl`. |

### Shared libraries (no code changes — read-only references)

| Path | Why we read it |
|---|---|
| `libs/atlas-constants/skill/constants.go` | Holds `FirePoisionWizardMpEaterId = Id(2100000)`, `IceLightningWizardMpEaterId = Id(2200000)`, `ClericMpEaterId = Id(2300000)`. The skill-id-to-Skill registry is `var Skills = map[Id]Skill{ ... }` (line 2358). Use `skill.Skills` for membership checks (the design doc calls it `skill.Registry` — that name does not exist in this codebase; use `skill.Skills`). |
| `libs/atlas-constants/job/constants.go` | Job ids: `MagicianId = Id(200)` (no MP Eater), `FirePoisonWizardId = Id(210)`, `FirePoisonMagicianId = Id(211)`, `FirePoisonArchMagicianId = Id(212)`, `IceLightningWizardId = Id(220)`, `IceLightningMagicianId = Id(221)`, `IceLightningArchMagicianId = Id(222)`, `ClericId = Id(230)`, `PriestId = Id(231)`, `BishopId = Id(232)`. `job.Id` is `uint16`. |
| `libs/atlas-packet/character/effect_body.go` | `CharacterSkillSpecialEffectBody(skillId)` (line 122) and `CharacterSkillSpecialEffectForeignBody(characterId, skillId)` (line 128). Mode: `CharacterEffectSkillSpecial = "SKILL_SPECIAL"`. |
| `libs/atlas-packet/character/clientbound` package (`charcb`) | Exposes `CharacterEffectWriter` and `CharacterEffectForeignWriter` writer names. Already used by `AnnounceSkillUse`. |

---

## Cross-service contract

The wire shape is identical on both sides. Field set:

```json
{
  "characterId": 12345,
  "skillId": 2300000,
  "amount": 73
}
```

Status-event body for `MP_CHANGED`:

```json
{
  "characterId": 12345,
  "skillId": 2300000,
  "reason": "MP_EATER",
  "amount": 73,
  "monsterMpAfter": 427
}
```

Both services must use the same JSON tags. The atlas-channel kafka message
package uses exported types (`Command[E]`, `DrainMpCommandBody`, etc.); the
atlas-monsters consumer package uses unexported types (`command[E]`,
`drainMpCommandBody`). This asymmetry is the existing convention — preserve
it.

---

## Conventions and gotchas

- **Skill registry symbol**: `skill.Skills` (not `skill.Registry`). Design doc
  is wrong on this point; the code uses `Skills`.
- **`job.Id` is `uint16`**: cast to `uint32` for arithmetic, then cast back to
  `skill.Id` (which is `uint32`).
- **`field.Model` construction in consumers**: use
  `field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()`
  — the existing `handleDamageReflected` (consumer.go:502) is the template.
- **Tenant scope on the channel-side consumer**: gate with
  `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` before any side
  effect, mirroring `handleDamageReflected`.
- **Failure logging**: per design §5, swallow errors with `Errorf` (Kafka emit,
  registry mutation) or `Debugf` (snapshot fetch miss); never abort the attack
  pipeline.
- **No mutation of monster snapshot in atlas-channel**: the channel only reads
  `MaxMp` / `Mp` from the snapshot. atlas-monsters owns the write.
- **One MP Eater roll per monster, not per damage line**: `DamageInfo` already
  groups lines per monster, so calling `mpEaterTryProc` once at the bottom of
  the per-`di` loop is exactly one roll per monster.
- **Reflect skip**: when `reflected == true` the loop hits `continue` *before*
  `mp.Damage` and *before* the `ApplyStatus` block. MP Eater must run *after*
  `ApplyStatus`, so the natural placement (after `ApplyStatus`, before the
  `}` closing the reflect-skip path) is already on the non-reflected branch.
- **Boss check authority**: the channel does **not** read the boss flag. The
  channel's pre-screen is `MaxMp > 0 && Mp > 0`. atlas-monsters does the boss
  check via `information.GetById`.
- **`Processor` interface vs `ProcessorImpl` in atlas-monsters**: when adding a
  method, update both the interface contract (`processor.go:25-55`) and the
  implementation. Mocks for tests may need an entry too — check
  `monster/processor_test.go` for fakes if test fails to compile.

---

## Test conventions

- Pure helpers in atlas-channel: table-driven `_test.go` co-located with the
  helper.
- atlas-monsters processor tests: see `monster/processor_test.go`. The pattern
  for boss/drain skip uses an in-memory monster registry and an injectable
  `emit` field on `ProcessorImpl` (see `emitter` type at processor.go:60).
- `testInformationLookup` (processor.go:64) is the existing seam for stubbing
  `information.GetById` in unit tests — reuse it for boss-flag testing if
  available, otherwise inject via the same pattern (test-only var).

---

## Build & test commands

```bash
# atlas-channel
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...

# atlas-monsters
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./...
```

The plan calls these explicitly in the verification step at the end.
