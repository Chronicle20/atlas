# Map Environment Object State & Field Obstacles — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

Atlas can encode four clientbound packets that move or toggle server-driven map objects —
`SetObjectState` (`CField::OnSetObjectState`), `FieldObstacleOnOff`
(`CField::OnFieldObstacleOnOff`), `FieldObstacleOnOffList`
(`CField::OnFieldObstacleOnOffStatus`), and `FieldObstacleAllReset`
(`CField::OnFieldObstacleAllReset`). All four codecs exist in
`libs/atlas-packet/field/clientbound/`, carry pinned `packet-audit:verify` evidence, and are
registered as writers in `services/atlas-channel/atlas.com/channel/main.go:835-836,871-872`.
**Nothing in the system ever emits them.** A repo-wide grep for the four writer constants
returns only the packet definitions, the thin writer wrappers, and the `main.go` registration.

The one script operation that should drive them is a logged stub. `move_environment` is declared
in the reactor script schema (`services/atlas-reactor-actions/docs/reactor_script_schema.json:112,220`),
dispatched in `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:62`, and its
implementation `executeMoveEnvironment` (`.../script/executor.go:278-292`) logs one line and
returns `nil` with the comment "TODO: This needs a new saga action for environment manipulation".
`services/atlas-reactor-actions/docs/domain.md:108` documents it as "not yet implemented".

The player-visible consequence: server-driven map objects never move. Party-quest and gate
environment nodes stay put, ship and level objects never animate, and obstacle on/off sequences
(Kulan Field pieces, Forest of Poison Haze) never fire. This PRD specifies the full path from a
script operation to the wire, with per-field-instance state owned by `atlas-maps` so a character
entering a map after a move sees the current object state rather than the WZ default.

## 2. Goals

Primary goals:

- Emit all four registered field-object packets from real gameplay triggers; retire the
  `executeMoveEnvironment` stub.
- Own environment-object and obstacle state per `field.Model` (world, channel, map, instance) in
  `atlas-maps`, so state survives across characters and is replayed to late joiners.
- Provide a generic saga action pair (`move_environment`, `reset_environment`) usable by any
  service, wired for `atlas-reactor-actions` and `atlas-map-actions` in this task.
- Expose a JSON:API resource on `atlas-maps` for reading and setting environment/obstacle state
  (GM and integration-test use).
- Provide an explicit reset operation that clears tracked state and returns the field's objects
  to their default appearance.

Non-goals:

- The sibling stubs `executeKillAllMonsters` and `executeWeakenAreaBoss`
  (`.../script/executor.go:257-276,294-303`) — separate tasks.
- Authoring the per-map obstacle/environment content for specific party quests (Kulan Field,
  Forest of Poison Haze, Herb Town PQ). This task delivers the mechanism; content lands as script
  data afterwards.
- A stage-transition action hook in `atlas-party-quests`. That service has no action-op executor
  today (its `kill_all_monsters` entries in `docs/party_quests/herb_town_pq.json` are stage
  *completion types*, not actions). PQ environment moves are driven by reactor scripts and map
  scripts inside PQ maps, matching the upstream Cosmic behaviour.
- New packet codecs or version gates. All four codecs are implemented and verified; no wire change
  is in scope.
- Frontend (`atlas-ui`) surfaces.

## 3. User Stories

- As a player in a party quest, I want the gate/platform node to move when the party clears the
  stage trigger, so the stage is actually traversable.
- As a player who enters a PQ map after the party moved a gate, I want to see the gate in its
  moved position, so I do not walk into an invisible wall or fall through a phantom floor.
- As a player in Kulan Field / Forest of Poison Haze, I want field obstacles to switch on and off
  on the server's schedule, so the map hazard behaves as designed.
- As a content author, I want `move_environment` in a reactor or map script to actually move the
  object named in the script, so the converted Cosmic scripts behave like their source.
- As a content author, I want a `reset_environment` operation, so a stage restart returns objects
  to their default state without tearing down the map instance.
- As a GM/operator, I want to read and set a field's environment object state over REST, so I can
  diagnose a stuck PQ without a client.

## 4. Functional Requirements

### 4.1 State ownership (`atlas-maps`)

