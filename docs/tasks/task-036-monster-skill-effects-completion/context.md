# task-036 — Execution Context

> Quick-reference companion to `plan.md`. Captures key file:line evidence, decisions, and dependencies an executing agent should not have to re-derive from scratch.

---

## 1. Authoritative Documents

| Doc | Path | Role |
|---|---|---|
| PRD | `docs/tasks/task-036-monster-skill-effects-completion/prd.md` | Goals, FRs (FR-4.x), acceptance criteria |
| Design | `docs/tasks/task-036-monster-skill-effects-completion/design.md` | **WINS** over PRD where they disagree (per design §1.1) |
| Data model | `docs/tasks/task-036-monster-skill-effects-completion/data-model.md` | Superseded by design §3.1 on venom slot encoding |
| API contracts | `docs/tasks/task-036-monster-skill-effects-completion/api-contracts.md` | Kafka body shapes |
| Risks | `docs/tasks/task-036-monster-skill-effects-completion/risks.md` | Edge cases + plan-phase actions |

**Critical override:** Design D3 supersedes PRD §4.4 / data-model §2-3. There are **NO** `VENOM_1`/`VENOM_2`/`VENOM_3` slot keys, no `IsVenomSlot` helper, no auxiliary `_LUCK`/`_MATK`/`_SOURCE` suffix keys. Each venom apply is its own `StatusEffect` in the existing `[]StatusEffect` slice; the codebase already supports this. The only behavioural fix in atlas-monsters for venom is the eviction policy (oldest-by-`ExpiresAt` instead of first-found).

---

## 2. Service Boundaries Touched

| Service | Path | Plan tasks |
|---|---|---|
| atlas-monsters | `services/atlas-monsters/atlas.com/monsters/` | T1, T4, T6-T9, T20-T21, T24 |
| atlas-channel | `services/atlas-channel/atlas.com/channel/` | T10-T12, T22-T23, T24, T25 |
| atlas-buffs | `services/atlas-buffs/atlas.com/buffs/` | T5 (verify only — already wired) |
| atlas-maps | `services/atlas-maps/atlas.com/maps/` | T13-T18 (new `mist` domain) |
| libs/atlas-packet | `libs/atlas-packet/field/clientbound/` | T2 |
| libs/atlas-constants | `libs/atlas-constants/monster/` | T3 |
| atlas-character | `services/atlas-character/atlas.com/character/` | T19 (verify reusing existing GET; only add if missing) |

No frontend (atlas-ui) changes.

---

## 3. Key Code Locations (file:line)

### atlas-monsters

