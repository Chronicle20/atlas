# Player Summons — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

Player **summons** are temporary, owner-bound creatures spawned by character skills. They appear on the field as their own map-object type, persist for a skill-defined duration, and exhibit one of three behaviors: a stationary **decoy/puppet** that draws monster aggro and absorbs damage until its HP is depleted, an **attacker** (hawk, phoenix, eagle, dragon, octopus, elemental, etc.) whose attacks damage nearby monsters and credit the owner for experience and drops, and a **buff aura** (Dark Knight's Beholder) that periodically heals and buffs its owner. Summons despawn when their owner logs out, changes channel, changes map, re-casts the same summon, or — for puppets — loses all HP.

Atlas does not implement summons today. The closest existing behavior is consumable "summoning sacks," which spawn *unowned, untracked* monsters with no despawn logic. The skill-cast inbound path exists (`character_skill_use.go`), the skill-effect data is already served per-tenant by `atlas-data`, and every cross-service integration seam (monster aggro/damage, character lifecycle events, buff application) already accepts an external caller. What is missing is ownership tracking, summon lifecycle, the summon wire protocol, and the skill→spawn wiring.

This feature introduces a new **`atlas-summons`** service that owns the summon lifecycle, modeled on `atlas-monsters` (per-tenant Redis registry, shared object-id allocator, Kafka command/event topics). The core engine is **version-agnostic**: summon behavior, lifecycle, and integration are identical across all supported client versions. Version variance is confined to the packet wire format and per-tenant opcode configuration. Cosmic (`~/source/Cosmic`) is the authoritative behavioral baseline for v83; the v83 roster of 21 summon skills is the functional target, implemented across **all supported versions** (GMS v12/83/84/87/92/95, JMS v185).

## 2. Goals

Primary goals:
- Stand up a new `atlas-summons` service that spawns, tracks, moves, and despawns owner-bound summons with skill-defined durations.
- Implement all three archetypes for the full v83 roster of 21 summon skills:
  - **Decoy/puppet** — HP, monster aggro-redirect, destruction at 0 HP (Ranger/Sniper/Wind-Archer Puppet).
  - **Attacker** — client-driven attacks that damage monsters and credit the owner; some apply stun/freeze (hawks, eagle, phoenix, frostprey, dragon, elementals, Bahamut, Gaviota, octopus, and the five Cygnus summons).
  - **Buff aura** — Beholder's periodic heal + buff applied to the owner on a server-side timer.
- Make the feature multi-version: the summon spawn/remove/move/attack/damage/skill packets encode correctly for **every supported version** using the established tenant-context branching idiom, with per-version opcodes seeded into tenant templates.
- Despawn summons correctly on owner logout, channel change, map change, summon re-cast, and (puppets) HP depletion.
- Forward owner credit so summon kills grant XP, drops, and quest progress to the owner.
- Validate client-reported summon attack damage server-side (autoban-style ceiling), matching Cosmic's trust model.

Non-goals:
- Summons that exist only in versions later than v83 (Dual Blade *Owl Spirit*, v88+; Evan's dragon, v84) — out of scope even where their data is present in v92/v95 tenants. The engine must not crash or misbehave when such a skill id is encountered; it simply has no summon mapping yet.
- Corsair **Aerial Strike** (a summon-enhancement skill, not a spawned creature) and **Battleship** (a mount).
- Player-NPC and pet behaviors (already handled elsewhere).
- Re-architecting `atlas-monsters` aggro; this feature consumes the existing aggro/control surface.

## 3. User Stories

- As a **Ranger**, I want to cast Silver Hawk so a hawk follows me and attacks nearby monsters, dealing damage credited to me.
- As a **Sniper**, I want to deploy a Puppet that monsters attack instead of me, until the puppet's HP runs out.
- As a **Bishop**, I want to summon Bahamut to follow me and attack, so I deal sustained damage hands-free.
- As a **Dark Knight**, I want Beholder to periodically heal and buff me while it follows me.
- As an **Outlaw**, I want to drop an Octopus that holds its ground and attacks monsters in range.
- As **any summoner**, I want my summon to disappear when I change maps, change channels, or log out — and to be replaced (not duplicated) when I re-cast the same skill.
- As a **player on any supported client version**, I want summons to render and behave correctly without version-specific bugs.
- As an **operator**, I want summon damage to be validated so a modified client cannot inflict impossible damage on monsters.

## 4. Functional Requirements

Requirements are grouped by capability. Each is specific and testable.

### 4.1 Summon roster & classification
- FR-1.1 The system SHALL recognize the 21 v83 summon skill ids (see Appendix A) and map each to a summon **type** (puppet / attacker / buff-aura) and a **movement type** (stationary / follow / circle-follow), matching Cosmic's `StatEffect` mapping.
- FR-1.2 Classification SHALL be data/config-driven, not hard-coded per call site, so additional summon skill ids can be added without code changes to the core engine.
- FR-1.3 When a skill cast resolves to a summon skill id with no roster mapping (e.g. a later-version-only summon), the system SHALL no-op gracefully (log at debug/info, no spawn, no error to client).

### 4.2 Spawn
- FR-2.1 On a summon skill cast, `atlas-summons` SHALL create a summon instance bound to `{ownerCharacterId, skillId, skillLevel, field(world/channel/map/instance), position, spawnTime, expiresAt}` and allocate a unique object id from the shared per-field object-id allocator (`libs/atlas-object-id`).
- FR-2.2 The spawn SHALL be broadcast to all characters in range on the owner's field via the summon spawn packet, with the correct movement type, puppet/attack flag, and animated flag.
- FR-2.3 Puppet HP SHALL be initialized from the skill effect's `x` value; Beholder SHALL receive the Cosmic-equivalent HP initialization.
- FR-2.4 Re-casting the same summon skill SHALL remove the prior instance of that skill for that owner before spawning the new one (no duplicates per skill id).
- FR-2.5 Casting a summon of a different mobility class SHALL cancel the conflicting class per Cosmic semantics (a new non-stationary summon cancels the existing non-stationary summon; a new stationary summon cancels the existing stationary one).

### 4.3 Movement
- FR-3.1 The system SHALL accept inbound summon-movement packets from the owner's client, validate the summon belongs to that owner, and rebroadcast the movement to other in-range characters.
- FR-3.2 Stationary summons SHALL not be expected to move; follow/circle-follow movement is client-driven and relayed verbatim.

### 4.4 Attacker behavior
- FR-4.1 The system SHALL accept inbound summon-attack packets (summon object id, direction, list of {monster object id, damage}) from the owner's client and rebroadcast the attack animation to in-range characters.
- FR-4.2 For each attacked monster, the system SHALL apply the reported damage to the monster **crediting the owner's `characterId`**, so XP/drops/kill-credit flow to the owner.
- FR-4.3 The system SHALL compute a server-side maximum per-hit damage from the owner's stats and the summon skill effect (physical via weapon attack multiplier, magic via magic attack multiplier, matching Cosmic's formulas) and clamp/flag any reported damage that exceeds it (autoban-style), rejecting the excess.
- FR-4.4 Summons whose effect carries a monster status (stun/freeze) SHALL apply that status to hit monsters with the effect's proc chance.
- FR-4.5 Gaviota SHALL self-cancel after a single attack (one grenade toss), per Cosmic.

### 4.5 Puppet decoy behavior
- FR-5.1 While a puppet is deployed in a field, monsters in the field SHALL redirect aggro/control toward the puppet's owner per the existing `atlas-monsters` aggro mechanism.
- FR-5.2 The system SHALL accept inbound summon-damage packets reporting damage dealt to a puppet, decrement the puppet's HP, and broadcast the puppet-damaged packet.
- FR-5.3 When a puppet's HP reaches 0, the system SHALL destroy the puppet (broadcast removal, release object id, remove aggro redirect).

### 4.6 Buff aura behavior (Beholder)
- FR-6.1 While Beholder is deployed, the system SHALL on a server-side interval apply a timed heal to the owner and, on a (possibly different) interval, apply the Beholder buff stat changes to the owner — via the existing `atlas-buffs` `APPLY` command, using a reserved source-id range that does not collide with player skill ids.
- FR-6.2 The aura timers SHALL stop and be cleaned up when the Beholder is removed for any reason.
- FR-6.3 Heal/buff values and intervals SHALL be derived from the Beholder skill effect data, matching Cosmic.

### 4.7 Lifecycle & despawn
- FR-7.1 Each summon SHALL expire automatically at `expiresAt` (skill-defined duration), broadcasting removal and releasing its object id.
- FR-7.2 The system SHALL consume character lifecycle events and despawn all of a character's summons on **logout**, **channel change**, and **map change**.
- FR-7.3 Summon state SHALL be tenant-scoped and survive `atlas-summons` pod restarts (Redis-backed registry), consistent with `atlas-monsters`.
- FR-7.4 Object ids SHALL be released back to the allocator on every despawn path (expiry, logout, map/channel change, re-cast, HP-0).

### 4.8 Multi-version protocol
- FR-8.1 The six summon packets (spawn, remove, move, attack, damage, summon-skill) SHALL be implemented in `libs/atlas-packet/summon/` with version-conditional encode/decode using the tenant-context branching idiom (`t.Region()`, `t.MajorAtLeast(n)`), producing byte-correct output for **every supported version**: GMS v12, v83, v84, v87, v92, v95, and JMS v185.
- FR-8.2 The per-version byte layout deltas SHALL be harvested from the IDA binaries (one IDB per version) and documented as a `summon-packet-delta.md`, following the task-083 precedent.
- FR-8.3 Each summon writer/handler SHALL be exercised by the per-variant packet test harness (`libs/atlas-packet/test`) across all configured version variants.
- FR-8.4 Summon opcodes SHALL be configured per version via the tenant socket-config seed templates (one opcode entry per summon writer/handler per `template_<region>_<major>_<minor>.json`), and resolved at runtime through the existing per-tenant opcode registry — never hard-coded.

## 5. API Surface

`atlas-summons` exposes a JSON:API REST surface modeled on `atlas-monsters`, plus Kafka topics. All REST payloads are tenant-scoped via context.

### 5.1 REST (read/diagnostic)
- `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/summons` — list summons in a field.
- `GET /summons/{summonId}` — fetch a single summon (resource type `summons`).
- (Internal/diagnostic only; summon creation is driven by skill casts via Kafka, not REST.)

JSON:API resource `summons` attributes (illustrative): `ownerCharacterId`, `skillId`, `skillLevel`, `summonType`, `movementType`, `x`, `y`, `hp`, `maxHp`, `expiresAt`, `worldId`, `channelId`, `mapId`, `instance`.

### 5.2 Kafka — produced by `atlas-summons`
- `EVENT_TOPIC_SUMMON_STATUS` — event types: `CREATED`, `MOVED`, `ATTACKED`, `DAMAGED`, `DESTROYED`/`EXPIRED`. Consumed by `atlas-channel` to broadcast packets.
- Commands emitted to existing topics:
  - `COMMAND_TOPIC_MONSTER` `DAMAGE` (attacker summons → monster, credited to owner).
  - `COMMAND_TOPIC_MONSTER` control/aggro command (puppet decoy redirect — see Open Questions for exact command).
  - `COMMAND_TOPIC_CHARACTER_BUFF` `APPLY` (Beholder aura).

### 5.3 Kafka — consumed by `atlas-summons`
- A summon **command** topic carrying skill-cast-derived spawn requests + inbound move/attack/damage relays from `atlas-channel`.
- `EVENT_TOPIC_CHARACTER_STATUS` — `LOGOUT`, `CHANNEL_CHANGED`, `MAP_CHANGED` (despawn cascade).
- `EVENT_TOPIC_MONSTER_STATUS` — for puppet aggro feedback and attacker `KILLED`/`DAMAGED` confirmation if needed.

### 5.4 `atlas-channel` packet surface
- Inbound handlers: `SummonMoveHandle`, `SummonAttackHandle`, `SummonDamageHandle`.
- Outbound writers: `SummonSpawn`, `SummonRemove`, `SummonMove`, `SummonAttack`, `SummonDamage`, `SummonSkill`.

### 5.5 Error cases
- Inbound summon packet referencing a summon not owned by the sender → drop, log at info (possible exploit).
- Reported summon damage exceeding server max → clamp + autoban alert.
- Spawn for an owner whose field is unknown → no spawn, log.

## 6. Data Model

`atlas-summons` registry entry (Redis-backed, per-tenant), shaped like the `atlas-monsters` stored model:

| Field | Type | Notes |
|---|---|---|
| `tenantId` | uuid | Multi-tenant isolation (key prefix). |
| `summonId` (object id) | uint32 | From shared `libs/atlas-object-id` per-field pool. |
| `ownerCharacterId` | uint32 | Owner. |
| `skillId` | uint32 | Summon skill id (Appendix A). |
| `skillLevel` | byte | Drives effect values. |
| `summonType` | enum | puppet / attacker / buff-aura. |
| `movementType` | enum | stationary(0) / follow(1) / circle-follow(3). |
| `worldId` / `channelId` / `mapId` / `instance` | field | Owner's field at spawn. |
| `x` / `y` | int16 | Position. |
| `hp` / `maxHp` | int32 | Puppets/Beholder; 0 for HP-less attackers. |
| `spawnTime` / `expiresAt` | time | Duration-driven expiry. |

Redis key shape: `atlas:summon:<tenantId>:<summonId>`; field index: `atlas:summon-map:<tenantId>:<world>:<channel>:<map>:<instance>` (modeled on the monster registry's map index). No relational migration; state is ephemeral cache, consistent with `atlas-monsters`.

`atlas-data` skill effect additions (if required by FR-4.3): expose `weaponAttack` / `magicAttack` getters on the channel-side `effect.Model`, and add an attack-interval field to the skill reader/model if Cosmic-parity attack timing is required server-side. (Most summon timing is client-driven; the interval is needed only if the server independently schedules attacks — see Open Questions.)

## 7. Service Impact

| Service | Change |
|---|---|
| **`atlas-summons`** (NEW) | Full service: `summon/` (model, builder, processor, registry, id allocator, resource/rest, kafka/producer), `kafka/consumer/` (summon commands, character lifecycle, data), `tasks/` (expiry sweep + Beholder timers), `main.go` wiring. |
| **`atlas-channel`** | New inbound handlers (move/attack/damage summon) routing to `atlas-summons` commands; new clientbound writers; consumer of `EVENT_TOPIC_SUMMON_STATUS`; register summon writers/handlers in `produceWriters`/`produceHandlers` and the per-tenant listener. Skill-cast handler routes summon skill ids to a summon spawn command. |
| **`libs/atlas-packet`** | New `summon/clientbound/` and `summon/serverbound/` packets with version-conditional encode/decode + per-variant tests. |
| **`atlas-monsters`** | Consume existing `DAMAGE` command with owner credit (no change expected); confirm/extend the aggro-control command path for puppet redirect (may need a small command addition — see Open Questions). |
| **`atlas-buffs`** | No code change expected; Beholder uses existing `APPLY`. Reserve a source-id range for summon auras. |
| **`atlas-data`** | Possible additive getters (`weaponAttack`/`magicAttack`) and an attack-interval field on skill effect. |
| **Repo/config** | `.github/config/services.json`, `docker-bake.hcl`, root `Dockerfile` (2 COPY lines if a new lib is added), `go.work`, `deploy/k8s/base/atlas-summons.yaml` + kustomization entry, `env-configmap.yaml` topic vars, and **summon opcode entries in every supported-version socket template**. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all summon state, Kafka headers, and Redis keys SHALL be tenant-scoped via `tenant.MustFromContext`. No cross-tenant leakage.
- **Multi-version correctness:** every summon packet SHALL pass the per-variant test harness for all 7 supported versions; opcodes resolved from tenant config, never hard-coded.
- **Security / anti-cheat:** client-reported summon damage SHALL be clamped to a server-computed maximum with autoban alerting (FR-4.3); inbound summon packets SHALL be ownership-validated.
- **Performance:** summon spawn/move/attack broadcast SHALL use the existing ranged-broadcast path; per-field summon counts are small (≤ a few per player). Beholder timers SHALL be leader-elected or otherwise single-fired per summon to avoid duplicate buffs across pods.
- **Observability:** summon create/destroy/damage SHALL emit structured logs and traces consistent with `atlas-monsters`; expose `/metrics`.
- **Resilience:** summon registry SHALL survive pod restarts; object ids SHALL never leak (released on every despawn path).
- **Redis discipline:** all Redis access SHALL route through `libs/atlas-redis` (redis-key-guard clean).

## 9. Open Questions

- **Q1 — Puppet aggro command:** does `atlas-monsters` already expose an inbound command to redirect aggro/control toward a puppet-owner, or is a new `COMMAND_TOPIC_MONSTER` command type required? (Discovery suggests the control surface exists but the explicit "add puppet" command path needs confirmation in design.) Relates to task-033.
- **Q2 — Server-scheduled attacks vs. client-driven:** Cosmic's attacker summons are client-driven (the client sends the attack packet). Confirm all 21 are client-driven on every supported version, or whether any version expects the server to schedule summon attacks (which would require the attack-interval data field).
- **Q3 — Beholder buff source-id range:** which reserved source-id range avoids collision with player skill ids and existing buff sources?
- **Q4 — Object-id pool sharing:** confirm summons must draw from the same per-field object-id pool as monsters/drops/reactors (shared client OID space on a map) — expected yes.
- **Q5 — Per-version roster gaps:** for v92/v95 (>v88), the Dual Blade summon and any other version-only summons are present in skill data but out of scope; confirm the graceful no-op (FR-1.3) is the desired handling rather than a hard error.
- **Q6 — Aerial Strike / summon-enhancement skills:** confirmed out of scope; flag if any supported version makes Aerial Strike a true spawned object (Cosmic does not).

## 10. Acceptance Criteria

- [ ] `atlas-summons` service exists, builds, and is registered in `services.json`, `docker-bake.hcl`, `go.work`, k8s manifests; `docker buildx bake atlas-summons` succeeds from the worktree root.
- [ ] All 21 v83 summon skills spawn the correct summon type with correct movement type, HP (where applicable), and duration, verified against Cosmic.
- [ ] Re-casting a summon replaces (does not duplicate) it; conflicting-class casts cancel per Cosmic semantics.
- [ ] Puppets draw monster aggro, take damage via the summon-damage packet, and are destroyed at 0 HP.
- [ ] Attacker summons damage monsters with damage credited to the owner; XP/drops/kill-credit reach the owner; stun/freeze summons apply their status; Gaviota self-cancels after one attack.
- [ ] Client-reported summon damage above the server-computed max is clamped and triggers an autoban alert.
- [ ] Beholder periodically heals and buffs its owner via `atlas-buffs`; timers stop on removal.
- [ ] Summons despawn on owner logout, channel change, map change, and skill duration expiry; object ids are released on every path.
- [ ] All six summon packets encode/decode byte-correctly for GMS v12/83/84/87/92/95 and JMS v185, covered by the per-variant test harness; deltas documented in `summon-packet-delta.md`.
- [ ] Summon opcodes are seeded into every supported-version socket template and resolved per-tenant at runtime.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `tools/redis-key-guard.sh` clean.
- [ ] A later-version-only summon skill id (e.g. Dual Blade) is handled as a graceful no-op, not an error.

---

## Appendix A — v83 summon roster (21 skills)

| Job | Summon | Skill ID | Type | Movement |
|---|---|---|---|---|
| Ranger | Puppet | 3111002 | Puppet | Stationary |
| Ranger | Silver Hawk | 3111005 | Attacker (stun) | Circle-follow |
| Bowmaster | Phoenix | 3121006 | Attacker | Circle-follow |
| Sniper | Puppet | 3211002 | Puppet | Stationary |
| Sniper | Golden Eagle | 3211005 | Attacker (stun) | Circle-follow |
| Marksman | Frostprey | 3221005 | Attacker (freeze) | Circle-follow |
| Priest | Summon Dragon | 2311006 | Attacker | Circle-follow |
| F/P Arch Mage | Elquines | 2121005 | Attacker (freeze) | Follow |
| I/L Arch Mage | Ifrit | 2221005 | Attacker | Follow |
| Bishop | Bahamut | 2321003 | Attacker | Follow |
| Dark Knight | Beholder | 1321007 | Buff aura | Follow |
| Outlaw | Octopus | 5211001 | Attacker | Stationary |
| Outlaw | Gaviota | 5211002 | Attacker (one-shot) | Circle-follow |
| Corsair | Wrath of the Octopi | 5220002 | Attacker | Stationary |
| Dawn Warrior | Soul | 11001004 | Attacker | Follow |
| Blaze Wizard (1st) | Flame | 12001004 | Attacker | Follow |
| Blaze Wizard (3rd) | Ifrit | 12111004 | Attacker | Follow |
| Wind Archer (1st) | Storm | 13001004 | Attacker | Follow |
| Wind Archer (3rd) | Puppet | 13111004 | Puppet | Stationary |
| Night Walker (1st) | Darkness | 14001005 | Attacker | Follow |
| Thunder Breaker (1st) | Lightning | 15001004 | Attacker | Follow |

All 21 skill ids already have constants in `libs/atlas-constants/skill/constants.go`.

## Appendix B — supported version matrix

GMS v12, v83, v84, v87, v92, v95; JMS v185 (`deploy/k8s/base/versions.json`). Packet test harness additionally exercises GMS v28/v86. See `discovery.md` for the version-conditional encoding idiom and worked references.