- **FR-1.** `atlas-maps` gains a `map/environment` package following the shape of the existing
  `map/weather` and `map/jukebox` packages: `registry.go`, `processor.go`, `producer.go`,
  `resource.go`, `rest.go`.
- **FR-2.** The registry is keyed by `FieldKey{Tenant tenant.Model, Field field.Model}`, matching
  `services/atlas-maps/atlas.com/maps/map/weather/registry.go:11-14`. Each entry holds an ordered
  map from object name (`string`) to `{Kind, State}` where `Kind ∈ {ENVIRONMENT, OBSTACLE}` and
  `State` is `uint32`.
- **FR-3.** Object names are opaque strings passed through from the script/command to the wire. The
  server does not validate a name against WZ data (no obstacle/environment name index exists in
  `libs/atlas-wz` or `atlas-data` today). An unknown name is a client-side no-op.
- **FR-4.** `Set(f, kind, name, state)` is idempotent: re-setting an already-tracked
  `(kind, name)` to the same state still emits the status event (scripts may rely on the
  re-broadcast) but does not duplicate the registry entry.
- **FR-5.** `Reset(f)` clears every tracked entry for the field and emits a reset event.
- **FR-6.** State is in-memory and scoped to the field instance. It disappears when the process
  restarts; there is no database persistence (matching weather/jukebox).
- **FR-7.** State entries are removed when the field instance is torn down. If `atlas-maps` has no
  existing instance-teardown hook that the weather/jukebox registries already use, the design phase
  must determine the hook; falling back to "cleared on explicit reset only" is acceptable but must
  be stated in the design doc, not silently assumed.

### 4.2 Commands (`COMMAND_TOPIC_MAP`)

- **FR-8.** Two new command types on the existing `COMMAND_TOPIC_MAP`
  (`services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:12-14`):
  `SET_ENVIRONMENT_STATE` and `RESET_ENVIRONMENT`.
- **FR-9.** `SET_ENVIRONMENT_STATE` body: `{ kind: "ENVIRONMENT"|"OBSTACLE", name: string, state: uint32 }`.
  `RESET_ENVIRONMENT` body: `{}` (field routing comes from the command envelope).
- **FR-10.** Both commands are keyed by `producer.CreateKey(int(f.MapId()))`, matching
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go`.
- **FR-11.** The `atlas-maps` map-command consumer applies the command to the registry and then
  emits the corresponding status event. Command handling is idempotent per FR-4.

### 4.3 Status events (`EVENT_TOPIC_MAP_STATUS`)

- **FR-12.** Three new status types alongside the existing set
  (`services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:12-19`):
  - `ENVIRONMENT_STATE_CHANGED` — body `{ kind, name, state }`
  - `ENVIRONMENT_RESET` — body `{}`
  - No third type is required if the bulk case is folded into FR-14's REST replay; if the design
    chooses to broadcast a bulk list, it adds `ENVIRONMENT_STATE_LIST` with body
    `{ obstacles: [{name, state}] }`.

### 4.4 Wire emission (`atlas-channel`)

- **FR-13.** The `atlas-channel` map consumer
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`) handles the new
  status types and broadcasts to the field via
  `_map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(...))`, the pattern used by
  `handleStatusEventWeatherStart` at `consumer.go:989-1010`.
- **FR-14.** Packet selection by `kind`:
  - `kind == ENVIRONMENT` → `fieldcb.SetObjectStateWriter` with
    `fieldcb.NewSetObjectState(name, state)`.
  - `kind == OBSTACLE` → `fieldcb.FieldObstacleOnOffWriter` with
    `fieldcb.NewFieldObstacleOnOff(name, state)`.
  - Bulk obstacle replay → `fieldcb.FieldObstacleOnOffListWriter` with
    `fieldcb.NewFieldObstacleOnOffList([]fieldcb.ObstacleState{...})`.
  - Reset → `fieldcb.FieldObstacleAllResetWriter` with `fieldcb.NewFieldObstacleAllReset()`.
- **FR-15.** `FieldObstacleOnOffList` has a GMS<61 single-obstacle wire divergence handled inside
  the codec (`libs/atlas-packet/field/clientbound/field_obstacle_on_off_list.go:32-45,71-88`) via
  `NewFieldObstacleLegacy(flag, itemId, name)`. The bulk-replay path must not send a
  multi-obstacle list to a GMS<61 session; on that tenant the channel sends one
  `FieldObstacleOnOff` per obstacle instead. No change to the codec.

