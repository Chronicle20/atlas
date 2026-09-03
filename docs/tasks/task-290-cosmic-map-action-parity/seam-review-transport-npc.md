# Seam review: transport (G1) and field NPC (G2) — shard 1 of 3

Range: `9613e7259..d058fd34a`. Scope: cross-service producer/consumer seams only,
for tasks C7 (play_sound), C9 (change_music/boat_effect), C10 (transportInTransit),
C11/C13 seeds, C12 (warp_to_map), C14/C15 (spawn_npc field NPC registry). No
style/guideline or plan-adherence re-check performed — those gates already ran
clean on this branch.

## Seam 1: `play_sound` (C7)

- Producer (saga action + payload): `libs/atlas-saga/model.go:183`,
  `libs/atlas-saga/payloads.go:502` — `PlaySound`/`PlaySoundPayload{CharacterId,
  WorldId, ChannelId, Path}`. Unmarshal case: `libs/atlas-saga/unmarshal.go:300`.
- Orchestrator → system-message command: `.../system_message/kafka.go:27,102`
  (`CommandPlaySound = "PLAY_SOUND"`, `PlaySoundBody{Path}`); processor method
  `.../system_message/processor.go:118-121`; handler
  `.../saga/handler.go:2907-2924` (synchronous, marks step complete immediately —
  matches the fire-and-forget contract documented at the same lines).
- Consumer (atlas-channel): `.../kafka/consumer/system_message/consumer.go:356-374`
  — `handlePlaySound` type/tenant/channel-scope guards match the `handleFieldEffect`
  pattern, and it announces via `fieldpkt.FieldEffectSoundBody(cmd.Body.Path)` under
  `fieldcb.FieldEffectWriter` — the pre-existing, already-in-production writer named
  in the plan (`field_effect_body.go:51`), so no new wire byte is introduced.
- Field/type agreement: `PlaySoundBody.Path` (json `path`) ↔ executor param `path`
  ↔ `FieldEffectSoundBody(path string)`. Confirmed consistent end to end.
- Tests asserting the NEW contract: `libs/atlas-saga/unmarshal_test.go:2161`
  (`TestUnmarshalPlaySoundStep`); `script/executor_test.go:887`
  (`TestExecutePlaySound`). No atlas-channel-side unit test exists for
  `handlePlaySound` specifically, but this mirrors the pre-existing `ShowInfo`
  baseline (confirmed: `system_message/processor.go`'s `ShowInfo` also has no
  saga-orchestrator-level test — `find .../system_message -name '*_test.go'`
  returns nothing at all). Not a regression introduced by this PR.
- **Verdict: seam correct.**

## Seam 2: `change_music` / `boat_effect` (C9)

- Producer/orchestrator wiring for both actions mirrors `play_sound` exactly
  (checked: `model.go`, `payloads.go`, `kafka.go`, `handler.go`, all present and
  paired).
- `boat_effect`'s consumer maps `Show bool` to a `writer.ContiMoveKey`
  (`kafka/consumer/system_message/consumer.go:483-486`): `Show==true → ContiMoveShow`,
  else `ContiMoveHide`. The payload carries no state/subState wire byte — confirmed
  by reading the payload struct and the handler; this matches the explicit
  `socket/writer/conti_move.go:13-18` rule the plan cites, which is read-only and
  unmodified in this diff.
- Tenant option keys the writer resolves (`SHOW_STATE`/`HIDE_STATE`) are present in
  all 10 seeded templates (`grep -rln "SHOW_STATE\|HIDE_STATE"
  services/atlas-configurations/seed-data/templates/` returns all 10 gms/jms
  template files; example: `template_gms_83_1.json:3292,3294`).
- Tests asserting the NEW contract: `libs/atlas-saga/unmarshal_test.go:2201,2241`;
  `script/executor_test.go:948,1009,1063`; and — unlike `play_sound` — atlas-channel
  **does** have dedicated consumer tests here:
  `kafka/consumer/system_message/consumer_test.go:164-296`
  (`TestHandleChangeMusic_Unconfigured_SkipsWrite`,
  `TestHandleChangeMusic_Configured_WritesExactlyAsBefore`,
  `TestHandleBoatEffect_Unconfigured_SkipsWrite`,
  `TestHandleBoatEffect_Configured_WritesExactlyAsBefore`) which pin the
  version-gated write guard.
