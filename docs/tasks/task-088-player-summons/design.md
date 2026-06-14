# Player Summons — Design

Status: Proposed
Created: 2026-06-12
Inputs: `prd.md`, `discovery.md`
Baseline: Cosmic (`~/source/Cosmic`, v83). Atlas paths relative to the worktree root.

---

## 1. Summary

Stand up a new **`atlas-summons`** service that owns the lifecycle of owner-bound
player summons — spawn, move, attack, take damage, despawn — modeled structurally
on `atlas-monsters` (Redis-backed per-tenant registry, shared object-id allocator,
Kafka command/event topics, leader-elected sweep tasks). The core engine is
**version-agnostic**; version variance is confined to six packet encoders/decoders
in `libs/atlas-packet/summon/` and per-version opcode entries in the tenant socket
templates.

`atlas-channel` is the thin packet edge: its skill-cast handler routes summon skill
ids to a spawn command, three new inbound handlers relay client move/attack/damage
packets as summon commands, and a new consumer of `EVENT_TOPIC_SUMMON_STATUS`
broadcasts six clientbound summon packets to in-range sessions — exactly mirroring
the existing `EVENT_TOPIC_MONSTER_STATUS` → broadcast path.

The feature reuses every existing cross-service seam (monster `DAMAGE`, character
`CHANGE_HP`, buff `APPLY`, character lifecycle events) and adds **one** new
cross-service capability: a puppet aggro-redirect command pair on
`COMMAND_TOPIC_MONSTER`, which does not exist today (resolves Q1).

---

## 2. Why a new service (not a fold-in)

Summons are owner-bound objects with a lifecycle orthogonal to monsters: they are
created by skill casts, despawn on owner lifecycle transitions, enforce
per-owner/per-class uniqueness, and run owner-targeted aura timers. Folding them
into `atlas-monsters` would entangle two unrelated aggro/lifecycle models in one
registry. A dedicated service keeps `atlas-monsters` focused, gives summons their
own Redis namespace and sweep cadence, and matches the established Atlas convention
of one service per object domain. The object-id space is still **shared** at the
field level (Q4): both services draw from `libs/atlas-object-id` (MinId 1,000,000),
so summon and monster oids never collide on a map.

---

## 3. Resolved open questions

| # | Question | Resolution |
|---|---|---|
| **Q1** | Puppet aggro command exists? | **No.** `atlas-monsters` has no inbound aggro/control-redirect command (control is server-assigned via `StartControl`). Add a new `ADD_PUPPET`/`REMOVE_PUPPET` command pair on `COMMAND_TOPIC_MONSTER` + a per-field puppet set that biases controller selection toward an in-vicinity puppet owner (Cosmic `MonsterAggroCoordinator`, vicinity `distanceSq < 177777`). See §9. |
| **Q2** | Server-scheduled attacks? | **All 21 are client-driven** in Cosmic (`SummonDamageHandler` reads the client packet). The server schedules **only** Beholder's heal/buff timers. ⇒ **no attack-interval field** is added to `atlas-data`. |
| **Q3** | Beholder buff source-id range? | Buff `SourceId` is `int32` and MUST be the positive, real skill id (`HEX_OF_THE_BEHOLDER` = `1320009`). ~~Originally negated for collision-avoidance~~ — **CORRECTED (live v83 testing):** the client writes `sourceId` into the give-buff packet as the per-stat `rSkillID` and calls `GetSkillTemplate(rSkillID)` to render the buff icon; a negative id resolves to null and **crashes the client**. There is no real collision (the hex skill is only ever applied by the Beholder), so the positive skill id is both safe and correct. See §10. |
| **Q4** | Shared object-id pool? | **Yes.** Reuse `libs/atlas-object-id` (per-tenant Redis allocator, MinId 1,000,000), same pool as monsters/drops/reactors. |
| **Q5** | Per-version roster gaps? | **Graceful no-op** (FR-1.3): a cast whose skill id has no roster entry logs at debug and spawns nothing. No hard error. |
| **Q6** | Aerial Strike / Battleship? | **Out of scope.** Aerial Strike (`5221003`) is a dead constant (not a `Summon`); Battleship (`5221006`) is a mount. Neither spawns a summon object. |