### 4.5 Enter replay

- **FR-16.** On `CHARACTER_ENTER`, `atlas-channel` fetches the field's current environment state
  from `atlas-maps` over REST and announces it to the entering session only — mirroring the
  weather replay at `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:355-365`.
- **FR-17.** Replay ordering: obstacles first as a single `FieldObstacleOnOffList` (or per-obstacle
  on GMS<61 per FR-15), then one `SetObjectState` per tracked environment object, in insertion
  order.
- **FR-18.** If the state fetch fails, the enter must still succeed; the failure is logged at
  `Error` and no environment packets are sent. Enter is never blocked on this call.
- **FR-19.** When the field has no tracked state, no environment packets are sent on enter.

### 4.6 Saga actions

- **FR-20.** Two new actions in `libs/atlas-saga/model.go` alongside `FieldEffectWeather`
  (`model.go:286`): `MoveEnvironment Action = "move_environment"` and
  `ResetEnvironment Action = "reset_environment"`.
- **FR-21.** Payloads in `libs/atlas-saga/payloads.go`:
  - `MoveEnvironmentPayload{ WorldId, ChannelId, MapId, Instance, Kind, Name, State }`
  - `ResetEnvironmentPayload{ WorldId, ChannelId, MapId, Instance }`
  Field types follow `FieldEffectWeatherPayload` (`payloads.go:1300+`).
