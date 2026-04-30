# Kafka FetchMessage Deadline + Tick-and-Escalate — Design Document

Version: v1
Status: Approved (brainstorm phase)
Created: 2026-04-30
Companion: [prd.md](./prd.md), [risks.md](./risks.md)
---

## 1. Purpose

Translate the PRD into an architecture and implementation contract that the
plan-task phase can decompose into TDD-sized steps. The PRD is highly
prescriptive — file paths, line numbers, default values, log strings, and
acceptance criteria are already locked. This design captures the
architectural shape, state-machine semantics, observability data-flow, test
scaffolding, and tradeoffs so the plan does not re-litigate them.

## 2. Architecture

The change is local to `libs/atlas-kafka/consumer`. Three layers of state,
each with a clear responsibility. Two of them already exist (task-016); only
the inner-loop layer is rewritten.

```
┌────────────────────────────────────────────────────────────────┐
│ Consumer.start  (outer reader-lifecycle loop, task-016)        │
│   for attempt := 0; ; attempt++:                               │
│     reader = rp(config)                                        │
│     onReaderCreated(attempt)                                   │
│       ├─ aliveSince = now                                      │
│       └─ if attempt > 0:                                       │
│            recreateCount++                                     │
│            lastError = ""                                      │
│            consecutiveTimeouts = 0   ← NEW                     │
│            lastTimeoutAt    = zero   ← NEW                     │
│     err = runFetchLoop(reader)                                 │
│     reader.Close()                                             │
│     if ctx cancelled: return                                   │
│     recordError(err)                                           │
│     <-time.After(backoff.next())                               │
└────────────────────────────────────────────────────────────────┘
                           ↓
┌────────────────────────────────────────────────────────────────┐
│ runFetchLoop  (inner fetch loop — REWRITTEN)                   │
│   for:                                                          │
│     check parent ctx                                            │
│     fetchCtx, cancel = context.WithTimeout(ctx, fetchTimeout)   │
│     msg, err = reader.FetchMessage(fetchCtx)                    │
│     cancel()  ← eager, NOT deferred                             │
│     branch err: see §3                                          │
└────────────────────────────────────────────────────────────────┘
                           ↓
┌────────────────────────────────────────────────────────────────┐
│ Consumer struct  (observable state — protected by c.mu)        │
│   aliveSince, lastFetchAt, lastErrorAt, lastError,             │
│   recreateCount             ← all task-016                     │
│   consecutiveTimeouts       ← NEW                              │
│   lastTimeoutAt             ← NEW                              │
│                                                                 │
│   fetchTimeout, maxConsecutiveTimeouts                         │
│     ← NEW; read-only after construction; no mutex needed       │
└────────────────────────────────────────────────────────────────┘
```

## 3. State machine — `runFetchLoop`

```
                    ┌─────────────┐
                    │ enter loop  │
                    └──────┬──────┘
                           ▼
                  ┌────────────────┐
                  │ ctx cancelled? │──yes──▶ return ctx.Err()
                  └────────┬───────┘
                          no│
                           ▼
              ┌─────────────────────────┐
              │ FetchMessage(fetchCtx)  │
              │  with deadline = fT     │
              └────────────┬────────────┘
                           │
        ┌──────────────────┼──────────────────────┐
        │                  │                      │
       err=nil       DeadlineExceeded          other err
        │           (parent still alive)          │
        │                  │                      │
        ▼                  ▼                      ▼
  recordFetch        recordTimeout          return err
  (resets counter)   counter++              (transport error,
        │           ┌───┴───┐                EOF, Canceled →
        │           │       │                outer recreate)
        │      counter      counter
        │      < max?       >= max?
        │           │       │
        │           ▼       ▼
        │       continue  log Warn,
        │       loop      return errFetchWedged
        ▼
  process(msg)
  commit
  continue loop
```

### 3.1 Branch semantics

| Branch | Test | Action | Counter | Returns? |
|---|---|---|---|---|
| Success | `err == nil` | recordFetch (resets counter, clears lastError); processMessage; commit | reset to 0 | no, continue |
| Idle tick | `errors.Is(err, DeadlineExceeded) && ctx.Err() == nil && counter+1 < max` | recordTimeout; Debug log | counter++ | no, continue |
| Wedge escalate | `errors.Is(err, DeadlineExceeded) && ctx.Err() == nil && counter+1 >= max` | recordTimeout; Warn log | counter++ | yes, `errFetchWedged` |
| Parent cancel | `ctx.Err() != nil` (any err) | none | unchanged | yes, `ctx.Err()` |
| Other error | otherwise | none | unchanged | yes, `err` |

