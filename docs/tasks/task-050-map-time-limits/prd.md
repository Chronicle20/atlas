# Map Time Limits — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03

---

## 1. Overview

Certain maps in MapleStory are time-limited: the player may only stay for a fixed number of seconds before the server warps them to the map's "forced return" destination. The map data already carries both fields (`timeLimit` in seconds, `forcedReturnMapId` as a map id) all the way from atlas-data through the JSON contract into atlas-channel — but atlas-channel discards them in `data/map/Extract` and never starts a countdown. Today, a player can stand on a time-limited map indefinitely.

This feature wires the timer end-to-end. When a character enters a map whose `timeLimit > 0` and `forcedReturnMapId != 999999999`, atlas-channel starts a per-character countdown, surfaces the remaining time to the v83 client via the map-clock packet, and on expiration issues a `CHANGE_MAP` command to warp the character to `forcedReturnMapId`. The same forced-return destination is used when the character disconnects, swaps channels, or is otherwise pulled off the map outside the normal portal/scroll path — preventing logout-camping in event maps and matching classic MS behavior. Death and return-scroll exits keep their existing semantics; this feature adds a parallel concept, not a replacement for either.

The countdown is per-character (not per-map and not per-instance) because two players in the same field will have entered at different real-times. Timer state is runtime-only inside atlas-channel and does not need to survive a channel restart.

## 2. Goals

Primary goals:

- A character entering a map with `timeLimit > 0` sees a client-side countdown matching the configured limit.
- When the countdown reaches zero, the character is warped to `forcedReturnMapId` via the existing `CHANGE_MAP` command flow.
- When a character disconnects (logout, network drop, channel-server crash signal) while on a time-limited map, the character's persisted map is rewritten to `forcedReturnMapId` so they re-enter the world there on next login.
- When a character changes channel while on a time-limited map, they arrive on the destination channel at `forcedReturnMapId`, not at the time-limited map.
- Timer is cancelled cleanly on any normal exit (portal, return scroll, death-respawn warp, change-map command from another service).
- Per-character, per-tenant isolation: one player's timer never affects another's.

Non-goals:

- No new map-data fields. Both `timeLimit` and `forcedReturnMapId` are already produced by atlas-data.
- No change to death respawn. `respawn/processor.go` continues to use `returnMapId`; dying on a time-limited map does **not** route through `forcedReturnMapId`.
- No change to return scrolls. Consumables that name a `targetMapId` keep going wherever the consumable says.
- No persistence of remaining time. Re-entering the same map (after leaving for any reason) restarts the timer at the full `timeLimit`.
- No party-quest timer integration. Party-quest stages have their own state machines and are tracked separately.
- No admin/GM override or "freeze timer" feature.
- No retroactive clean-up. If a channel pod crashes mid-timer, the in-flight character is NOT auto-relocated to forced-return on next channel boot — they keep whatever map they were on (treated as a normal session-resume). Future work, not in this task.

## 3. User Stories

- As a player on a time-limited event map, I want to see a visible countdown so I can plan my exit before I'm yanked away.
- As a player whose timer expired, I want to be warped to the map's forced-return destination so I'm not stuck in a glitched zone.
- As a player who logs out on a time-limited map, I want to log back in on the forced-return map, so the timer can't be defeated by camping.
- As a player who swaps channels while on a time-limited map, I want to arrive at the forced-return map on the new channel, so channel-swap can't be used to bypass the limit either.
- As a player who takes a portal off the time-limited map, I want my timer to silently stop and not chase me — taking the portal was a normal exit.
- As an operator, I want timers to be ephemeral so a channel restart doesn't leave behind stale countdowns or fire phantom warps.

## 4. Functional Requirements

### 4.1 Map data extraction

- atlas-channel `services/atlas-channel/atlas.com/channel/data/map/model.go` adds two fields: `forcedReturnMapId _map.Id` and `timeLimit int32`. Provide `ForcedReturnMapId()` and `TimeLimit()` getters.
- `Extract` in `data/map/rest.go` populates both fields from the existing `RestModel`.
- A map is **time-limited** iff `TimeLimit() > 0` AND `ForcedReturnMapId() != 999999999`. Both conditions must hold; either alone is a no-op (e.g., FM has neither).
- Helper `Model.IsTimeLimited() bool` encodes the predicate.

