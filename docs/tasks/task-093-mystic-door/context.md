# Mystic Door — Implementation Context

Companion to `plan.md`. Captures the key files, decisions, dependencies, and
**research-driven corrections to the design** that an executing engineer must know.
Read this before starting any task.

---

## 1. What we are building

Mystic Door (Priest skill `2311002`). A new version-agnostic engine service
**`atlas-doors`** owns door lifecycle (Redis registry, shared object-id allocator,
leader-elected expiry sweep, per-party town-slot allocation, Kafka command/event
topics, REST). **`atlas-channel`** is the thin per-version packet edge: it routes the
cast, decodes the enter-door packet, warps via the existing portal path, and
broadcasts spawn/remove/party-minimap packets to eligible viewers. All version
variance lives in a new `libs/atlas-packet/door` package + per-version tenant socket
template opcodes.

Source of truth for behavior: Cosmic v83 (`~/source/Cosmic`:
`server/maps/Door.java`, `DoorObject.java`, `net/server/channel/handlers/DoorHandler.java`,
`net/server/world/Party.java`, `tools/PacketCreator.java`).

---

## 2. CRITICAL corrections to the design (verified during planning)

The design (`design.md`) is sound, but research surfaced concrete facts that override
or sharpen it. **These are load-bearing.**

1. **`atlas-summons` does NOT exist in this worktree.** It lives in the sibling
   worktree `.worktrees/task-088-player-summons/` (task-088, unmerged). Do **not**
   reference or import it. The concrete in-tree mirror is **`atlas-monsters`**
   (`services/atlas-monsters/atlas.com/monsters/`), which has the identical patterns we
   need: `main.go` leader-election block, `leaderconfig.go`, `tasks/task.go`, the
   `libs/atlas-object-id` allocator wrapper, the Redis registry, and Kafka
   command/event consumers. **Mirror `atlas-monsters`, not `atlas-summons`.**

2. **Opcodes are never hardcoded in Go.** Clientbound writers are referenced by a
   string *name* (`const XxxWriter = "Xxx"`); the tenant socket config maps that name →
   opcode at runtime (`writerProducer(writerName)`). Inbound handlers are keyed by a
   *handle name* (`const XxxHandle`) likewise resolved per tenant. **Consequence:** the
   Go encoders/decoders branch on packet *structure* only; all per-version *opcode*
   wiring is a tenant-template/live-config task (Part H), not Go code.