A seventh decision, **where summon-attack damage is validated**, refines PRD §6:
validation lives in **`atlas-summons`** (it owns the summon's skillId/level and is
the natural emitter of the credited monster `DAMAGE`). Consequently the
PRD's tentative "add `weaponAttack`/`magicAttack` getters to the *channel-side*
effect model" is **not** done; instead `atlas-summons` carries its own skill-effect
data client exposing those fields. See §8.3.

---

## 4. Component architecture

`atlas-summons` mirrors `services/atlas-monsters/atlas.com/monsters/`:

```
services/atlas-summons/atlas.com/summons/
  main.go                      boot: logger → redis → InitIdAllocator/InitRegistry
                               (sync.Once) → teardown mgr → tracing → consumers →
                               handlers → REST → leader-elected tasks → teardown
  summon/
    model.go                   immutable Model (private fields + getters)
    builder.go                 Clone() + fluent ModelBuilder
    registry.go                Redis registry + field index + owner index (sync.Once,
                               RWMutex via atlasredis.Registry)
    id_allocator.go            wraps objectid.NewRedisAllocator (Allocate/Release)
    processor.go               Interface + Impl; Spawn/Move/Attack/Damage/Despawn,
                               pure Method(mb) + MethodAndEmit() split
    resource.go / rest.go      JSON:API "summons" resource + GET handlers
    kafka.go                   EVENT_TOPIC_SUMMON_STATUS event structs + type consts
    producer.go                event providers (created/moved/attacked/damaged/destroyed)
  world/resource.go            GET .../maps/{mapId}/instances/{instanceId}/summons
  data/                        skill-effect data client (duration/x/watk/matk/
                               monsterStatus/prop) — reads atlas-data REST
  kafka/consumer/
    summon/                    COMMAND_TOPIC_SUMMON (SPAWN/MOVE/ATTACK/DAMAGE)
    character/                 EVENT_TOPIC_CHARACTER_STATUS (despawn cascade)
  tasks/                       expiry sweep + beholder-aura sweep (leader-elected)
  leaderconfig.go
  logger/init.go
```

Boot order, registry singleton pattern, id-allocator wrapping, and leader-election
gating are copied verbatim from `atlas-monsters/main.go:53-139`.

---

## 5. Data model

Redis-backed, per-tenant, ephemeral (no relational migration), consistent with
`atlas-monsters`.

| Field | Type | Notes |
|---|---|---|
| `summonId` (object id) | `uint32` | From `libs/atlas-object-id`. |
| `ownerCharacterId` | `uint32` | Owner. |
| `skillId` | `uint32` | Summon skill id (Appendix A). |
| `skillLevel` | `byte` | Drives effect values. |
| `summonType` | enum | `puppet` / `attacker` / `buff-aura`. |
| `movementType` | enum | `stationary(0)` / `follow(1)` / `circle-follow(3)`. |
| field (`world`/`channel`/`map`/`instance`) | `field.Model` | Owner's field at spawn. |
| `x` / `y` | `int16` | Position (updated by move packets for non-stationary). |
| `hp` / `maxHp` | `int32` | Puppets and Beholder; 0 for HP-less attackers. |
| `spawnTime` / `expiresAt` | `time` | Duration-driven expiry. |
| `nextHealAt` / `nextBuffAt` | `time` | Beholder only; aura timers. |
| `healAmount` | `int16` | Beholder only; snapshot of `AURA_OF_BEHOLDER` effect `hp`. |
| `buffChanges` / `buffSourceId` / `buffLevel` / `buffDuration` | snapshot | Beholder only; snapshot of `HEX_OF_BEHOLDER` effect. |

**Redis keys** (mirroring the monster registry):
- store: `atlas:summon:<tenantId>:<summonId>`
- field index (broadcast range): `atlas:summon-map:<tenantId>:<w>:<c>:<m>:<instance>`
- **owner index** (re-cast / conflict-cancel / despawn cascade):
  `atlas:summon-owner:<tenantId>:<characterId>`

