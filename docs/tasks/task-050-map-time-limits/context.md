# Map Time Limits — Implementation Context

Quick reference for executing agents. Read alongside `plan.md`.

---

## Source artifacts

- PRD: [`prd.md`](./prd.md)
- Design (authoritative): [`design.md`](./design.md) — note design re-homes the timer to **atlas-maps**, deviating from PRD §7. Honor the design.

---

## Owning service

**atlas-maps** — `/home/tumidanski/source/atlas-ms/atlas/services/atlas-maps/atlas.com/maps/`

Atlas-channel becomes a dumb renderer for the new `MAP_TIMER_STARTED` event.

---

## Key code anchors

### atlas-maps (timer owner)

| Path | Purpose / what it currently does |
|---|---|
| `kafka/consumer/character/consumer.go` | Existing CHARACTER_STATUS consumer. Hosts MAP_CHANGED (line 77) + CHANNEL_CHANGED (line 90) handlers. **Add timer cancel/register + force-return-fallback calls here.** |
| `kafka/message/character/kafka.go` | `EVENT_TOPIC_CHARACTER_STATUS` envelope (consumed). **Add CHANGE_MAP command envelope (`COMMAND_TOPIC_CHARACTER`, ChangeMapBody, Command[E]) here so the timer can produce.** |
| `kafka/message/map/kafka.go` | `EVENT_TOPIC_MAP_STATUS` envelope (produced). Currently has CHARACTER_ENTER/EXIT/WEATHER_*. **Add `MAP_TIMER_STARTED` constant + body.** |
| `kafka/producer/producer.go` | `producer.Provider` definition. |
| `kafka/message/message.go` | `Buffer` + `Emit(p)(func(buf *Buffer) error)` pattern. |
| `map/processor.go` | Map domain processor (Enter/Exit/TransitionMap). Do **not** add timer hooks here — keep them in the consumer. |
| `map/producer.go` | Existing map-status providers (enterMapProvider/exitMapProvider). Mirror style for the new `mapTimerStartedProvider`. |
| `mist/registry.go` | Reference for tenant-scoped, sync.RWMutex registry pattern. |
| `mist/processor.go` | Reference for `Processor` interface + ProcessorImpl + tenant-from-ctx + producer.Provider injection. |
| `tasks/mist_tick.go:131` | Reference for OTel `tracer.Start(context.Background(), ...)` + `tenant.WithContext` rebuild used by detached goroutines. |
| `tasks/mist_tick_test.go:25-54` | `recordingProducer` recipe — copy verbatim into timer tests. |
| `data/map/script/{model,processor,rest,requests}.go` | Existing pattern for atlas-data REST fetches. Mirror for the new `data/map/info` package. |
| `main.go` | Wires consumers + handlers + tasks. **Add session consumer + handler registration; no task to register because timers self-schedule per-character.** |

### atlas-channel (renderer only)

