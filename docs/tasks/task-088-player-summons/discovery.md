# Player Summons — Discovery Notes

Created: 2026-06-12
Status: Reference (feeds design.md)

Source maps gathered during the pre-PRD discovery session. Cosmic = `~/source/Cosmic` (Java, v83 baseline). Atlas = this repo. All Atlas paths relative to repo root.

---

## 1. Cosmic baseline (v83)

### 1.1 Core model & lifecycle
- `server/maps/Summon.java` — fields: `owner`, `skill`, `skillLevel`, `hp`, `movementType`. `isStationary()` / `isPuppet()` switch on skill id. Puppet ids: `3111002`, `3211002`, `13111004`.
- `server/maps/SummonMovementType.java` — `STATIONARY(0)`, `FOLLOW(1)`, `CIRCLE_FOLLOW(3)`.
- `server/StatEffect.java:1766-1797` — skill-id → movement-type mapping (the authoritative classification table).
- `server/StatEffect.java:1022-1059` — spawn path: cancel conflicting buff (`PUPPET` vs `SUMMON`), `new Summon(...)`, `map.spawnSummon`, `addSummon`, `addHP(x)` (puppet HP from effect `x`; Beholder `+1`).
- `client/Character.java:729-735` (`addSummon`, puppet → `map.addPlayerPuppet`), `:2898-2924` (buff-expire task), `:3769-3791` (removal cascade), `:4460-4490` (Beholder heal/buff schedules).
- `server/maps/MapleMap.java:2045-2051` (`spawnSummon`), `:2600-2610` (`addPlayerPuppet`/`removePlayerPuppet`), `:2646-2652` (despawn on map leave).

### 1.2 Behavior handlers
- Attack: `net/server/channel/handlers/SummonDamageHandler.java:54-145` — reads targets, validates vs `calcMaxDamage` (physical vs magic), applies damage, applies monster status, Gaviota self-cancels (`Outlaw.GAVIOTA`).
- Puppet damage: `net/server/channel/handlers/DamageSummonHandler.java:35-54` — `addHP(-dmg)`; at ≤0, `cancelEffectFromBuffStat(PUPPET)`.
- Movement: `net/server/channel/handlers/MoveSummonHandler.java:36-59`.
- Puppet aggro: `server/life/Monster.java:1804-2165` (`aggroAddPuppet`, vicinity check, controller repick/visibility).

### 1.3 Packets (v83) — `tools/PacketCreator.java`
- `spawnSummon` (`:1149`) — ownerId, oid, skillId, `0x0A`(v83), skillLevel, pos, stance, short(0), movementType, `!isPuppet()` (attack flag), `!animated`.
- `removeSummon` (`:1172`) — ownerId, oid, byte(4 animated / 1 instant).
- `moveSummon` (`:2284`) — cid, oid, startPos, raw movement bytes.
- `summonAttack` (`:2308`) — cid, oid, byte(0), direction, count, per target {oid, byte(6), int dmg}.
- `damageSummon` (`:4076`) — cid, oid, byte(12), int dmg, monsterIdFrom, byte(0).
- `summonSkill` (`:4569`) — cid, summonSkillId, newStance (Beholder buff effect).
- Opcode **names** matter; v83 **byte values** must come from Atlas tenant config / IDA, not Cosmic constants.

### 1.4 Roster confirmation
- 21 summons exist in v83 (see prd.md Appendix A). `constants/skills/*.java` per job.
- `Corsair.AERIAL_STRIKE = 5221003` is a **dead constant** (zero references) — not a `Summon`. Out of scope.
- `Corsair.BATTLE_SHIP = 5221006` is a mount (MONSTER_RIDER), not a summon.
- No thief/Night-Lord summon (Owl Spirit is Dual Blade, v88). Evan is v84. See memory `reference_maplestory_version_timeline`.

---

## 2. Atlas current state & gap

### 2.1 Skill cast & effect data (already present)
- Inbound cast: `services/atlas-channel/.../socket/handler/character_skill_use.go:35` — resolves `skill3.NewProcessor(l,ctx).GetEffect(skillId, level)` (`:70`). Comment references `CUserLocal::DoActiveSkill_Summon`.
- Effect model: `services/atlas-channel/.../data/skill/effect/model.go` — exposes `Duration()`, `X()` (puppet HP), `MonsterStatus()` (stun/freeze + chance map), `Prop()` (proc chance). **`weaponAttack`/`magicAttack` present but no getter**; **no attack-interval field**.
- Data source: `services/atlas-data/.../skill/reader.go` parses `Skill.wz` (`time`, `pad`, `mad`, `x`, `y`, prop, monsterStatus). REST `GET /data/skills/{skillId}` returns the full field set. Per-tenant/version scoped (`services/atlas-data/.../wzinput/scope.go`).
- Skill-id constants: all 21 exist in `libs/atlas-constants/skill/constants.go` (e.g. `RangerPuppetId=3111002`, `DarkKnightBeholderId=1321007`).
- Local WZ dump for value extraction: `tmp/<uuid>/GMS/83.1/Skill.wz`.

