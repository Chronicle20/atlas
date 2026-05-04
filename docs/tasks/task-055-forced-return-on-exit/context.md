# task-055 — Forced Return on Exit — Implementation Context

> Companion to `plan.md`. Captures the surface area an executor needs to read before touching code, the decisions already made, and the parts of the design that are still open.

## What's being built

Move durable character location ownership from atlas-character to atlas-maps. Introduce one in-process resolver in atlas-maps that decides the destination map on disconnect and channel-change, honoring WZ `forcedReturn`. Retire three duplicate per-feature warp emits (atlas-maps timer, atlas-transports `HandleLogin`, atlas-party-quests disconnect leave).

## Files an executor must read before changing anything

### atlas-maps (gains location ownership)
- `services/atlas-maps/atlas.com/maps/data/map/info/processor.go` — existing tenant-cached map info loader; `Resolve` will consume `info.Processor.GetById`.
- `services/atlas-maps/atlas.com/maps/data/map/info/model.go` — `ForcedReturnMapId()` already exists; no field additions needed.
- `services/atlas-maps/atlas.com/maps/character/processor.go` + `requests.go` + `rest.go` — closest existing GORM+REST pattern (NPC visit-style processor in atlas-maps); copy the structure for `character/location/`.
- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` — current LOGIN/LOGOUT/MAP_CHANGED/CHANNEL_CHANGED handlers. The plan modifies four of these and adds two new consumers (CHANGE_MAP, CHANGE_CHANNEL_REQUEST).
- `services/atlas-maps/atlas.com/maps/map/timer/processor.go` lines 140-170 — `ForceReturnIfTracked` emits `CHANGE_MAP`; this emit is removed.
- `services/atlas-maps/atlas.com/maps/map/timer/model.go` — `Entry.ForcedReturnMapId()` field; can be retired entirely after the emit goes.
- `services/atlas-maps/atlas.com/maps/map/character/registry.go` — singleton in-memory presence cache moving to Redis-backed (D10).
- `services/atlas-maps/atlas.com/maps/migration.go` (or wherever AutoMigrate runs) — register the new entity.

### atlas-character (subtractive)
- `services/atlas-character/atlas.com/character/character/entity.go` lines 41-42 — `MapId` and `Instance` columns dropped.
- `services/atlas-character/atlas.com/character/character/model.go` lines 99-104, 401-410 — `MapId()`/`Instance()` getters and builder setters dropped.
- `services/atlas-character/atlas.com/character/character/rest.go` lines 40-41, 105-106, 143-144 — `MapId`/`Instance` REST fields hydrated via atlas-maps lookup (D11 backward-compat shim).
- `services/atlas-character/atlas.com/character/character/processor.go`:
  - Line 391: `Login` reads `c.MapId()`/`c.Instance()` to populate LOGIN event → must query atlas-maps instead.
  - Line 405: `Logout` same pattern → query atlas-maps.
  - Lines 410-424: `ChangeChannel` / `ChangeChannelAndEmit` removed entirely.
  - Lines 426-468: `ChangeMap`, `ChangeMapAndEmit`, `positionAtPortal`, `announceMapChangedWithBuffer`, `announceMapChanged` removed entirely.
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` lines 38, 117-129 — `handleChangeMap` and its registration removed.
- `services/atlas-character/atlas.com/character/kafka/consumer/session/consumer.go` line 85 — drop the `ChangeChannelAndEmit` call from the StateTransition branch (registry/history bookkeeping stays).

