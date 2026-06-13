# task-093-mystic-door — Implementation Context

Companion to `plan.md`. Captures the key files, decisions, and dependencies an
implementer needs before starting. Read this once; refer back as needed.

Paths are repo-relative to the worktree root unless noted. `<repo-root>` =
`.worktrees/task-093-mystic-door`. Sibling worktrees are referenced as
`.worktrees/<other-task>/...`.

## What we're building

Mystic Door (Priest skill `2311002`): the first persistent, party-shared,
two-map field object in Atlas. A new version-agnostic engine service
**`atlas-doors`** owns door lifecycle (registry, id allocation, slot allocation,
leader-elected expiry, Kafka command/event topics, REST). **`atlas-channel`**
stays the thin per-version packet edge: it routes the cast, decodes the
enter-door packet, performs the warp through the existing portal path, and
broadcasts spawn/remove/party-minimap packets. All version variance lives in a
new `libs/atlas-packet/door` package plus per-version tenant socket-template
opcodes.

PRD: `prd.md`. Design: `design.md`. Both resolve the five open questions.

## Structural template: atlas-summons (CRITICAL)

`atlas-doors` is modeled one-for-one on **`atlas-summons`**, which lives in a
**sibling in-flight worktree**, NOT in this worktree and NOT on `main`:

```
.worktrees/task-088-player-summons/services/atlas-summons/atlas.com/summons/
```

- atlas-doors **borrows patterns** from atlas-summons (copy a file, rename
  `summon`→`door`, apply the deltas the plan calls out). It **must NOT import**
  `atlas-summons` — they are independent services.
- atlas-doors does **not** depend on task-088 merging. It is a brand-new module.
- If the task-088 worktree is gone at execution time, the same patterns exist in
  `atlas-monsters` (registry/id-allocator/expiry) and any status-event consumer;
  atlas-summons is just the closest match (owner-bound, expiring, oid-occupying
  field object).

The atlas-summons skeleton (verified during planning):

```
main.go            bootstrap: logger → redis → InitIdAllocator/InitRegistry →
                   tracing → kafka consumers (AddConsumer + InitHandlers) →
                   REST routes (InitResource) → leader-elected tasks.Register
leaderconfig.go    DOOR_LEADER_* env parsing (copy verbatim, rename SUMMON→DOOR)
tasks/task.go      Task interface {Run(); SleepTime()} + Register goroutine loop
logger/            logrus + ECS hook (copy verbatim)
summon/
  model.go         immutable Model, getters, copy-on-write mutators
  builder.go       NewBuilder + Clone + chainable setters + Build
  id_allocator.go  wraps libs/atlas-object-id (per-tenant, MinId 1_000_000)
  registry.go      atlasredis.Registry[string,stored] + KeyedSet indices
  processor.go     Processor interface + Impl, NewProcessor(l, ctx)
  kafka.go         Command[E]/StatusEvent[E] envelopes + topic env consts
  producer.go      event provider helpers (model.Provider[[]kafka.Message])
  resource.go      RestModel + Transform (JSON:API; GetName, GetID/SetID)
  rest.go          InitResource(si) → GET /summons/{id}
  expiry_task.go   leader-elected sweep (GetAll grouped by tenant)
kafka/consumer/
  consumer.go      NewConfig curry + LookupBrokers
  summon/          COMMAND_TOPIC_SUMMON handlers (InitConsumers/InitHandlers)
  character/       EVENT_TOPIC_CHARACTER_STATUS handlers (LOGOUT/CHANNEL/MAP)
rest/handler.go    type aliases + ParseXxxId helpers
world/resource.go  GET /worlds/.../maps/{m}/instances/{i}/summons
data/skill/        REST client → atlas-data skill effect
```

## Key channel-side seams (this worktree)