3. **No `WritePos` helper exists.** `libs/atlas-socket/response/writer.go` offers
   `WriteByte/WriteBool/WriteShort/WriteInt/WriteInt8/WriteInt16/WriteInt32/
   WriteAsciiString/WriteByteArray`. `WriteInt16 == WriteShort(uint16)`,
   `WriteInt32 == WriteInt`. Door map coords are two ints; minimap coords are two shorts
   (see the reserved party block, correction #8).

4. **`PortalTypeDoor` is NOT a shared constant.** It is package-private to atlas-data:
   `services/atlas-data/atlas.com/data/map/reader.go:137` `PortalTypeDoor uint8 = 6`.
   Portal `Type` on the REST model is `uint8`. atlas-doors classifies a door portal by
   `RestModel.Type == 6`.

5. **`FieldLimitNoMysticDoor` is a bare `uint32` (0x02), not a typed enum.**
   `libs/atlas-constants/map/field_limit.go:9`. There is **no** `NoMysticDoor(limit)`
   helper (only `NoExpLossOnDeath`). Test the bit inline:
   `fieldLimit & _map.FieldLimitNoMysticDoor != 0`.

6. **Skill effect carries the cost data.** `effect.Model` (channel side,
   `data/skill/effect/model.go`): `Duration() int32` (milliseconds, **`-1` = no
   duration** sentinel), `MPConsume()`, `ItemConsume() uint32` (the Magic Rock item id,
   sourced from WZ `itemCon` — Magic Rock `4006000` has **no** Go constant), X/Y. The
   channel effect getters for X/Y are **not yet exposed** — add them if needed.
   Duration-by-level: `data/skill.Processor.GetEffect(skillId, level)` (level **1-based**).

7. **Magic Rock / MP consume is already handled generically.** `skill/handler/
   common.go:73-95` (`UseSkill`) consumes HP/MP and emits `REQUEST_ITEM_CONSUME` for
   `e.ItemConsume()` *before* per-skill dispatch (`Lookup` at `common.go:121`), then
   only applies a character buff when `Duration>0 && len(StatUps)>0`. **OQ-1 resolved:
   no new cost logic, no double-consume.** We hook the existing per-skill `Lookup`
   dispatcher (the seam Heal/Dispel use), not `character_skill_use.go`.

8. **Party door fields are already reserved (hard-zeroed) in
   `libs/atlas-packet/party/clientbound/created.go`.** The block is **int townMapId,
   int targetMapId, short x, short y** (currently `EmptyMapId,EmptyMapId,0,0`). There is
   **no** separate "partyPortal"/door-update operation packet today; per-member door
   data also lives in `party/member_data.go` (`WritePartyData`). The stale TODO
   `docs/TODO.md:146` cites a deleted path — update it to point at `created.go`.

9. **`PlayPortalSound` already exists — do NOT add a packet.** It is a saga action
   (`libs/atlas-saga/model.go:95`) and a character *simple effect*
   (`character/clientbound/effect.go` `EffectSimple`/`EffectSimpleForeign`). Reuse the
   existing simple-effect path for the portal sound on warp (FR-5.3); do not invent a
   `playPortalSound` packet. This removes one packet from the FR-7.1 list.

10. **Per-version door opcode *bytes* are NOT yet verified.** The design defers this to
    a plan-phase IDA matrix (OQ-5). The plan provides Cosmic-derived **structure**
    (field order) for the encoders and a per-version **opcode-resolution + golden-test**
    procedure. **Unresolved fnames are stop-and-escalate** (memory:
    `feedback_unresolved_fname_escalate`) — never guess an opcode or fake a hash.

11. **Channel map/portal clients are trimmed.** `data/map` (channel) omits portals and
    `forcedReturnMapId`; `data/portal` (channel) `Model` has only `Id()` and only a
    by-name request. atlas-doors needs its **own** atlas-data clients (full map +
    all-portals with X/Y/Type/TargetMapId getters). Do not assume the channel clients
    suffice.

12. **Party member order is join order, leader-seeded at index 0 — but index 0 ≠ leader
    after a leadership change** (`party/registry.go:42` seeds `[leaderId]`; `SetLeader`
    only swaps the scalar). Slot = member's 0-based index in `Members()` (matches the
    existing `recipients.go:91-99 SelectInRangePartyMembers` precedent). Order is
    deterministic and stable across the Redis/REST round-trip.

---

## 3. Key files & seams (verified, in the task-093 worktree)

### In-tree mirror service — `services/atlas-monsters/atlas.com/monsters/`
- `main.go` — leader-election block (`lock.New(rc, "<name>-sweep", …)`, `le.Run`),
  consumer + REST init order, `tasks.Register`.
- `leaderconfig.go` — `<PREFIX>_LEADER_ELECTION_ENABLED/_TTL/_REFRESH/_BACKOFF` parsing.
- `tasks/task.go` — `Task` interface (`Run()`, `SleepTime()`) + `Register(l,ctx)(t)`
  goroutine loop.
- `monster/` — registry (`atlasredis.Registry` + `KeyedSet` indices, tenant id in key
  suffix), id allocator wrapper (`objectid.NewRedisAllocator`), processor
  (`NewProcessor(l,ctx)`, `tenant.MustFromContext`), producer/kafka envelope, rest +
  resource (JSON:API), in-field list route.
- `kafka/consumer/…` — command + character-status consumers
  (`SetHeaderParsers(Span,Tenant)`, `SetStartOffset(LastOffset)`, curried
  `InitConsumers(l)(cmf)(groupId)` / `InitHandlers`, `message.AdaptHandler(
  message.PersistentConfig(handler))`, per-type `if c.Type != … { return }` guard).

> The summons-documented patterns are the architectural ideal; the **in-tree** code to
> copy verbatim is atlas-monsters.

