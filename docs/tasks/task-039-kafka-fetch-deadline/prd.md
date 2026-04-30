# Kafka FetchMessage Deadline + Tick-and-Escalate — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-30
---

## 1. Overview

The Kafka consumer scaffolding in `libs/atlas-kafka/consumer` was made self-healing by task-016, which wraps reader lifecycle in an outer recreate loop and exposes per-consumer state (`lastFetchAt`, `recreateCount`, `aliveSince`, etc.) via `GET /api/debug/consumers`. That self-heal is **error-driven**: the reader is recreated only when `reader.FetchMessage(ctx)` returns an error. Task-016 explicitly excluded staleness heuristics in its non-goals.

That carve-out has now produced an outage. On 2026-04-30 both `atlas-maps` and `atlas-monsters` were observed alive for ~18 hours with `lastFetchAt: "0001-01-01T00:00:00Z"`, `recreateCount: 0`, empty `lastError`. The pods looked healthy by every existing signal. The `/api/debug/consumers` endpoint reported all consumers (`EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_MAP_STATUS`, etc.) as registered with handlers attached but never having fetched a message. Meanwhile `atlas-channel`, on the same broker but using a per-pod UUID-suffixed consumer group, fetched normally. The visible end-user symptom was that monsters never spawned in maps containing characters: `atlas-maps`'s respawn task in `tasks/respawn.go:35` consults its in-memory character registry, which is populated by the `EVENT_TOPIC_CHARACTER_STATUS` consumer, which was wedged. The cascade produced a fully silent failure — log-clean, error-clean, alive-by-aliveness-check, broken in practice.

The root cause is the failure shape itself: `FetchMessage` blocks indefinitely instead of returning an error. Static consumer groups (`"Map Service"`, `"Monster Registry Service"`) appear to have ended up with broker-side state that prevents partition assignment from completing, so the long-poll inside `FetchMessage` never returns. No error means no recreate. The reader is alive, the goroutine is alive, the FetchMessage call is alive, and nothing makes progress.

This task adds a **per-call deadline** on `FetchMessage` so that any indefinite block becomes an observable event in bounded time. Deadline expirations are *not* fatal — the consumer ticks back to the top of the inner loop with a fresh deadline, the same reader, the same partition assignment. Only after `maxConsecutiveTimeouts` consecutive deadline expirations with no successful fetch in between does the loop return a sentinel error (`errFetchWedged`) to the existing outer recreate-with-backoff path, which rebuilds the reader and rejoins the consumer group from scratch. A successful fetch in between resets the counter. The change is self-contained to `libs/atlas-kafka`; every service inherits it on rebuild without code changes.

## 2. Goals

Primary goals:
- A Kafka consumer whose `reader.FetchMessage(ctx)` blocks indefinitely (stuck partition assignment, stuck long-poll, broker-coordinator hang) is detected within `maxConsecutiveTimeouts × fetchTimeout` of process start (~15 minutes with defaults) and recovered by the existing outer recreate-with-backoff path without operator action.
- Idle topics — topics that legitimately produce no messages for hours in a dev cluster — do **not** cause spurious reader recreations or consumer-group rebalances. Tick-and-continue is silent in steady state.
- Active topics — `COMMAND_TOPIC_MONSTER_MOVEMENT`, `EVENT_TOPIC_CHARACTER_STATUS` under load, etc. — never observe the deadline because messages return in milliseconds.
- The wedge case is *visible* in `/api/debug/consumers` before recreate fires (via new `consecutiveTimeouts`, `lastTimeoutAt` fields) and after (via the existing `recreateCount` jump and `lastError` containing the sentinel string).
- The change ships as a `libs/atlas-kafka` library update. Every service that owns Kafka consumers picks up the new behavior on the next image rebuild. No `main.go` changes, no manifest changes, no env-var changes.