### 3.2 Two key invariants

**Invariant 1 — eager cancel.** `cancel()` is called immediately after
`FetchMessage` returns, never via `defer`. The fetch loop is unbounded; a
deferred cancel would leak one timer per iteration until function return,
which on a healthy active consumer is functionally never. Test 2 guards
this via a `runtime.NumGoroutine()` margin check.

**Invariant 2 — counter authority.** `consecutiveTimeouts` lives on the
`Consumer` struct, not just function-local. The Snapshot endpoint reads
it between iterations, and the function-local view in `runFetchLoop` is
just a convenience mirror of `c.consecutiveTimeouts`. Source of truth is
the struct field, accessed under `c.mu`.

### 3.3 Counter reset sites (locked: Option A from brainstorm)

The counter resets only at two structurally-guaranteed sites:

1. `recordFetch()` — every successful fetch (resets to 0, clears lastError).
2. `onReaderCreated(attempt > 0)` — every reader recreate (resets to 0;
   also zeroes `lastTimeoutAt`).

`runFetchLoop` does **not** explicitly reset the counter on non-deadline
error. The structural guarantee: every non-cancel return from
`runFetchLoop` lands in the outer `start` loop, which calls
`reader.Close()`, applies backoff, and creates a new reader, which calls
`onReaderCreated(attempt > 0)`, which resets. No code is needed in
`runFetchLoop` for this.

### 3.4 Backoff non-reset on wedge recreate

The outer-loop `fetchBackoff` is created once per `start` invocation and
is never reset. A wedge → recreate → wedge → recreate cycle climbs to the
10s cap and stays there until the underlying broker condition resolves
and a successful fetch happens. This is intentional: a flapping broker
should not amortize back to fast recreates. (See risks doc R3.)

## 4. Sentinel error

Defined at package scope in `manager.go` (locked: brainstorm Q3 = A —
co-locate with sole producer; package is small):

```go
// errFetchWedged is returned from runFetchLoop when FetchMessage has hit
// its deadline maxConsecutiveTimeouts times in a row without a successful
// fetch in between. The outer start loop treats it identically to any
// other recreate-eligible error: close reader, backoff, rebuild.
var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")
```

Unexported. The outer `start` loop's existing `c.recordError(err)` writes
the sentinel's message string into `lastError`, so the debug endpoint
surfaces `"consumer fetch wedged: exceeded consecutive timeouts"`
automatically without a new code path.

## 5. Config surface

Two new decorators in `config.go`:

```go
func SetFetchTimeout(d time.Duration) model.Decorator[Config]
func SetMaxConsecutiveTimeouts(n int) model.Decorator[Config]
```

Two new fields on `Config`, with defaults set in `NewConfig`:

| Field | Default | Override decorator |
|---|---|---|
| `fetchTimeout` | `5 * time.Minute` | `SetFetchTimeout` |
| `maxConsecutiveTimeouts` | `3` | `SetMaxConsecutiveTimeouts` |

Defaults yield 15-minute worst-case detection latency. No current
monorepo consumer requires an override. Decorators stack with the
existing `SetStartOffset`, `SetMaxWait`, `SetHeaderParsers` without
ordering constraints.

`AddConsumer` copies both fields onto the resulting `Consumer`. They are
read-only after construction (no mutex needed).

## 6. Snapshot extensions

Two new fields on `Snapshot`:

```go
type Snapshot struct {
    // existing fields…
    LastTimeoutAt       time.Time
    ConsecutiveTimeouts int
}
```

`Snapshot()` copies them under the existing `c.mu`. New `recordTimeout`
method:

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

`recordFetch` modified to also reset `consecutiveTimeouts = 0` (additive
to its existing `lastFetchAt = now; lastError = ""`).

`onReaderCreated` modified: when `attempt > 0`, also reset
`consecutiveTimeouts = 0` and `lastTimeoutAt = time.Time{}` alongside the
existing `recreateCount++; lastError = ""`.

## 7. Debug-route extensions

`debug.go`'s `debugAttributes` gains two fields:

```go
LastTimeoutAt       time.Time `json:"lastTimeoutAt"`
ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
```

`snapshotToAttributes` wires them through. RFC 3339 with `Z`, consistent
with the existing `lastFetchAt` / `lastErrorAt` serialization.

### 7.1 Operator decision tree

| `consecutiveTimeouts` | `lastFetchAt` | `recreateCount` | `lastError` | Interpretation |
|---|---|---|---|---|
| 0 | recent | 0 | "" | healthy active |
| 0 | zero | 0 | "" | just started, hasn't fetched yet |
| 0 | recent | 0 | "" | healthy idle (last tick was followed by a fetch) |
| ≥1 | zero or stale | 0 | "" | wedge in progress; will escalate at max |
| 0 | varies | ≥1 | `"consumer fetch wedged…"` | wedge already detected and recovered |

The `consecutiveTimeouts > 0` row with no recent fetch is the unique
pre-escalation signature that was missing during the 2026-04-30 incident.

## 8. Component-by-component changes

| File | Change |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` | Rewrite `runFetchLoop` per §3. Add `errFetchWedged` package-level sentinel. Add 4 fields to `Consumer`: `fetchTimeout`, `maxConsecutiveTimeouts`, `consecutiveTimeouts`, `lastTimeoutAt`. Add `recordTimeout` method. Modify `recordFetch` to reset the counter. Modify `onReaderCreated` to reset the counter and timestamp on recreate. Wire `Config` fields through `AddConsumer` into `Consumer`. Extend `Snapshot` struct + `Snapshot()` method. Drop the inner `retry.Try` block. Drop `retry` import if unused after the rewrite. |
| `libs/atlas-kafka/consumer/config.go` | Add `fetchTimeout` and `maxConsecutiveTimeouts` to `Config`. Set defaults in `NewConfig`. Add `SetFetchTimeout` and `SetMaxConsecutiveTimeouts` decorators. |
| `libs/atlas-kafka/consumer/debug.go` | Add `LastTimeoutAt` and `ConsecutiveTimeouts` to `debugAttributes`. Wire in `snapshotToAttributes`. |
| `libs/atlas-kafka/consumer/manager_test.go` | (a) Test-fake fix (§9.1). (b) Three new wedge tests (§9.2). |
| `libs/atlas-kafka/consumer/debug_test.go` | Extend any attribute-key assertions to include the two new fields. |

No service-side code changes. No manifest changes. No env-var changes.
No producer changes. Every consumer-owning service inherits the new
behavior on Docker rebuild against the updated library.

## 9. Test plan

### 9.1 Test-fake correctness fix (3 fakes, 1 line each)

`MockReader`, `ChannelMockReader`, and `scriptedReader` currently return
the literal `context.Canceled` after `<-ctx.Done()`. Real kafka-go
returns `ctx.Err()`, which is `DeadlineExceeded` on timeout and
`Canceled` on cancel. The new state machine distinguishes these.

```go
// before
return kafka.Message{}, context.Canceled
// after
return kafka.Message{}, ctx.Err()
```

This is a fidelity improvement, not a behavior change for existing
tests. Existing tests only ever cancel the parent ctx (never time it
out), and `ctx.Err()` returns `Canceled` in that path — same value as
the literal.

### 9.2 New tests

All three use `SetFetchTimeout(50*time.Millisecond)` and
`SetMaxConsecutiveTimeouts(3)`; all three run in <250ms wall-clock.

**Test 1 — `TestFetchTimeoutTicksWithoutRecreate`**
- Single `scriptedReader` with empty script (always blocks on ctx).
- Wait ~75ms (> 1 timeout, < 2 timeouts).
- Assert: `Snapshot.ConsecutiveTimeouts >= 1`,
  `Snapshot.RecreateCount == 0`, `Snapshot.LastError == ""`,
  `Snapshot.LastTimeoutAt` non-zero.
- Cancel parent ctx; assert reader closed exactly once (no recreate).

**Test 2 — `TestFetchTimeoutEscalatesAfterMaxToWedge`**
- Two readers via `readerFactory`: r1 empty (always blocks), r2 delivers
  one message.
- Wait for handler invocation (signals r2 was reached → r1 was wedged →
  recreated).
- Assert: `r1.Closes() == 1`, `Snapshot.RecreateCount >= 1`,
  `Snapshot.LastError == "consumer fetch wedged: exceeded consecutive timeouts"`,
  `Snapshot.ConsecutiveTimeouts == 0` (reset by `onReaderCreated`).
- **Goroutine-leak guard** (PRD risk R2): capture
  `runtime.NumGoroutine()` before the test starts and after settling
  ~50ms post-handler-invocation; assert the delta is bounded (e.g.,
  ≤ a small constant). This catches the case where the eager `cancel()`
  fails to unblock the fake — if the assumption holds, no goroutines
  leak; if it fails, this test fires.

**Test 3 — `TestFetchTimeoutResetsOnSuccessfulFetch`**
- Single reader with custom `FetchMessage` that alternates: first call
  blocks until deadline, second call delivers a message, third blocks,
  fourth delivers, … Repeat for ~6 iterations (3 timeouts interleaved
  with 3 successes).
- Wait for 3 handler invocations.
- Assert: `Snapshot.RecreateCount == 0` (counter resets between
  successes; never reaches max), `Snapshot.ConsecutiveTimeouts == 0`
  after final success.

### 9.3 Existing tests — no scaffolding change (locked: brainstorm Q2 = A)

All existing `manager_test.go` tests finish in <1s; the 5-minute default
deadline never fires. None require `SetFetchTimeout` injection. The
test-fake `ctx.Err()` fix in §9.1 is the only existing-test change.

## 10. Tradeoffs locked from PRD + risks doc

For plan-task's reference — the implementation plan should not
re-litigate these:

| Decision | Choice | Source |
|---|---|---|
| Detection mechanism | Per-call deadline on `FetchMessage` | PRD §1, §4.1; risks R1 |
| Default `fetchTimeout` | `5 * time.Minute` | PRD §4.5; risks R6 |
| Default `maxConsecutiveTimeouts` | `3` | PRD §4.5 |
| Inner `retry.Try` | Removed | PRD §4.3 |
| Sentinel export | Unexported (`errFetchWedged`) | PRD §4.4 |
| Sentinel file location | `manager.go` | brainstorm Q3 |
| Counter reset sites | `recordFetch` + `onReaderCreated(attempt>0)` only | brainstorm Q1 |
| Backoff reset on wedge recreate | None | risks R3 |
| Tick log level | `Debug` | PRD §4.2 |
| Wedge log level | `Warn` (one-shot per wedge) | PRD §4.2 |
| Counter visibility | `Snapshot` + `/api/debug/consumers` | PRD §4.5, §5.2 |
| Process restart on wedge | None | PRD §2 non-goals |
| Per-topic overrides | Available via decorators; none used today | PRD §4.7 |
| Existing-test scaffolding | Untouched | brainstorm Q2 |
| Test-fake `ctx.Err()` fix | Applied (3 fakes, 1 line each) | brainstorm |

## 11. Out of scope

Explicit non-goals carried from PRD §2 plus deferred follow-ups:

- Producer-side outage detection.
- Per-topic cadence learning or expected-traffic modeling.
- Anything resembling task-016's explicitly-rejected staleness
  heuristics (external watchdog, wall-clock comparison).
- Prometheus metrics, Grafana dashboards, alert rules, k8s liveness
  probes.
- Process restart / pod-kill recovery.
- Operator runbook doc (risks R5 mentions one; deferred to a follow-up
  task).
- Any service code changes beyond Docker rebuilds.

## 12. Risk acknowledgment

The full risk register lives in [risks.md](./risks.md). In summary:

- **R1** — task-016 non-goal reversal: addressed by structural
  distinction (idle vs. wedge falls out of the loop shape, not a
  heuristic).
- **R2** — kafka-go `FetchMessage` ctx-cancellation behavior: covered
  by the goroutine-leak guard in Test 2; if the assumption fails, the
  guard fires and the design needs re-discussion.
- **R3** — reader-recreate storm on flapping broker: bounded by the
  existing 10s outer-backoff cap; no change required.
- **R4** — test-fake behavior cascading: addressed by the audit in
  §9.3; no scaffolding changes needed beyond the §9.1 fidelity fix.
- **R5** — operator misinterpretation of `consecutiveTimeouts > 0` on
  idle: deferred to a follow-up runbook doc; the §7.1 decision tree
  serves as the inline reference for now.
- **R6** — default-timeout collision with future low-cadence topics:
  none today; override site documented in PRD §4.7.