### Shared libs (all in-tree, verified)
- `libs/atlas-redis` — `NewRegistry[K,V]`, `NewKeyedSet[K]`; rediskeyguard requires all
  keyed Redis access through these (`tools/redis-key-guard.sh`, `GOWORK=off`).
- `libs/atlas-object-id` — `NewRedisAllocator(rc)`, `Allocate(ctx,t)`, `Release(ctx,t,id)`,
  `MinId = 1_000_000`. **Two allocations per door** (area + town); release both on
  removal; **fail the spawn** on allocation error (no silent `MinId` fallback).
- `libs/atlas-lock` — `lock.New(rc, name, WithTTL/WithRefreshInterval/WithBackoff/
  WithLogger)`, `le.Run(ctx, onElected)`.
- `libs/atlas-kafka/producer` — `CreateKey(int)`, `SingleMessageProvider(key,&val)`,
  `ProviderImpl(l)(ctx)(topic)(provider)`; header decorators auto-applied.
- `libs/atlas-tenant` — `MustFromContext(ctx)`, `WithContext(ctx,t)`; version API
  `Region()`, `IsRegion("GMS")`, `MajorVersion()`, `MajorAtLeast(v)`, `MajorAtMost(v)`.
  **Use `IsRegion("GMS") && MajorAtLeast(87)`, never `> 83`** (off-by-one for v84-86).
- `libs/atlas-constants` — `skill.PriestMysticDoorId` (`skill/constants.go:3069`,
  =2311002), `map.FieldLimitNoMysticDoor` (`map/field_limit.go:9`, 0x02), `world.Id=byte`,
  `channel.Id=byte`, `_map.Id=uint32`, `field.NewBuilder(w,c,m).SetInstance(uuid).Build()`.
- `libs/atlas-packet` — clientbound `const XxxWriter`, `Encode(l,ctx)(options)[]byte`
  with `t := tenant.MustFromContext(ctx)` version branch; serverbound `const XxxHandle`,
  `Decode(l,ctx)(r,options)`; test helpers `test/context.go` (`pt.Variants`,
  `pt.CreateContext`), `test/roundtrip.go` (`pt.RoundTrip` asserts zero unconsumed bytes).
  Reserved party door block: `party/clientbound/created.go`.

### atlas-channel edge seams — `services/atlas-channel/atlas.com/channel/`
- `skill/handler/common.go` — `UseSkill` (cost at :73-95), `Lookup` dispatch (:121).
- `skill/handler/registry.go` — `Handler` type, `Register(id,h)`, `Lookup(id)`.
- `skill/handler/heal/heal.go` — per-skill handler template (`init(){ Register(...) }`).
- `skill/handler/registrations/registrations.go` — blank-import list (add mysticdoor).
- `skill/handler/recipients.go:91-99` — `SelectInRangePartyMembers` (party-slot precedent).
- `socket/handler/handle.go` — `LoggedInValidator` const; handler/validator types.
- `socket/handler/portal_script.go` + `character_drop_meso.go` — inbound handler template.
- `kafka/consumer/mist/consumer.go` — status-event→broadcast template
  (`InitConsumers`/`InitHandlers`, `sc.Is(tenant,…)` guard, package-var broadcaster).
- `kafka/consumer/map/consumer.go:189-211` — SpawnForSelf block (per-object spawn to
  arriving session); `:494 spawnReactorsForSession` operator template.
- `map/processor.go` — `ForSessionsInMap`, `ForOtherSessionsInMap`.
- `session/processor.go:170` — `Announce(l)(ctx)(wp)(writerName)(encode)`;
  `IfPresentByCharacterId(channel)(charId, op)`.
- `portal/processor.go:43` — `Warp(f, characterId, targetMapId _map.Id)` (emits
  `COMMAND_TOPIC_PORTAL`).
- `kafka/producer/producer.go:12` — `ProviderImpl(l)(ctx)(topic)(provider)`.
- `kafka/message/portal/kafka.go` + `portal/producer.go` — command envelope + provider
  template (mirror for the door SPAWN command).
- `main.go` — `produceWriters()` (~600), `produceHandlers()` (~695),
  `produceValidators()` (~772), consumer init (~180), handler init (~445).