| What | Path | Lines |
|---|---|---|
| StatusEffect struct, ctor, getters | `services/atlas-monsters/atlas.com/monsters/monster/status.go` | 14-108 |
| Source type constants | `services/atlas-monsters/atlas.com/monsters/monster/status.go` | 10-11 |
| Model.statusEffects + ApplyStatus / CancelStatus | `services/atlas-monsters/atlas.com/monsters/monster/model.go` | 33-55, 214-253 |
| Builder.AddStatusEffect (VENOM cap branch) | `services/atlas-monsters/atlas.com/monsters/monster/builder.go` | 130-156 |
| executeStatBuff already-active gate | `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | 537-543 |
| executeStatBuff body | `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | 647-689 |
| executeDebuff routing | `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | 724-753 |
| applyDiseaseCommandProvider | `services/atlas-monsters/atlas.com/monsters/monster/disease.go` | 45-63 |
| processDoTTick (POISON + VENOM iteration) | `services/atlas-monsters/atlas.com/monsters/monster/status_task.go` | 61-119 |
| Kafka event bodies | `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` | 108-132 |
| Picker AREA_POISON exclusion | `services/atlas-monsters/atlas.com/monsters/monster/picker.go` | 144-149 |
| Picker reflect/immunity already-active gate | `services/atlas-monsters/atlas.com/monsters/monster/picker.go` | 183-191 |
| MobSkill model (X, Y, Lt/Rb, Duration) | `services/atlas-monsters/atlas.com/monsters/monster/mobskill/model.go` | 1-94 |

### atlas-channel

| What | Path | Lines |
|---|---|---|
| handleStatusEffectApplied | `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` | 299-321 |
| handleStatusEffectExpired | same file | 323-345 |
| handleStatusEffectCancelled | same file | 347-369 |
| handleDamageReflected (UNCHANGED) | same file | 371-384 |
| handleStatusEventDestroyed | same file | 119-135 |
| handleStatusEventKilled | same file | 203-228 |
| processAttack (TODOs at 144-145) | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | 23-153 |
| character.Model.X()/Y() | `services/atlas-channel/atlas.com/channel/character/model.go` | 239-245 |
| monster.Processor.GetById | `services/atlas-channel/atlas.com/channel/monster/processor.go` | 27 |
| monster.Model.X()/Y() | `services/atlas-channel/atlas.com/channel/monster/model.go` | 53-59 |

### atlas-buffs

| What | Path | Lines |
|---|---|---|
| tasks.Expiration (template) | `services/atlas-buffs/atlas.com/buffs/tasks/expiration.go` | 1-33 |
| tasks.PoisonTick (already exists) | `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` | 1-31 |
| main.go task wiring (PoisonTick at line 64) | `services/atlas-buffs/atlas.com/buffs/main.go` | 41-79 |
| ProcessPoisonTicks (already exists) | `services/atlas-buffs/atlas.com/buffs/character/processor.go` | 114-157 |
| HasImmunity gate (Holy Shield) | `services/atlas-buffs/atlas.com/buffs/character/processor.go` | 43-44 |
| GetPoisonCharacters / GetLastPoisonTick / UpdatePoisonTick | `services/atlas-buffs/atlas.com/buffs/character/registry.go` | 209-251 |
| changeHPCommandProvider topic + body | `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` | 82-96 |

### atlas-maps

| What | Path | Lines |
|---|---|---|
| Reactor model (template for mist) | `services/atlas-maps/atlas.com/maps/reactor/model.go` | 1-184 |
| Reactor processor (template) | `services/atlas-maps/atlas.com/maps/reactor/processor.go` | 1-92 |
| Reactor producer (template) | `services/atlas-maps/atlas.com/maps/reactor/producer.go` | 1-34 |
| Reactor command Kafka shape | `services/atlas-maps/atlas.com/maps/kafka/message/reactor/kafka.go` | 1-34 |
| Map character membership registry | `services/atlas-maps/atlas.com/maps/map/character/registry.go` | 1-99 |

### libs/atlas-constants

| What | Path | Lines |
|---|---|---|
| field.Model (key for `(world, channel, map, instance)`) | `libs/atlas-constants/field/model.go` | 13-199 |
| Skill category constants | `libs/atlas-constants/monster/skill.go` | 6-12, 187-218 |
| Skill type → status name mapping | `libs/atlas-constants/monster/skill.go` | 61-86 |
| Reflect skill ids: 143 (`SkillTypePhysicalCounter`) → `WEAPON_COUNTER`; 144 (`SkillTypeMagicCounter`) → `MAGIC_COUNTER`; 145 (`SkillTypePhysicalMagicCounter`) | `libs/atlas-constants/monster/skill.go` | 61-86, 187-218 |
| TemporaryStatType constants | `libs/atlas-constants/monster/temporary_stat.go` | 3-44 |

### libs/atlas-packet

| What | Path | Lines |
|---|---|---|
| field/clientbound/ template (clock writer) | `libs/atlas-packet/field/clientbound/clock.go` | 1-102 |
| reactor/clientbound/spawn writer (close template) | `libs/atlas-packet/reactor/clientbound/spawn.go` | 1-65 |
| **AffectedArea writers** | `libs/atlas-packet/field/clientbound/affected_area_*.go` | **MISSING — created in T2** |

### atlas-character

| What | Path | Lines |
|---|---|---|
| Character RestModel includes X, Y, MapId, Instance | `services/atlas-character/atlas.com/character/character/rest.go` | 40-45, 109-111 |
| **Position-only endpoint** | — | **None today; T19 verifies whether to add a thin endpoint or reuse existing GET** |

---

## 4. Locked Architectural Decisions (from design)

- **D1.** Reflect math placement: atlas-channel `StatusMirror` + local computation, emits `DAMAGE_REFLECTED` to existing consumer.
- **D2.** Mist owner: atlas-maps `mist` domain (new package), modelled on `reactor/`.
- **D3.** Venom encoding: each apply = its own `StatusEffect`; eviction-policy fix only. **No slot keys.**
- **D4.** Reflect range gate: bounding box (`LtX/LtY/RbX/RbY`), reusing AoE pattern at `processor.go:683`.
- **D5.** AffectedArea writers in `libs/atlas-packet/field/clientbound/`.
- **D6.** Mist character lookup: existing `map.character` registry → per-character REST position from atlas-character, filter via `mist.Contains(x,y)`.
- **D7.** `SourceSkillClass` (`"PHYSICAL"`/`"MAGIC"`/`""`) populated by atlas-channel on `STATUS_CANCEL` command body; atlas-monsters reads it directly.
- **D8.** Immunity mutual exclusion: inline cancel-then-apply in `executeStatBuff`, two events, partition-keyed by `uniqueId`.
- **D9.** `PoisonTick` is a thin wrapper around existing `character.ProcessPoisonTicks`. **Already exists** at `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` and is wired in `main.go:64`. Plan adds tests + verifies wiring; no new task creation.

---

## 5. Resolved Open Questions (design §1.2)

- **AffectedArea writers absent** in `libs/atlas-packet/field/clientbound/` → T2 adds them.
- **Reflect mapping** → `sd.X()` is `ReflectPercent`; bounding box is the range gate; `ReflectMaxDamage` is constant `32767` (replace with `sd.<field>()` if WZ inspection during T8 reveals a per-skill cap).
- **PoisonTick character damage producer** → uses existing `EnvCommandTopicCharacter` / `CommandChangeHP` / `ChangeHPCommandBody`. No new topic.
- **Mist on instance maps** → `field.Model` already encodes `(world, channel, mapId, instance)`; `MapKey` already disambiguates instances in atlas-maps registries.

---

## 6. Reflect skill semantics (locked)

| Skill type | id | Status name (`SkillTypeToStatusName`) | `ReflectKind` |
|---|---|---|---|
| `WEAPON_REFLECT` (`SkillTypePhysicalCounter`) | 143 | `WEAPON_COUNTER` | `PHYSICAL` |
| `MAGIC_REFLECT` (`SkillTypeMagicCounter`) | 144 | `MAGIC_COUNTER` | `MAGICAL` |
| `SkillTypePhysicalMagicCounter` | 145 | (existing mapping) | both kinds (verify in T8) |

`ReflectKind` constants live in `libs/atlas-constants/monster/skill.go` (added by T3 next to the existing skill-category constants).

---

## 7. Cross-service Kafka contracts

### `EVENT_TOPIC_MONSTER_STATUS` — `StatusEffectAppliedBody` (extended)

```go
type statusEffectAppliedBody struct {
    EffectId          uuid.UUID        `json:"effectId"`
    SourceType        string           `json:"sourceType"`
    SourceCharacterId uint32           `json:"sourceCharacterId"`
    SourceSkillId     uint32           `json:"sourceSkillId"`
    SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
    Statuses          map[string]int32 `json:"statuses"`
    Duration          int64            `json:"duration"`
    TickInterval      int64            `json:"tickInterval"`
    // NEW (T7)
    ReflectKind       string           `json:"reflectKind"`       // "" | "PHYSICAL" | "MAGICAL"
    ReflectPercent    int32            `json:"reflectPercent"`
    ReflectLtX        int16            `json:"reflectLtX"`
    ReflectLtY        int16            `json:"reflectLtY"`
    ReflectRbX        int16            `json:"reflectRbX"`
    ReflectRbY        int16            `json:"reflectRbY"`
    ReflectMaxDamage  int32            `json:"reflectMaxDamage"`
}
```

No `omitempty` (cjson safety per FR-4.10).

### `EVENT_COMMAND_TOPIC_MIST` (new — T15/T16)

```go
const (
    EnvCommandTopic    = "COMMAND_TOPIC_MIST"
    CommandTypeCreate  = "CREATE"
    CommandTypeCancel  = "CANCEL"
)

