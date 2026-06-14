# Player Summons — Implementation Context

Companion to `plan.md`. Captures the key files, decisions, and dependencies an
engineer needs before executing the plan. All paths are relative to the worktree
root (`<repo-root>/.worktrees/task-088-player-summons`).

Cosmic baseline lives at `~/source/Cosmic` (Java, v83). Authoritative source docs
(read first): `prd.md`, `design.md`, `discovery.md`.

---

## 1. What we're building

A new **`atlas-summons`** Go microservice owning the lifecycle of owner-bound
player summons (spawn / move / attack / take-damage / despawn), structurally
modeled on `atlas-monsters`. Plus six clientbound + three serverbound summon
packets in `libs/atlas-packet/summon/`, a roster table in
`libs/atlas-constants/summon/`, `atlas-channel` wiring, one new
`atlas-monsters` command pair (`ADD_PUPPET`/`REMOVE_PUPPET`), and per-version
opcodes in the seven socket-config seed templates.

---

## 2. Blueprint service — `atlas-monsters`

`services/atlas-monsters/atlas.com/monsters/` is the structural template for the
new service. Mirror these files:

| File | Role | Key lines |
|---|---|---|
| `main.go` | Boot order | logger(54) → redis+`InitIdAllocator`/`InitMonsterRegistry`(57-63) → teardown mgr(65) → tracing(67-70) → consumers `InitConsumers(l)(cmf)(group)`(72-84) → REST `AddRouteInitializer`(88-97) → leader-elected `registerSweepTasks`(99-131) → teardown hooks(133) → `tdm.Wait()`(136) |
| `monster/model.go` | Immutable model (private fields + getters) | struct 33-55; `Move`/`Control`/`Damage` return Clone()…Build() |
| `monster/builder.go` | `Clone(m)` + fluent `ModelBuilder` + `Build()` | 12, 41, 195 |
| `monster/registry.go` | sync.Once singleton; `atlasredis.NewRegistry`+`NewKeyedSet`; optimistic `Update` | `InitMonsterRegistry`(275-285); key suffixes(294-313); `atomicUpdate`(338-356) |
| `monster/id_allocator.go` | wraps `objectid.NewRedisAllocator`; `Allocate`/`Release` | whole file |
| `monster/processor.go` | `Processor` interface + `ProcessorImpl`; `NewProcessor(l,ctx)`; `emit` closure(85-86) | 26-93 |
| `monster/resource.go`/`rest.go` | JSON:API `monsters` resource; `RegisterHandler` GET | resource 18-23; rest 25-43 |
| `monster/kafka.go`/`producer.go` | event topic env var, `statusEvent[E]` envelope, type consts, providers | kafka 14-60; producer 26-37 |
| `kafka/consumer/monster/consumer.go` | `InitConsumers(l)(cmf)(group)` + `InitHandlers(l)(rf)`; `handleX(l,ctx,c)` | 18-85 |
| `tasks/task.go` | `Task{Run(); SleepTime()}`; `Register(l,ctx)(t)` tick loop | 10-30 |
| `world/resource.go` | field-scoped list/`in-rect`/POST/DELETE | 30-68 |
| `leaderconfig.go` | leader env parsing | whole file |

**Boot detail:** registries init via `sync.Once` (`InitX(rc)` + `GetX()`); leader
election (`lock.New(rc, "summons-sweep", …)`) gates the sweep tasks; non-leader
pods register no sweep tasks.

---

## 3. Object-id and Redis libs

- **`libs/atlas-object-id`** (`allocator.go`): `Allocator` interface
  `Allocate(ctx,t)(uint32,error)` / `Release(ctx,t,id)error` / `Clear`. Constructor
  `NewRedisAllocator(rc)`. `MinId=1_000_000`, `MaxId=2147483647`. **Per-tenant,
  Redis-backed, shared per-field pool** (keys `atlas:oid:<tenantId>:next|free`) —
  summon oids never collide with monster/drop/reactor oids (Q4 = yes).
- **`libs/atlas-redis`**: `Registry[K,V]` (`NewRegistry(rc, namespace, keyFn)`;
  `Get`/`Put`/`Remove`/`Update(ctx,key,fn)`/`GetAll`/`Exists`). `KeyedSet[K]`
  (`NewKeyedSet(rc, namespace, keyFn)`; `Add`/`Remove`/`Members`/`IsMember`/`Clear`).
  Namespace → key `atlas:<namespace>:<…>`. All summon Redis access MUST route through
  these (redis-key-guard).