The owner index is the one structural addition over the monster registry; it makes
FR-2.4/FR-2.5 (replace same-skill, cancel conflicting class) and FR-7.2 (despawn all
of a character's summons) O(owner) instead of a field scan. All Redis access routes
through `libs/atlas-redis` (redis-key-guard clean).

---

## 6. Roster classification (FR-1.1/1.2)

Movement type and summon type are **not** WZ data in Cosmic — they are hard-coded in
`StatEffect`. Atlas needs the same skill-id → `{summonType, movementType}` table in
a place both `atlas-summons` (authoritative behavior) and `atlas-channel` (the
"is this a summon?" predicate) can read.

**Decision:** add `libs/atlas-constants/summon/roster.go` — a static table of the 21
v83 entries keyed by the skill-id constants already in
`libs/atlas-constants/skill/constants.go`, exposing:
- `Lookup(skillId) (Entry, bool)` — full `{summonType, movementType}` (atlas-summons)
- `IsSummonSkill(skillId) bool` — predicate (atlas-channel)

This satisfies FR-1.2 (adding a summon = one table row, no core-engine change) and
keeps the classification out of call sites. Because it is a subpackage of the
already-vendored `atlas-constants` module, it needs **no** new `go.work` entry and
**no** Dockerfile `COPY` lines.

Puppet ids (`3111002`, `3211002`, `13111004`) and stationary ids
(`+ 5211001`, `5220002`) match Cosmic `Summon.isPuppet()` / `isStationary()`.
Movement classes per Cosmic `StatEffect.java:1766-1797`: see Appendix A.

---

## 7. Spawn & despawn lifecycle

### 7.1 Spawn (FR-2.x)

```
client casts summon skill
  → atlas-channel character_skill_use.go: if summon.IsSummonSkill(skillId)
      emit COMMAND_TOPIC_SUMMON SPAWN{ownerCharId, skillId, level, field, x, y}
  → atlas-summons handleSpawn:
      roster.Lookup(skillId) miss → debug log, return (FR-1.3, Q5)
      via owner index: remove same-skill instance (FR-2.4) and the
        conflicting-mobility-class instance (FR-2.5) — each removal runs the full
        despawn (event + oid release + ADD/REMOVE_PUPPET + timer stop)
      allocate oid; fetch effect (duration→expiresAt, x→hp); Beholder hp = x+1
      persist (store + field index + owner index)
      emit EVENT_TOPIC_SUMMON_STATUS CREATED
      if puppet: emit COMMAND_TOPIC_MONSTER ADD_PUPPET (field, ownerCharId, x, y)
      if Beholder: snapshot aura/hex effects, set nextHealAt/nextBuffAt (§10)
  → atlas-channel consumes CREATED → broadcast SummonSpawn to ForSessionsInMap
```

Spawn position is the caster's position at cast time, resolved channel-side and
carried in the SPAWN command (owner `GET /location` returns field only, not x/y).

The conflicting-class rule, restated from Cosmic `StatEffect.java:1024-1029`: a new
**stationary** summon cancels the owner's existing **stationary** summon; a new
**non-stationary** summon cancels the owner's existing **non-stationary** summon.

### 7.2 Despawn (FR-7.x)

Every despawn path runs one `despawn(summon, mode)` routine: emit `DESTROYED`
(animated vs instant per mode), release the oid, drop the field/owner index entries,
emit `REMOVE_PUPPET` if puppet, and stop Beholder timers if Beholder.

| Trigger | Source |
|---|---|
| Expiry at `expiresAt` | leader-elected expiry sweep task (scan field index) |
| Logout / channel change / map change | consume `EVENT_TOPIC_CHARACTER_STATUS` `LOGOUT` / `CHANNEL_CHANGED` / `MAP_CHANGED`; despawn all via owner index |
| Re-cast same skill / conflicting class | inline in spawn (§7.1) |
| Puppet HP ≤ 0 | inline in damage handler (§9) |

Object ids are released on **every** path (FR-7.4) because all paths funnel through
the single `despawn` routine.

---

## 8. Attacker behavior (FR-4.x)

All attacker summons are client-driven (Q2). Round-trip mirrors the monster pattern;
the rebroadcast carries clamped damage so other clients render validated numbers.

```
client → atlas-channel SummonAttackHandle(summonOid, direction, [{monsterOid, dmg}])
  → emit COMMAND_TOPIC_SUMMON ATTACK{summonOid, senderCharId, direction, targets[]}
  → atlas-summons handleAttack:
      load summon by oid; reject if ownerCharacterId != senderCharId (drop+info, §11)
      compute maxPerHit (§8.3); clamp each target; if clamped → autoban alert (§8.4)
      for each target: emit COMMAND_TOPIC_MONSTER DAMAGE{characterId: owner, damages}
        (credit flows to owner via existing DAMAGED/KILLED DamageEntries — FR-4.2)
      if effect carries MonsterStatus and prop succeeds: include stun/freeze (FR-4.4)
      emit EVENT_TOPIC_SUMMON_STATUS ATTACKED{direction, clamped targets}
      if skillId == Gaviota (5211002): despawn self (FR-4.5)
  → atlas-channel consumes ATTACKED → broadcast SummonAttack to other in-range sessions
```

### 8.1 Owner credit

The monster `DAMAGE` command body already carries `CharacterId` (the creditor);
`atlas-monsters` aggregates `DamageEntries[]` by character and emits them on
`DAMAGED`/`KILLED`. Sending the owner's `characterId` is sufficient for XP/drops/
quest credit — **no `atlas-monsters` change** for the attacker path.

### 8.2 Monster status

Stun/freeze ride the existing monster status mechanism. The summon effect's
`MonsterStatus()` map + `Prop()` proc chance drive it; apply with the same
`APPLY_STATUS` command `atlas-channel` already uses for mob-affecting skills.

### 8.3 Server-side damage ceiling (FR-4.3)

Port Cosmic `SummonDamageHandler.calcMaxDamage` (`:123-145`) verbatim:

- `magic := effect.weaponAttack == 0`
- **magic:** `matk = max(owner.totalMagic, 14)`;
  `max = owner.maxBaseMagicDamage(matk) * 0.05 * effect.magicAttack`
- **physical:** `watk = max(owner.totalWatk, 14)`;
  `base = owner.maxBaseDamage(watk, weaponType)`;
  `mod = base >= 438 ? 0.054 : 0.077`; `max = base * mod * effect.weaponAttack`

This requires the owner's live combat stats (`effective-stats` REST, as
`atlas-character` already uses) and a weapon-type-aware base-damage computation.
**Planning risk:** porting `calculateMaxBaseDamage`/`MagicDamage` faithfully is the
largest single piece of behavioral logic here; planning must first check whether
Atlas already computes a per-hit ceiling for player attacks that can be reused. If a
faithful port is deferred, a conservative ceiling still clamps egregious values, with
the exact formula as the parity target — but this must be an explicit, logged
limitation, never a silent `// TODO`.