type Command[E any] struct {
    Tenant   uuid.UUID  `json:"tenant"`
    Type     string     `json:"type"`
    Body     E          `json:"body"`
}

type CreateCommandBody struct {
    WorldId         byte      `json:"worldId"`
    ChannelId       byte      `json:"channelId"`
    MapId           uint32    `json:"mapId"`
    Instance        uuid.UUID `json:"instance"`
    OwnerType       string    `json:"ownerType"`
    OwnerId         uint32    `json:"ownerId"`
    OriginX         int16     `json:"originX"`
    OriginY         int16     `json:"originY"`
    LtX             int16     `json:"ltX"`
    LtY             int16     `json:"ltY"`
    RbX             int16     `json:"rbX"`
    RbY             int16     `json:"rbY"`
    Disease         string    `json:"disease"`
    DiseaseValue    int32     `json:"diseaseValue"`
    DiseaseDuration int64     `json:"diseaseDuration"`
    Duration        int64     `json:"duration"`
    TickIntervalMs  int64     `json:"tickIntervalMs"`
    SourceSkillId   uint32    `json:"sourceSkillId"`
    SourceSkillLevel uint32   `json:"sourceSkillLevel"`
}

type CancelCommandBody struct {
    MistId uuid.UUID `json:"mistId"`
}
```

### `EVENT_TOPIC_MIST` (new — T15)

```go
const (
    EnvEventTopic    = "EVENT_TOPIC_MIST"
    EventTypeCreated = "MIST_CREATED"
    EventTypeDestroyed = "MIST_DESTROYED"
)