- **Verdict: seam correct, with the strongest consumer-side test coverage of any
  seam in this shard.**

## Seam 3: `transportInTransit` condition (C10) and its seed encoding

- Shared constant: `libs/atlas-saga/validation.go:40` —
  `TransportInTransitCondition = "transportInTransit"`.
- Aggregator alias + evaluation arm: `validation/model.go:52` (`ConditionType`
  alias) and `validation/model.go:557-566` (`EvaluateWithContext` arm) — `state ==
  "in_transit" → actualValue = 1`, else `0`. Verified `Evaluate` (the
  character-scoped sibling function, `model.go:111-405`) has no
  `TransportAvailable`/`TransportInTransit` arm either — consistent with the
  existing `transportAvailable` condition, so there is no missing parallel switch.
- Accepted-type / referenceId validation touchpoints all present and consistent:
  `builder.go:216` (accepted-type list), `builder.go:360` (`FromInput`),
  `builder.go:451` (`Validate()`), `rest.go:275` (REST-input arm).
- Seed encoding cross-checked against the aggregator arm directly:
  `deploy/seed/gms/83_1/map-actions/onUserEnter/map-200090000.json` —
  `{"type":"transportInTransit","referenceId":"200090000","operator":"=","value":"0"}`
  — `referenceId` is read by `ctx.GetTransportState(_map.Id(c.referenceId))`
  (a map id, matching the plan's C11-step-1 finding), `operator`/`value` follow
  the standard numeric-comparison path already used by `transportAvailable`.
  Encoding matches the aggregator's parsing exactly.
- Tests: `validation/model_test.go:2414` (`TestTransportInTransitCondition`, all
  five route states + both `value=1`/`value=0` polarities per the plan's table),
  `:2500` (`TestTransportInTransitRequiresReferenceId`), `:2516`
  (`TestTransportInTransitAcceptedBySetType` — the exact regression the plan
  flagged for a `builder.go:210`-style omission).
- Seed replication verified mechanically: 22 files for the two dock-arrival seeds
  (C11), 132 files for the twelve transport catch-up seeds (C13) — both match the
  plan's expected counts exactly. `./tools/gen-map-action-schema.sh --check` and
  `catalog-lint` both exit 0 against `deploy/seed`.
- The `101000301`/`200000112` PRD correction called out in the plan is present
  verbatim in the seed content (reciprocal pair, not the PRD's `200090010`).
- **Verdict: seam correct.**

## Seam 4: `warp_to_map` (C12)

- Producer: `libs/atlas-saga/model.go:125`, `payloads.go:53`
  (`WarpToMapPayload{CharacterId, WorldId, ChannelId, MapId}`), unmarshal case
  `unmarshal.go:72`.
- Executor arm: `script/executor.go` (`case "warp_to_map"` → `executeWarpToMap`,
  confirmed via `executor_test.go:1097` `TestExecuteWarpToMap`) — asserts the
  payload's `MapId` is the destination, not `f.MapId()`, per the plan's explicit
  anti-regression instruction.
- Orchestrator handler: `saga/handler.go:966-984` (`handleWarpToMap`) — unpacks
  `WarpToMapPayload` and calls `h.charP.WarpToMapAndEmit(...)`.
- Consumer-side resolution (the actual cross-service-relevant logic, since this is
  where Cosmic's `getRandomPlayerSpawnpoint()` semantics must be reproduced):
  `character/processor.go:108-124` — `WarpToMapAndEmit` rebuilds a `field.Model`
  from `worldId/channelId/mapId` and delegates to the existing `WarpRandom`, which
  already resolves a random spawn point via `portal.Processor.RandomSpawnPointIdProvider`
  (`data/portal/processor.go:46-58`, filtered by `SpawnPoint`/`NoTarget`) — this is
  pre-existing, unmodified code, so the "random player spawn point, not a portal"
  contract the plan requires is satisfied by composition rather than new logic.
- `event_acceptance.go:336` correctly gates `WarpToMap` on
  `EventKindCharacterMapChanged` (asynchronous completion via the map-changed
  event, matching `WarpToRandomPortal`'s semantics — not the synchronous
  fire-and-forget shape used by `PlaySound`).
- Test gap (non-blocking): unlike `WarpToPortal`/`WarpToRandomPortal`, which both
  have a dedicated case in `saga/handler_test.go` (lines 218, 313, 339) exercising
  `handleWarpToRandomPortal` against a mock `character.Processor`, there is **no**
  `saga/handler_test.go` case for `handleWarpToMap`, and no `character/processor_test.go`
  exists at all (only `character/mock/processor_test.go` and `producer_test.go`).
  The only test that would catch a `payload.MapId` vs `f.MapId()` mixup at this
  layer is the executor-side `TestExecuteWarpToMap`, which only proves the payload
  is built correctly, not that the orchestrator's handler and
  `WarpToMapAndEmit` consume it correctly. Given the sibling actions in the same
  family (`WarpToPortal`, `WarpToRandomPortal`) do have this coverage, this is a
  gap specific to the new action, not an established repo baseline.
- **Verdict: seam correct by inspection; test-honesty gap flagged non-blocking.**

## Seam 5: `spawn_npc` field NPC registry (C14/C15)

Traced end to end: map-actions executor → saga action → saga-orchestrator
`npc_spawn` client → atlas-maps `map/npc` REST → atlas-maps in-memory registry
→ atlas-maps Kafka NPC-status event → atlas-channel consumer → `SPAWN_NPC` packet,
plus the "replay to a newly-entering session" half of the seam.

- Executor: `script/executor.go:350-411` (`executeSpawnNpc`) — required `npcId`,
  `x`, `y`; optional `spawnIfAbsent`. Payload:
  `saga.SpawnNpcPayload{CharacterId, WorldId, ChannelId, MapId, Instance, NpcId, X,
  Y, SpawnIfAbsent}` — matches `payloads.go` field-for-field. Verified via
  `TestExecuteSpawnNpc`-shaped tests in `executor_test.go`.
- Orchestrator handler `saga/handler.go:1969-2001` (`handleSpawnNpc`) resolves a
  foothold via `h.footholdP.GetFootholdBelow` (falls back to `fh=0` on lookup
  failure, logged as a warning — matches `handleSpawnMonster`'s pattern) and
  rebuilds `field.Model` from `payload.Instance`, then calls
  `h.npcSpawnP.SpawnNpc(f, req)`.
- Orchestrator → atlas-maps REST client: `npc_spawn/requests.go:11-22` —
  `worlds/%d/channels/%d/maps/%d/instances/%s/npcs` — **matches** atlas-maps'
  route exactly: `map/npc/resource.go:29-30`
  (`/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/npcs`,
  POST → `handleCreateNpcInMap`). `SpawnInputRestModel{NpcId, X, Y, Fh,
  SpawnIfAbsent}` field names/JSON tags match atlas-maps' `RestModel` exactly
  (`map/npc/rest.go`, confirmed field-for-field).
- atlas-maps registry: `map/npc/registry.go` — keyed by `FieldKey{Tenant,
  Field}` (tenant-scoped, confirmed `tenant.MustFromContext` is read into
  `ProcessorImpl.t` and used in every `FieldKey` construction in
  `processor.go:48,58`), instance-scoped via `field.Model`'s own `Instance()`
  field (satisfies the plan's
  `TestCreateNpcSpawnIfAbsentIsFieldScoped` requirement). ID allocation
  (`Registry.NextId()`) is a single global monotonic counter shared across all
  tenants/fields — not tenant-scoped, but this is fine: the ids only need to be
  unique within the process, and are never persisted or compared cross-tenant.
- `SpawnIfAbsent` idempotency: `processor.go:34-47` — pre-check against
  `GetInField`, returns a zero `Model` (`UniqueId()==0`) and nil error on
  suppression; REST handler maps that to `204 No Content`
  (`resource.go:70-76`), correctly distinguishing "suppressed" from "created."
- atlas-maps → atlas-channel event contract: producer
  `map/npc/producer.go:27-44` emits on topic token `EVENT_TOPIC_NPC_STATUS`
  (`kafka/message/npc/kafka.go:13`) with `Type: "CREATED"`
  (`kafka.go:17`) and body `{npcId, x, y, fh}`. Consumer
  `kafka/message/npc/kafka.go` in atlas-channel independently redeclares the
  same topic token string (`EnvEventTopicStatus = "EVENT_TOPIC_NPC_STATUS"`) and
  the same type string (`EventStatusTypeCreated = "CREATED"`), with an identical
  JSON-tagged body struct. Both sides checked field-by-field: they agree.
- atlas-channel handler `kafka/consumer/map/npc.go:52-70`
  (`handleStatusEventNpcCreated`) broadcasts to every session already in the
  field via `ForSessionsInMap`, building the packet through `ScriptedNpcSpawn`
  which calls the pre-existing `npcpkt.NewNpcSpawn(id, template, x, cy, f, fh,
  rx0, rx1)` (`libs/atlas-packet/npc/clientbound/spawn.go:52`) — argument order
  verified positionally against the call site
  (`ScriptedNpcSpawn(uniqueId, npcId, x, y, 0, uint16(fh), x, x)` →
  `id=uniqueId, template=npcId, x=x, cy=y, f=0, fh=fh, rx0=x, rx1=x`); no new
  packet introduced.
- Second half of the seam (a character entering the field **after** the NPC was
  created, which the plan's own consumer-write-up calls out as "task-BC2" and
  documents as a real gap that was found and closed within this branch):
  `kafka/consumer/map/npc.go:87-102` (`spawnScriptedNpcsForSession`) is called
  from `kafka/consumer/map/consumer.go:269` at field-entry time, and reads via
  atlas-channel's own `map/npc` REST client
  (`map/npc/requests.go:16` — same `worlds/%d/channels/%d/maps/%d/instances/%s/npcs`
  path template, `map/npc/processor.go:39` `InMapModelProvider`). Verified this
  client's test `TestForEachInMap_RequestsInstanceScopedPath`
  (`map/npc/rest_test.go`) actually asserts the requested path is instance-scoped
  and that the NPC already present on the field is returned to a newly-arriving
  session — this is exactly a test that would fail without the fix (it's named,
  and its own comment states, that it exists to catch a real regression).
- Tests overall: atlas-maps side —
  `map/npc/processor_test.go`/`registry_test.go` (idempotency + field-scoping,
  per the plan's four required test names); orchestrator side —
  `npc_spawn/processor_test.go:43,116` (`TestSpawnNpcCarriesSpawnIfAbsent`,
  `TestSpawnNpcPropagatesUpstreamFailure`); atlas-channel side —
  `map/npc/rest_test.go`, `kafka/consumer/map/npc_test.go` (not read in full but
  present). No `saga/handler_test.go` case for `handleSpawnNpc` specifically —
  same non-blocking gap pattern noted for `warp_to_map`, consistent with the
  pre-existing `handleSpawnMonster`/`handleSpawnReactorDrops` baseline (also
  untested at that layer).
- Seed replication verified: 55 files for the five explorer-route NPC seeds
  (11 roots × 5 scripts), matching the plan exactly. `catalog-lint`'s spawn rule
  was correctly extended to cover both `spawn_monster` and `spawn_npc`
  (`tools/catalog-lint/mapactions.go:225-229`), and both `--check` and the linter
  pass clean against `deploy/seed`.
- **Verdict: seam correct, and unusually well-instrumented — the branch caught and
  fixed a real producer/consumer gap (late-entering session never seeing an
  already-placed scripted NPC) that a green build alone would not have surfaced.**

## Not evaluable

- Nothing in this shard's assigned seams was out of reach of the diff plus the
  read-only files the plan named as contract-bearing. `kafka/consumer/map/npc_test.go`
  and `player_npc_test.go` were located but not read line-by-line beyond
  confirming they exist and cover the writer-call-ordering pattern; this does not
  change the verdict since the contract-defining code (producer/consumer field
  agreement, path templates, topic/type strings) was independently verified by
  hand against both sides.

## Findings summary

No blocking findings. Two related non-blocking findings, both the same class
(missing `saga/handler_test.go` coverage for a new handler, where a sibling
action in the same family has that coverage and the new one does not):

1. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:966`
   (`handleWarpToMap`) has no `saga/handler_test.go` case, unlike its siblings
   `handleWarpToPortal`/`handleWarpToRandomPortal` (`handler_test.go:218,313,339`).
   A `payload.MapId`/`f.MapId()` mixup here would only be caught by the
   executor-side test, which cannot see the orchestrator's consumption of the
   payload.
2. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:1969`
   (`handleSpawnNpc`) has no `saga/handler_test.go` case either — lower severity,
   since this one mirrors the already-untested `handleSpawnMonster` baseline
   rather than a tested sibling.

Neither finding blocks: the actual wire/REST contracts were verified by hand
against both producer and consumer, field-for-field, and both are exercised by a
test at either the producer boundary (executor) or the REST-client boundary
(npc_spawn/processor_test.go), just not at the orchestrator-internal handler
layer.