### 4.2 Timer registry

- New package `services/atlas-channel/atlas.com/channel/map/timer/` (or under an existing `map/` sub-package — final placement per the design phase).
- Registry holds entries keyed by `(tenant.Id, characterId)`. Entry contents:
  - `mapId _map.Id`
  - `instance uuid.UUID`
  - `forcedReturnMapId _map.Id`
  - `expiresAt time.Time`
  - `cancel context.CancelFunc` (or equivalent stop handle)
- Singleton via `sync.Once`, `sync.RWMutex` per Atlas registry conventions (see project memory).
- One outstanding timer per character. Starting a new timer for an already-tracked character cancels the old one (transparent map-to-map move within the same time-limited zone family).
- Registry is initialized at channel-server startup and stopped on `ctx.Done()`. All in-flight timers stop when the parent ctx cancels — they do **not** fire warps during shutdown.

### 4.3 Timer lifecycle hooks

- **Start** — In the `MAP_CHANGED` status-event consumer (`kafka/consumer/character/consumer.go::handleStatusEventMapChanged`), after the `SetField` packet is sent, look up the destination map's metadata. If `IsTimeLimited()`, register a timer with duration `time.Duration(timeLimit) * time.Second`.
  - Channel scoping: the consumer already drops events not targeting this channel via `sc.Is(...)`; the timer is only started for characters resident on this channel.
  - The timer is started after the warp completes so the registry only ever holds timers for characters this channel actually owns.
- **Stop on normal exit** — Same handler: when a `MAP_CHANGED` event lands and the *old* map was time-limited, cancel any existing timer for that character before starting a (possibly new) timer for the destination. This covers portal exits, return scrolls, death-respawn warps, and any other `CHANGE_MAP`-initiated move.
- **Stop on session end** — The session/disconnect path in atlas-channel (the existing logout/disconnect handling that emits the session-destroyed message) must call into the timer registry to cancel and trigger forced-return persistence (see 4.4).
- **Channel handoff** — When the character is migrated to another channel (channel-change flow), the disconnecting channel cancels the timer and forces persistence to `forcedReturnMapId` so the destination channel reads the new map on session resume. From the timer's perspective this is identical to a logout.

### 4.4 Forced-return on disconnect / channel change

- When the timer registry observes a session-end event for a character whose entry is still present (i.e., the character disconnected without a normal map exit), it must publish a `character.ChangeMapProvider`-equivalent `CHANGE_MAP` command with:
  - `MapId = forcedReturnMapId`
  - `Instance = uuid.Nil` (forced-return goes to a non-instanced field, matching the instance-transports convention)
  - `PortalId = 0` (default spawn portal)
- Because atlas-character persists `mapId` in its `ChangeMapAndEmit` flow, this rewrites the character's stored map. On next login the character spawns at `forcedReturnMapId`.
- Order of operations on logout:
  1. Channel detects session ended.
  2. Timer registry checks for a tracked entry.
  3. If present, emit `CHANGE_MAP` to forced-return.
  4. Cancel the timer entry.
- The `CHANGE_MAP` command is fire-and-forget from the channel's perspective — the client is already gone, so no `SetField` packet needs to be sent.

### 4.5 Timer expiration (in-session)

