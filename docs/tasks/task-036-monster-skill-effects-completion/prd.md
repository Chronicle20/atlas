# Monster Skill Effects Completion — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-29
---

## 1. Overview

Atlas already supports the basic monster-skill dispatch pipeline: a picker selects a skill, an executor applies it, and most stat-buff / debuff broadcasts already reach clients via `MonsterStatSet` / `MonsterStatReset`. What is missing is the long tail of mechanics that make those skills *do* anything beyond bookkeeping. Reflect skills set a status but never reflect damage. Venom applies one effect instead of stacking up to three. Mist (area-poison) skills are explicitly skipped by the picker. The buff service tracks poisoned characters but never ticks their HP. And several invariants — physical/magic immunity exclusivity, dispel guard against a matching reflect — are unenforced.

This task closes the long tail end-to-end: reflect actually reflects in the attack handler, venom stacks correctly with snapshot stats, mist zones spawn as first-class map objects in atlas-maps with a 1 s tick, atlas-buffs gets a dedicated `PoisonTick` task, and the immunity / dispel invariants are enforced. Scope is deliberately bounded: boss phase mechanics are deferred to spec-task 4, and only skill *types* already present in WZ data are addressed.

## 2. Goals

Primary goals:
- Reflect skills reflect damage to attacking characters in atlas-channel.
- Venom debuff supports up to 3 independent stacks with per-stack snapshot stats from the attacker.
- Mist (`AREA_POISON`, skill type 131) zones spawn, tick, expire, and broadcast as first-class entities — including the picker un-skip.
- Player-side poison HP DoT is driven by a dedicated `PoisonTick` task in atlas-buffs.
- Physical / magical immunity are mutually exclusive on a monster.
- Dispel cannot remove a status while a reflect of the matching damage class is active.
- All `cjson` empty-array landmines on existing and new status-event payloads are eliminated.
- TDD coverage for every behavior added (status_task, reflect mirror, mist tick, poison tick, immunity exclusion, dispel guard).

Non-goals:
- Boss-specific phase mechanics or transition skills (deferred — spec-task 4).
- New skill *types* not present in mob WZ data.
- Banish destination redesign — verify-only.
- Player-cast skills that *spawn* mists (Poison Mist, Smokescreen). The atlas-maps mist model added by this task should be reusable, but no player-skill wiring is in scope.
- Holy Shield handling on the apply path — already implemented in atlas-buffs (`character/processor.go:43-44`); no atlas-monsters → atlas-buffs REST query is needed.

## 3. User Stories