type Event[E any] struct {
    Tenant    uuid.UUID  `json:"tenant"`
    WorldId   byte       `json:"worldId"`
    ChannelId byte       `json:"channelId"`
    MapId     uint32     `json:"mapId"`
    Instance  uuid.UUID  `json:"instance"`
    MistId    uuid.UUID  `json:"mistId"`
    Type      string     `json:"type"`
    Body      E          `json:"body"`
}

type CreatedBody struct {
    OwnerType string `json:"ownerType"`
    OwnerId   uint32 `json:"ownerId"`
    OriginX   int16  `json:"originX"`
    OriginY   int16  `json:"originY"`
    LtX       int16  `json:"ltX"`
    LtY       int16  `json:"ltY"`
    RbX       int16  `json:"rbX"`
    RbY       int16  `json:"rbY"`
    Duration  int64  `json:"duration"`
}

type DestroyedBody struct {
    Reason string `json:"reason"` // "EXPIRED" | "CANCELLED"
}
```

Disease metadata is intentionally absent from the outbound event (internal to atlas-maps tick task only).

### `STATUS_CANCEL` command body extension (T24)

```go
type statusCancelCommandBody struct {
    UniqueId         uint32 `json:"uniqueId"`
    StatusName       string `json:"statusName"`
    SourceCharacterId uint32 `json:"sourceCharacterId"`
    SourceSkillId    uint32 `json:"sourceSkillId"`
    // NEW
    SourceSkillClass string `json:"sourceSkillClass"` // "" | "PHYSICAL" | "MAGIC"
}
```

(Plan-phase action: confirm whether a `STATUS_CANCEL` command channel currently exists between atlas-channel and atlas-monsters. If it does not — and the explore in T-prep suggests it does not — T24 introduces it.)

---

## 8. Things NOT to do (anti-pattern guard)

- **Do NOT** add `VENOM_1`/`VENOM_2`/`VENOM_3` slot keys, an `IsVenomSlot` helper, or any `_LUCK`/`_MATK`/`_SOURCE` suffix keys. The codebase already supports multi-stack venom natively via `[]StatusEffect`. (Design D3, supersedes PRD §4.4.)
- **Do NOT** use a 1-D X-axis distance check for reflect. Use the bounding box per design D4.
- **Do NOT** introduce a new `STATUS_REPLACED` event for immunity mutual exclusion. Use cancel-then-apply with two existing events. (Design D8.)
- **Do NOT** re-derive dispel skill class in atlas-monsters from skill data. atlas-channel populates `SourceSkillClass` on the cancel command. (Design D7.)
- **Do NOT** add disease metadata to `MIST_CREATED` outbound events. It is internal to atlas-maps. (Design §3.3, api-contracts §5.)
- **Do NOT** create a new `tasks.PoisonTick` — it already exists at `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` and is wired in `main.go:64`. Plan adds tests only.
- **Do NOT** use `omitempty` on any field that backs a Lua/cjson consumer (the four reflect numeric fields, `reflectKind`, all status-event slice fields).

---

## 9. Build/test commands

```bash
# Per-service build
cd services/atlas-monsters/atlas.com/monsters && go build ./...
cd services/atlas-channel/atlas.com/channel && go build ./...
cd services/atlas-buffs/atlas.com/buffs && go build ./...
cd services/atlas-maps/atlas.com/maps && go build ./...
cd libs/atlas-packet && go build ./...
cd libs/atlas-constants && go build ./...