`atlas-summons` owns this because it holds the summon's `skillId`/`level`; it reads
the effect via its own `data/` client (duration/x/`weaponAttack`/`magicAttack`/
monsterStatus/prop) rather than adding getters to the channel-side effect model.

### 8.4 Autoban

On clamp, emit a structured warning (owner, skillId, mob, reported vs max) and, if
Atlas exposes a cheat-report/autoban topic, a report. Damage is clamped regardless.
**Planning item:** confirm whether an Atlas autoban surface exists; if not, the
warning log is the honest floor (Cosmic clamps and continues — does not disconnect).

---

## 9. Puppet behavior (FR-5.x) — the one cross-service addition

Puppets are stationary, so their position is fixed at spawn (no move packets). The
gap is monster aggro redirect (Q1): no inbound monster command exists.

**Add to `COMMAND_TOPIC_MONSTER`:**

- `ADD_PUPPET{worldId, channelId, mapId, instance, ownerCharacterId, x, y}`
- `REMOVE_PUPPET{worldId, channelId, mapId, instance, ownerCharacterId}`

`atlas-monsters` maintains a per-field puppet set (Redis) keyed like the monster map
index. When the set changes (or a monster (re)selects a controller), controller
selection **biases toward a puppet owner whose puppet is within Cosmic's vicinity
threshold** (`distanceSq < 177777`, `Monster.java:1804-1942`), reproducing
`aggroAddPuppet` / `aggroUpdatePuppetController`. This is the minimal port: the
summon service signals add/remove + position; `atlas-monsters` owns the vicinity
test and controller repick.