| Seam | File | Use |
|---|---|---|
| Per-skill cast dispatch | `services/atlas-channel/atlas.com/channel/skill/handler/common.go:121` | `Lookup(skillId)` is invoked after cost/cooldown/buff. Register a door handler here (same seam as Heal). MP + Magic Rock already consumed at lines 73–95; Mystic Door has duration but no statups so no phantom buff. |
| Handler registry | `.../skill/handler/registry.go` | `Register(id, Handler)` / `Lookup(id)`. |
| Registration driver | `.../skill/handler/registrations/registrations.go` | blank-import new handler subpackage here; `main.go` imports this package. |
| Heal handler (example) | `.../skill/handler/heal/` | the closest existing per-skill handler to copy structure from. |
| Portal warp | `.../portal/processor.go` | `NewProcessor(l, ctx)`; `Warp(f field.Model, characterId uint32, targetMapId _map.Id) error`. Emits a warp command on `portal.EnvPortalCommandTopic`. |
| Inbound handler + validator | `.../socket/handler/handle.go` (`LoggedInValidator`/`LoggedInValidatorFunc`), `main.go:695-777` (`produceHandlers`/`produceValidators`) | register the enter-door handler name→func in `produceHandlers`, ensure its tenant-template entry uses `LoggedInValidator` (a validator-less handler is silently dropped). |
| Inbound handler example | `.../socket/handler/buddy_operation.go` | signature `Func(l, ctx, wp) func(s session.Model, r *request.Reader, opts map[string]any)`. |
| Map-enter spawn-for-self | `.../kafka/consumer/map/consumer.go` `SpawnForSelf(...)` (~line 154-310) | where monsters/npcs/drops/summons get spawned to an entering session; add door spawns after line 310. |
| Status-event consumer (example) | `.../kafka/consumer/monster/consumer.go` | `SetHeaderParsers(Span, Tenant)`; handler uses `_map.NewProcessor(l,ctx).ForSessionsInMap(field, fn)` + `session.Announce(l)(ctx)(wp)(Writer)(body)(s)`. |
| Party membership | `.../party/processor.go` (`GetByMemberId`), `.../character/processor.go` `PartyDecorator`; filters `party.MemberInMap(field)`, `party.OtherMemberInMap(field, charId)` | read the caster's party + same-channel members in a map. |
| Map data model | `.../data/map/model.go` | `ReturnMapId()`, `ForcedReturnMapId`, `FieldLimit()`, `Town()` — but does **not** expose portals. atlas-doors needs its own atlas-data map client for portals. |
| Party packet door fields | `libs/atlas-packet/party/clientbound/created.go` | already reserves door map x/y (int) + minimap x/y (short), currently hard-zeroed. `partyPortal` = populate these from live door state. |

## atlas-data portal facts (resolves OQ-3 mechanics)

- `GET /data/maps/{mapId}/portals` returns all portals; door portals have
  `Type == 6` (`PortalTypeDoor`, `services/atlas-data/atlas.com/data/map/reader.go:137`).
- atlas-data assigns **door-type portals sequential ids starting at 1 per map
  load** (`atomic.AddUint32(&portalId, 1)`) — it does **NOT** expose `0x80+slot`
  ids. So: fetch portals, filter to `Type==6` in load order, index by party slot
  for the **position**; encode `0x80+slot` as the **wire** portal id (the client
  addresses door portals as `0x80+slot`). See design §6.3.
- **Open data risk (verify first, Task 0):** confirm the towns players actually
  return to (Henesys 100000000, Ellinia 101000000, Perion 102000000, Kerning
  103000000, Sleepywood 105040300, Lith Harbor 104000000, Nautilus 120000000,
  Orbis 200000000, etc.) expose ≥6 `Type==6` door portals across versions. If a
  town has <6, the §6.3 fallback (default door position near spawn portal) is
  exercised in normal play.

## Constants already present (reuse — do not redefine)

