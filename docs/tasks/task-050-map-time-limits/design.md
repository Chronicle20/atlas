# Map Time Limits — Design

Version: v1
Status: Approved
Created: 2026-05-03
PRD: [`prd.md`](./prd.md)

---

## 1. Owning service: atlas-maps (deviates from PRD §7)

The PRD assumes atlas-channel owns the per-character map-stay timer. This design re-homes the timer to **atlas-maps** for the following reasons:

- **Conceptual fit.** The timer's state is "this character has been on *this map* for X seconds." atlas-maps already owns the (character, map) join via its `visit/` and `map/character/` packages. atlas-channel owns (character, **session**), a different join.
- **Existing signal coverage.** atlas-maps already consumes the full character lifecycle stream — LOGIN, LOGOUT, MAP_CHANGED, CHANNEL_CHANGED, DELETED — at `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`. All start/stop signals the timer needs are already there.
- **Existing producer infrastructure.** atlas-maps already publishes character-targeted commands across the wire (e.g., mist tick → `COMMAND_TOPIC_CHARACTER_BUFF`).
- **Existing scheduled-tick pattern.** atlas-maps already runs per-map ticks (mist, weather, monster respawn). A per-character map-stay timer is the same shape.
- **Survives channel-pod restarts.** PRD §8.6 listed loss-on-channel-restart as an accepted v1 limitation. Under atlas-maps ownership it disappears for free.
- **Channel-change is a non-event for the timer state itself** — the character is on the same map across the channel boundary. (Forced-return on channel change is still implemented; see §3.)

**Service Impact (revised):**

| Service | Change |
|---|---|
| atlas-maps | Owns timer registry. Hooks MAP_CHANGED + SESSION_DESTROYED + CHANNEL_CHANGED + LOGOUT consumers. Emits MAP_TIMER_STARTED event + CHANGE_MAP command. New `data/map/Model` fields (`forcedReturnMapId`, `timeLimit`). |
| atlas-channel | Subscribes to new `MAP_TIMER_STARTED` event from atlas-maps' map-status topic. Renders `clientbound.NewTimerClock` packet on event. **No timer state. No new map data fields.** |
| atlas-character | None. Existing `handleChangeMap` consumer already does the persisted-map rewrite. |
| atlas-data | None. `timeLimit` and `forcedReturnMapId` already in JSON contract. |

Other PRD non-goals stand unchanged: no death-respawn change, no return-scroll change, no PQ-timer integration, no admin override.

---

## 2. Wire shape — atlas-maps tells atlas-channel via an event

Adding a new **status event** to atlas-maps' existing outbound topic `EVENT_TOPIC_MAP_STATUS` (`services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:11`). atlas-channel becomes a new consumer of this topic.

**New event type: `MAP_TIMER_STARTED`**

| Field | Type | Notes |
|---|---|---|
| (envelope) `transactionId` | uuid.UUID | per existing StatusEvent envelope shape |
| (envelope) `worldId` | world.Id | |
| (envelope) `characterId` | uint32 | |
| `Body.channelId` | channel.Id | for channel-scope filtering by atlas-channel's `sc.Is(...)` |
| `Body.seconds` | uint32 | duration to display |

**Why an event, not a command.** Commands are imperative ("change this character's HP"); events are facts ("a stat changed"). atlas-maps is communicating a fact: a map timer has been started. atlas-channel renders the consequence the same way it renders consequences of every other status event today (`STAT_CHANGED → StatChanged packet`, `MAP_CHANGED → SetField packet`, `EXPERIENCE_CHANGED → CharacterStatusMessage packet`). atlas-channel never makes the "is this map time-limited?" decision itself — atlas-maps owns it.

**Why `EVENT_TOPIC_MAP_STATUS`, not a new topic.** It is atlas-maps' authoritative outbound stream. Existing inhabitants are weather start/end events. Future map-status concerns (timer pause/resume, dynamic adjustment, party-PQ countdowns) drop in without new wiring.

**No `MAP_TIMER_CANCELLED` event.** The v83 client clears its clock overlay on the next `MAP_CHANGED`, per PRD §4.6. No explicit stop is needed.

---

## 3. Timer lifecycle hooks (atlas-maps consumers)

### 3.1 Start / cancel — `handleStatusEventMapChangedFunc` (existing, line 77)

Existing handler currently calls `TransitionMapAndEmit`. Add timer-registry interaction:

1. If the **old** map had a registry entry, cancel it. (Covers normal exits via portal, scroll, death-respawn warp, generic CHANGE_MAP.)
2. Look up the **new** map's metadata (`map.Processor.GetById`).
3. If the new map `IsTimeLimited()`, register a fresh entry with duration = `time.Duration(timeLimit) * time.Second` and emit `MAP_TIMER_STARTED` event so atlas-channel renders the clock.