# Per-service test
cd services/atlas-monsters/atlas.com/monsters && go test ./...
cd services/atlas-channel/atlas.com/channel && go test ./...
cd services/atlas-buffs/atlas.com/buffs && go test ./...
cd services/atlas-maps/atlas.com/maps && go test ./...
cd libs/atlas-packet && go test ./...
cd libs/atlas-constants && go test ./...

# Targeted single-test runs use `-run` with a regex; see plan tasks for the exact invocation per step.
```

CI Docker builds for all five services + the two libs are required after final integration.

---

## 10. Dependency DAG (plan task numbers)

```
  Leaves (parallelisable):  T1, T2, T3, T4, T5
                              │   │   │   │
              ┌───────────────┘   │   │   │
              ▼                   ▼   ▼   ▼
      T6 (StatusEffect fields) ──┐ (uses T3)
      T7 (Kafka body extension) ─┤
      T8 (executeStatBuff reflect)
              │
              ▼
      T9 (immunity exclusion — independent of reflect path)
              │
              ├────────────────────────────────────────────────┐
              ▼                                                ▼
      T10 (StatusMirror)                                T13-T18 (mist domain in atlas-maps)
              │                                                │
              ▼                                                │
      T11 (wire mirror into status consumers)                  │
              │                                                │
              ▼                                                │
      T12 (VENOM wire collapse)                                │
              │                                                │
              ▼                                                ▼
      T23 (reflect math in attack handler)            T20 (executeMist + producer)
              │                                                │
              ▼                                                ▼
      T24 (STATUS_CANCEL + dispel guard)              T21 (picker un-skip AREA_POISON)
                                                               │
                                                               ▼
                                                      T22 (atlas-channel mist consumer + AffectedArea broadcast — uses T2 + T19)
                                                               │
                                                               ▼
                                                      T19 (verify atlas-character position lookup)
                                                               │
                                                               ▼
                                                      T25 (player-skill venom snapshot DPT — atlas-channel)
                                                               │
                                                               ▼
                                                      T26 (end-to-end verification)
                                                               │
                                                               ▼
                                                      T27 (final commits + audits)
```

Tasks T1–T5 fan out at full width. Mid-tier T6–T9 and T13–T18 split into two parallel branches once their leaves land. Top-tier converges at T26.