- As a player attacking a Pap with Power Guard active, my next physical hit applies zero damage to Pap and reduces my own HP, exactly as the legacy server behaved.
- As a player applying Venom to a boss three times in a row from one job, all three stacks tick concurrently with the Luck × MagicAttack snapshotted at the moment each was applied.
- As a player walking into an Anego's poison cloud, I receive the POISON disease, my HP ticks down once a second from a `PoisonTick` task, and the cloud disappears at the configured duration.
- As a player who applies a magic-immunity-canceling skill (e.g. Heal vs Undead, where applicable), I expect the monster's prior physical immunity to be cleared since the two are mutually exclusive in this game's data.
- As a player attempting to dispel a boss currently running Weapon Reflect, the dispel rejects the reflect-related status (so I don't get a free immunity strip while the reflect is hot).
- As an oncall engineer, I never see a "lua expected list got map" error from a downstream consumer because every status-event array field marshals as `[]` even when empty.

## 4. Functional Requirements

### 4.1 Reflect — atlas-monsters apply path

- **FR-4.1.1** When `executeStatBuff` applies a status of category `SkillCategoryReflect` (skill types `WEAPON_REFLECT` and `MAGIC_REFLECT`), it MUST populate structured reflect metadata onto the resulting `StatusEffect`: `ReflectKind` (`PHYSICAL` | `MAGICAL`), `ReflectPercent` (from `sd.X()`), `ReflectRange` (from `sd.Y()`, in field-units), and `ReflectMaxDamage` (from `sd.X()` cap if Y is unused — see §6.2 for the exact source-field mapping).
- **FR-4.1.2** The reflect status MUST flow through the same `StatusEffectAppliedBody` event used for any other status, but the body MUST carry the structured reflect fields (see §5.1).
- **FR-4.1.3** Re-applying a reflect status of the same kind to a monster that already has it active SHOULD reject the second apply (existing picker gate at `picker.go:185-191` is sufficient; verify with a regression test).

### 4.2 Reflect — atlas-channel consumer mirror

- **FR-4.2.1** atlas-channel MUST maintain an in-process per-tenant `monster.StatusMirror` keyed by `(uniqueId)` that stores active status effects on each visible monster. The mirror is populated by the existing `handleStatusEffectApplied`, `handleStatusEffectExpired`, and `handleStatusEffectCancelled` handlers and pruned by `handleStatusEventDestroyed` / `handleStatusEventKilled`.
- **FR-4.2.2** Mirror entries MUST decode the structured reflect metadata from the event body and store it alongside the status name.
- **FR-4.2.3** Lookup `Mirror.GetReflect(uniqueId, kind) (ReflectInfo, bool)` MUST return `false` if the monster has no active reflect of that kind, and `true` with the structured data otherwise. O(1) lookup; reads under `sync.RWMutex`.
- **FR-4.2.4** The mirror MUST clear when an `EventStatusDestroyed` or `EventStatusKilled` arrives for a uniqueId.

### 4.3 Reflect — atlas-channel attack handler

- **FR-4.3.1** In `socket/handler/character_attack_common.go`, replace the `TODO Monster Weapon Atk Reflect` and `TODO Monster Magic Atk Reflect` placeholders. Per damage entry, before the entry is applied to the monster:
  - Resolve the `ReflectKind` from the attack type (Close-range / Ranged → `PHYSICAL`; Magic → `MAGICAL`).
  - Consult `Mirror.GetReflect(monsterId, kind)`. If absent: continue normally.
  - If present and the attacker's distance to the monster on the X axis is within `ReflectRange`: compute `reflected := damage * ReflectPercent / 100`, capped at `ReflectMaxDamage`; produce `EventStatusDamageReflected` with `{CharacterId: attackerId, ReflectDamage: reflected, MonsterUniqueId: monsterId}` and set the entry's monster damage to zero (no `DAMAGED` event for that entry).
- **FR-4.3.2** Damage zeroing MUST occur before the existing `damage_taken` write so the monster's HP is unaffected by the reflected entry.
- **FR-4.3.3** The existing `handleDamageReflected` consumer (`consumer.go:371-384`) already applies HP loss to the character and is unchanged.

### 4.4 Venom — 3-stack model

- **FR-4.4.1** Atlas-monsters MUST expose three internal venom slot statuses, `VENOM_1`, `VENOM_2`, `VENOM_3`, in addition to the existing wire-side `VENOM`.
- **FR-4.4.2** When a venom apply arrives:
  1. If any slot is empty (no active StatusEffect with that slot key), apply into the lowest-numbered free slot.
  2. If all three are occupied, replace the slot with the **earliest** `expiresAt` (oldest-first replacement).
- **FR-4.4.3** Each slot's `StatusEffect` MUST carry the per-attacker snapshot stats inside `Statuses`: `{VENOM_N: snapshotDamagePerTick}` plus auxiliary keys `{VENOM_N_LUCK: luck, VENOM_N_MATK: matk, VENOM_N_SOURCE: characterId}`. The damage-per-tick computation `0.1-0.2 * Luck * MagicAttack` (uniform random per apply) is performed at apply time, then frozen.
- **FR-4.4.4** Each venom slot ticks independently in `StatusExpirationTask.processDoTTick`. The combined tick damage is the sum of all active slots, applied as a single `DAMAGED` event, then capped by the kill-prevention rule (`currentHp - 1`).
- **FR-4.4.5** On the wire, atlas-channel MUST translate `VENOM_N` to a single `VENOM` `MonsterStatSet` / `MonsterStatReset` entry. Subsequent applies of additional slots MUST NOT re-broadcast `VENOM` (idempotent re-broadcast of the same wire stat is acceptable but not required). When the **last** slot expires/cancels, atlas-channel MUST emit `MonsterStatReset` for `VENOM`.
- **FR-4.4.6** A new `VENOM_N` apply event flowing through `handleStatusEffectApplied` MUST be coalesced — the consumer translates `VENOM_N` keys to a single `VENOM` stat in the temporary-stat encoding.

### 4.5 Poison — replacement model (re-affirm)

- **FR-4.5.1** Re-applying `POISON` to a monster that already has `POISON` MUST replace the existing `StatusEffect` (new `expiresAt`, new `lastTick = now`, new source character / skill level). The poison damage formula `maxHp / (70 - skillLevel)` continues to apply post-replacement.
- **FR-4.5.2** Test must cover: a poison applied at skill level 5 then re-applied at skill level 10 ticks at the new formula immediately after replacement.

### 4.6 Mist zones — atlas-maps area effect

- **FR-4.6.1** Add a new `mist` (or generic `area-effect`) domain to atlas-maps with an immutable `Mist` model: `Id (uuid.UUID)`, `Field`, `OwnerType` (`MONSTER` | `CHARACTER` reserved for future), `OwnerId`, `Origin (X,Y)`, `LtX/LtY/RbX/RbY` (relative bounding box), `Disease` (e.g. `POISON`), `DiseaseValue int32`, `DiseaseDuration time.Duration`, `Duration time.Duration`, `TickInterval time.Duration`, `CreatedAt`, `ExpiresAt`, `LastTick`.
- **FR-4.6.2** A `MistRegistry` (singleton, `sync.Once`, `sync.RWMutex`-guarded) maintained per tenant + field stores active mists.
- **FR-4.6.3** A `MistTickTask` runs at 1 s cadence. Per active mist:
  - If expired: remove from registry, emit `MistDestroyed` event.
  - Else: list character IDs in `mist.Field()`, filter to those whose `(x,y)` lies within absolute bounds, and for each, **re-apply** the disease via the existing `EnvCommandTopicCharacterBuff` apply-disease command (resets duration on re-apply per existing buff replacement semantics).
- **FR-4.6.4** Atlas-maps MUST publish `EVENT_TOPIC_MIST` with `MIST_CREATED` and `MIST_DESTROYED` events. Body fields: `MistId`, `OwnerType`, `OwnerId`, `MapId`, `Instance`, `Origin`, `Bounds`, `Duration`. (Disease metadata is internal; not broadcast.)
- **FR-4.6.5** Atlas-monsters' new `executeMist(m, sd)` produces a `MIST_CREATE` command on `EVENT_COMMAND_TOPIC_MIST` with `{OwnerType: MONSTER, OwnerId: m.UniqueId, MapId: m.MapId, Instance: m.Instance, Origin: (m.X, m.Y), Bounds: {LtX/LtY/RbX/RbY from sd}, Disease: POISON, DiseaseValue: sd.X, DiseaseDuration: sd.Duration → reuse existing field, Duration: sd.Duration*sec}`.
- **FR-4.6.6** Atlas-channel consumes `EVENT_TOPIC_MIST` and broadcasts `AffectedAreaCreated` (writer to be added if not present — see §6.4) on `MIST_CREATED` and `AffectedAreaRemoved` on `MIST_DESTROYED` to all sessions in the field.
- **FR-4.6.7** The `AREA_POISON` exclusion at `picker.go:144-149` MUST be removed once `executeMist` is wired.
- **FR-4.6.8** Mist disease re-apply MUST NOT bypass Holy Shield: the existing `HasImmunity` check in atlas-buffs already handles this and is exercised on every tick.

### 4.7 Player poison HP DoT — atlas-buffs `PoisonTick` task

- **FR-4.7.1** Add a new `tasks.PoisonTick` task (sibling of `tasks.Expiration`) running on a 1 s cadence (configurable via env, default 1000 ms).
- **FR-4.7.2** Per tick: walk `Registry.GetPoisonCharacters(ctx)` for each tenant. For each entry, check `GetLastPoisonTick`; if ≥ 1 s elapsed (or never ticked), produce a `CHARACTER_DAMAGE` command on the existing character damage topic with `{CharacterId, Amount: entry.Amount, Source: POISON}` then call `UpdatePoisonTick(now)`.
- **FR-4.7.3** Damage application MUST respect Holy Shield (apply path already gates on `HasImmunity` so no poisoned entry is created in the first place — verify with a regression test).
- **FR-4.7.4** If the character is offline (no session), the produce SHOULD still occur; downstream consumers handle absence.
- **FR-4.7.5** `tasks.Expiration` is unchanged (continues to expire all buffs including poison). The two tasks coexist; an expired poison buff stops appearing in `GetPoisonCharacters` and therefore stops ticking.

### 4.8 Immunity mutual exclusion

- **FR-4.8.1** When `executeStatBuff` applies `PHYSICAL_IMMUNE` to a monster that has an active `MAGIC_IMMUNE`, the existing `MAGIC_IMMUNE` MUST be cancelled (full status-cancel flow including the `EventStatusEffectCancelled` event) before the new `PHYSICAL_IMMUNE` is applied. Symmetric for `MAGIC_IMMUNE` displacing `PHYSICAL_IMMUNE`.
- **FR-4.8.2** This rule MUST run **before** the existing already-active gate at `processor.go:540-543` so that a stale opposite-immunity does not block the new one.

### 4.9 Dispel guard against active reflect

- **FR-4.9.1** When a player skill produces a `STATUS_CANCEL` for a monster's status, atlas-monsters MUST refuse the cancel if both:
  1. The status being cancelled is *not* itself a reflect (reflect cancel by player is normal expiration).
  2. The monster currently has an active reflect of the same damage kind as the player's attack class (PHYSICAL skills cannot dispel while WEAPON_REFLECT is active; MAGIC skills cannot while MAGIC_REFLECT is active).
- **FR-4.9.2** The check lives in atlas-monsters' status cancel handler (or processor). Source skill kind is derived from the existing `SourceSkillId` on the cancel command — atlas-monsters has the skill data already via the same provider used by the picker.
- **FR-4.9.3** Rejected dispels MUST be logged at debug level and counted in metrics (existing logger is acceptable; no new metric required).

### 4.10 `cjson` empty-array safety

- **FR-4.10.1** Audit every status-event body in `services/atlas-monsters/atlas.com/monsters/monster/kafka.go`, including types added in this task, for any slice fields that may marshal as `[]` when empty. Apply the established fix pattern from commits `2c0ac23f2` and `afc3bd28a` (typically a custom `MarshalJSON` or sentinel "always-non-nil slice" guard).
- **FR-4.10.2** Add a regression test per type that round-trips an empty slice and asserts `[]` (not `null` or `{}`) in the JSON output.

## 5. API Surface

### 5.1 Kafka — extend `StatusEffectAppliedBody` (atlas-monsters)

Existing body in `kafka/message/monster/...` is extended with structured reflect fields. `omitempty` is **not** used — empty values must serialize predictably for cjson consumers.

```
StatusEffectAppliedBody {
  EffectId         uuid.UUID
  SourceType       string
  SourceCharacterId uint32
  SourceSkillId    uint16
  SourceSkillLevel uint16
  Statuses         map[string]int32
  Duration         int64    // ms
  TickInterval     int64    // ms
  // NEW — populated only for reflect statuses; zero values otherwise.
  ReflectKind      string   // "" | "PHYSICAL" | "MAGICAL"
  ReflectPercent   int32
  ReflectRange     int32
  ReflectMaxDamage int32
}
```

### 5.2 Kafka — new topic `EVENT_TOPIC_MIST` (atlas-maps)

```
MistEvent[T] {
  Tenant   tenant.Model
  WorldId  byte
  ChannelId byte
  MapId    uint32
  Instance uuid.UUID
  MistId   uuid.UUID
  Type     "MIST_CREATED" | "MIST_DESTROYED"
  Body     T
}

MistCreatedBody {
  OwnerType string // "MONSTER" | "CHARACTER"
  OwnerId   uint32
  Origin    Point  // {X int16, Y int16}
  LtX,LtY,RbX,RbY int16
  Duration  int64  // ms
}

MistDestroyedBody {
  Reason string // "EXPIRED" | "CANCELLED"
}
```

### 5.3 Kafka — new command topic `EVENT_COMMAND_TOPIC_MIST` (atlas-maps)

```
MistCommand[T] {
  Tenant   tenant.Model
  Type     "MIST_CREATE" | "MIST_CANCEL"
  Body     T
}

MistCreateBody {
  WorldId   byte
  ChannelId byte
  MapId     uint32
  Instance  uuid.UUID
  OwnerType string
  OwnerId   uint32
  Origin    Point
  LtX,LtY,RbX,RbY int16
  Disease         string // e.g. "POISON"
  DiseaseValue    int32
  DiseaseDuration int64  // ms — passed to apply-disease command on each tick
  Duration        int64  // ms — total mist lifetime
  TickIntervalMs  int64  // default 1000
}

MistCancelBody {
  MistId uuid.UUID
}
```

### 5.4 Kafka — extend `StatusEventDamageReflectedBody` (already exists)

No change to the body itself. New thing: this event is now actually produced — by the atlas-channel attack handler (see §4.3).

### 5.5 No REST changes

This task adds no new REST endpoints. atlas-channel reflect lookups are local (mirror); mist applies flow over Kafka.

## 6. Data Model

### 6.1 atlas-monsters — `StatusEffect` extension

```
StatusEffect {
  // existing fields …
  // NEW
  reflectKind      string // "" | "PHYSICAL" | "MAGICAL"
  reflectPercent   int32
  reflectRange     int32
  reflectMaxDamage int32
}
```

Builder + getters follow the immutable-model pattern. New constructor `NewReflectStatusEffect(...)` for clarity, plus an extended `NewStatusEffect` overload. Existing calls keep their default zero values.

### 6.2 Source-field mapping for reflect

Until concrete WZ inspection, the plan phase MUST verify against actual mob skill data and pick one mapping; the PRD locks the structure, not the bit-for-bit mapping. Default working assumption (subject to verification in plan phase):

| Source           | Target              |
|------------------|---------------------|
| `sd.X()`         | `ReflectPercent`    |
| `sd.Y()`         | `ReflectRange`      |
| (constant 32767) | `ReflectMaxDamage`  |

If WZ inspection shows `sd.Y()` is the reflected damage cap rather than the range, the plan phase swaps `ReflectRange` and `ReflectMaxDamage` and uses skill bounding-box as the radius.

### 6.3 atlas-channel — `monster.StatusMirror`

```
type ReflectInfo struct {
  Kind             string
  Percent, Range   int32
  MaxDamage        int32
  ExpiresAt        time.Time
}

type StatusMirror struct {
  mu       sync.RWMutex
  byMonster map[tenantKey]map[uint32]map[string]StatusEntry // status name → entry
}
```

Per-tenant scoping uses the existing `tenantKey` pattern in atlas-channel registries. No persistence — purely in-memory mirror of Kafka events.

### 6.4 atlas-maps — `mist.Mist`

Immutable model + builder; registry indexed by `(tenant, mistId)` with secondary index `(tenant, field) → []mistId`.

```
type Mist struct {
  id              uuid.UUID
  field           field.Model
  ownerType       string
  ownerId         uint32
  origin          Point
  ltX,ltY,rbX,rbY int16
  disease         string
  diseaseValue    int32
  diseaseDuration time.Duration
  duration        time.Duration
  tickInterval    time.Duration
  createdAt, expiresAt, lastTick time.Time
}
```

Absolute bounds for the in-zone test: `[origin.X+ltX, origin.X+rbX] × [origin.Y+ltY, origin.Y+rbY]`. Verify left/right vs. top/bottom convention from existing skill bounding-box code in atlas-monsters (`processor.go:683-687`).

### 6.5 Packet — `AffectedArea` writers (libs/atlas-packet)

Confirm in plan phase whether v83 mist packet writers exist; if not, add `AffectedAreaCreated` and `AffectedAreaRemoved` to `libs/atlas-packet/map/clientbound/` (or wherever similar map-object writers live). Reference Cosmic v83 affected-area opcodes.

## 7. Service Impact

| Service | Changes |
|---|---|
| **atlas-monsters** | Extend `StatusEffect` with reflect metadata. Venom 3-slot apply logic + tick aggregation. Immunity mutual exclusion in `executeStatBuff`. New `executeMist` producing mist-create commands. Dispel guard against active reflect. cjson audit on existing status events. Remove `AREA_POISON` picker exclusion. |
| **atlas-channel** | New `monster.StatusMirror` populated by existing status consumers. Reflect check + damage-zeroing in `character_attack_common.go` (replace 2 TODOs at `:144-145`). Translate `VENOM_N` → `VENOM` in `handleStatusEffectApplied/Expired/Cancelled`. New mist consumer broadcasting `AffectedAreaCreated/Removed`. |
| **atlas-buffs** | New `tasks.PoisonTick` task running at 1 s cadence, producing character-damage commands for each entry from `Registry.GetPoisonCharacters`. Wire into `main.go` task loop. |
| **atlas-maps** | New `mist` domain: model + builder, registry, processor (`Create`, `Destroy`, `ByFieldProvider`), `MistTickTask`, command consumer (`MIST_CREATE`/`MIST_CANCEL`), event producer (`MIST_CREATED`/`MIST_DESTROYED`). |
| **libs/atlas-constants** | Add `VENOM_1`, `VENOM_2`, `VENOM_3` status names plus an `IsVenomSlot(name) bool` helper. Add reflect kind constants if not already centralised. |
| **libs/atlas-packet** | Add `AffectedAreaCreated`/`AffectedAreaRemoved` writers + tests if not already present. |

No frontend (atlas-ui) changes.

## 8. Non-Functional Requirements

### 8.1 Performance

- Reflect check is on every damage entry of every player attack. Mirror lookup MUST be O(1) under a read lock — no maps-of-maps that require multi-hop traversal beyond `tenant → uniqueId → status name`.
- Mist tick is per-second per-mist; per-tick work is `O(charactersInField × activeMists)`. Acceptable while N is small (typical map: ≤ 1-2 active mists, ≤ 30 characters). Plan phase MUST add an upper-bound assertion in tests (e.g. 10 mists × 50 characters ≤ 50 ms per tick on dev hardware) and revisit if exceeded.
- `PoisonTick` is per-second per-tenant per-poisoned-character. Bounded by simultaneous-online players × poisoned-fraction; well within budget.

### 8.2 Multi-tenancy

- Every new registry (`StatusMirror`, `MistRegistry`) MUST scope by `tenant.Model` exactly like existing patterns (`monster.GetMonsterRegistry()`).
- Every new producer / consumer MUST use the existing tenant header parser.
- Every new task MUST iterate tenants and call `tenant.WithContext(...)` before producing.

### 8.3 Observability

- Reflect zero-out MUST log at debug level: `"reflect: char [%d] hit on monster [%d] reflected %d damage."`
- Mist tick MUST log at debug level: `"mist: zone [%s] applied %s to %d characters."`
- Dispel rejection MUST log at debug level (FR-4.9.3).
- No new metrics required for v1; revisit if reflect or mist proves operationally noisy.

### 8.4 Concurrency / safety

- `StatusMirror` reads under `RWMutex.RLock`; writes (apply / expire / cancel / destroy) under `RWMutex.Lock`.
- `MistRegistry` follows the same pattern as `monster.GetMonsterRegistry()`.
- Venom slot allocation MUST be atomic — apply-and-find-slot under one lock acquisition to avoid two concurrent applies racing into the same slot.

### 8.5 Security

- No new external surfaces. Mist commands flow over internal Kafka topics.
- Reflect-induced HP damage is bounded by `ReflectMaxDamage` to avoid one-shot griefing scenarios from misconfigured WZ data.

## 9. Open Questions

1. **`AffectedArea` packet opcodes** — confirm whether `libs/atlas-packet/map/clientbound/` already has an affected-area writer for v83. Plan phase verifies; if missing, add as part of the mist task chain.
2. **`sd.X()` vs `sd.Y()` for reflect** — locked structure, deferred bit-for-bit mapping to plan phase (§6.2).
3. **PoisonTick character damage producer** — confirm the existing topic / payload that atlas-character (or whichever service is authoritative for character HP) consumes for "monster did N damage." Plan phase to identify; if no precedent exists for player poison ticks, reuse `EVENT_COMMAND_TOPIC_CHARACTER_DAMAGE` if present, otherwise plumb a small command on an existing buff topic.
4. **Mist on instance maps** — confirm `Instance` UUID propagates correctly through `MistCreatedBody` and that consumers honour it for instance-scoped broadcast (mirrors monster instance handling).

## 10. Acceptance Criteria

A reviewer can verify completion by:

1. **Reflect end-to-end** — Spawn a Pap, apply Weapon Reflect (GM command if needed). Have a player melee attack within range; observe (a) Pap's HP unchanged, (b) attacker's HP drops by `damage * reflectPercent / 100`, (c) `MonsterDamage` is **not** announced for that hit, (d) the corresponding `StatusEventDamageReflected` event is observable in the broker.
2. **Reflect range gate** — Same setup, attacker outside range: damage applies normally.
3. **Venom 3-stack** — Apply Venom three times in rapid succession from the same player; observe (a) three `STATUS_APPLIED` events with `VENOM_1`, `VENOM_2`, `VENOM_3`, (b) `MonsterStatSet` for `VENOM` once on the wire, (c) DoT tick damage equals the sum of all three slots, (d) on the 4th apply the slot with the earliest `expiresAt` is replaced.
4. **Venom expire collapse** — Let two of three slots expire; on the last expire, observe one `MonsterStatReset` for `VENOM`.
5. **Mist** — A mist-firing monster (e.g. AREA_POISON-bearing skill) can be picked, the mist is broadcast as `AffectedAreaCreated` to all sessions in the field, characters standing in the zone receive the POISON disease, the buff service ticks their HP via `PoisonTick`, the mist disappears at duration with `AffectedAreaRemoved`. Holy Shield characters in the zone do not receive the disease.
6. **Picker un-skip** — `picker_test.go::TestPicker_AreaPoisonExcluded` is removed (or inverted) and the picker can fire `AREA_POISON` skills.
7. **Player poison DoT** — A player poisoned by any means observes 1 s HP ticks until the buff expires, with the per-second damage equal to `Registry.GetPoisonCharacters` `Amount`.
8. **Immunity mutual exclusion** — Apply MAGIC_IMMUNE then PHYSICAL_IMMUNE; observe a `STATUS_CANCELLED` event for MAGIC_IMMUNE before the `STATUS_APPLIED` for PHYSICAL_IMMUNE.
9. **Dispel guard** — A monster running WEAPON_REFLECT cannot have its other statuses dispelled by a physical-class player skill; logs include the rejection at debug level. Magical-class dispel still works.
10. **cjson** — A JSON round-trip test for every status event with a slice field asserts `[]` not `null` / `{}` for the empty case.
11. **Test coverage** — New tests for: venom snapshot + 3-slot replacement, poison kill-cap (existing — re-affirm), reflect mirror apply/expire/destroy, mist tick disease re-apply, mist Holy Shield bypass, immunity mutual exclusion, dispel guard, PoisonTick.
12. **No regressions** — `go test ./...` green for all five touched services. Picker / cooldown / aggro tests untouched and still pass.