### 3.2 Forced-return on disconnect — **new consumer for `EVENT_TOPIC_SESSION_STATUS`**

atlas-channel's `session.Destroy` (`services/atlas-channel/atlas.com/channel/session/processor.go:335`) publishes `DestroyedStatusEventProvider(sessionId, accountId, characterId, channel.Model)` immediately on TCP teardown.

atlas-maps adds a new consumer for `EVENT_TOPIC_SESSION_STATUS`. On the `DESTROYED` event:

1. Registry lookup keyed by `(tenantId, characterId)`.
2. If an entry exists, atomically claim+remove it and emit `CHANGE_MAP` with `MapId = forcedReturnMapId`, `Instance = uuid.Nil`, `PortalId = 0`.

This single hook covers **both true logout AND channel-change** because both produce `SESSION_DESTROYED` in real time. atlas-character processes the `CHANGE_MAP` and rewrites the persisted `mapId` even when no session is active. Player re-logs in (or destination channel reads atlas-character) to find `mapId = forcedReturnMapId`.

**Why not LOGOUT?** atlas-character debounces SESSION_DESTROYED → LOGOUT by 5+ seconds (`services/atlas-character/atlas.com/character/session/task.go:45`) to disambiguate logout from in-flight channel-change. SESSION_DESTROYED is the real-time signal; LOGOUT is too late.

### 3.3 Channel-change fallback — `handleStatusEventChannelChangedFunc` (existing, line 90)

Race window: client reconnects to destination channel faster than the SESSION_DESTROYED → CHANGE_MAP → atlas-character DB write chain (~20-30ms). Worst case the destination channel briefly serves the time-limited map.