- When the timer fires while the character is still connected:
  1. Verify the registry entry is still current (race with a concurrent `MAP_CHANGED` — the entry's `expiresAt` and the entry pointer must match what was scheduled).
  2. Publish a `CHANGE_MAP` command with `MapId = forcedReturnMapId`, `Instance = uuid.Nil`, `PortalId = 0`.
  3. The existing `MAP_CHANGED` event-loop (atlas-character emits status event → atlas-channel `handleStatusEventMapChanged`) handles the actual session field swap and `SetField` packet — no new packet path is needed for the warp itself.
  4. The same handler will then cancel the registry entry as part of the normal exit branch (4.3 stop-on-exit).
- Idempotency: if a `MAP_CHANGED` already moved the character off the map between the timer firing and the warp completing, the second `CHANGE_MAP` is harmless (atlas-character will just persist the second value). Acceptable for v1 — alternatively, the registry's "is entry still current" check (step 1) prevents this.

### 4.6 Client-visible countdown

- The v83 client supports a per-field countdown clock packet (the same one used by event maps and PQ stages). atlas-channel emits this packet on `MAP_CHANGED` for time-limited maps, with the seconds remaining = full `TimeLimit`.
- The packet's exact opcode and payload format are an implementation detail for the design phase — refer to existing v83 writer references in `socket/writer/` for analogous packet shapes.
- Client overlay does not poll; the server sends once on entry and the client counts down locally. If the timer is cancelled (normal exit), the new map's `MAP_CHANGED` resets the overlay; no explicit "stop clock" packet is required.
- On forced-return warp at expiration, the destination map's `MAP_CHANGED` clears the clock the same way as any normal map change.

### 4.7 Tenant scoping

- Registry keys include the tenant id so two tenants' timers cannot collide on identical character ids.
- All timer-emitted Kafka commands carry the originating tenant header, populated from the `MAP_CHANGED` event's context.
- A timer entry stores the originating tenant context (or enough to reconstruct it) so the expiration goroutine can re-establish `tenant.MustFromContext(ctx)` semantics when emitting the `CHANGE_MAP` command.

## 5. API Surface

No HTTP/REST changes. No JSON:API additions. No new public Kafka topics.

Internal Kafka traffic is unchanged from existing flows: this feature publishes to the existing `EnvCommandTopic` for `CHANGE_MAP` (using `character.ChangeMapProvider`) and reads from the existing `EnvEventTopicCharacterStatus` for `MAP_CHANGED`. Both already carry tenant headers.

## 6. Data Model

No persistent storage. The timer registry is in-process state in atlas-channel.

- `Entry` (in-memory):
  - `tenantId tenant.Id`
  - `characterId uint32`
  - `mapId _map.Id`
  - `instance uuid.UUID`
  - `forcedReturnMapId _map.Id`
  - `expiresAt time.Time`
  - `cancel func()`
- `Registry` (in-memory): map keyed by `(tenantId, characterId)`.

No DB migrations. No new shared library types. atlas-character's existing `mapId` column carries the persisted forced-return value.

## 7. Service Impact

| Service | Change | Reason |
| --- | --- | --- |
| atlas-channel | Add `forcedReturnMapId` and `timeLimit` to `data/map/Model` + extract. New per-character timer registry. Hook `MAP_CHANGED` consumer for start/stop. Hook session-end path for forced-return. Emit map-clock packet on time-limited-map entry. | Owns the runtime gameplay loop; only place where per-character session state lives. |
| atlas-character | None expected. The existing `CHANGE_MAP` consumer + `ChangeMapAndEmit` handles the persisted-map rewrite at logout/expiration. | Already does what we need. |
| atlas-data | None. `timeLimit` and `forcedReturnMapId` are already in the JSON contract. | Already done. |
| atlas-ui | None. | Server-only feature, client-side count is rendered by the v83 game client. |
| atlas-saga-orchestrator | None. The forced-return is a single `CHANGE_MAP` command, not a multi-step saga. | Existing respawn saga is for death; this path is simpler. |

## 8. Non-Functional Requirements

### 8.1 Performance

- Registry lookups are O(1). Map-change happens at human-input frequency (single-digit Hz per player), so a `sync.RWMutex` is sufficient — no need for sharded locks.
- Each timer is a single `time.AfterFunc` (or `context.WithTimeout` + select) — sub-microsecond scheduling overhead. Worst case, a busy channel with thousands of concurrent players in time-limited zones runs thousands of pending timers; well within Go runtime capacity.
- No Kafka traffic until the timer fires or the session ends, so steady-state cost is zero per-tick.

### 8.2 Correctness / race conditions

- Race A: timer fires concurrently with a `MAP_CHANGED` for the same character. Mitigation: check that the registry entry the timer captured is still the active one before emitting `CHANGE_MAP`.
- Race B: session-end fires concurrently with timer expiration. Both paths attempt to cancel + emit. Mitigation: registry uses compare-and-swap-style remove (`remove if entry == captured`), and `CHANGE_MAP` is idempotent on atlas-character.
- Race C: `MAP_CHANGED` for a different character that happens to share the same ID across tenants. Mitigation: registry key includes tenant id.

### 8.3 Observability

- Log at `Info` on timer start, `Info` on natural cancel, `Warn` on forced-return-via-expiration, `Warn` on forced-return-via-disconnect.
- Each log line includes tenant id, character id, map id, instance uuid, forcedReturnMapId.
- Metrics (counter, OpenTelemetry per task-040 conventions): `map_time_limit_started_total`, `map_time_limit_expired_total`, `map_time_limit_disconnect_total`, `map_time_limit_cancelled_total`. Tagged by tenant id and (where bounded) map id.

### 8.4 Multi-tenancy

- All registry operations carry tenant id in the key.
- All emitted Kafka commands inherit the tenant header from the originating event.
- No cross-tenant aggregation, no shared registries.

### 8.5 Security

- No new network surface.
- No user input fed into the timer beyond what already passes through atlas-data → atlas-channel (already trusted).

### 8.6 Failure modes

- Channel pod crash: in-flight timers are lost. Characters resident on those timers either (a) reconnect to a new channel-server with their original map persisted (so they re-enter the time-limited map; a fresh timer starts) or (b) were already mid-warp. Acceptable for v1.
- Kafka producer unavailable when timer fires: the forced-return command fails to publish. Log + metric. Player remains on the map until they exit manually. Acceptable for v1; could be retried in future work.

## 9. Open Questions

1. Exact placement of the timer registry package — `map/timer/`, `character/timer/`, or co-located with the consumer? Defer to design phase.
2. Map-clock packet — the v83 opcode and payload need to be confirmed against existing references (`socket/writer/`). Defer to design phase.
3. Whether the timer registry should be exposed for unit tests via dependency injection or as a singleton — design phase.
4. Whether channel-change should be modeled as session-end (current proposal) or get its own dedicated hook — review once the channel-change flow is re-read in the design phase.
5. Whether the `instance != uuid.Nil` case (character on a time-limited map *inside* an instance) should warp to `forcedReturnMapId` in `uuid.Nil` (proposed) or in the same instance. The instance-transports memory says forced-return targets `uuid.Nil`; confirm this generalizes.

## 10. Acceptance Criteria

- [ ] `data/map/Model` exposes `ForcedReturnMapId()`, `TimeLimit()`, `IsTimeLimited()` and the values match what atlas-data publishes for known time-limited maps.
- [ ] Entering a map with `timeLimit > 0` and `forcedReturnMapId != 999999999` registers a timer for the character, scoped by tenant + character id.
- [ ] Entering a map with no time limit (FM, towns, etc.) registers no timer.
- [ ] Re-entering the same time-limited map resets the timer to full duration.
- [ ] Taking a portal to a non-time-limited map cancels the timer and emits no `CHANGE_MAP` warp.
- [ ] Taking a portal directly to another time-limited map cancels the old timer and starts a new one with the new map's `timeLimit`.
- [ ] When the timer expires while the character is connected, a `CHANGE_MAP` command is published with the map's `forcedReturnMapId`, the character is warped to that map, and the registry entry is cleaned up.
- [ ] Logging out / disconnecting on a time-limited map publishes a `CHANGE_MAP` command to forced-return; reconnecting puts the character on `forcedReturnMapId`, not on the time-limited map.
- [ ] Channel change while on a time-limited map results in arrival at `forcedReturnMapId` on the destination channel.
- [ ] Death on a time-limited map continues to follow `respawn/processor.go` (uses `returnMapId`, NOT `forcedReturnMapId`).
- [ ] Channel-server graceful shutdown cancels all timers and emits zero `CHANGE_MAP` commands.
- [ ] Two tenants' characters with identical character ids on identical maps maintain independent timers.
- [ ] Player sees a client-side countdown on entering a time-limited map.
- [ ] Logs and metrics fire per §8.3 for start, natural cancel, expiration, and disconnect-forced-return.