- `skill.PriestMysticDoorId = Id(2311002)` — `libs/atlas-constants/skill/constants.go:3069`.
- `map.FieldLimitNoMysticDoor uint32 = 0x02` — `libs/atlas-constants/map/field_limit.go:9`.
- `world.Id` = `byte`, `channel.Id` = `byte`, `_map.Id` = `uint32`.
- `field.Model` via `field.NewBuilder(worldId, channelId, mapId).SetInstance(uuid).Build()`.
- object-id shared pool: `libs/atlas-object-id`, `MinId = 1_000_000`.

## Service registration (hand-synced — all required)

- `.github/config/services.json` — add a `go-service` entry (see plan Task 1).
- `docker-bake.hcl` — add `"atlas-doors"` to the hardcoded `go_services` list
  (HCL can't read JSON; both must be edited — memory:
  `reference_docker_bake_hand_synced`).
- `go.work` — add `./services/atlas-doors/atlas.com/doors`.
- Root `Dockerfile` — **no edit** (atlas-doors adds no new shared lib; the shared
  Dockerfile builds via `COPY services/${SERVICE}/`).
- `deploy/k8s/base/atlas-doors.yaml` — Deployment + Service mirroring
  atlas-summons; readiness probe path **`/api/readyz`** (not `/readyz`); do NOT
  hard-code `*_SERVICE_URL` from the kustomize base — rely on `BASE_SERVICE_URL`
  (memory: `bug_service_url_hardcoded_base_namespace`,
  `bug_readiness_probe_path_under_api_basepath`).

## Per-version packet work (OQ-5) — the honest constraint

Six packets (`spawnDoor`, `removeDoor`, `spawnPortal`, `playPortalSound`,
`partyPortal`, enter-door) × six versions (`gms_v83/84/87/92/95`, `jms_v185`).

- **v83 is fully specified from Cosmic** (`~/source/Cosmic` `PacketCreator.java`,
  `DoorHandler.java`) — transcribe the byte sequence, write golden-byte tests.
- **Other versions are IDA/WZ-verified, not ported blindly from v83** (opcode
  table shifts ≥0x3D, `MajorVersion()>83` off-by-one for v84 — v84≡v83 for
  structure; use `>=87` not `>83`). Memory: `bug_v84_opcode_table_shifted_vs_v83`,
  `bug_majorversion_gt83_is_off_by_one_v87`.
- **An opcode whose fname doesn't resolve in the IDB is a STOP-AND-ESCALATE**
  (memory: `feedback_unresolved_fname_escalate`). Park that version/packet like
  v92 mount-food; never guess a hash or substitute an fname.
- New opcodes must be added to tenant socket **templates** AND patched into
  **live tenant config** (existing tenants don't auto-receive them; channel
  restart required — memory: `bug_new_opcodes_not_in_live_tenant_config`,
  `bug_socket_handler_missing_validator_silently_dropped`).

## Decisions locked in (from design)

- **OQ-1:** no new cost logic — `UseSkill` already consumes MP + Magic Rock.
- **OQ-2:** warp is channel-side via `portal.Warp`; atlas-doors never warps.
- **OQ-4:** `FieldLimitNoMysticDoor` gate covers instances; return map resolves to
  non-instanced town (`uuid.Nil`).
- One `door.Model` represents the **pair** (area + town) with one `pairId =
  areaDoorId`. Two oid allocations per door; both released on remove.
- Allocation failure **fails the spawn** (no fallback to MinId).
- Recast handled inside SPAWN (remove existing owner door first).
- Town index keyed by `partyId` (per-party slot scope); solo casters namespaced
  by owner so two solo slot-0 doors at one town don't collide.
- Deploy grace ~3000ms before remove broadcast (FR-6.3 client-crash guard).

## Verification gates (CLAUDE.md)

Per changed module: `go test -race ./...`, `go vet ./...`, `go build ./...`.
`docker buildx bake atlas-doors` and `docker buildx bake atlas-channel` from the
worktree root. `tools/redis-key-guard.sh` clean (run with `GOWORK=off`).