**Puppet damage / destruction:**

```
client → atlas-channel SummonDamageHandle(summonOid, dmg, monsterIdFrom)
  → emit COMMAND_TOPIC_SUMMON DAMAGE{summonOid, senderCharId, dmg, monsterIdFrom}
  → atlas-summons handleDamage:
      load + ownership check; addHP(-dmg); emit DAMAGED{dmg, monsterIdFrom}
      if hp ≤ 0: despawn(self)  → DESTROYED + REMOVE_PUPPET + oid release
  → atlas-channel: DAMAGED → broadcast SummonDamage; DESTROYED → broadcast SummonRemove
```

**Phasing note:** if the controller-repick parity proves large, the puppet HP/damage/
destruction loop and the `ADD_PUPPET`/`REMOVE_PUPPET` signaling can land first with a
simple "puppet owner is preferred controller while in vicinity" bias, and the full
visibility/repick nuance follows. Both are real work, not stubs.

---

## 10. Beholder buff aura (FR-6.x)

Beholder (`1321007`) is `buff-aura` + `follow`. Its heal and buff come from **two
separate owner skills** the player has trained — `AURA_OF_BEHOLDER` (`1320008`) and
`HEX_OF_BEHOLDER` (`1320009`) — not from the summon skill itself
(`Character.java:4448-4491`).