Non-goals:
- Producer-side outage detection. A topic going silent because the upstream service stopped producing is *not* a consumer wedge and is out of scope.
- Per-topic cadence learning or expected-traffic modeling. The deadline is a fixed value; we do not infer "this topic should see traffic every N seconds."
- Anything resembling task-016's explicitly-excluded staleness heuristics — comparing `lastFetchAt` against wall-clock at intervals, sweeping consumers from a watchdog goroutine, etc. This task is purely a per-call timeout inside `runFetchLoop`.
- No Prometheus metrics, Grafana dashboards, alert rules, or k8s liveness probes. The signal lives in `/api/debug/consumers` and structured logs.
- No process crash / restart on a wedge. Atlas services maintain stateful registries (and atlas-channel maintains live game sessions); recovery without restart remains a hard design constraint inherited from task-016.
- No producer changes, no topic-config changes, no consumer-group naming changes (atlas-channel's per-pod UUID groups stay UUID; static-group services stay static).

## 3. User Stories

- As an Atlas developer running a stack overnight, I want a wedged consumer to recover on its own within ~15 minutes so that I don't wake up to monsters not spawning and have to manually delete pods.
- As an operator inspecting `/api/debug/consumers`, I want to see `consecutiveTimeouts` advancing on a wedged consumer *before* it recreates, so that I can distinguish "currently mid-detection" from "currently healthy and idle."
- As an operator inspecting `/api/debug/consumers` after a wedge has self-recovered, I want `recreateCount` to be non-zero and `lastError` to contain `consumer fetch wedged: exceeded consecutive timeouts`, so that I can attribute the recreate to a wedge versus a transient broker error.
- As a developer reading logs after a wedge incident, I want one `Warn`-level log line per wedge-detected event with consumer name and topic, so that the diagnostic that was missing during the 2026-04-30 incident is present in future incidents.
- As a developer of a consumer with unusual cadence (none today, but hypothetically), I want to override `fetchTimeout` and `maxConsecutiveTimeouts` per-consumer via decorators on `consumer.NewConfig`, without forking the library.
- As a developer running `libs/atlas-kafka/consumer` tests, I want deterministic coverage of the three behaviors — tick on idle, escalate on wedge, counter resets on success — using the existing `KafkaReader` test fake or a minimal extension.

## 4. Functional Requirements

### 4.1 Per-call deadline on `FetchMessage`

In `libs/atlas-kafka/consumer/manager.go`, rewrite `runFetchLoop` (currently lines 339–374). Each iteration:

1. Check parent `ctx` for cancellation; if cancelled, return `ctx.Err()` (existing shutdown path).
2. Build a child context with deadline: `fetchCtx, cancel := context.WithTimeout(ctx, c.fetchTimeout)`.
3. Call `reader.FetchMessage(fetchCtx)`.
4. **Immediately** call `cancel()` after the call returns (success or failure). Do not `defer cancel()` — we are inside an unbounded `for` loop and a deferred cancel would leak until function return.
5. Branch on the result (see §4.2).

`c.fetchTimeout` is a new `time.Duration` field on `Consumer`, populated from the `Config` at `AddConsumer` time (see §4.5). Default `5 * time.Minute`.

### 4.2 Tick-and-escalate state machine

Local to `runFetchLoop`: `consecutiveTimeouts := 0`. Branch on the `err` returned by `reader.FetchMessage(fetchCtx)`:

- **Successful fetch** (`err == nil`):
  - `consecutiveTimeouts = 0`
  - `c.recordFetch()` (existing behavior — updates `lastFetchAt`, clears `lastError`)
  - Process the message via `c.processMessage` and commit (existing behavior, lines 366–372).
  - Continue the loop.

- **Deadline expiration with parent ctx still alive** (`errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil`):
  - `consecutiveTimeouts++`
  - `c.recordTimeout()` — new method that updates `lastTimeoutAt` and increments the snapshot-visible `consecutiveTimeouts` field (see §4.5). **Does not** touch `lastError` or `lastErrorAt`. Idle is not an error.
  - Log at `Debug`: `"FetchMessage deadline expired (consecutive=%d/%d); ticking."` Single line per tick.
  - If `consecutiveTimeouts >= c.maxConsecutiveTimeouts`:
    - Log at `Warn`: `"FetchMessage wedged: %d consecutive timeouts on topic [%s] (group [%s]); forcing reader recreate."`
    - Return `errFetchWedged` (new package-level sentinel — see §4.4) so the outer `start` loop closes the reader and recreates.
  - Otherwise, `continue` the loop (same reader, fresh deadline next iteration).

- **Parent ctx cancellation** (`ctx.Err() != nil` or `errors.Is(err, context.Canceled)`):
  - Return the error. Outer loop treats as shutdown (existing behavior).

- **Any other error** (kafka-go transport error, broker rejection, EOF, etc.):
  - Return the error. Outer loop treats as recreate-eligible with backoff (existing behavior).

### 4.3 Drop the inner `retry.Try` block

Remove the existing `retry.Try` wrapping `FetchMessage` (manager.go:346–361). The current retry shape (3 attempts, 100ms initial, 500ms max — roughly 1 second before falling through) was task-016's allowance for "a transient kafka-go hiccup that self-resolves within ~1s stays on the current reader." With the new per-call deadline driving the loop, that allowance is unnecessary and actively harmful: `retry.Try` would interpret `DeadlineExceeded` as a retryable error and retry the call three times against the same already-deadlined `fetchCtx`, all of which fail immediately, producing log noise and confusing the new state machine.

The outer `start` loop (manager.go:291–331) is the proper retry mechanism for non-deadline errors — it closes the reader, applies capped exponential backoff, and rebuilds. That is sufficient. A genuine 1-second transient kafka-go hiccup will now cause one reader recreate instead of being absorbed silently. Acceptable cost: reader recreates are cheap relative to the cost of obscuring a wedge.

Confirm by reading `runFetchLoop` post-rewrite: only one `reader.FetchMessage` call site, no `retry.Try` import, the `retry` package import in `manager.go` is removed if unused elsewhere in the file (it currently appears only in this function).

### 4.4 Sentinel error

Define at package scope in `manager.go` (or a sibling file):

```go
// errFetchWedged is returned from runFetchLoop when FetchMessage has hit
// its deadline maxConsecutiveTimeouts times in a row without a successful
// fetch in between. The outer start loop treats it identically to any
// other recreate-eligible error: close reader, backoff, rebuild.
var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")
```

The sentinel is unexported. External callers (services) don't introspect consumer errors today; if a future need arises, export it then. Tests in the same package can reference it directly via `errors.Is`.

The outer `start` loop's existing `c.recordError(err)` call (manager.go:322) records the sentinel's message into `lastError` / `lastErrorAt`, so the debug endpoint surfaces `"consumer fetch wedged: exceeded consecutive timeouts"` automatically without further changes.

### 4.5 New `Consumer` state and `Snapshot` fields

Add to the `Consumer` struct (manager.go:180):

- `fetchTimeout time.Duration` — copied from `Config` at `AddConsumer` time. Default `5 * time.Minute`. Read-only after construction; no mutex needed.
- `maxConsecutiveTimeouts int` — copied from `Config` at `AddConsumer` time. Default `3`. Read-only after construction; no mutex needed.
- `consecutiveTimeouts int` — protected by `c.mu`. Updated by `recordTimeout` and `recordFetch`. Snapshot-visible.
- `lastTimeoutAt time.Time` — protected by `c.mu`. Updated by `recordTimeout`. Snapshot-visible.

Add to `Snapshot` (manager.go:201):

- `LastTimeoutAt time.Time`
- `ConsecutiveTimeouts int`

Update `Snapshot()` (manager.go:215) to copy them out under the existing mutex. Update the debug serializer in `libs/atlas-kafka/consumer/debug.go` (created by task-016) to expose them in the JSON:API attributes block as `lastTimeoutAt` (RFC 3339 with `Z`) and `consecutiveTimeouts` (integer).

New methods on `Consumer`:

```go
// recordTimeout marks one deadline expiration; called per tick by runFetchLoop.
// Idle, not an error: lastError / lastErrorAt are untouched.
func (c *Consumer) recordTimeout() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.lastTimeoutAt = time.Now()
    c.consecutiveTimeouts++
}
```

Modify `recordFetch` (manager.go:243) to also reset `consecutiveTimeouts = 0`. The existing `lastError = ""` clear stays.

Note on lifecycle of `consecutiveTimeouts` across reader recreates: it lives on the `Consumer` (not on `runFetchLoop` state alone) so the snapshot can read it between iterations. When a wedge fires and `runFetchLoop` returns `errFetchWedged`, the outer `start` loop calls `c.onReaderCreated(attempt)` for the new reader (manager.go:233). Extend `onReaderCreated` to reset `consecutiveTimeouts = 0` and `lastTimeoutAt = time.Time{}` when `attempt > 0`, alongside the existing `recreateCount++` and `lastError = ""`. This keeps the counter scoped to a single reader's lifetime and prevents stale counts from spilling into the next one.

### 4.6 New `Config` decorators

Add to `libs/atlas-kafka/consumer/config.go`:

```go
//goland:noinspection GoUnusedExportedFunction
func SetFetchTimeout(d time.Duration) model.Decorator[Config] {
    return func(c Config) Config {
        c.fetchTimeout = d
        return c
    }
}

//goland:noinspection GoUnusedExportedFunction
func SetMaxConsecutiveTimeouts(n int) model.Decorator[Config] {
    return func(c Config) Config {
        c.maxConsecutiveTimeouts = n
        return c
    }
}
```

Add the corresponding fields to `Config` (currently lines 22–30):

```go
fetchTimeout           time.Duration
maxConsecutiveTimeouts int
```

Set defaults in `NewConfig` (line 11):

```go
return Config{
    brokers:                brokers,
    name:                   name,
    topic:                  topic,
    groupId:                groupId,
    maxWait:                50 * time.Millisecond,
    startOffset:            kafka.FirstOffset,
    fetchTimeout:           5 * time.Minute,
    maxConsecutiveTimeouts: 3,
}
```

Wire the new fields through `AddConsumer` (manager.go:96–134) so the resulting `Consumer` carries them. Add field assignments alongside the existing copy of `c.brokers`, `c.headerParsers`, etc.

### 4.7 No service-side changes

No service `main.go` requires modification. Defaults apply. Decorators are opt-in. Docker rebuilds of services that depend on `libs/atlas-kafka` are sufficient to deploy the change.

If, in the future, a topic with cadence slower than 15 minutes is introduced and tick-and-continue overhead becomes undesirable, that consumer can be configured at registration time:

```go
character.InitConsumers(l)(cmf)(consumerGroupId, consumer.SetFetchTimeout(20*time.Minute))
```

Per the design conversation, **no current topic requires an override**. Defaults (5m / 3) are correct for every consumer in the monorepo today.

## 5. API Surface

### 5.1 New Go APIs

In `libs/atlas-kafka/consumer`:

- `func SetFetchTimeout(d time.Duration) model.Decorator[Config]`
- `func SetMaxConsecutiveTimeouts(n int) model.Decorator[Config]`

Unexported additions:

- `var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")`
- `func (c *Consumer) recordTimeout()`
- New fields on `Consumer`: `fetchTimeout`, `maxConsecutiveTimeouts`, `consecutiveTimeouts`, `lastTimeoutAt`
- New fields on `Config`: `fetchTimeout`, `maxConsecutiveTimeouts`

Modified:

- `Snapshot` gains two fields: `LastTimeoutAt time.Time`, `ConsecutiveTimeouts int`. This is a compatible addition — task-016's debug-route serializer is the only consumer of `Snapshot`, and adding optional JSON:API attributes is a non-breaking extension.

No existing exported signatures change. `AddConsumer`, `RegisterHandler`, `NewConfig`, `SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`, `ConfigReaderProducer`, `ResetInstance`, `KafkaReader`, `ReaderProducer`, `Manager`, `Consumer.Snapshot` — all source-compatible.

### 5.2 New HTTP surface

The existing `GET /api/debug/consumers` endpoint (added by task-016) gains two new attributes per consumer entry:

```json
{
  "type": "consumers",
  "id": "EVENT_TOPIC_CHARACTER_STATUS",
  "attributes": {
    "name": "status_event",
    "topic": "EVENT_TOPIC_CHARACTER_STATUS",
    "groupId": "Map Service",
    "brokers": ["kafka.home:9093"],
    "aliveSince": "2026-04-30T15:00:00Z",
    "lastFetchAt": "2026-04-30T15:02:14.712Z",
    "lastTimeoutAt": "2026-04-30T15:07:14.712Z",
    "lastErrorAt": "0001-01-01T00:00:00Z",
    "lastError": "",
    "consecutiveTimeouts": 1,
    "recreateCount": 0,
    "handlerCount": 5
  }
}
```

`lastTimeoutAt` is RFC 3339 with `Z`; the zero value (no timeouts ever) serializes as `"0001-01-01T00:00:00Z"`, consistent with how task-016 serializes `lastFetchAt` / `lastErrorAt`. `consecutiveTimeouts` is an integer; resets on successful fetch and on reader recreate.

No URL change. No new endpoints.

### 5.3 New Kafka surface

None.

### 5.4 Config surface

No new env vars. No new k8s manifest changes.

## 6. Data Model

No persisted data. All new state is in-process and resets on process restart, consistent with task-016. No migrations.

## 7. Service Impact

| Service / Library | Change |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` | Rewrite `runFetchLoop` per §4.1–§4.4 (per-call deadline, tick-and-escalate state machine, drop inner `retry.Try`). Add `errFetchWedged` sentinel. Extend `Consumer` struct with four new fields per §4.5. Add `recordTimeout` method. Modify `recordFetch` to reset `consecutiveTimeouts`. Modify `onReaderCreated` to reset `consecutiveTimeouts` and `lastTimeoutAt` on recreate. Wire `Config.fetchTimeout` / `Config.maxConsecutiveTimeouts` through `AddConsumer` into `Consumer`. Extend `Snapshot` and `Snapshot()` with the new fields. Remove `retry` package import if no longer used. |
| `libs/atlas-kafka/consumer/config.go` | Add `fetchTimeout` and `maxConsecutiveTimeouts` fields to `Config`. Set defaults (`5 * time.Minute`, `3`) in `NewConfig`. Add `SetFetchTimeout` and `SetMaxConsecutiveTimeouts` decorators. |
| `libs/atlas-kafka/consumer/debug.go` | Update the JSON:API serializer (created by task-016) to include `lastTimeoutAt` and `consecutiveTimeouts` in the attributes block. RFC 3339 with `Z` for the timestamp; integer for the counter. |
| `libs/atlas-kafka/consumer/manager_test.go` | Add three test cases per §11. May need to extend the existing `KafkaReader` fake so its `FetchMessage` blocks until the supplied ctx is cancelled or its deadline fires (instead of returning immediately). Existing tests must continue to pass; if any rely on `FetchMessage` returning instantly, adapt them by setting `SetFetchTimeout(1*time.Hour)` so the deadline never fires during the test. |
| `libs/atlas-kafka/consumer/debug_test.go` | If task-016's debug tests assert exact attribute keys, extend assertions to include `lastTimeoutAt` and `consecutiveTimeouts`. |
| All 49 consumer-owning services | No code change. Pick up new behavior on Docker rebuild against the updated `libs/atlas-kafka`. |
| `deploy/k8s/*.yaml` | No change. |

No producer changes, handler changes, topic changes, payload changes, database changes, REST resource changes, UI changes, saga changes, conversation script changes, or any game-facing behavior changes.

## 8. Non-Functional Requirements

**Performance.**

- Steady-state cost for active topics: zero. `FetchMessage` returns in milliseconds with a message; the deadline never fires; one `time.Now()` and one mutex-guarded counter reset per message — already paid by `recordFetch`. The new `consecutiveTimeouts = 0` reset is a single integer write inside the existing critical section.
- Steady-state cost for idle topics: one cancelled syscall per `fetchTimeout` (5 minutes) per consumer. Negligible at any plausible consumer count. Cancelled FetchMessage on segmentio/kafka-go returns `context.DeadlineExceeded` without broker round-trips when the long-poll is already established.
- Wedge-recovery cost: one reader recreate (which incurs one consumer-group rejoin / rebalance) every `maxConsecutiveTimeouts × fetchTimeout` (~15 minutes) until the underlying broker condition resolves. Bounded above by task-016's 10s outer-loop backoff cap.
- HTTP cost: two integer/timestamp lookups added to the existing `Snapshot()` mutex-guarded section. Non-measurable.

**Reliability & availability.**

- Detection bound: any indefinite block on `FetchMessage` is detected within `maxConsecutiveTimeouts × fetchTimeout` of process start (default ~15 minutes).
- No process restart, no game-session disconnection. Recovery is in-process via the existing task-016 outer loop.
- Backwards compatible at runtime: a service running the new library against a healthy broker behaves identically to the old library.

**Backwards compatibility.**

- Public `libs/atlas-kafka/consumer` API is source-compatible. The two new decorators are additive and optional.
- The JSON:API debug response gains two attributes; clients that ignore unknown attributes (the only sane behavior) are unaffected. No known external consumers exist today — the route is dev/ops-internal.
- On-wire Kafka traffic is unchanged. No producer, topic, or payload changes.
- Consumer-group behavior is unchanged in steady state. The wedge-recovery path uses the same reader-recreate mechanism task-016 already established.

**Observability.**

- New `Debug` log: `"FetchMessage deadline expired (consecutive=%d/%d); ticking."` per tick. Routine; suppressible at deployed log level if noisy.
- New `Warn` log: `"FetchMessage wedged: %d consecutive timeouts on topic [%s] (group [%s]); forcing reader recreate."` once per wedge.
- Existing `Info` log on reader recreate (task-016) fires immediately after the wedge log, providing the bridge from "detection" to "recovery" in the log timeline.
- New `/api/debug/consumers` attributes:
  - `consecutiveTimeouts` — integer, current consecutive-timeout count for this reader.
  - `lastTimeoutAt` — RFC 3339, time of most recent deadline expiration.
- An operator triaging "is consumer X wedged?" reads, in order: `consecutiveTimeouts > 0` (in progress), `recreateCount > 0` and `lastError == "consumer fetch wedged: exceeded consecutive timeouts"` (recovered).

**Security.**

- No new credentials, no new network paths, no auth changes. The debug route's existing exposure model from task-016 is unchanged.

**Multi-tenancy.**

- The deadline operates at consumer-group granularity, not tenant. No tenant scoping required. The debug route remains tenant-agnostic per task-016.

**Testing.**

- Existing `KafkaReader` test fake from task-016 may need to be extended so its `FetchMessage` respects ctx deadlines. The contract for the fake should be: block until ctx is cancelled (returning `ctx.Err()`) or until the test pumps a scripted message. This is the realistic kafka-go behavior; it should also make existing tests more representative.
- Tests use very short `fetchTimeout` (e.g. `50 * time.Millisecond`) to keep wall-clock duration sub-second.

## 9. Open Questions

- Whether the existing `KafkaReader` test fake in `manager_test.go` already respects ctx cancellation in `FetchMessage`. If not, extend it during implementation; document the change in the test file's preamble. If yes, no test scaffolding change required.
- Whether to also reset `consecutiveTimeouts` on a non-deadline error that triggers reader recreate (e.g., kafka-go transport error). The current spec resets in `onReaderCreated`, which fires on every recreate including non-wedge ones — implicitly handled. Reconfirm during implementation.

## 10. Acceptance Criteria

### Behavioral

- [ ] A consumer whose `FetchMessage` blocks indefinitely (modeled in tests by a fake that ignores incoming messages and only returns when ctx is cancelled) hits its deadline, ticks, and **does not** force a reader recreate after one timeout.
- [ ] The same consumer, after `maxConsecutiveTimeouts` consecutive deadline expirations with no successful fetch in between, returns `errFetchWedged` from `runFetchLoop`. The outer `start` loop closes the reader, applies the existing backoff, and creates a new reader. `recreateCount` increments to 1.
- [ ] A successful `FetchMessage` between deadline expirations resets `consecutiveTimeouts` to 0 and the wedge escalation never fires, even across many idle-busy-idle cycles.
- [ ] On parent ctx cancellation, the consumer exits cleanly within one `fetchTimeout` tick without recreating the reader.
- [ ] On a non-deadline `FetchMessage` error (e.g., simulated EOF or transport error), `runFetchLoop` returns the error directly to the outer recreate path. `consecutiveTimeouts` is not incremented on this path.
- [ ] Active topics in production never observe the deadline. (Verifiable post-deploy by inspecting `consecutiveTimeouts == 0` on `/api/debug/consumers` for `COMMAND_TOPIC_MONSTER_MOVEMENT` or any other busy topic during normal play.)

### Observability

- [ ] `GET /api/debug/consumers` returns `consecutiveTimeouts` and `lastTimeoutAt` for every consumer entry. Zero values serialize as `0` and `"0001-01-01T00:00:00Z"` respectively.
- [ ] `consecutiveTimeouts` increments on each deadline expiration and resets to 0 on successful fetch or reader recreate. Verifiable from a test that checks the `Snapshot()` returned values across the loop.
- [ ] After a wedge-driven recreate, `lastError` contains the literal string `"consumer fetch wedged: exceeded consecutive timeouts"`.
- [ ] A wedge produces exactly one `Warn`-level log line containing topic name and group id, immediately followed by the existing `Info`-level recreate log.

### Configuration

- [ ] `consumer.SetFetchTimeout(d time.Duration)` overrides the default `5 * time.Minute` per consumer.
- [ ] `consumer.SetMaxConsecutiveTimeouts(n int)` overrides the default `3` per consumer.
- [ ] Calling `consumer.NewConfig(...)` without either decorator yields the documented defaults.
- [ ] Decorators stack with existing decorators (`SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`) without ordering issues.

### Non-regression

- [ ] All existing `libs/atlas-kafka/consumer` tests pass without modification, except for tests that need a longer `fetchTimeout` decorator added to keep the new deadline from firing during the test (acceptable and minor).
- [ ] All 49 consumer-owning services build cleanly against the new library.
- [ ] Docker builds for the primary affected services (atlas-maps, atlas-monsters, atlas-channel, atlas-character, atlas-quest, atlas-inventory, atlas-saga-orchestrator) succeed against the new library. (Per project CLAUDE.md: shared-library changes require Docker build verification.)
- [ ] Steady-state behavior is unchanged: a service running against a healthy broker shows no extra reader recreates, no extra log lines (beyond the new `Debug` ticks if log level is `debug`), and no consumer-group rebalances attributable to the new code.
- [ ] `go vet ./libs/atlas-kafka/...` and `go vet ./services/...` clean.

### Tests

- [ ] `manager_test.go` has a test asserting one deadline expiration → one tick → no recreate. Test uses `SetFetchTimeout(50*time.Millisecond)` and `SetMaxConsecutiveTimeouts(3)`; runs for ~75ms.
- [ ] `manager_test.go` has a test asserting `maxConsecutiveTimeouts` consecutive deadline expirations → `errFetchWedged` returned → outer loop recreates reader (`recreateCount == 1`). Test uses `SetFetchTimeout(50*time.Millisecond)` and `SetMaxConsecutiveTimeouts(3)`; runs for ~200ms.
- [ ] `manager_test.go` has a test asserting alternating timeout/success/timeout/success → never recreates → counter resets across cycles.
- [ ] If the existing `KafkaReader` fake is extended, the change is documented inline and existing test cases that depend on the prior behavior are updated minimally.
- [ ] If `debug_test.go` asserts exact attribute keys, it is updated to include `lastTimeoutAt` and `consecutiveTimeouts`.

### Build

- [ ] `libs/atlas-kafka/consumer` builds.
- [ ] `go build ./...` from the monorepo root succeeds.
- [ ] Docker builds for at minimum `atlas-maps`, `atlas-monsters`, `atlas-channel` succeed against the new library.