Summon registry namespaces: store `"summon"`, field index `"summon-map"`, owner
index `"summon-owner"`. Key suffix shape mirrors monster:
`<tenantId>:<id>` (store), `<tenantId>:<w>:<c>:<m>:<instance>` (field index),
`<tenantId>:<characterId>` (owner index).

---

## 4. Packet library — `libs/atlas-packet/summon/`

- Already in `go.work` (line 12) and root `Dockerfile` (COPY lines 38/66) — a new
  subpackage needs **no** `go.work`/Dockerfile edits.
- Layout: `summon/clientbound/` (6 writers) + `summon/serverbound/` (3 decoders).
- **Writer pattern** (`monster/clientbound/spawn.go`): `Encode(l,ctx) func(opts) []byte`;
  `w := response.NewWriter(l)`; `t := tenant.MustFromContext(ctx)`; chain
  `w.WriteInt/WriteByte/WriteShort/WriteInt16/WriteAsciiString/WriteByteArray`;
  return `w.Bytes()`.
- **Decoder pattern** (`chat/serverbound/general.go`): `Decode(l,ctx) func(r *request.Reader, opts)`;
  `r.ReadInt/ReadByte/ReadInt16/ReadUint32/ReadAsciiString`.
- **Version idiom**: branch on `t.Region()`, `t.IsRegion("GMS")`, `t.MajorAtLeast(n)`,
  `t.MajorAtMost(n)`, `t.MajorInRange(lo,hi)`. Per
  `bug_majorversion_gt83_is_off_by_one_v87`, gate new structure on `>=87`, NOT `>83`
  (v84/v86 are byte-identical to v83).
- **Tests**: `libs/atlas-packet/test` — `test.Variants` (GMS v28/83/84/86/87/95,
  JMS v185 — note **no v92** in harness), `test.CreateContext(region,maj,min)`,
  `test.RoundTrip(t,ctx,encode,decode,opts)` (asserts zero leftover bytes). Loop all
  variants per `character/clientbound/spawn_test.go:18-27`.

**v83 packet layouts (Cosmic `tools/PacketCreator.java`), confirmed/extended per
version via IDA in Phase 6:**

| Packet | Cosmic | Body |
|---|---|---|
| `SummonSpawn` | :1149 | int ownerId, int oid, int skillId, byte `0x0A`(v83 marker), byte level, pos(short x, short y), byte stance, short 0, byte movementType, bool `!isPuppet`, bool `!animated` |
| `SummonRemove` | :1172 | int ownerId, int oid, byte (4 animated / 1 instant) |
| `SummonMove` | :2284 | int cid, int oid, pos(short x, short y) startPos, raw movement bytes |
| `SummonAttack` | :2308 | int cid, int oid, byte 0, byte direction, byte count, per target {int oid, byte 6, int dmg} |
| `SummonDamage` | :4076 | int cid, int oid, byte 12, int dmg, int monsterIdFrom, byte 0 |
| `SummonSkill` | :4569 | int cid, int summonSkillId, byte newStance |

Serverbound decoders mirror the move/attack/damage reads (Cosmic
`MoveSummonHandler`/`SummonDamageHandler`/`DamageSummonHandler`).

---

## 5. Cross-service contracts

### Emit (atlas-summons → existing topics)
- **Monster `DAMAGE`** — `COMMAND_TOPIC_MONSTER`, `command[damageCommandBody]`
  (`WorldId/ChannelId/MapId/Instance/MonsterId/Type/Body`; body
  `{CharacterId uint32, Damages []uint32, AttackType byte}`). Set `CharacterId` =
  owner ⇒ XP/drops/kill credit (FR-4.2). `CommandTypeDamage="DAMAGE"`.
- **Monster `APPLY_STATUS`** — same topic, body
  `{SourceType, SourceCharacterId, SourceSkillId, SourceSkillLevel,
  Statuses map[string]int32, Duration uint32, TickInterval uint32}`,
  `CommandTypeApplyStatus="APPLY_STATUS"` (stun/freeze, FR-4.4).
- **Monster `ADD_PUPPET`/`REMOVE_PUPPET`** — **NEW** command types added in Phase 4
  (the only `atlas-monsters` code change).