- **FR-22.** Both actions are registered in `libs/atlas-saga/unmarshal.go`, re-exported in
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`, added to the
  event-acceptance set (`saga/event_acceptance.go:324` neighbourhood), and given handlers in
  `saga/handler.go` following `handleFieldEffectWeather` (`handler.go:3715-3744`).
- **FR-23.** The handlers delegate to two new methods on the `map_command` processor
  (`saga-orchestrator/map_command/processor.go:15-18`): `MoveEnvironment(...)` and
  `ResetEnvironment(...)`.
- **FR-24.** Both actions are fire-and-forget with respect to saga completion: the step completes
  when the command is produced. There is no compensating action — reversing an environment move is
  the script author's responsibility via a second `move_environment`.

### 4.7 Script operations

- **FR-25.** `executeMoveEnvironment` (`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:278-292`)
  builds and creates a saga step with the `MoveEnvironment` action, using the same builder shape as
  `executeSpawnMonster` (`.../executor.go:225-242`). The TODO comment and the bare `return nil` are
  removed.
- **FR-26.** `move_environment` parameters: required `name` (string); required `value` (parsed as
  `uint32`; a non-numeric value is an error, not a silent zero); optional `kind` defaulting to
  `"ENVIRONMENT"`. A missing `name` or `value` returns an error in the style of
  `executeDropMessage`'s missing-parameter check (`.../executor.go:311-314`).
- **FR-27.** A new `reset_environment` operation is added to the reactor script dispatcher
  (`.../executor.go:62` neighbourhood), the reactor script JSON schema
  (`services/atlas-reactor-actions/docs/reactor_script_schema.json`), and `docs/domain.md`. It
  takes no parameters.
- **FR-28.** `atlas-map-actions` gains the same two operations in its script executor
  (`services/atlas-map-actions/atlas.com/map-actions/script/executor.go:37-47`), with identical
  parameter semantics, plus its own schema/doc updates.
- **FR-29.** `services/atlas-reactor-actions/docs/domain.md:108` must no longer say
  `move_environment` is "not yet implemented".

### 4.8 Client-version behaviour

- **FR-30.** Environment moves are emitted for every supported tenant version. The four codecs are
  verified across `gms_v48`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, and `jms_v185` (see the
  `packet-audit:verify` markers in `libs/atlas-packet/field/clientbound/*_test.go`), except
  `FieldObstacleOnOffList`, which has no `gms_v48` marker and takes the legacy single-obstacle
  shape below GMS 61 (FR-15).

## 5. API Surface

New JSON:API resource on `atlas-maps`, registered next to the existing weather and jukebox
resources (`services/atlas-maps/atlas.com/maps/map/weather/resource.go`,
`.../map/jukebox/resource.go`). Route shape follows whatever those resources already use for
field scoping; the design phase pins the exact path from `map/resource.go`.

### `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/environment`

Query parameter `instance` (UUID, optional) selects the field instance.

Response (`200`):

```json
{
  "data": [
    {
      "type": "environment-objects",
      "id": "ENVIRONMENT:gate01",
      "attributes": { "kind": "ENVIRONMENT", "name": "gate01", "state": 1 }
    },
    {
      "type": "environment-objects",
      "id": "OBSTACLE:obs3",
      "attributes": { "kind": "OBSTACLE", "name": "obs3", "state": 0 }
    }
  ]
}
```

An untracked field returns `200` with an empty `data` array, never `404`.

### `POST /worlds/{worldId}/channels/{channelId}/maps/{mapId}/environment`

Request:

```json
{
  "data": {
    "type": "environment-objects",
    "attributes": { "kind": "ENVIRONMENT", "name": "gate01", "state": 1 }
  }
}
```

Applies the state and emits the status event, exactly as the Kafka command path does — the REST
handler and the command handler call the same processor method.

Responses: `202 Accepted` on success; `400` for a missing/blank `name` or an unrecognised `kind`;
`404` only when the world/channel/map is not routable.

### `DELETE /worlds/{worldId}/channels/{channelId}/maps/{mapId}/environment`

Resets the field: clears tracked state and emits `ENVIRONMENT_RESET`. `204 No Content`. Resetting
an untracked field is a success (`204`), not a `404`.

All three endpoints are tenant-scoped by the standard tenant header middleware; a request for a
tenant with no state for that field sees an empty result and cannot observe another tenant's state.

## 6. Data Model

No database entities and no migrations. State is an in-memory singleton registry, matching
`services/atlas-maps/atlas.com/maps/map/weather/registry.go`:

```go
type FieldKey struct {
    Tenant tenant.Model
    Field  field.Model
}

type ObjectKind string // "ENVIRONMENT" | "OBSTACLE"

type ObjectEntry struct {
    Kind  ObjectKind
    Name  string
    State uint32
}

type Registry struct {
    mutex   sync.RWMutex
    entries map[FieldKey][]ObjectEntry // insertion-ordered
}
```

Tenant isolation is structural: `Tenant` is part of the key, so two tenants running the same map
never share entries. Insertion order is preserved because replay order (FR-17) is observable to the
client.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-saga` | `MoveEnvironment` / `ResetEnvironment` actions in `model.go`; payloads in `payloads.go`; cases in `unmarshal.go` |
| `atlas-saga-orchestrator` | Re-export in `saga/model.go`; accept in `saga/event_acceptance.go`; `handleMoveEnvironment` / `handleResetEnvironment` in `saga/handler.go`; two new `map_command` processor methods + command providers |
| `atlas-maps` | New `map/environment` package (registry, processor, producer, resource, rest); two new `COMMAND_TOPIC_MAP` types + consumer arms; new `EVENT_TOPIC_MAP_STATUS` types; route registration in `main.go` |
| `atlas-channel` | New map-consumer arms emitting the four writers; enter-replay fetch via a new `environment` REST client package (shaped like `channel/weather/`); GMS<61 list fallback |
| `atlas-reactor-actions` | Implement `executeMoveEnvironment`; add `reset_environment`; update `docs/reactor_script_schema.json` and `docs/domain.md` |
| `atlas-map-actions` | Add `move_environment` and `reset_environment` ops + schema/doc updates |
| `libs/atlas-packet` | **No change.** All four codecs exist and are verified. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every registry read/write is keyed by `tenant.Model` from context via
  `tenant.MustFromContext`. Kafka commands and events carry the standard tenant headers. No code
  path may read state without a tenant in context.
- **Concurrency.** The registry uses `sync.RWMutex` with reads under `RLock`. A concurrent
  `Set` and enter-replay `Get` must not tear a partial list; `Get` returns a copied slice, never the
  backing array.
- **Enter-path latency.** The replay fetch (FR-16) must not block map entry: it runs on the same
  code path as the existing weather replay and, on failure or timeout, logs and continues (FR-18).
- **Ordering.** Commands and events are keyed by map id, so all state changes for one map land on
  one partition and are applied in order.
- **Observability.** Each state change logs at `Debug` with tenant, field (world/channel/map/
  instance), kind, name, and state. Failure to broadcast logs at `Error` with the same field
  identifiers, matching `consumer.go:1002-1004`.
- **Memory.** The registry holds only fields with at least one moved object; a field with no
  tracked state occupies no entry. Reset and instance teardown delete the key entirely.
- **Testing.** Unit tests for the registry (including tenant isolation and copy-on-read), the
  processor, and the command/event round trip; a channel-consumer test asserting the correct writer
  is chosen per `kind` and that GMS<61 takes the per-obstacle fallback; an executor test asserting
  `move_environment` produces a saga step with the right payload (replacing today's "returns nil"
  behaviour).
- **Security.** The new REST endpoints are internal-service endpoints on the same trust boundary as
  the existing weather/jukebox resources; no new external exposure.

## 9. Open Questions

1. **Default state on reset for `ENVIRONMENT` objects.** `FieldObstacleAllReset` is a payload-free
   packet the client uses to restore obstacle defaults, but there is no equivalent reset packet for
   `SetObjectState` objects. Options: (a) send `SetObjectState(name, 0)` for each tracked
   environment object, assuming `0` is the WZ default; (b) record the pre-move state — which the
   server does not know, since it never read it; (c) restrict `reset_environment` to obstacles and
   document that environment objects must be moved back explicitly. **This must be resolved from
   client evidence in the design phase — do not assume `0` is the default without confirming it
   against the client's `CField` object handling.**
2. **Field-instance teardown hook (FR-7).** Whether `atlas-maps` already has a teardown signal that
   weather/jukebox hook into, or whether one must be added, has not been confirmed.
3. **Bulk vs. per-object status events (FR-12).** Whether to add `ENVIRONMENT_STATE_LIST` depends on
   whether any script needs to toggle many obstacles atomically. Reactor scripts today issue one
   `move_environment` at a time.
4. **`kind` inference.** Whether converted Cosmic scripts can be relied on to specify `kind`, or
   whether the default `ENVIRONMENT` will silently mis-route obstacle toggles. May need a per-map
   name convention, or explicit `kind` in every converted script.
5. **Exact REST route shape.** Pinned in the design phase from `atlas-maps`'s existing
   weather/jukebox route registration in `main.go` / `map/resource.go`.

## 10. Acceptance Criteria

- [ ] `grep -rn "SetObjectStateWriter\|FieldObstacleOnOffWriter\|FieldObstacleOnOffListWriter\|FieldObstacleAllResetWriter" services/atlas-channel --include="*.go"` shows at least one emitting call site per writer outside `main.go` and the writer wrappers.
- [ ] `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` contains no
      `TODO: This needs a new saga action for environment manipulation`, and `executeMoveEnvironment`
      creates a saga rather than returning `nil`.
- [ ] `move_environment` in a reactor script moves the named object for every character in the map.
- [ ] A character entering the map after the move sees the moved state, verified by a test on the
      `CHARACTER_ENTER` replay path.
- [ ] `reset_environment` clears tracked state, emits `FieldObstacleAllReset`, and a subsequent
      enter sends no environment packets.
- [ ] A GMS<61 tenant receives per-obstacle `FieldObstacleOnOff` packets on bulk replay, never a
      multi-entry `FieldObstacleOnOffList`.
- [ ] `MoveEnvironment` and `ResetEnvironment` round-trip through `libs/atlas-saga/unmarshal.go`
      and are present in the orchestrator's event-acceptance set, with handler tests.
- [ ] `GET`/`POST`/`DELETE` on the `atlas-maps` environment resource behave per §5, including the
      empty-state and cross-tenant isolation cases.
- [ ] `move_environment` and `reset_environment` are documented in
      `services/atlas-reactor-actions/docs/domain.md`, `.../docs/reactor_script_schema.json`, and
      the `atlas-map-actions` equivalents; nothing still reads "not yet implemented".
- [ ] `libs/atlas-packet` is unchanged by the branch diff.
- [ ] Flagless `tools/verify.sh` exits 0.
