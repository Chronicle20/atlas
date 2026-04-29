# Monster Skill Effects Completion — Design

Companion to `prd.md`, `data-model.md`, `api-contracts.md`, `risks.md`. Locks the architectural decisions deferred by the PRD's §9 open questions and supersedes any data-model / api-contract details that this design changes. Where this design and the earlier companion docs disagree, **this design wins**.

---

## 1. Decisions

### 1.1 Architectural decisions made this phase

| # | Topic | Decision | Reasoning |
|---|---|---|---|
| D1 | Reflect math placement | atlas-channel `StatusMirror` + local math + emits `DAMAGE_REFLECTED` | REST or Kafka round-trip per damage entry would put atlas-monsters in the attack hot path (≥6 entries / hit). Reflect is fire-and-forget gameplay; the eventual-consistency window is acceptable per `risks.md §1`. |
| D2 | Mist domain owner | atlas-maps (new `mist` domain) | Mist is a first-class field-scoped object (like reactors). atlas-maps already owns per-field state and `field.Model` already encodes instance UUID. Future player-cast mists (Poison Mist, Smokescreen) compose without a service-boundary change. |
| D3 | Venom encoding | Each apply = its own `StatusEffect` in the existing `[]StatusEffect` slice; the codebase already supports this | The audit at `monster.Model.statusEffects` (slice), `cancelByEffectId` (UUID-keyed), and `builder.go:130-156` (existing `VENOM`-cap branch) shows multi-stack VENOM is already implemented. The PRD's `VENOM_1`/`VENOM_2`/`VENOM_3` slot keys would add a magic-string layer on top of a model that natively supports stacking. **The PRD's `data-model.md §2-3` is superseded.** |
| D4 | Reflect range gate | Bounding box (`LtX/LtY/RbX/RbY`), reusing the AoE pattern at `processor.go:683-687` | The bounding-box fields are already authoritative for AoE skills on the same skill data. Using them for reflect keeps spatial semantics consistent across mob skills and avoids the 1-D simplification flagged by `risks.md §4`. |
| D5 | `AffectedArea` writer location | `libs/atlas-packet/field/clientbound/affected_area_{created,removed}.go` | Co-locates with the existing 10 field packet writers (clock, weather, kites, field setup); affected areas are field-scoped objects. A separate `mist/clientbound/` package would be premature for two writers. |
| D6 | Mist character lookup | `MistTickTask` calls `atlas-maps.character.GetCharactersInMap(field)` (existing) for membership, then a per-character REST query to atlas-character for `(x,y)` to apply the bounding-box `Contains` filter | atlas-maps already maintains a per-`(tenant, field)` character-membership registry at `services/atlas-maps/atlas.com/maps/map/character/registry.go:13-99`. Position is the only piece missing — a bounded REST cost (`O(charactersInField × activeMists)` per second). If profiling later shows the position lookup is hot, the fallback is to extend the registry with `(x,y)` fed by movement events; no architectural change required. |
| D7 | Dispel-class classification | atlas-channel populates `SourceSkillClass` (`"PHYSICAL"` \| `"MAGIC"` \| `""`) on the `STATUS_CANCEL` command body; atlas-monsters reads it directly | atlas-channel already differentiates physical / magic / ranged for the attack handler today. Re-deriving it in atlas-monsters from skill data would require a new player-skill-metadata provider for what atlas-channel already knows. The cjson audit (PRD FR-4.10) covers the new field at no extra cost. |
| D8 | Immunity mutual exclusion | Inline cancel-then-apply in `executeStatBuff`; two events; partition-keyed by `uniqueId` for ordering | Reuses existing cancel/apply pipelines so downstream consumers (mirror, packet broadcasts) get correct state through standard handlers. The transient "no immunity" window is microseconds at the consumer when both events share a Kafka partition. A new `STATUS_REPLACED` event would touch every existing status consumer for one rare interaction. |
| D9 | `PoisonTick` task wiring | Standalone `tasks.PoisonTick` wrapping the existing `character.ProcessPoisonTicks` (`services/atlas-buffs/atlas.com/buffs/character/processor.go:114-140`) on a 1 s cadence; sibling to `tasks.Expiration` | The producer logic already exists. The PRD's "new task" is purely the recurring loop. Folding it into `tasks.Expiration` would conflate buff-cleanup cadence with DoT cadence. |

### 1.2 Resolutions to PRD §9 open questions