- **Character `CHANGE_HP`** — `COMMAND_TOPIC_CHARACTER`,
  `CharacterCommand[ChangeHPCommandBody]` (`{CharacterId, WorldId, Type, Body}`,
  body `{ChannelId, Amount int16}`), `CommandChangeHP="CHANGE_HP"` (Beholder heal).
- **Buff `APPLY`** — `COMMAND_TOPIC_CHARACTER_BUFF`, `Command[ApplyCommandBody]`
  (`{WorldId,ChannelId,MapId,Instance,CharacterId,Type,Body}`, body
  `{FromId uint32, SourceId int32, Level byte, Duration int32, Changes []StatChange}`,
  `StatChange{Type string, Amount int32}`), `CommandTypeApply="APPLY"`. Beholder buff
  uses `SourceId = -int32(1320009)` (Q3 negated skill id, collision-free).

### Consume (existing topics → atlas-summons)
- **`EVENT_TOPIC_CHARACTER_STATUS`** — `StatusEvent[E]`
  (`{TransactionId, WorldId, CharacterId, Type, Body}`). Despawn cascade on:
  - `LOGOUT` (`StatusEventTypeLogout`) body `{ChannelId, MapId, Instance}`
  - `CHANNEL_CHANGED` (`StatusEventTypeChannelChanged`) body `{ChannelId, OldChannelId, MapId, Instance}`
  - `MAP_CHANGED` (`StatusEventTypeMapChanged`) body `{ChannelId, OldMapId, OldInstance, TargetMapId, TargetInstance, TargetPortalId}`

  All carry `CharacterId` + `WorldId` at envelope level ⇒ despawn via owner index.

### New topics
- **`COMMAND_TOPIC_SUMMON`** (channel → summons): `SPAWN`/`MOVE`/`ATTACK`/`DAMAGE`.
- **`EVENT_TOPIC_SUMMON_STATUS`** (summons → channel): `CREATED`/`MOVED`/`ATTACKED`/`DAMAGED`/`DESTROYED`.

Both env vars added to `deploy/k8s/base/env-configmap.yaml`.

---

## 6. atlas-channel wiring

- **Skill-cast branch**: `socket/handler/character_skill_use.go` — after
  `GetEffect` (line ~70), add `if summon.IsSummonSkill(skillId)` → emit
  `COMMAND_TOPIC_SUMMON SPAWN{owner, skillId, level, field, x, y}` (caster position
  from session).
- **Writers**: register 6 summon writer names in `produceWriters()` (`main.go:586-686`).
- **Handlers**: register 3 summon handler entries in `produceHandlers()`
  (`main.go:688-762`); handler funcs in `socket/handler/summon_*.go` mirror
  `character_move.go:14-22` signature
  `func(l,ctx,wp) func(s session.Model, r *request.Reader, opts)`.
- **Status consumer**: new `kafka/consumer/summon/consumer.go` mirrors
  `kafka/consumer/monster/consumer.go` (InitConsumers 34-40, InitHandlers 42-118,
  `handleStatusEventX`(120-156), `ForSessionsInMap` broadcast(`map/processor.go:45`)).
- **Command emit**: `summon/processor.go` mirrors `monster/processor.go:56-59`
  (`producer.ProviderImpl(l)(ctx)(TOPIC)(Provider(...))`).

---

## 7. Skill-effect data client (atlas-summons `data/`)

Channel-side effect `Model` (`services/atlas-channel/.../data/skill/effect/model.go`)
has getters `Duration()`, `X()`, `Y()`, `Prop()`, `MonsterStatus()` but
`weaponAttack`/`magicAttack` are private with **no getter** (intentional). The
atlas-data REST `GET /data/skills/{skillId}` returns a JSON:API resource whose
effect attributes include `weaponAttack`, `magicAttack`, `duration`, `x`, `y`,
`prop`, `monsterStatus`. So `atlas-summons` adds its own `data/` REST client
(env `DATA_SERVICE_URL`, path `data/skills/%d`, `requests.GetRequest[RestModel]` +
`Extract`) exposing `WeaponAttack()`/`MagicAttack()`/`Duration()`/`X()`/`Y()`/
`Prop()`/`MonsterStatus()`. Mirror `services/atlas-channel/.../data/skill/requests.go`.

---

## 8. Damage ceiling (FR-4.3) — resolved approach