Belt-and-suspenders fallback: when CHANNEL_CHANGED arrives at atlas-maps, if the registry STILL has an entry for the character (meaning the SESSION_DESTROYED-driven CHANGE_MAP didn't land first), emit CHANGE_MAP again. atlas-character is idempotent for this — second CHANGE_MAP rewrites again. Player gets warped from the brief flash to forced-return.

In the common case (>95% per typical client reconnect timing) the SESSION_DESTROYED path wins and the CHANNEL_CHANGED handler finds an empty registry and no-ops.

### 3.4 Timer expiration in-session

`time.AfterFunc` callback runs:

1. `registry.Claim(key, capturedToken)` — atomic remove-if-token-still-matches.
2. If claim succeeds, emit `CHANGE_MAP` with the entry's `forcedReturnMapId`, `Instance = uuid.Nil`, `PortalId = 0`.
3. atlas-character processes the command, rewrites `mapId`, emits `MAP_CHANGED`.
4. atlas-channel's existing `handleStatusEventMapChanged` warps the live session.
5. atlas-maps' MAP_CHANGED handler (3.1) sees the destination is non-time-limited and the prior entry is already gone (claimed in step 1), so registration logic for the new map is the only side effect.

### 3.5 LOGOUT — unchanged

The existing handler at line 65 keeps calling `ExitAndEmit`. No registry interaction needed; SESSION_DESTROYED already handled the forced-return. (LOGOUT-driven forced-return would always be a no-op since SESSION_DESTROYED arrives ≥5s earlier.)

---

## 4. Timer registry — Go-internal design

### 4.1 Placement

New package: `services/atlas-maps/atlas.com/maps/map/timer/`. Standard atlas Processor pattern (interface + ProcessorImpl + `getRegistry()` singleton via `sync.Once`).

### 4.2 Entry shape

| Field | Type | Notes |
|---|---|---|
| tenant | tenant.Model | captured from originating ctx; used to rebuild ctx in goroutine |
| characterId | uint32 | |
| field | field.Model | the time-limited map (worldId, channelId, mapId, instance) |
| forcedReturnMapId | _map.Id | |
| seconds | uint32 | original duration; used in MAP_TIMER_STARTED event |
| token | uuid.UUID | per-registration token for race-safe claim |
| timer | *time.Timer | the underlying scheduled callback |
| expiresAt | time.Time | for diagnostics |

Key: `(tenant.Id, characterId)`. One entry per character.

### 4.3 Registry API (race-safe)

```
Register(key, entry) -> token             # adds entry, schedules goroutine, returns token
Cancel(key)                               # atomic remove-and-stop; idempotent
Claim(key, token) -> (entry, claimed)     # atomic remove-if-token-matches; used by goroutine
ClaimAny(key) -> (entry, claimed)         # atomic remove-anything; used by SESSION_DESTROYED handler
```

All operations behind a `sync.RWMutex` per atlas registry conventions.

**Race A: timer fires concurrently with MAP_CHANGED-cancel.** Goroutine calls `Claim(key, token)`. If MAP_CHANGED ran `Cancel` first, Claim finds nothing or finds a different token — goroutine bails. If goroutine ran first, Cancel finds nothing — also a no-op.

**Race B: timer fires concurrently with SESSION_DESTROYED.** Goroutine calls `Claim(key, token)`; DESTROYED handler calls `ClaimAny(key)`. Whichever locks first wins. The loser sees an empty registry and bails. Both paths emit CHANGE_MAP, so even the unlikely double-fire (if our locking is broken) is idempotent on atlas-character.

**Race C: cross-tenant character ID collision.** Registry key includes `tenant.Id`.

### 4.4 Tenant context propagation into goroutine

Mirror `services/atlas-character/atlas.com/character/session/task.go:33-41`:

```
sctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), "MapTimer.Expire")
defer span.End()
tctx := tenant.WithContext(sctx, entry.tenant)
// emit CHANGE_MAP using tctx
```

`context.Background()` deliberately decouples from the originating Kafka-consumer ctx (which may be cancelled before the timer fires). `context.WithoutCancel` is not in use in this codebase.

### 4.5 Lifecycle

- Initialized at atlas-maps startup; cancelled via `tdm.Context().Done()`. All in-flight timers stop on shutdown — they do NOT fire forced-return commands during shutdown (PRD §4.2 acceptance).

---

## 5. atlas-channel side: dumb renderer

New package: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/` (mirrors `kafka/consumer/character/`).

- Subscribes to `EVENT_TOPIC_MAP_STATUS`.
- Handler `handleMapStatusEventMapTimerStarted` filters `event.Type == "MAP_TIMER_STARTED"`, then `sc.Is(tenant, worldId, channelId)` for channel scope.
- Looks up session via `session.NewProcessor.IfPresentByCharacterId(sc.Channel())`.
- Writes `clientbound.NewTimerClock(seconds)` via `session.Announce`.

**No `data/map/Model` changes in atlas-channel.** atlas-channel does not need to know which maps are time-limited; atlas-maps tells it via the event.

---

## 6. atlas-maps map data extraction

atlas-maps' map-data layer must learn `timeLimit` and `forcedReturnMapId`. Pattern mirrors atlas-channel's `data/map/rest.go` extract — both fields are already in the JSON contract from atlas-data; current atlas-maps Extract drops them.

- `data/map/Model`: add `timeLimit int32` + `forcedReturnMapId _map.Id` fields with getters.
- `data/map/rest.go::Extract`: populate both.
- `Model.IsTimeLimited()` predicate: `TimeLimit() > 0 && ForcedReturnMapId() != 999999999`.

(Whether the file path is exactly `data/map` in atlas-maps, or a different sub-package, will be confirmed by the planning phase by reading the current atlas-maps tree. The pattern is the same regardless.)

---

## 7. CHANGE_MAP producer in atlas-maps

Mirror `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go:17`. New `ChangeMapProvider` in atlas-maps targeting `EnvCommandTopicCharacter`. Body is the existing `ChangeMapBody`. atlas-character's existing `handleChangeMap` consumer is unchanged.

The saga orchestrator's `ChangeMapProvider` is NOT shared as a library: the saga orchestrator and atlas-maps are independent services and the project conventions favor straightforward duplication of small producer functions over cross-service indirection (`CLAUDE.md` Code Patterns).

---

## 8. Instance handling

PRD §9.5 — character on a time-limited map inside an instance.

**Decision: forced-return always targets `Instance = uuid.Nil`** (matches the instance-transports convention per project memory; matches PRD §4.5).

The timer entry stores the originating `field.Model` including the instance UUID, so we know where the character was. But the warp destination drops the instance. v83 content has effectively no time-limited maps inside instances (FM, town entries, event lobbies are all non-instanced; PQ stages are non-goal per PRD §2). If a future map needs return-inside-instance, that's a follow-up extending data.

---

## 9. Testing

Standard atlas-maps Processor pattern; reference `services/atlas-maps/atlas.com/maps/map/character/processor_test.go`.

**Approach:**

- `Processor` interface + `ProcessorImpl`. Constructor takes the Kafka producer, allowing tests to inject a recorder (`tasks/mist_tick_test.go:153` shows the pattern).
- **Test the registry state machine, not the goroutine scheduling.** Cover:
  - Register-then-Cancel — entry removed, goroutine stopped.
  - Register-then-Claim with matching token — entry removed, returns entry.
  - Register-then-Claim with stale token — no-op, returns nothing.
  - Register-then-ClaimAny — entry removed regardless of token.
  - Race simulations: Cancel followed by Claim (loser bails); ClaimAny followed by Claim (loser bails).
- **End-to-end "timer fires" test** uses 100ms durations + `time.Sleep(150 * time.Millisecond)`; minor flake risk acceptable per existing project patterns.
- **Mock Kafka producer** uses the existing recorder pattern; assert that CHANGE_MAP and MAP_TIMER_STARTED messages were emitted with expected bodies.

No injectable `Clock` abstraction. The state-machine split makes time mocking unnecessary.

---

## 10. Observability

PRD §8.3 wording (`map_time_limit_started_total` etc.) is interpreted under the codebase's actual pattern from task-040: **OTel spans, not direct counters.** Spanmetrics (Tempo → Prometheus) auto-derives `traces_spanmetrics_calls_total{span_name=...}` from named spans. The codebase has no `Int64Counter` calls today and task-040 explicitly directs against new ones.

**Spans:**

| Span name | Trigger | Attributes (curated for spanmetrics) |
|---|---|---|
| `MapTimer.Start` | Registry registers entry | `tenant.id`, `world.id`, `map.id`, `forced.return.map.id` |
| `MapTimer.Cancel` | MAP_CHANGED off the time-limited map (or different time-limited map) | `tenant.id`, `world.id`, `map.id` |
| `MapTimer.Expire` | Goroutine claims and fires CHANGE_MAP | `tenant.id`, `world.id`, `map.id`, `forced.return.map.id` |
| `MapTimer.Disconnect` | SESSION_DESTROYED handler claims and fires CHANGE_MAP | `tenant.id`, `world.id`, `map.id`, `forced.return.map.id` |

`character.id` stays on the span body for trace search but is NOT a spanmetric dimension (cardinality, per task-040 §3 explicit exclusion).

**Logs:** Info on Start and Cancel; Warn on Expire and Disconnect. Each line includes tenant, character, map, instance, forcedReturnMapId, per PRD §8.3.

---

## 11. PRD revisions captured here

For audit-phase clarity, this design deviates from the PRD as follows. All changes preserve PRD acceptance criteria (§10).

| PRD section | Original | Revised in this design | Reason |
|---|---|---|---|
| §4.2, §4.3, §6, §7 | Timer registry in atlas-channel | Timer registry in atlas-maps | Conceptual fit; existing signal coverage; survives channel restart |
| §4.6 | atlas-channel sends clock packet on its own | atlas-maps emits `MAP_TIMER_STARTED`; atlas-channel renders | atlas-channel does not own the time-limit decision |
| §4.3 (logout hook) | session-end path inside atlas-channel | new `EVENT_TOPIC_SESSION_STATUS` consumer in atlas-maps | Real-time signal vs. 5s-delayed LOGOUT |
| §4.3 / §4.4 (channel-change) | session-end path inside atlas-channel | SESSION_DESTROYED + CHANNEL_CHANGED fallback in atlas-maps | Same hook covers both cases; CHANNEL_CHANGED is belt-and-suspenders |
| §8.3 | Direct OTel counters with explicit names | Span-derived metrics via task-040 conventions | Codebase pattern; no Int64Counter calls exist |
| §8.6 | Channel restart loses timers (accepted limitation) | Channel restart does not lose timers (atlas-maps survives it) | Free win from re-homing |

---

## 12. Acceptance criteria mapping

Cross-references PRD §10:

- ✓ **Map data exposes ForcedReturnMapId/TimeLimit/IsTimeLimited** — in atlas-maps' map model (§6).
- ✓ **Timer registered on entry to time-limited maps** — handled by 3.1.
- ✓ **No timer for non-time-limited maps** — `IsTimeLimited()` predicate.
- ✓ **Re-entering resets timer** — Register cancels prior entry first (3.1).
- ✓ **Portal exit cancels timer, no warp** — 3.1's cancel branch.
- ✓ **Direct portal to another time-limited map** — 3.1 cancel-then-register.
- ✓ **Expiration → CHANGE_MAP → warp** — 3.4.
- ✓ **Logout → forced-return persisted** — 3.2 via SESSION_DESTROYED.
- ✓ **Channel change → arrival at forced-return** — 3.2 (primary) + 3.3 (fallback).
- ✓ **Death follows respawn/processor.go (returnMapId)** — no change to respawn flow.
- ✓ **Graceful shutdown cancels all timers, zero CHANGE_MAP** — 4.5.
- ✓ **Cross-tenant character isolation** — registry key includes tenantId (4.3, race C).
- ✓ **Client-side countdown** — MAP_TIMER_STARTED → atlas-channel renders TimerClock.
- ✓ **Logs and metrics fire** — §10 spans.