- **§9-1 (`AffectedArea` packet writers).** Confirmed missing in `libs/atlas-packet/field/clientbound/`. Plan adds two new writers there (D5).
- **§9-2 (`sd.X()` vs `sd.Y()` for reflect).** Superseded by D4. Reflect uses the bounding box, not `X/Y` as range. `ReflectPercent` still comes from `sd.X()`. `ReflectMaxDamage` is a constant cap of `32767` (a v83-typical sentinel matching the PRD's default). If WZ inspection during plan-phase TDD shows a per-skill cap, the constant is replaced by `sd.<field>()`.
- **§9-3 (PoisonTick character damage producer).** Resolved. atlas-buffs' existing `character.ProcessPoisonTicks` produces `CHANGE_HP` to `COMMAND_TOPIC_CHARACTER` already (see `kafka/message/character/kafka.go:82-96`). No new topic.
- **§9-4 (Mist on instance maps).** Resolved. `field.Model` encodes `(world, channel, mapId, instance)` (`libs/atlas-constants/field/model.go:13-18`); `MapKey{Tenant, Field}` already disambiguates instances in atlas-maps' character registry.

---

## 2. Architecture

```
                       atlas-monsters
                       ───────────────
   picker fires AREA_POISON
       └─► executeMist ──► MIST_CREATE command  ──────────┐
   picker fires WEAPON_REFLECT / MAGIC_REFLECT             │
       └─► executeStatBuff (with bounding-box metadata)   │
            └─► STATUS_APPLIED (extended body)            │
   picker fires PHYSICAL_IMMUNE / MAGIC_IMMUNE            │
       └─► executeStatBuff:                                │
            cancel opposite immunity → STATUS_CANCELLED   │
            apply new immunity → STATUS_APPLIED           │
   STATUS_CANCEL command in (from atlas-channel)           │
       └─► dispel guard reads SourceSkillClass            │
            reject if reflect of matching kind active     │
   builder.AddStatusEffect (VENOM branch)                  │
       └─► count == 3? evict min(ExpiresAt) instead        │
            of first-found                                 │
                                                           │
                       atlas-maps                          │
                       ───────────                         │
   command consumer ◄──────────────────────────────────────┘
       └─► mist.Processor.Create
            └─► MistRegistry.Add
            └─► emit MIST_CREATED ──────────────► atlas-channel
   MistTickTask (1 s cadence)
       └─► for each tenant:
            for each mist in registry:
              expired? Destroy + emit MIST_DESTROYED(EXPIRED)
              else: GetCharactersInMap(field) (existing)
                    REST atlas-character for (x,y)
                    filter via mist.Contains(x,y)
                    apply-disease cmd on EnvCommandTopicCharacterBuff

                       atlas-channel
                       ─────────────
   StatusMirror (per tenant → uniqueId → status name → []StatusEntry)
       fed by existing handleStatusEffectApplied / Expired / Cancelled
       and StatusEventDestroyed / Killed
   character_attack_common.go (replace 2 TODOs)
       └─► Mirror.GetReflect(t, monsterId, kind)
       └─► bounding-box check (attacker x,y vs monster + Lt/Rb)
       └─► reflected = damage*Percent/100, capped at MaxDamage
       └─► emit DAMAGE_REFLECTED, zero entry damage
   handleDamageReflected (existing, unchanged)
       └─► character.NewProcessor(...).ChangeHP(...)
   STATUS_CANCEL producer (wherever it lives today)
       └─► populate SourceSkillClass from player-skill metadata
   mist consumer (NEW)
       └─► MIST_CREATED → AffectedAreaCreated to sessions in field
       └─► MIST_DESTROYED → AffectedAreaRemoved to sessions in field

                       atlas-buffs
                       ───────────
   tasks.PoisonTick (NEW, 1 s cadence)
       └─► character.NewProcessor(l, ctx).ProcessPoisonTicks()
            (existing — produces CHANGE_HP per poisoned character)

                       libs/atlas-packet
                       ─────────────────
   field/clientbound/affected_area_created.go   (NEW)
   field/clientbound/affected_area_removed.go   (NEW)
       (modeled on reactor/clientbound/spawn.go:24-52)

                       libs/atlas-constants
                       ────────────────────
   ReflectKindPhysical, ReflectKindMagical constants  (NEW)
   No venom slot constants — multi-stack uses existing []StatusEffect
```

---

## 3. Component-by-Component Design

### 3.1 atlas-monsters

**`monster.StatusEffect` extension.** Add four fields (kind, percent, LtX/LtY/RbX/RbY) to the existing immutable model. Defaults zero for non-reflect statuses. New `IsReflect()` returns `kind != ""`. Both an extended `NewStatusEffect` overload **and** a builder-style helper `NewReflectStatusEffect` exist; the plan picks one and stays consistent. Existing callers continue to compile because new fields default to zero values.

**`executeStatBuff` (`monster/processor.go` ~`:540-700`).** Three behavioural changes:

1. **Reflect metadata population.** When the new status is `WEAPON_REFLECT` or `MAGIC_REFLECT`, fetch the mob skill via the existing `mobskill.GetByIdAndLevel` provider and populate `reflectKind` (`PHYSICAL` for weapon, `MAGICAL` for magic), `reflectPercent = sd.X()`, and the four bounding-box fields directly from `sd.LtX/LtY/RbX/RbY`. `reflectMaxDamage` is the constant `32767`.

2. **Immunity mutual exclusion.** When applying `PHYSICAL_IMMUNE`, if the monster has an active `MAGIC_IMMUNE` (via `m.HasStatusEffect(MAGIC_IMMUNE)`), cancel that effect through the existing internal cancel path **before** the new apply runs. Symmetric for `MAGIC_IMMUNE` displacing `PHYSICAL_IMMUNE`. This block runs **before** the existing already-active gate so a stale opposite immunity doesn't block the new one. Both cancel and apply emit Kafka events partition-keyed by `uniqueId`.

3. **No change to existing reflect already-active gate** at `picker.go:185-191` — that gate correctly prevents re-applying same-kind reflect (PRD FR-4.1.3).

**`executeMist(m, sd)` (new).** Produces a `MIST_CREATE` command on `EVENT_COMMAND_TOPIC_MIST`. Body fields lifted from `sd`:
- `Origin = (m.X, m.Y)`
- `LtX/LtY/RbX/RbY` from `sd.LtX/LtY/RbX/RbY` (relative offsets)
- `Disease = "POISON"`, `DiseaseValue = sd.X()`, `DiseaseDuration = sd.Duration()` (ms)
- `Duration = sd.Duration() * 1000` (mist lifetime in ms — verify the multiplier in plan phase against WZ; AREA_POISON's `duration` field is in seconds in v83 data dumps but the plan must confirm against atlas-data's exposure)
- `TickIntervalMs = 1000`
- `OwnerType = "MONSTER"`, `OwnerId = m.UniqueId`

**Picker un-skip.** Remove the `AREA_POISON` exclusion at `picker.go:144-149`. Flip / delete `picker_test.go::TestPicker_AreaPoisonExcluded` to assert the picker now fires it.

**Venom eviction policy fix (`builder.go:130-156`).** The current "VENOM at cap → evict first-found" branch becomes "evict the effect with the minimum `ExpiresAt`." Single-method change plus a new test asserting eviction order under non-trivial expiry timestamps.

**Per-effect snapshot DPT — important scoping note.** VENOM is a *player-cast* debuff (Night Lord / Shadower skills); attacker stats (`Luck`, `MagicAttack`) are not visible to atlas-monsters' mob-cast `executeStatBuff` (which hard-codes `SourceCharacterId = 0` at `processor.go:661`). The snapshot DPT formula `damagePerTick = round(rand.Float(0.1, 0.2) * attackerLuck * attackerMagicAttack)` therefore runs on the **player-skill apply path**, not in `executeStatBuff`. atlas-monsters' role in the venom flow is purely:

1. Receive an apply request that already carries the snapshot DPT in `Statuses["VENOM"]` and the attacker's character id in `SourceCharacterId`.
2. Build the `StatusEffect` and call `builder.AddStatusEffect`, which now uses the corrected min-`ExpiresAt` eviction (D3).
3. `processDoTTick` reads each effect's `Statuses["VENOM"]` directly (`status_task.go:61-102`, already correct) — sums all stacks, applies `currentHp - 1` cap, emits `DAMAGED`.

The plan phase must locate the actual player-skill apply entry into atlas-monsters (likely a command consumer somewhere — grep for `STATUS_APPLY` command handling). The snapshot DPT computation lives at that entry's *upstream* producer side, in atlas-channel where attacker stats are known. The atlas-monsters changes for venom are limited to the eviction-policy fix in `builder.go` plus the existing DoT-tick code which already iterates all stacks correctly.

**Dispel guard.** In the `STATUS_CANCEL` command consumer (or processor — plan picks the cleaner placement based on existing structure), read the new `SourceSkillClass` field. If non-empty and the monster has any active reflect effect with `reflectKind == SourceSkillClass`, log at debug `"dispel rejected: monster [%d] has active %s reflect"` and return without applying the cancel. Empty `SourceSkillClass` falls through to normal cancel behaviour (preserves backwards-compat for non-skill cancels — e.g., expiration).

**cjson empty-array audit.** Audit every body in `monster/kafka.go` per PRD FR-4.10. For each slice field, add a regression test that round-trips `nil` and asserts the JSON contains `"field":[]` not `"field":null`. The audit covers the new fields too — `ReflectKind` is a string with default `""` (never `null`), the four reflect numeric fields are scalars (no slice gotcha applies). The `Statuses` map guard is already in place; verify it.

### 3.2 atlas-channel

**`monster.StatusMirror` (new singleton).** Per-tenant → per-`uniqueId` → per-status-name → `[]StatusEntry`. Multiple entries per status name supported (D3 — for VENOM stacking). `RWMutex`-guarded; reads under `RLock`, writes under `Lock`. Singleton via `sync.Once`. Tenant key follows the existing in-tree convention (string-cast UUID).

```go
type ReflectInfo struct {
    Kind                       string
    Percent                    int32
    LtX, LtY, RbX, RbY         int32
    MaxDamage                  int32
    ExpiresAt                  time.Time
}

type StatusEntry struct {
    EffectId          uuid.UUID
    Statuses          map[string]int32
    Reflect           *ReflectInfo // nil for non-reflect
    SourceCharacterId uint32
    ExpiresAt         time.Time
}

type StatusMirror struct {
    mu       sync.RWMutex
    perTenant map[string]map[uint32]map[string][]StatusEntry
}

// Public API.
func GetStatusMirror() *StatusMirror
func (m *StatusMirror) OnApplied(t tenant.Model, uniqueId uint32, body StatusEffectAppliedBody)
func (m *StatusMirror) OnExpired(t tenant.Model, uniqueId uint32, effectId uuid.UUID, statuses map[string]int32)
func (m *StatusMirror) OnCancelled(t tenant.Model, uniqueId uint32, effectId uuid.UUID, statuses map[string]int32)
func (m *StatusMirror) OnMonsterGone(t tenant.Model, uniqueId uint32)
func (m *StatusMirror) GetReflect(t tenant.Model, uniqueId uint32, kind string) (ReflectInfo, bool)
func (m *StatusMirror) VenomCount(t tenant.Model, uniqueId uint32) int
```

`OnExpired` / `OnCancelled` remove by `effectId` (not by status name) so concurrent VENOM stacks each remove independently.

**Mirror wiring.** The existing `handleStatusEffectApplied`, `handleStatusEffectExpired`, `handleStatusEffectCancelled`, `handleStatusEventDestroyed`, `handleStatusEventKilled` consumers each call the corresponding mirror method **after** their existing wire-broadcast logic.

**VENOM wire collapse.** atlas-channel's existing consumer iterates `e.Body.Statuses` and emits per-key `MonsterStatSet` packets. With multi-stack VENOM, multiple `STATUS_APPLIED` events for the same monster carry `Statuses["VENOM"] = <DPT>` independently. To avoid spamming `MonsterStatSet(VENOM)` per apply:

- Apply path: before emitting `MonsterStatSet(VENOM)`, query `Mirror.VenomCount(t, uniqueId)` **before** writing the new entry. If the prior count was 0, emit `MonsterStatSet`; else suppress (idempotent re-broadcast acceptable but not desired).
- Expire/cancel path: after the mirror write, query `VenomCount`. If the new count is 0, emit `MonsterStatReset(VENOM)`; else suppress.

This rule replaces the PRD's `VENOM_N → VENOM` translator entirely (D3).

**`character_attack_common.go` reflect math.** Replace the two `TODO Monster Weapon/Magic Atk Reflect` placeholders. Per damage entry, before applying it to the monster:
1. Resolve `kind` from attack type: close-range / ranged → `PHYSICAL`, magic → `MAGICAL`.
2. `info, ok := Mirror.GetReflect(t, monsterId, kind)`. If `!ok`, continue normally.
3. Compute `dx, dy = attacker.X - monster.X, attacker.Y - monster.Y`. If **not** `dx >= info.LtX && dx <= info.RbX && dy >= info.LtY && dy <= info.RbY`, continue normally.
4. `reflected := min(damage * info.Percent / 100, info.MaxDamage)`.
5. Emit `DAMAGE_REFLECTED` (`{CharacterId: attackerId, ReflectDamage: reflected, MonsterUniqueId: monsterId}`) via the existing producer.
6. **Set the entry's monster-side damage to 0 before any `damage_taken` write**; do not emit `DAMAGED` for that entry.

The existing `handleDamageReflected` consumer at `consumer/monster/consumer.go:371-384` is unchanged.

**`STATUS_CANCEL` producer extension.** Wherever atlas-channel produces `STATUS_CANCEL` for monster statuses today, populate `SourceSkillClass` from the player skill the cancel originated from (atlas-channel already needs this classification for the attack handler in §3.2). Empty string for non-skill-driven cancels.

**Mist consumer (new).** Subscribes to `EVENT_TOPIC_MIST`. On `MIST_CREATED`: `ForSessionsInMap(field, AffectedAreaCreatedWriter(mistId, origin, bounds, duration))`. On `MIST_DESTROYED`: `ForSessionsInMap(field, AffectedAreaRemovedWriter(mistId))`. Disease metadata is intentionally absent from the outbound event (api-contracts §5).

### 3.3 atlas-maps (new `mist` domain)

Path: `services/atlas-maps/atlas.com/maps/mist/`. Pattern follows the existing `reactor/` package as a template.

**`mist.Mist` immutable model.** Builder + private fields + getters (per the codebase convention). Fields per `data-model.md §6`. Helpers: `Contains(x, y int16) bool`, `Expired() bool`, `ShouldTick() bool`. Absolute bounding box: `[origin.X+ltX, origin.X+rbX] × [origin.Y+ltY, origin.Y+rbY]`. `Contains` uses inclusive bounds.

**`MistRegistry`.** Singleton via `sync.Once`. `RWMutex`-guarded. Per-tenant index: `map[string]map[uuid.UUID]Mist`. Secondary index: `map[string]map[FieldKey][]uuid.UUID` for fast field lookup. `Add`, `Remove(returns the removed Mist for event emission)`, `GetByField`, `UpdateLastTick`. `FieldKey` derives from `field.Model` (string-cast or struct — match existing in-tree pattern).

**`mist.Processor`.** Interface + `ProcessorImpl` per Atlas convention. `NewProcessor(l, ctx)`. Methods: `Create(body MistCreateBody) (Mist, error)` — generates UUID, builds the model, registry-inserts, emits `MIST_CREATED`. `Destroy(mistId, reason)` — registry-removes, emits `MIST_DESTROYED`.

**`MistTickTask`.** Pattern follows the existing `tasks.Respawn` task. 1 s cadence. `Run()`:

```
for each tenant t:
  ctx := tenant.WithContext(taskCtx, t)
  for each mist m in MistRegistry.AllByTenant(t):
    if m.Expired():
      Processor.Destroy(m.Id(), "EXPIRED")
      continue
    if !m.ShouldTick(): continue
    members := character.NewProcessor(l, ctx).GetCharactersInMap(_, m.Field())
    for each cid in members:
      pos, err := atlas-character REST: GET /characters/{cid}/position
      if err != nil: log + skip
      if !m.Contains(pos.X, pos.Y): continue
      produce apply-disease command on EnvCommandTopicCharacterBuff
        with (cid, m.Disease(), m.DiseaseValue(), m.DiseaseDuration(), source skill ids from owner)
    MistRegistry.UpdateLastTick(t, m.Id(), now)
```

The atlas-character position endpoint must exist — the plan confirms (it does in some shape; if not, the plan adds a minimal `GET /characters/{id}/position` returning `{x, y, mapId, channel}`). Per-character REST is bounded by `O(charactersInField × activeMists) ≤ O(50 × 10) = 500` queries / sec on an extreme map. Acceptable; profile in plan-phase test bench.

**Command consumer.** Subscribes to `EVENT_COMMAND_TOPIC_MIST`. `MIST_CREATE` → `Processor.Create`. `MIST_CANCEL` → `Processor.Destroy(mistId, "CANCELLED")`. Standard tenant-header parsing.

**Event producer.** Emits `EVENT_TOPIC_MIST` with `MIST_CREATED` and `MIST_DESTROYED` per api-contracts §5. Partition key: `mistId` (string).

**Holy Shield bypass safety.** atlas-buffs already gates apply on `HasImmunity` (`character/processor.go:43-44`). No additional check in atlas-maps. Mist tick re-applying disease every second simply gets rejected by atlas-buffs for shielded characters (PRD FR-4.6.8 + risks §8).

### 3.4 atlas-buffs

**`tasks.PoisonTick` (new).**

```go
type PoisonTick struct {
    l        logrus.FieldLogger
    interval int // ms; default 1000
}

func NewPoisonTick(l logrus.FieldLogger, interval int) *PoisonTick

func (r *PoisonTick) Run() {
    // For each tenant context (matches tasks.Expiration's iteration pattern):
    //   ctx := tenant.WithContext(...)
    //   _ = character.NewProcessor(r.l, ctx).ProcessPoisonTicks()
}

func (r *PoisonTick) SleepTime() time.Duration {
    return time.Duration(r.interval) * time.Millisecond
}
```

Wired in `main.go` next to `tasks.Expiration`. Default interval 1000 ms; env-overridable via existing config-loading pattern. The producer logic — `GetPoisonCharacters` → `GetLastPoisonTick` → `CHANGE_HP` command → `UpdatePoisonTick` — already exists at `services/atlas-buffs/atlas.com/buffs/character/processor.go:114-140`. The new task is purely the recurring wrapper.

**No Holy Shield changes.** Apply path already gates per FR-4.7.3.

### 3.5 libs/atlas-packet

**New writers in `field/clientbound/`:**
- `affected_area_created.go` — `Created` immutable model with `MistId uuid.UUID`, `Origin (X,Y int16)`, `LtX/LtY/RbX/RbY int16`, `Duration int64`, `OwnerId uint32`. Builder + getters. `Operation()` returns the writer name. `Encode(l, ctx) func(opts) []byte` writes the v83 affected-area-create packet.
- `affected_area_removed.go` — `Removed` model with `MistId uuid.UUID`. Same pattern, encodes the affected-area-remove packet.

Modeled on `reactor/clientbound/spawn.go:24-52`. v83 opcode constants: `AFFECTED_AREA_CREATED`, `AFFECTED_AREA_REMOVED`. Plan-phase TDD verifies opcode values against MapleStory protocol references; if values are unknown, a temporary placeholder is fine for the unit test, replaced once a v83 packet capture is consulted.

### 3.6 libs/atlas-constants

**Reflect kind constants** (new):
```go
const (
    ReflectKindPhysical = "PHYSICAL"
    ReflectKindMagical  = "MAGICAL"
)
```

Located alongside the existing skill-category / status-name constants. Used by atlas-monsters (in `executeStatBuff` and the dispel guard), atlas-channel (in the mirror, attack handler, and `SourceSkillClass` population), and the test suite.

**No venom-slot constants needed** (D3). The PRD's `data-model.md §2` constants (`StatusVenom1/2/3`, `IsVenomSlot`, suffix constants) are not added.

---

## 4. Data Flow Diagrams

### 4.1 Reflect end-to-end

```
1. Mob picker fires WEAPON_REFLECT skill on monster M.
2. atlas-monsters.executeStatBuff:
   - Fetch mobskill.Model for skill.
   - Build StatusEffect with reflectKind="PHYSICAL", reflectPercent=sd.X(),
     reflectLtX/...=sd.LtX/..., reflectMaxDamage=32767.
   - Apply to M, emit STATUS_APPLIED with extended body.
3. atlas-channel.handleStatusEffectApplied:
   - Existing wire broadcast (MonsterStatSet WEAPON_REFLECT).
   - StatusMirror.OnApplied → mirror entry with Reflect populated.
4. Player attacks M with melee:
   - character_attack_common.go iterates damage entries.
   - For each: kind=PHYSICAL.
   - Mirror.GetReflect(t, M.uniqueId, PHYSICAL) → ReflectInfo.
   - Compute dx, dy from attacker (x,y) to M (x,y).
   - If inside [LtX..RbX]×[LtY..RbY]:
       reflected = min(damage * Percent / 100, MaxDamage)
       produce DAMAGE_REFLECTED(M.uniqueId, attacker.id, reflected)
       set entry damage to 0
   - Else: continue normally.
5. atlas-channel.handleDamageReflected (existing):
   - character.NewProcessor(...).ChangeHP(field, attacker.id, -reflected).
6. Reflect status expires:
   - StatusExpirationTask emits STATUS_EXPIRED.
   - atlas-channel: existing wire broadcast (MonsterStatReset).
   - Mirror.OnExpired → entry removed.
   - Subsequent attacks no longer reflect.
```

### 4.2 Venom 3-stack

```
1. Player applies VENOM to M (apply 1):
   - atlas-channel attack handler (or the player-skill apply path —
     plan locates the exact entry): snapshot DPT computed from attacker
     stats: round(rand(0.1, 0.2) * Luck * MAtk).
   - Apply command sent to atlas-monsters carrying Statuses["VENOM"] = DPT
     and SourceCharacterId = attacker.id.
   - atlas-monsters builds StatusEffect, calls builder.AddStatusEffect:
     VENOM branch, count=0, no eviction. Append.
   - atlas-monsters emits STATUS_APPLIED.
2. atlas-channel: VenomCount before write = 0 → emit MonsterStatSet(VENOM).
   Mirror.OnApplied → entry added.
3. Player applies VENOM to M (applies 2 and 3):
   - Same as above, fresh effect ids, fresh snapshot DPT.
   - VenomCount before write > 0 → suppress MonsterStatSet (no spam).
4. Player applies VENOM to M (apply 4):
   - builder.AddStatusEffect: count=3, find effect with min(ExpiresAt),
     remove it from slice, append the new one.
   - The evicted effect generates a STATUS_CANCELLED event so atlas-channel
     can prune the mirror.
   - Mirror state: 3 entries (newest replaces oldest).
5. processDoTTick (atlas-monsters, every 1 s per status_task):
   - Iterates all StatusEffects on M.
   - For each effect with HasStatus("VENOM"): totalDamage += DPT from that
     effect's Statuses["VENOM"].
   - Sums all stacks, applies cap (currentHp - 1), produces DAMAGED event.
6. Effects expire one by one:
   - Each expiry → STATUS_EXPIRED → Mirror.OnExpired → VenomCount decrements.
   - On the last expiry (count transitions to 0): emit MonsterStatReset(VENOM).
```

### 4.3 Mist tick

```
1. Mob fires AREA_POISON. picker now allows it (D2 + un-skip).
2. atlas-monsters.executeMist:
   - Build MistCreateBody from sd.
   - Produce MIST_CREATE on EVENT_COMMAND_TOPIC_MIST.
3. atlas-maps command consumer:
   - mist.Processor.Create:
     - Generate uuid.UUID for the mist.
     - Build mist.Mist from body.
     - MistRegistry.Add.
     - Emit MIST_CREATED on EVENT_TOPIC_MIST.
4. atlas-channel mist consumer:
   - ForSessionsInMap(field, AffectedAreaCreatedWriter(mistId, ...)).
5. MistTickTask (every 1 s):
   - For each tenant:
     for each mist m:
       expired? → Processor.Destroy(m.id, "EXPIRED"), emit MIST_DESTROYED.
       else:
         members = atlas-maps.character.GetCharactersInMap(m.Field())
         for each cid:
           pos = REST GET /characters/{cid}/position
           if m.Contains(pos.x, pos.y):
             produce apply-disease command (POISON, m.DiseaseValue,
               m.DiseaseDuration) on EnvCommandTopicCharacterBuff
6. atlas-buffs apply path:
   - HasImmunity check (existing). Holy Shield → no apply.
   - Otherwise: poison buff applied (or duration reset on existing).
7. atlas-buffs tasks.PoisonTick (every 1 s):
   - ProcessPoisonTicks → CHANGE_HP per poisoned character.
8. Mist expires → MIST_DESTROYED → AffectedAreaRemoved on the wire.
   Poisoned characters continue ticking until their buff duration ends.
```

### 4.4 Immunity mutual exclusion

```
1. Monster M has active MAGIC_IMMUNE.
2. Mob picker fires PHYSICAL_IMMUNE.
3. executeStatBuff:
   - Detect m.HasStatusEffect(MAGIC_IMMUNE).
   - Cancel the existing MAGIC_IMMUNE through internal cancel path:
     - Registry write: remove the StatusEffect by EffectId.
     - Emit STATUS_CANCELLED with the cancelled status name.
   - (Both events are partition-keyed by uniqueId so they arrive in order.)
   - Run the existing already-active gate (now passes since opposite was
     cancelled).
   - Apply PHYSICAL_IMMUNE: registry write, emit STATUS_APPLIED.
4. atlas-channel:
   - Mirror.OnCancelled (MAGIC_IMMUNE entry removed) + wire broadcast.
   - Mirror.OnApplied (PHYSICAL_IMMUNE entry added) + wire broadcast.
```

### 4.5 Dispel guard

```
1. Player casts dispel-class skill (PHYSICAL) on M.
2. atlas-channel produces STATUS_CANCEL command:
   - SourceSkillId = player skill id.
   - SourceSkillClass = "PHYSICAL" (atlas-channel already knows).
3. atlas-monsters cancel handler:
   - Read SourceSkillClass from body.
   - Iterate active StatusEffects on M; if any has reflectKind == "PHYSICAL":
     log debug "dispel rejected: monster [%d] has active PHYSICAL reflect"
     return without cancelling anything.
   - Else: proceed with normal cancel for the targeted statuses.
4. Magic-class dispel against same monster runs the same check with
   SourceSkillClass="MAGIC" — succeeds (no MAGICAL reflect active).
```

---

## 5. Sequencing & Dependencies

The plan phase will produce TDD tasks; this section gives the dependency DAG so independent leaves can be parallelised.

**Leaves (no inter-task deps):**
- L1. cjson empty-array audit + regression tests on existing status bodies (atlas-monsters).
- L2. `affected_area_{created,removed}.go` writers + tests (libs/atlas-packet).
- L3. `ReflectKindPhysical/Magical` constants (libs/atlas-constants).
- L4. `tasks.PoisonTick` wrapper around existing producer (atlas-buffs) + wiring in `main.go`.
- L5. Venom eviction-policy fix in `builder.go` (one-method change + test).

**Mid-tier (depend on leaves):**
- M1. `StatusEffect` reflect fields + extended `executeStatBuff` reflect path (depends on L3).
- M2. Immunity mutual exclusion in `executeStatBuff` (independent of M1; can parallelise).
- M3. `StatusMirror` in atlas-channel (depends on M1 — needs the extended body shape).
- M4. `mist` domain in atlas-maps: model, registry, processor, command consumer, event producer, tick task (depends loosely on L2 — atlas-maps doesn't import the writers; the tick task uses existing membership and a REST position call).

**Top-tier (depend on mid-tier):**
- T1. Reflect math in `character_attack_common.go` (depends on M3).
- T2. Dispel guard in atlas-monsters cancel handler + `SourceSkillClass` population in atlas-channel (depends on M1; also on a small Kafka body addition the plan tracks).
- T3. `executeMist` in atlas-monsters + picker un-skip (depends on M4 being deployable).
- T4. atlas-channel mist consumer broadcasting via L2 writers (depends on M4 + L2).

**Parallelisation hint:** L1–L5 can fan out at full width. M1–M4 can run as four parallel branches once their leaf deps land. Top-tier converges per-branch.

---

## 6. Testing Strategy

Mirrors PRD §10 acceptance criteria. Notable tests required by **this design's** decisions (in addition to PRD-listed ones):

- **D3 venom eviction (4 applies).** Apply 4 VENOM stacks with non-trivial expiry timestamps; assert the evicted effect is the one with minimum `ExpiresAt`, not the one inserted first.
- **D3 venom multi-stack tick.** Apply 3 VENOM stacks with different snapshot DPTs; tick once; assert summed damage equals sum of all three stacks; cap at `currentHp - 1`.
- **D3 venom wire collapse.** Three rapid `STATUS_APPLIED` events for VENOM on the same monster: assert exactly one `MonsterStatSet(VENOM)` is broadcast (transition 0→1). Three sequential expires: assert exactly one `MonsterStatReset(VENOM)` is broadcast (last transition to 0).
- **D4 reflect bounding-box.** Per attack-class: in-box → reflects, out-of-box (each axis individually) → applies normally; edge case at `attacker.X == monster.X + RbX` → inclusive (reflects).
- **D6 mist position filter.** Two characters in same field at different positions: one inside the bounding box, one outside; assert only the inside one receives the disease apply per tick.
- **D6 mist on instance maps.** Two `field.Model` instances with same `mapId` but different `instance` UUIDs; mist created in instance A; characters in instance B receive nothing.
- **D7 dispel guard parametric.** Pairs `(reflectKind on monster, SourceSkillClass on cancel)`: `(PHYSICAL, PHYSICAL)` → reject, `(PHYSICAL, MAGIC)` → allow, `(MAGIC, PHYSICAL)` → allow, `(MAGIC, MAGIC)` → reject, `(none, anything)` → allow, `(anything, "")` → allow (backwards-compat).
- **D8 immunity exclusion ordering.** Apply MAGIC_IMMUNE then PHYSICAL_IMMUNE; assert two events emitted in order: `STATUS_CANCELLED(MAGIC_IMMUNE)` precedes `STATUS_APPLIED(PHYSICAL_IMMUNE)` and both share the same Kafka partition key (`uniqueId`).
- **D9 PoisonTick wiring.** Start `tasks.PoisonTick` with a 100 ms interval (test override); assert it invokes `ProcessPoisonTicks` repeatedly; assert the existing producer is unchanged (regression test for FR-4.7.3 and FR-4.7.5).
- **`StatusEffectAppliedBody` empty-reflect serialization.** A non-reflect body round-trips with `"reflectKind":""` and `"reflectPercent":0` etc. — not absent, not `null`.

---

## 7. Out-of-Scope Confirmations

PRD §2 non-goals re-affirmed:
- No boss phase mechanics (deferred to spec-task 4).
- No new mob skill *types* not present in WZ data.
- Banish destination redesign: verify-only.
- No player-cast mist wiring (Poison Mist, Smokescreen). The atlas-maps mist domain is *reusable* for it but not wired here.
- No Holy Shield apply-path changes in atlas-buffs (already implemented at `character/processor.go:43-44`).

No frontend changes (PRD §7).