No reusable per-hit ceiling exists in Atlas (server currently trusts client damage).
Port Cosmic `SummonDamageHandler.calcMaxDamage` (`:123-145`). Owner combat stats via
**`atlas-effective-stats`**: `GET /worlds/{w}/channels/{c}/characters/{id}/stats`
(env `EFFECTIVE_STATS`), RestModel exposes `WeaponAttack`/`MagicAttack`/`Strength`/
`Dexterity`/`Intelligence`/`Luck` (all uint32). Plan lands a **conservative ceiling
first** (real clamp, not a stub — magic/physical attack-multiplier bound) then the
weapon-type-aware `maxBaseDamage` port as a follow-on task in the same phase. The
conservative phase is an explicit, logged limitation (never a silent `// TODO`).

## 9. Autoban (FR-4.3 alert) — resolved approach

`atlas-ban` (`COMMAND_TOPIC_BAN`, `Command[CreateCommandBody]`) exists. Cosmic
**clamps-and-continues** (does not ban). Plan: clamp damage + emit a structured
warning alert (owner, skillId, mob, reported vs max). The ban topic is documented as
available but **intentionally not auto-fired** for summon-damage clamps (false-positive
risk across versions); promoting to a real ban is a deliberate later decision, not part
of this task.

---

## 10. Multi-version & opcodes

- 7 seed templates: `services/atlas-configurations/seed-data/templates/template_{gms_12_1,gms_83_1,gms_84_1,gms_87_1,gms_92_1,gms_95_1,jms_185_1}.json`.
  Each has `socket.handlers[]` (`{opCode:"0x..",validator,handler}`) and
  `socket.writers[]` (`{opCode:"0x..",writer,options?}`). Add 6 writer + 3 handler
  opcode entries to every template (Phase 1 seeds v83 only; Phase 6 fills the rest).
- Resolution: `libs/atlas-opcodes/producer.go` (`BuildWriterProducer`/`BuildHandlerMap`
  parse `opCode` → uint16); wired in `atlas-channel/main.go:366-391`.
- **Opcode byte values and per-version layout deltas are client-fixed — harvest from
  IDA, never invent.** Precedent: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`.
  IDBs loaded one at a time (`reference_ida_harvest_subagents`): v83/v84/v87/v95/JMS185
  (no v12/v92 IDB — derive those from config/known deltas + tests, documented).
- Live tenants need a config patch + channel restart to pick up new opcodes
  (`bug_new_opcodes_not_in_live_tenant_config`) — operational note, not code.

---

## 11. Roster (FR-1.1) — `libs/atlas-constants/summon/roster.go`

Static table keyed by the 21 skill-id constants already in
`libs/atlas-constants/skill/constants.go`. `Lookup(skillId)(Entry,bool)` +
`IsSummonSkill(skillId)bool`. Subpackage of already-vendored `atlas-constants` ⇒ no
`go.work`/Dockerfile change. Full roster in `design.md` Appendix A:
- **Puppet / stationary(0):** 3111002, 3211002, 13111004
- **Attacker / stationary(0):** 5211001, 5220002
- **Attacker / circle-follow(3):** 3111005, 3211005, 3121006, 3221005, 2311006, 5211002 (Gaviota one-shot)
- **Attacker / follow(1):** 2121005, 2221005, 2321003, 11001004, 12001004, 12111004, 13001004, 14001005, 15001004
- **Buff-aura / follow(1):** 1321007 (Beholder; HP = effect x+1; other puppet HP = effect x)
- Stun: 3111005, 3211005. Freeze: 3221005, 2121005.

Out of scope (graceful no-op): Dual Blade (v88+), Evan dragon (v84), Aerial Strike
(5221003 dead constant), Battleship (5221006 mount).

---

## 12. Verification gate (every phase end, from worktree root)

1. `go test -race ./...` clean in changed modules.
2. `go vet ./...` clean.
3. `go build ./...` clean.
4. `docker buildx bake atlas-summons` (and `atlas-monsters`/`atlas-channel` when their
   code changes) from worktree root.
5. `tools/redis-key-guard.sh` clean (run with `GOWORK=off`).

New service ⇒ no new shared lib ⇒ no Dockerfile COPY edits; but
`.github/config/services.json` + `docker-bake.hcl` derive + `go.work` must list
`atlas-summons`.