- `data/skill` (channel) — `GetById`, `GetEffect(id,level)` (level 1-based).
- `party/requests.go` / `party/processor.go` — `GetByMemberId`, `GetById`, `Members()`.

### Registration / deploy
- `.github/config/services.json` — add the `atlas-doors` go-service object.
- `docker-bake.hcl` — add `"atlas-doors"` to the hardcoded `go_services` list
  (HCL can't read JSON; both must be edited).
- `go.work` — add `./services/atlas-doors/atlas.com/doors`.
- Root `Dockerfile` — **no edit** (parameterized by `ARG SERVICE`; no new shared lib).
- `deploy/k8s/base/atlas-doors.yaml` — mirror `atlas-monsters.yaml`; if a readiness
  probe is added, path **`/api/readyz`**. `deploy/k8s/base/kustomization.yaml` — add
  `- atlas-doors.yaml`. `deploy/k8s/base/env-configmap.yaml` — add `COMMAND_TOPIC_DOOR`
  + `EVENT_TOPIC_DOOR_STATUS`.

---

## 4. Architectural decisions (locked)

- **New standalone `atlas-doors` service** (design §3-A). Rejected: folding into channel
  (B), generalizing summons (C).
- **One `door.Model` per pair** (area + town share one record, one `pairId = areaDoorId`)
  — avoids cross-record consistency on expiry/removal.
- **Channel-side warp** (OQ-2): enter-door handler validates against atlas-doors (REST)
  then `portal.Warp(...)`. atlas-doors never warps.
- **Recast handled inside SPAWN** (FR-1.4): processor removes any existing owner door
  before deploying — single atomic command, no separate REMOVE round-trip.
- **Slot = 0-based party index**, town portal wire id = `0x80 + slot`; the atlas-data
  internal door-portal id supplies only the *position*. Fallback when a town has <6
  door portals: default placement near spawn, still encode `0x80+slot` (design §6.3).
- **Visibility is party-scoped + per-channel** (FR-3, FR-6.5). Channel intersects map
  session enumeration with party membership; caster always included. Leaver visibility
  revocation (send `removeDoor` to the one session, door survives) is a channel concern.
- **Cleanup**: LOGOUT/CHANNEL_CHANGED always remove owner's door; MAP_CHANGED removes
  only when the owner left the **source field** (walking into own town door is a warp,
  not abandonment). Leader-elected expiry sweep is the orphan backstop.
- **Deploy grace (FR-6.3)**: honor ~3000ms from deploy before a remove broadcast to
  avoid the rapid cast→cancel client crash.

---

## 5. Dependencies & ordering

1. **Service scaffold + registration** (Part A) precedes all atlas-doors code.
2. **Model → registry → id allocator → slot/data clients → processor → producer/REST →
   consumers/expiry → main wiring** (Parts B-E) is the engine build order.
3. **`libs/atlas-packet/door`** (Part F) is independent of the engine; gates the channel edge.
4. **atlas-channel edge** (Part G) depends on F (packets) + E (event/command contracts).
5. **Per-version opcode/template/live-config matrix** (Part H) depends on F+G; the
   "version done" gate — includes IDA opcode verification (escalate unresolved fnames).
6. **Final verification** (Part I): `go test -race`, `go vet`, `go build`,
   `docker buildx bake atlas-doors` + `atlas-channel`, `tools/redis-key-guard.sh`.

## 6. Risks (carried from design §12)

| Risk | Action |
|---|---|
| Towns may lack ≥6 `PortalTypeDoor` portals in atlas-data. | Verify via `GET /api/data/maps/{id}/portals` per version (Part C); fallback placement coded. **Highest pre-impl risk.** |
| Per-version door opcodes/bytes unknown until IDA-verified. | Part H matrix; escalate unresolved fnames, never guess. |
| Rapid cast→cancel client crash. | Deploy grace (FR-6.3) before remove broadcast. |
| Concurrent same-party casts race on slot. | Town+party Redis index as slot source of truth under registry serialization. |
| Live tenants don't auto-get new opcodes. | Patch live config + restart channel (Part H). |