**At spawn**, snapshot onto the summon model (so the sweep task never re-fetches):
- resolve owner levels in `1320008`/`1320009` and those skills' effects
- `healAmount = aura.effect.hp`; heal interval = `aura.effect.x` seconds
- `buffChanges/buffLevel/buffDuration = hex.effect.*`; buff interval = `hex.effect.x` s
- `buffSourceId = int32(1320009)` (Q3 CORRECTED: positive real skill id — negative crashes the v83 client's icon lookup)
- set `nextHealAt` / `nextBuffAt`

**Leader-elected beholder-aura sweep task** scans beholder summons; when a timer is
due it:
- heal: `COMMAND_TOPIC_CHARACTER` `CHANGE_HP{channelId, amount: +healAmount}`
  (delta int16, clamped to MaxHP by `atlas-character` — the same path the Heal skill
  uses, `heal/heal.go` + `character/processor.go ChangeHP`)
- buff: `COMMAND_TOPIC_CHARACTER_BUFF` `APPLY{fromId: owner, sourceId: 1320009,
  level, duration, changes}`
- advance `nextHealAt` / `nextBuffAt`

Leader election guarantees single-fire per pod set, preventing duplicate heals/buffs
(NFR). Timers are cleaned up by the `despawn` routine (§7.2), which clears the
beholder fields, so no orphaned ticks survive removal.

---

## 11. Multi-version protocol (FR-8.x)

Six packets in `libs/atlas-packet/summon/`:

| Packet | Dir | v83 layout source (Cosmic `PacketCreator.java`) |
|---|---|---|
| `SummonSpawn` | client-bound | `:1149` ownerId, oid, skillId, `0x0A`, level, pos, stance, short(0), movementType, `!isPuppet`, `!animated` |
| `SummonRemove` | client-bound | `:1172` ownerId, oid, byte(4 animated / 1 instant) |
| `SummonMove` | client-bound | `:2284` cid, oid, startPos, raw movement bytes |
| `SummonAttack` | client-bound | `:2308` cid, oid, byte(0), direction, count, per target {oid, byte(6), int dmg} |
| `SummonDamage` | client-bound | `:4076` cid, oid, byte(12), int dmg, monsterIdFrom, byte(0) |
| `SummonSkill` | client-bound | `:4569` cid, summonSkillId, newStance (Beholder buff effect) |

Three inbound decoders mirror the move/attack/damage reads
(`MoveSummonHandler`/`SummonDamageHandler`/`DamageSummonHandler`).

**Version idiom** (from `monster/clientbound/spawn.go:41-57`,
`character/clientbound/spawn.go:79-145`): each writer/decoder pulls
`t := tenant.MustFromContext(ctx)` and branches on `t.Region()` /
`t.MajorAtLeast(n)` / `t.MajorAtMost(n)`. The v83 byte (`0x0A`) in `spawnSummon` is a
known version-variant marker — its per-version value comes from IDA, not the Cosmic
constant.

**Delta harvest (FR-8.2):** the per-version byte layout for v12/v84/v87/v92/v95 and
JMS185 is harvested from the IDA binaries — one IDB loaded at a time
(`reference_ida_harvest_subagents`) — and documented in
`summon-packet-delta.md`, following the task-083 `v84-packet-delta.md` precedent.
Per `bug_majorversion_gt83_is_off_by_one_v87`, v84 is byte-identical to v83; new
branches must gate on `>=87`, not `>83`.

**Opcodes (FR-8.4):** six writer + three handler opcode entries are added to **every**
template `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`
(gms_12/83/84/87/92/95, jms_185). Opcode **byte values are also harvested from IDA**
(the client dispatches on them — they are not free to invent), then resolved at
runtime through `libs/atlas-opcodes` and wired in `atlas-channel/main.go`'s
`produceWriters`/`produceHandlers`. Per `bug_new_opcodes_not_in_live_tenant_config`,
seed templates apply only at tenant creation; live tenants need a config patch +
channel restart to pick up the new opcodes (an operational note, not a code change).

**Tests (FR-8.3):** each writer/decoder is exercised across all
`libs/atlas-packet/test` variants (GMS v28/83/84/86/87/95, JMS v185 — note the harness
has no v92 and adds v28/v86), per `character/clientbound/spawn_test.go`.

---

## 12. Cross-service contract (new surface)

**New topics**
- `COMMAND_TOPIC_SUMMON` (channel → summons): `SPAWN`, `MOVE`, `ATTACK`, `DAMAGE`.
- `EVENT_TOPIC_SUMMON_STATUS` (summons → channel): `CREATED`, `MOVED`, `ATTACKED`,
  `DAMAGED`, `DESTROYED`.

**New commands on an existing topic**
- `COMMAND_TOPIC_MONSTER`: `ADD_PUPPET`, `REMOVE_PUPPET` (the only `atlas-monsters`
  code addition).

**Reused unchanged**
- `COMMAND_TOPIC_MONSTER` `DAMAGE` (owner-credited attacker damage).
- `COMMAND_TOPIC_MONSTER` `APPLY_STATUS` (stun/freeze).
- `COMMAND_TOPIC_CHARACTER` `CHANGE_HP` (Beholder heal).
- `COMMAND_TOPIC_CHARACTER_BUFF` `APPLY` (Beholder buff, positive source id 1320009).
- `EVENT_TOPIC_CHARACTER_STATUS` `LOGOUT`/`CHANNEL_CHANGED`/`MAP_CHANGED` (despawn).

**Error handling (FR-5.5 / §5.5)**
- Inbound summon packet whose summon is not owned by the sender → drop, log info.
- Reported damage > server max → clamp + autoban alert.
- Spawn for an owner with unknown field → no spawn, log.

---

## 13. Repo registration touchpoints

New **service** (no new shared lib ⇒ no Dockerfile `COPY` edits):
- `.github/config/services.json` (single source; `docker-bake.hcl` derives targets)
- `go.work` (the new `atlas-summons` module)
- `deploy/k8s/base/atlas-summons.yaml` + `kustomization.yaml` entry
- `deploy/k8s/base/env-configmap.yaml` (the two new topic vars)

New **packets/roster** live under already-vendored modules (`libs/atlas-packet`,
`libs/atlas-constants`) ⇒ no `go.work`/Dockerfile change for them.

`atlas-channel` wiring: register the three inbound handlers + six writers in
`produceHandlers`/`produceWriters`, add the `EVENT_TOPIC_SUMMON_STATUS` consumer, and
add the summon branch in `character_skill_use.go`.

Verification gate (CLAUDE.md): `go test -race`, `go vet`, `go build`,
`docker buildx bake atlas-summons`, and `tools/redis-key-guard.sh` — all from the
worktree root.

---

## 14. Alternatives considered

- **Fold summons into `atlas-monsters`.** Rejected — entangles two aggro/lifecycle
  models; summons are owner-bound with distinct despawn triggers (§2).
- **Validate attack damage in `atlas-channel`.** Rejected — the channel does not hold
  the summon's `skillId`/level (the attack packet carries only the oid), so it would
  round-trip to `atlas-summons` anyway; validating where the state lives is cleaner
  and avoids new channel-side effect getters (§3, §8.3).
- **Roster as a tenant config resource.** Rejected as overkill — a static
  `libs/atlas-constants/summon` table satisfies "config-driven, addable without core
  change" (FR-1.2) with far less machinery and zero new module plumbing (§6).
- **Per-summon goroutine timers for Beholder.** Rejected — does not survive pod
  restarts and risks duplicate fires across pods. A leader-elected sweep over the
  registry matches the `atlas-monsters` recovery/status task model (§10).
- **Direct channel-side movement rebroadcast (skip `atlas-summons`).** Rejected —
  ownership validation and authoritative position both live in `atlas-summons`;
  the Kafka round-trip matches the monster movement precedent (§8 framing).

---

## 15. Risks & suggested phasing

**Top risks**
1. **Damage-ceiling port (§8.3)** — weapon-type-aware base-damage math is the largest
   behavioral unknown; reuse-vs-port must be settled early in planning.
2. **Puppet controller-repick parity (§9)** — the only `atlas-monsters` change;
   vicinity bias is straightforward, full visibility/repick nuance is the long tail.
3. **IDA delta + opcode harvest (§11)** — six packets × seven versions, one IDB at a
   time; the per-version `0x0A`-style markers and opcode bytes are client-fixed and
   must be harvested, not guessed.

**Suggested phases** (planning sets final granularity):
- **P0** — scaffold `atlas-summons` (registry, oid, REST, repo registration); builds
  green, no behavior.
- **P1** — roster lib + spawn/despawn lifecycle (`CREATED`/`DESTROYED`, character-event
  cascade, expiry sweep) + v83 `SummonSpawn`/`SummonRemove` + channel wiring + v83
  opcodes.
- **P2** — movement (`SummonMove`, follow/circle).
- **P3** — attacker (credit `DAMAGE` + status + Gaviota self-cancel + `SummonAttack`),
  then layer in the §8.3 ceiling.
- **P4** — puppet (`ADD_PUPPET`/`REMOVE_PUPPET` + aggro bias + `SummonDamage` +
  destruction).
- **P5** — Beholder aura (`CHANGE_HP` + buff `APPLY` + timers + `SummonSkill`).
- **P6** — multi-version: `summon-packet-delta.md` harvest, version-conditional
  encode/decode, opcodes across all 7 templates, per-variant tests.

---

## 16. Out of scope (restated)

Version-only summons (Dual Blade *Owl Spirit* v88+, Evan's dragon v84) — graceful
no-op, not an error (Q5). Aerial Strike and Battleship (Q6). Player-NPC and pet
behaviors. Re-architecting `atlas-monsters` aggro beyond the `ADD_PUPPET` addition.

---

## Appendix A — roster & movement classes (Cosmic-confirmed)

`StatEffect.java:1766-1797`, `Summon.isStationary()/isPuppet()`:

| Movement | Skill ids |
|---|---|
| **Stationary (0)** | 3111002 (Ranger Puppet), 3211002 (Sniper Puppet), 13111004 (Wind Archer Puppet), 5211001 (Octopus), 5220002 (Wrath of the Octopi) |
| **Circle-follow (3)** | 3111005 (Silver Hawk), 3211005 (Golden Eagle), 3121006 (Phoenix), 3221005 (Frostprey), 2311006 (Summon Dragon), 5211002 (Gaviota) |
| **Follow (1)** | 1321007 (Beholder), 2121005 (Elquines), 2221005 (Ifrit), 2321003 (Bahamut), 11001004 (Soul), 12001004 (Flame), 12111004 (Ifrit/Blaze 3rd), 13001004 (Storm), 14001005 (Darkness), 15001004 (Lightning) |

Puppets: 3111002, 3211002, 13111004. Beholder spawn HP = effect `x + 1`; other
puppet HP = effect `x`. All 21 skill-id constants already exist in
`libs/atlas-constants/skill/constants.go`.