| Path | Purpose |
|---|---|
| `kafka/consumer/map/consumer.go` | Existing EVENT_TOPIC_MAP_STATUS consumer. **Add `handleStatusEventMapTimerStarted` here + register it in InitHandlers (line 60-82 init, line 84+ handlers).** |
| `kafka/message/map/kafka.go` | Mirror of atlas-maps' EVENT_TOPIC_MAP_STATUS envelope. **Add `EventTopicMapStatusTypeMapTimerStarted` constant + `MapTimerStarted` body (must match atlas-maps payload byte-for-byte).** |
| `data/map/{model,rest,processor}.go` | Reference for the `IsTimeLimited()`-style predicate placement (NOT modified — atlas-channel doesn't decide time-limited any more). |

### Shared / external (do NOT modify)

| Path | Purpose |
|---|---|
| `libs/atlas-packet/field/clientbound/clock.go` | `NewTimerClock(seconds uint32)` + `ClockWriter` already exist (lines 12, 42). atlas-channel calls `session.Announce(...)(ClockWriter)(NewTimerClock(seconds).Encode)(s)`. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go:17` | Reference shape for `ChangeMapProvider`. atlas-maps will implement its own copy. |
| `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:119` | atlas-character's existing `CHANGE_MAP` consumer. **Unchanged.** Confirms our published command is consumed. |
| `services/atlas-channel/atlas.com/channel/session/processor.go:335` | Channel publishes `DestroyedStatusEventProvider` to `EVENT_TOPIC_SESSION_STATUS` on TCP teardown — the real-time signal atlas-maps will subscribe to. |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/kafka/{message,consumer}/session/` | Reference for SESSION_STATUS consumer pattern. Copy the envelope and consumer registration shape. |

---

## Key decisions locked in

1. **Timer registry lives in atlas-maps**, NOT atlas-channel (PRD §7 superseded by design §1).
2. **atlas-channel is a dumb renderer** for `MAP_TIMER_STARTED` — no map-data fields, no decision logic.
3. **Forced-return target is always `Instance = uuid.Nil`** (matches instance-transports convention; design §8).
4. **`time.AfterFunc` (or `*time.Timer`) per entry**, NOT a shared ticker. Each per-character timer schedules its own goroutine.
5. **Race-safe registry API**: `Register / Cancel / Claim(token) / ClaimAny`. Token is a UUID stamped on the entry at register time.
6. **Detached goroutine context**: `context.Background()` + `tenant.WithContext(ctx, entry.tenant)` + a fresh OTel span. Never inherit the Kafka-consumer ctx.
7. **No persisted state.** Channel restart = atlas-maps process restart; in-flight timers are lost. Acceptable per PRD §8.6 (and atlas-maps' existing crash semantics).
8. **`forcedReturnMapId` sentinel for "no forced return"** is `999999999` (matches atlas-data and atlas-channel convention).
9. **`tasks.Register` is NOT used** for the timer registry — there's no tick loop. Each registered entry self-schedules its own `time.Timer`.
10. **Observability via OTel spans only**, no `Int64Counter` calls (per task-040 conventions).

## Naming conventions

- Package for new map-info data: **`info`** at `data/map/info/` (sibling to `data/map/script/`). Imported as `mapInfo "atlas-maps/data/map/info"`.
- Package for new timer: **`timer`** at `map/timer/`. Imported as `mapTimer "atlas-maps/map/timer"`.
- New event type constant: `EventTopicMapStatusTypeMapTimerStarted = "MAP_TIMER_STARTED"`.
- New command type constant: `CommandChangeMap = "CHANGE_MAP"` (matches atlas-character's value at `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:15`).
- New command topic env var: `EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"`.

## Race semantics summary

| Scenario | Resolution |
|---|---|
| Timer goroutine vs MAP_CHANGED-cancel | Goroutine: `Claim(key, token)`; MAP_CHANGED: `Cancel(key)`. Whichever locks first wins; loser sees no entry/different token and bails. |
| Timer goroutine vs SESSION_DESTROYED | Goroutine: `Claim(key, token)`; SESSION_DESTROYED: `ClaimAny(key)`. Whichever wins emits CHANGE_MAP exactly once; loser bails. |
| SESSION_DESTROYED vs CHANNEL_CHANGED order | SESSION_DESTROYED arrives first (real-time TCP teardown). CHANNEL_CHANGED is the 5s-debounced derivative — by the time it lands, the registry should already be empty. If not (rare race), CHANNEL_CHANGED also runs `ForceReturnIfTracked` as belt-and-suspenders. atlas-character's `CHANGE_MAP` is idempotent. |
| Cross-tenant character-id collision | Registry key is `(tenant.Id, characterId)`. Two tenants are always in different buckets. |

## Build / test commands

```bash
# atlas-maps full build + tests
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./...

# atlas-channel full build + tests
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...

# Targeted — just the new packages
cd services/atlas-maps/atlas.com/maps && go test ./map/timer/... ./data/map/info/... ./kafka/consumer/session/...
```

## What NOT to touch

- `respawn/processor.go` (death respawn — uses `returnMapId`, NOT `forcedReturnMapId`).
- atlas-character's CHANGE_MAP consumer (already does the persisted-map rewrite).
- atlas-data (already publishes both fields).
- atlas-saga-orchestrator (forced-return is a single command, not a saga).
- atlas-channel's `data/map/Model` (no new fields — atlas-channel does NOT decide time-limited).
- atlas-channel's `kafka/consumer/character/consumer.go` MAP_CHANGED handler (the timer hooks live in atlas-maps' consumer, not channel's).
- `libs/atlas-packet/...` — `NewTimerClock` already exists.

## Common gotchas (read before coding)

- `tenant.MustFromContext(ctx)` panics outside a tenant ctx — always rebuild via `tenant.WithContext(context.Background(), entry.tenant)` in detached goroutines.
- `field.NewBuilder(worldId, channelId, mapId).SetInstance(uuid).Build()` — instance setter is mandatory; default is uuid.Nil otherwise.
- `time.AfterFunc(d, f)` returns a `*time.Timer`; call `.Stop()` to cancel. Stop returns false if the timer already fired or stopped — ignore that return value (cancel is idempotent through registry semantics).
- `producer.Provider` in atlas-maps = `func(token string) producer.MessageProducer`. Build via `producer.ProviderImpl(l)(ctx)`.
- The mock map processor at `services/atlas-maps/atlas.com/maps/map/mock/processor.go` is missing `GetCharactersInMapAllInstances`. Don't expand it for this task.
