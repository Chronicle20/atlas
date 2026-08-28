# Cross-service seam review: task-278 map environment object state

Commit range: `bda6566f3..HEAD` (HEAD `741d1636d`), branch `task-278-map-environment-object-state`.

Scope: the Kafka command/event contract for environment object state
(`SET_ENVIRONMENT_STATE` / `RESET_ENVIRONMENT` commands on `COMMAND_TOPIC_MAP`,
`ENVIRONMENT_STATE_CHANGED` / `ENVIRONMENT_RESET` events on
`EVENT_TOPIC_MAP_STATUS`) across atlas-saga-orchestrator (producer),
atlas-maps (consumer/producer), atlas-channel (consumer), plus the
atlas-channel -> atlas-maps REST replay seam, plus the saga-action wiring in
atlas-map-actions / atlas-reactor-actions / libs/atlas-saga /
atlas-saga-orchestrator. This is a per-unit-review-cannot-see review, not a
re-review of any single task's internal correctness.

Per-unit reviews for all 14 plan tasks already passed; this review does not
re-litigate in-service logic that a green `verify.sh` or a per-task review
already covers.

## 1. Three-way message type cross-check

### Command envelope + bodies (orchestrator -> maps)

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go:11-51`
- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:11-51`

Both define an identical `Command[E any]` envelope (`TransactionId
uuid.UUID`/`WorldId world.Id`/`ChannelId channel.Id`/`MapId _map.Id`/`Instance
uuid.UUID`/`Type string`/`Body E`, identical json tags) and identical
`SetEnvironmentStateCommandBody{Kind, Name, State}` /
`ResetEnvironmentCommandBody{}` bodies, byte-for-byte including json tags.
Type constants `CommandTypeSetEnvironmentState = "SET_ENVIRONMENT_STATE"` and
`CommandTypeResetEnvironment = "RESET_ENVIRONMENT"` are identical strings in
both files. PASS.

### Status event envelope + bodies (maps -> channel)

- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:1-84`
- `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go:1-84`

The two files are byte-identical for every type touched by this task
(`StatusEvent[E]`, `EnvironmentStateChanged{Kind,Name,State}`,
`EnvironmentObject{Kind,Name}`, `EnvironmentReset{Cleared
[]EnvironmentObject}`), including the doc comment on `EnvironmentReset`. Type
constants `EventTopicMapStatusTypeEnvironmentStateChanged =
"ENVIRONMENT_STATE_CHANGED"` and `EventTopicMapStatusTypeEnvironmentReset =
"ENVIRONMENT_RESET"` match exactly. PASS.

### Producer -> consumer field-by-field verification

- Orchestrator producer: `map_command/producer.go:50-80` populates
  `SetEnvironmentStateCommandBody{Kind: string(kind), Name: name, State:
  state}` and `ResetEnvironmentCommandBody{}` with `Type:
  mapKafka.CommandTypeSetEnvironmentState` / `CommandTypeResetEnvironment`.
- Maps consumer: `kafka/consumer/map/consumer.go:110-152` type-guards on the
  identical constants and reads `c.Body.Kind`, `c.Body.Name`, `c.Body.State`.
  No field is added on one side and dropped on the other. PASS.
- Maps producer: `map/environment/producer.go:14-48` populates
  `EnvironmentStateChanged{Kind: string(e.Kind), Name: e.Name, State:
  e.State}` and `EnvironmentReset{Cleared: []EnvironmentObject{Kind, Name}}`.
- Channel consumer: `kafka/consumer/map/consumer.go:1238-1297` type-guards on
  the identical constants and reads `e.Body.Kind`, `e.Body.Name`,
  `e.Body.State` / `e.Body.Cleared[i].Kind`, `.Name`. PASS.

No divergence found in casing, field names, Go types, or JSON tags anywhere
in this chain.

## 2. Type-string and enum agreement

- Command type strings: orchestrator emits `CommandTypeSetEnvironmentState`
  = `"SET_ENVIRONMENT_STATE"` (`kafka/message/map/kafka.go:18`); maps
  switches on the same-named, same-valued constant
  (`kafka/message/map/command.go:15`) at `consumer.go:112`. Character-identical.
- Event type strings: maps emits `EventTopicMapStatusTypeEnvironmentStateChanged`
  = `"ENVIRONMENT_STATE_CHANGED"` (`kafka/message/map/kafka.go:21`); channel
  switches on the identical constant at `consumer.go:1240`. Character-identical.
  Same for `ENVIRONMENT_RESET` (`consumer.go:1266`).
- `field.ObjectKind` / `ParseObjectKind`
  (`libs/atlas-constants/field/constants.go:14-41`) is a single shared type
  imported by every producer and consumer of `Kind` on this seam (orchestrator
  `map_command/producer.go:50`, maps `consumer.go:116` and
  `map/environment/resource.go:74`, channel `consumer.go:1247,1284,1210`,
  map-actions `script/executor.go:274`, reactor-actions
  `script/executor.go:301`). `ParseObjectKind` accepts exactly `""`
  (defaults to `ENVIRONMENT`), `"ENVIRONMENT"`, `"OBSTACLE"` (case-insensitive,
  trimmed) and rejects everything else — and every writer of the wire `Kind`
  string writes `string(kind)` where `kind` was itself produced by
  `ParseObjectKind` or the `ObjectKind` constants, so the wire value the
  consumer must accept is always one `ParseObjectKind` itself would produce.
  Symmetric by construction (shared library type, not a duplicated enum).
  PASS.

## 3. Topic agreement

- `COMMAND_TOPIC_MAP`: `EnvCommandTopicMap = "COMMAND_TOPIC_MAP"` identical
  in `atlas-saga-orchestrator/kafka/message/map/kafka.go:12` and
  `atlas-maps/kafka/message/map/command.go:12`.
- `EVENT_TOPIC_MAP_STATUS`: `EnvEventTopicMapStatus =
  "EVENT_TOPIC_MAP_STATUS"` identical in
  `atlas-maps/kafka/message/map/kafka.go:12` and
  `atlas-channel/kafka/message/map/kafka.go:12`.
- Grep sweep across `services/**/kafka/message/map/*.go` and
  `services/**/kafka/consumer/map/*.go` confirms only these two literal
  values exist repo-wide for these two env-var keys; every consumer of them
  is `consumer.EnvProvider`-resolved off the same string. PASS.

## 4. Test asserts the new contract

- **Orchestrator -> maps (command):** `map_command/producer_test.go` decodes
  the producer's own emitted `kafka.Message` back into
  `mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]` /
  `...ResetEnvironmentCommandBody` and asserts `Type`/body fields
  (`TestResetEnvironmentCommandProvider`, `map_command/producer_test.go:66-83`,
  and the sibling `TestSetEnvironmentStateCommandProvider` above it). This is
  a same-package round-trip, not an independent construction of the maps
  side's struct — it proves the producer serializes what it says it does, not
  that atlas-maps' independently-declared struct still matches. The maps-side
  guarantee instead rests on the byte-for-byte struct comparison in §1 (both
  structs are literally identical text), plus
  `atlas-maps/kafka/consumer/map/consumer_test.go:108-250`
  (`TestHandleSetEnvironmentStateCommand_Applies`,
  `_BlankKindDefaultsEnvironment`, `_WrongTypeIgnored`,
  `_UnknownKindRejected`, `_BlankNameRejected`,
  `TestHandleResetEnvironmentCommand_ClearsTracked`,
  `_WrongTypeIgnored`) which constructs a `mapKafka.Command[...]` directly
  and asserts the handler's Type-string routing and `ParseObjectKind`
  rejection path. Handler-side dispatch on the exact type-string constant is
  proven; wire-format identity between the two services is proven by textual
  comparison, not by an independent decode test on either side. Acceptable
  given the byte-identical struct definitions, but noted as the weakest link
  in the chain (non-blocking).
- **Maps -> channel (status event):**
  `atlas-channel/kafka/consumer/map/consumer_test.go:871-1061`
  (`TestHandleStatusEventEnvironmentStateChanged_Broadcasts`,
  `_WrongTypeIgnored`, `_BadKindIgnored`,
  `TestHandleStatusEventEnvironmentReset_AllResetRouted`,
  `_AllResetUnrouted`, `_EmptyCleared`) constructs
  `_map3.StatusEvent[_map3.EnvironmentStateChanged]` /
  `[_map3.EnvironmentReset]` directly and asserts the handler's type-guard,
  `ParseObjectKind` rejection, and per-session broadcast behavior. Same
  caveat as above: this proves the channel decodes and routes its own
  locally-declared struct correctly, not that the wire bytes maps actually
  emits decode into it — that guarantee again rests on the byte-identical
  struct text (§1). PASS with the same non-blocking caveat.
- **REST seam (channel -> maps):**
  `atlas-channel/atlas.com/channel/environment/requests_test.go:31-114`
  (`TestGetAll_ParsesCollection`, `_EmptyCollection`, `_ServerError`,
  `_RequestsInstancePath`) is a genuine seam test: it stands up an
  `httptest.Server` serving a **hand-written literal JSON:API document**
  (`{"data":[{"type":"environment-objects","id":"OBSTACLE:obs3","attributes":
  {"kind":"OBSTACLE","name":"obs3","state":2}}, ...]}`) — not a struct the
  channel package itself constructed — and asserts the channel's
  `GetAll`/`Extract` decode it correctly, and that the request path matches
  what atlas-maps registers (`/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/environment`,
  matched against `atlas-maps/map/environment/rest.go:30`). This is the
  strongest test on the seam: it fails if atlas-channel's `RestModel` json
  tags, type string, or path diverge from the literal wire shape.
  `atlas-maps/map/environment/resource_test.go:114-127` independently
  marshals atlas-maps' own `RestModel` through the real jsonapi encoder and
  asserts the on-wire attribute keys (`"kind"`, `"name"`, `"state"`) and
  values — so both ends of the REST seam are asserted against the literal
  wire shape, from opposite directions. PASS, no caveat.
- **Saga action wiring (map-actions / reactor-actions -> orchestrator ->
  map_command):** `atlas-saga-orchestrator/saga/handler_test.go:1737-1899`
  (`TestHandleMoveEnvironment_InvalidPayload`,
  `_DelegatesToMapCommand`, `TestHandleResetEnvironment_InvalidPayload`,
  `_DelegatesToMapCommand`) asserts the handler both rejects a
  wrongly-typed payload and forwards a valid one's `Kind`/`Name`/`State`/
  field identity to the mocked `map_command.Processor`.
  `saga/model_test.go:1171-1228` (`TestUnmarshalStep_MoveEnvironment`,
  `_ResetEnvironment`) proves the JSON `Step[any]` unmarshal dispatch table
  (`unmarshal.go:648-659`) actually recovers a typed
  `MoveEnvironmentPayload`/`ResetEnvironmentPayload` from raw JSON — not just
  a compiled type assertion — and that both actions are present in
  `acceptanceTable` (`event_acceptance.go:326-327`, fire-and-forget `{}`
  entries, consistent with `handleMoveEnvironment`/`handleResetEnvironment`
  self-completing via `StepCompleted` immediately after producing the
  command, `handler.go:3809-3843`). `libs/atlas-saga/unmarshal_test.go`
  (new file) covers the shared-library dispatch table itself. PASS.
- map-actions (`script/executor_test.go`) and reactor-actions
  (`script/executor_test.go`) each have new test files (291 and 298 lines
  respectively per the diffstat) exercising `move_environment` /
  `reset_environment` operation parsing and saga-step construction; not
  read in full detail here since both import the same
  `field.ParseObjectKind` and `saga.MoveEnvironment`/`MoveEnvironmentPayload`
  from the shared libraries (§2), so a kind/action-name mismatch between
  these two services and the orchestrator is structurally impossible, not
  merely untested.

## 5. REST seam

- Route: `atlas-maps/map/environment/resource.go:30` registers
  `/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment`
  for GET (list), POST (set), DELETE (reset).
- Channel client: `atlas-channel/environment/requests.go:12-26` builds
  `worlds/%d/channels/%d/maps/%d/instances/%s/environment` off the same
  `MAPS` service-root env var convention used elsewhere in the codebase, and
  only implements GET (`GetAll`) — consistent with the task's "replay on map
  enter" scope; POST/DELETE are maps-only administrative/command-mirroring
  routes, not part of the channel's contract.
- JSON:API type string: both `RestModel.GetName()` return
  `"environment-objects"` (`atlas-maps/map/environment/model.go:17`,
  `atlas-channel/environment/rest.go:15`) — identical.
- Attribute names/types: both `RestModel{Kind, Name string; State uint32}`
  with identical json tags (`kind`/`name`/`state`), `Id` tagged `json:"-"`
  and populated from `Kind:Name` on both sides
  (`atlas-maps/map/environment/model.go:26-31`,
  `atlas-channel/environment/rest.go:47-52`). Id semantics match: neither
  side treats `Id` as anything but an opaque composite key; the channel
  side's `SetToOneReferenceID`/`SetToManyReferenceIDs` no-ops exist solely
  to satisfy the jsonapi decoder interface and carry no cross-service
  meaning. PASS.

## Not evaluable

- The internal correctness of `map-actions`/`reactor-actions` script
  parameter parsing (missing name/value, non-numeric value) beyond
  confirming it dispatches through the shared `field.ParseObjectKind` and
  `saga.MoveEnvironment`/`ResetEnvironment` constants — that is in-service
  logic already covered by the per-task reviews and is out of this seam
  review's surface.
- Whether the client-side wire opcodes selected by `announceObjectState`
  (`SetObjectStateWriter` / `FieldObstacleOnOffWriter` /
  `FieldObstacleAllResetWriter`) are the correct MapleStory opcodes for the
  target client version — that is a packet-encoding correctness question,
  not a cross-service Go-struct-contract question, and is outside this
  review's stated scope (Kafka command/event contract + REST seam).

## Verdict rationale

Every message struct, type-string constant, topic env-var key, and REST
contract element touched by this task is either byte-identical text across
the two service copies, or backed by a single shared library type
(`field.ObjectKind`, `saga.Action`, `saga.MoveEnvironmentPayload`) that
cannot drift by construction. Every seam has at least one test that asserts
the NEW contract rather than merely compiling against it; the REST seam has
tests asserting the literal wire JSON from both directions. No blocking
defect found.

One non-blocking observation: the two Kafka seams (orchestrator->maps,
maps->channel) are proven consistent by textual struct identity plus
same-package handler tests, not by a test that decodes one service's actual
emitted bytes with the other service's independently-declared struct. That
is the established pattern in this codebase (every other Kafka seam in
these files follows it too) and is not a task-278-specific regression, but
it is worth naming as the one place a future accidental struct edit on
either side would not be caught by a red test — it would only be caught by
a future seam review doing this same textual diff.