### atlas-channel (modified at three sites)
- `services/atlas-channel/atlas.com/channel/socket/handler/channel_change.go` — emits `CHANGE_CHANNEL_REQUEST` on the new topic in addition to (or replacing) the existing `as.UpdateState(2, ...)` chain. Account state machine call remains so atlas-account session state still progresses (IP/port handoff). The new emit is what triggers atlas-maps to publish CHANNEL_CHANGED.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go` line 169 — session bootstrap reads `c.MapId()`. Pivot to atlas-maps `GET /characters/{id}/location`. On atlas-maps unreachable: fail closed (D12).
- `services/atlas-channel/atlas.com/channel/respawn/processor.go` — `Respawn` already takes `currentMapId` from the caller (`map_change.go:54` passes `s.MapId()` from the live session). Live-session source is unchanged — no pivot needed at the caller site. Re-confirm during Phase 6.

### atlas-login
- `services/atlas-login/atlas.com/login/socket/writer/character_list.go:41` — replace `c.MapId()` with atlas-maps location lookup per character. Keep `c.SpawnPoint()` (atlas-character still owns it).

### atlas-transports (subtractive)
- `services/atlas-transports/atlas.com/transports/instance/processor.go` lines 283-299 — remove the entire `HandleLogin` transit-map detection branch. `HandleLogout` is unchanged.

### atlas-party-quests (subtractive)
- `services/atlas-party-quests/atlas.com/party-quests/instance/processor.go` lines 917-963 — guard the `mb.Put(character2.EnvCommandTopic, warpCharacterProvider(...))` at line 953 with `if reason != "disconnect" { ... }`.

### libs/atlas-constants
- `libs/atlas-constants/map/constants.go:2267` — `EmptyMapId = Id(999999999)` already exists.
- `libs/atlas-constants/map/model.go` — add `IsSentinel()` method on `Id`.

## Design decisions already made

Read `design.md` §2 (D1–D12). Headlines:

- **D1**: Resolver lives in atlas-maps; exposed via `GET /characters/{id}/location` for cross-service callers.
- **D7**: Rule 1 (HP ≤ 0 → returnMap) is dropped. Resolver does not take HP; atlas-channel respawn handles dead-state. *Acceptance criteria PRD §10 "disconnect at HP=0" cases are documented parity deltas, not tasks.*
- **D8**: atlas-character drops `map_id` and `instance` columns. Location ownership transfers to atlas-maps' new `character_locations` table.
- **D9**: Channel-change goes via new `COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST`. atlas-maps emits `CHANNEL_CHANGED`.
- **D10**: atlas-maps presence registries move to Redis.
- **D11**: LOGIN/LOGOUT/character-REST `MapId`/`Instance` populated by an in-flight atlas-maps lookup as a backward-compat shim (TODO §10.1).
- **D12**: atlas-channel session bootstrap fails closed on atlas-maps unreachable.

## Things deferred to follow-up tasks (not blockers)

- Strip `MapId`/`Instance` from LOGIN/LOGOUT event payloads (§10.1).
- Remove `MapId`/`Instance` from atlas-character `RestModel` (§10.1).
- Investigate `drop` command body's `MapId`/`Instance` source (§10.2).
- Retire PQ JSON `def.Exit()` for non-disconnect leaves (§10.3).
- Movement command source-of-truth review (§10.4).
- Future channel-change validators (§10.5).
- Retire `route.StartMapId()` if unused (§10.6).

## Things still open in the plan

- **Topic ownership for `COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST`**: by convention, env var lives where the producer lives (atlas-channel), but the message-shape struct typically lives alongside other character commands. Plan introduces it under `services/atlas-channel/atlas.com/channel/kafka/message/character/` with the env var; atlas-maps imports the type. Alternate: live in `services/atlas-character/atlas.com/character/kafka/message/character/` for symmetry with other character commands. Executor: pick one and stay consistent. Plan preserves the atlas-channel placement.
- **Backfill execution**: 2 rows per user confirmation. Plan has a SQL snippet; an operator runs it manually between the atlas-maps deploy and the atlas-character column-drop deploy.
- **Single deploy vs. staged**: design §5 lists 6 steps but says they can be one deploy if migrations land cleanly. Plan groups them into rollout-ordered phases (Phase 1 atlas-maps additive, then everything else) so subagents can verify each layer in isolation.

## Conventions to follow

Project conventions documented in `CLAUDE.md`:

- **Immutable models**: private fields + getters + Builder pattern. Apply to the new `location.Model`.
- **Processors**: Interface + Impl. `NewProcessor(l, ctx, db)`. Pure logic in `Method(mb)`; side-effecting in `MethodAndEmit()`.
- **Kafka**: `message.Buffer` for batching; curried `InitConsumers(l)(cmf)(groupId)`.
- **REST**: JSON:API via `api2go/jsonapi`. `RestModel.GetName()` returns the resource type. Tenant-scoped via existing handler middleware.
- **Multi-tenancy**: `tenant.MustFromContext(ctx)`. Persistence keys must include `tenant_id`.
- **Constants**: reuse `libs/atlas-constants/`. Per CLAUDE.md DOM-21: do not redefine map ids, world ids, channel ids, etc.
- **Test-first**: prefer TDD. The new resolver and location processor are pure-logic units that should land with table-driven tests before code.

## Verification gates per phase

Each phase ends with a verification gate before moving on. Minimum:

1. Affected services build cleanly: `go build ./...` per service.
2. Affected services pass tests: `go test ./...` per service.
3. For Kafka changes, `docker compose build` + smoke run if the change altered consumer wiring.

Heavyweight verification (live KPQ disconnect → relog at lobby) belongs to the integration test in Phase 11.