### 2.2 New-service blueprint (model on atlas-monsters)
- Template service: `services/atlas-monsters/atlas.com/monsters/` — `monster/` (model, builder, processor, registry, id_allocator, resource, rest, kafka, producer), `kafka/consumer/`, `tasks/`, `main.go`.
- `main.go` boot order: logger → Redis + `InitIdAllocator`/`InitRegistry` (sync.Once) → teardown mgr → tracing → consumers (`InitConsumers(l)(cmf)(group)`) + handlers → REST server (`AddRouteInitializer`) → leader-elected sweep tasks → teardown hooks.
- Object ids: `libs/atlas-object-id` (Redis-backed, per-tenant, MinId 1,000,000; `Allocate`/`Release`). **Shared per-field pool** with monsters/drops/reactors.
- Repo registration for a new service (≈7 touchpoints): `.github/config/services.json`, `docker-bake.hcl` (`go_services`), `go.work`, `deploy/k8s/base/atlas-summons.yaml` (new) + `kustomization.yaml`, `deploy/k8s/base/env-configmap.yaml` (topics), `services/atlas-channel/.../main.go` (consumer+writer+handler wiring). Plus root `Dockerfile` 2× COPY only if a new shared lib is added.

### 2.3 Cross-service integration seams (all accept external callers today)
- **Puppet aggro** → `COMMAND_TOPIC_MONSTER` (control) / `EVENT_TOPIC_MONSTER_STATUS` feedback. `services/atlas-monsters/.../monster/processor.go:292-323` (`StartControl`). Whether an explicit "add puppet" command exists is an **open question** (Q1).
- **Attacker damage** → `COMMAND_TOPIC_MONSTER` `DAMAGE` with owner `characterId` (`processor.go:340-476`; `DamageCommandBody.CharacterId`). `DAMAGED`/`KILLED` events carry `DamageEntries[]` for credit.
- **Despawn triggers** → `EVENT_TOPIC_CHARACTER_STATUS` `LOGOUT`/`CHANNEL_CHANGED`/`MAP_CHANGED` (`services/atlas-character/.../character/producer.go:65-115`; event consts in `.../kafka/message/character/kafka.go:212-290`).
- **Owner location** → `GET /characters/{id}/location` (field only, not x/y; spawn coords come from the cast).
- **Beholder aura** → `COMMAND_TOPIC_CHARACTER_BUFF` `APPLY` (`services/atlas-buffs/.../character/processor.go:43-56`; `ApplyCommandBody{FromId,SourceId,Level,Duration,Changes}`). Reserve a source-id range (Q3).

### 2.4 Existing summon trace (what little exists)
- `libs/atlas-packet/field/clientbound/effect.go` — `EffectSummon` / `NewFieldEffectSummon` (visual field effect only; NOT object spawning).
- Consumable summoning sacks: `services/atlas-consumables/.../consumable/processor.go:412-453` (`ConsumeSummoningSack`) spawns **unowned, untracked** monsters — not the player-summon mechanic.
- **No** summon spawn/move/attack/damage writers or handlers, **no** summon opcodes in repo.

---

## 3. Multi-version mechanics

### 3.1 Supported matrix
- `deploy/k8s/base/versions.json` — GMS 12, 83, 84, 87, 92, 95; JMS 185. Test harness variants (`libs/atlas-packet/test/context.go:18-29`) also include GMS v28, v86.
- Tenant model: `libs/atlas-tenant/tenant.go` — `Region()`, `MajorVersion()`, helpers `IsRegion`, `MajorAtLeast`, `MajorAtMost`, `MajorInRange`.

### 3.2 Version-conditional encoding idiom
- Writers/handlers pull `t := tenant.MustFromContext(ctx)` and branch inline. Examples:
  - `libs/atlas-packet/monster/clientbound/spawn.go:41-57` — control byte for `GMS>12 || JMS`.
  - `libs/atlas-packet/character/clientbound/spawn.go:79-145` — worked multi-band reference (≤87 / >87 / <95 / JMS).
  - `libs/atlas-packet/chat/serverbound/general.go:56-66` — inbound decode version branch.
- Per-variant tests loop all variants: `libs/atlas-packet/character/clientbound/spawn_test.go:18-27`.

### 3.3 Opcodes per version
- Per-tenant opcode registry built from socket config templates: `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json` (handlers + writers with `opCode`). Resolved at runtime via `libs/atlas-opcodes/producer.go:14-35`, wired in `services/atlas-channel/.../main.go:366-391`. **Summon opcodes must be added to every supported-version template.**

### 3.4 Data per version
- `atlas-data` serves skill data per-tenant/version, so the available roster is version-correct automatically. Later-version-only summons (Dual Blade v88+) surface in v92/v95 data but are out of scope (graceful no-op, FR-1.3 / Q5).

### 3.5 Precedent
- `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` — the model for harvesting per-version byte deltas from IDA (IDBs: v83/v84/v87/v95/JMS185). Per memory `reference_ida_harvest_subagents`, one IDB is loaded at a time.
