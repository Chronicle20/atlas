# Kafka Consumer Fetch-Wedge — Design

Task: task-136-consumer-fetch-wedge
Status: Proposed
PRD: `docs/tasks/task-136-consumer-fetch-wedge/prd.md`

---

## 1. Verified Facts (grounding for every hypothesis below)

All facts below were read from source in this worktree or from the pinned
`segmentio/kafka-go v0.4.51` module (the version in `libs/atlas-kafka/go.mod`).
Nothing in this section is inferred.

**Atlas shared consumer (`libs/atlas-kafka/consumer`):**

- F1 — Defaults: `maxWait = 50ms`, `fetchTimeout = 5m`,
  `maxConsecutiveTimeouts = 3` (`config.go:17-20`). `maxWait` is passed
  straight into `kafka.ReaderConfig.MaxWait` (`manager.go:120`).
- F2 — The serial loop wraps each `FetchMessage` in
  `context.WithTimeout(ctx, fetchTimeout)`; on `DeadlineExceeded` it counts a
  timeout, and at `maxConsecutiveTimeouts` returns `errFetchWedged`
  (`manager.go:396-415`). The outer `start` loop then **closes the reader**,
  backs off (500ms→10s cap), and recreates it (`manager.go:350-367`).
- F3 — One `Consumer` (one `kafka.Reader`) per topic; **every consumer in a
  service shares one `GroupID`** — e.g. atlas-saga-orchestrator registers 15
  consumers, all with `consumergroup.Resolve("Saga Orchestrator Service")`
  (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go:38,90-104`).
- F4 — **No service overrides any consumer decorator**: zero non-test call
  sites for `SetFetchTimeout`, `SetMaxConsecutiveTimeouts`, `SetMaxWait`,
  `SetMaxInFlight` across `services/`. Defaults rule everywhere, and the
  parallel loop (`maxInFlight > 1`) is **unused in production** — this answers
  PRD §9 Q3.
- F5 — The wedge machinery was added 2026-04-30
  (`a8244ff3de`, `4c99fe36de`, `e238596734`) as a defense against readers going
  silently dead; its recovery value for genuine stalls must be preserved.

**kafka-go v0.4.51 (module cache, cited by file:line in the pinned module):**

- F6 — `Reader.FetchMessage(ctx)` merely `select`s on `ctx.Done()` vs the
  internal message channel (`reader.go:815-838`). **A per-call deadline
  expiring does NOT touch the group session** — the background run loop and
  heartbeats continue. PRD §9's leading hypothesis ("deadline-cancel drops the
  group session and forces a rejoin") is **refuted at the FetchMessage level**.
  The damage is what our wrapper does next (F2): `Close()` + recreate.
- F7 — `Reader.Close()` cancels the run loop and the consumer-group member
  sends `LeaveGroup` to the coordinator (`reader.go:757-771`,
  `consumergroup.go:737,753,1205-1230`). A member leaving triggers a
  **group-wide rebalance**: every member of that GroupID — including the
  service's *active*-topic readers — has its generation ended and must
  rejoin/resync before fetching again.
- F8 — Group-protocol timing defaults: `HeartbeatInterval = 3s`,
  `SessionTimeout = 30s`, `RebalanceTimeout = 30s`, `JoinGroupBackoff = 5s`
  (`consumergroup.go:39-51`). None are set by Atlas, so these govern how long
  a rebalance stalls the group.
- F9 — kafka-go's **own** `MaxWait` default is **10s** (`reader.go:655-657`);
  `MinBytes` defaults to 1 (`reader.go:31`). With `MinBytes=1` the broker
  answers a fetch **immediately when data exists**; `MaxWait` only bounds how
  long the broker parks an *empty* long-poll. Atlas's 50ms override therefore
  buys zero delivery latency and costs a 200× higher idle fetch-request rate
  vs the library default.
- F10 — `Reader.Stats()` returns counter deltas since the previous call
  (`reader.go:1089-1096`): `Dials`, `Fetches`, `Messages`, `Errors`,
  `Rebalances`, `Timeouts`, plus `Lag`/`QueueLength` gauges
  (`reader.go:575-596`). No Atlas code calls `Stats()` today (the reader is
  constructed privately inside the lib), so the lib may consume the deltas.

## 2. Causal Model — hypotheses the harness must confirm or kill

The dwell is modeled as three independent contributors. The harness (§4)
measures each in isolation; `findings.md` attributes the observed dwell.

**H1 (leading): wedge-recreate churn causes group-wide rebalance storms.**
Chain: idle topic → 3 × 5m deadlines → `errFetchWedged` → `Close()`
(LeaveGroup, rebalance #1) → backoff → new reader joins (rebalance #2).
A service with N mostly-idle topics in ONE group (F3) generates
`2N rebalances / 15min`; for N=15 that is a group-wide rebalance every ~30s
on average once wedge phases de-synchronize. Each rebalance suspends *all*
members (F7) for the rejoin/sync window (multi-second under F8's
JoinGroupBackoff=5s, worse when joins queue behind each other). Overlapping
recreates can chain rebalances back-to-back, so an active topic's member
spends most of a window without a valid generation — a ~55s dwell for one
message is consistent with 2-5 chained rebalances. The observed "~20
wedge/recreate lines across 3 services in 25min" matches the cadence math.

**H2: 50ms MaxWait idle-spin loads the single broker.**
Every idle reader long-polls and gets an empty response every ~50ms (F1, F9).
Cluster-wide (hundreds of readers × ~20 req/s each, ~481 partitions, 1 broker,
~1.1 core) this is thousands of fetch requests/sec of pure idle overhead,
inflating fetch-path latency for the readers that *do* have data. H2 is a
latency *amplifier* rather than the 55s source.

**H3 (refuted in source, kept as a harness control): per-call deadline
drops the group session.** F6 shows it does not. The harness still runs a
control (deadline ticks with recreate disabled, no churn) to demonstrate ~0
added dwell, closing PRD §9 Q2 with evidence rather than only a code citation.

**H4: head-of-line blocking in the serial loop.** A slow handler delays the
next fetch on the same topic, but cross-topic delivery is independent (one
reader each). The observed dwell was cross-message on a single-partition
topic whose *previous* message completed in 184ms, so H4 cannot explain it;
the harness's phase attribution will show handler-dispatch time as negligible.

## 3. Approaches Considered

### Approach A (chosen): stop recreating on idle; detect real stalls by progress, not silence; align MaxWait with the library default

Keep the one-reader-per-topic / one-group-per-service topology. Change the
loop's *interpretation* of a deadline tick:

- An expired fetch deadline is an **idle tick**, never a wedge. No warn log,
  no recreate, no LeaveGroup. With the churn generator gone, group membership
  is stable in steady state, so rebalances happen only on real membership
  change (deploys, crashes) — H1's cause is removed at the source.
- A **genuine stall** is detected by *absence of reader progress*, not by
  absence of messages: at each idle tick, consume `reader.Stats()` deltas
  (F10). A healthy idle reader still issues fetch attempts every MaxWait
  (`Fetches` delta > 0). A dead reader shows `Fetches == 0 && Dials == 0`
  across a full tick. Only no-progress ticks count toward the wedge
  threshold; `errFetchWedged` + recreate is retained for that case (F5's
  defensive value survives, now correctly targeted).
- Raise the default `maxWait` from 50ms to **10s** (kafka-go's own default,
  F9). Zero delivery-latency cost with `MinBytes=1`; ~200× fewer idle fetch
  requests against the 1.1-core broker (H2). Per-consumer `SetMaxWait`
  override keeps working.
- Shorten the default `fetchTimeout` from 5m to **1m**: it is now a cheap
  liveness-check cadence, not a recreate trigger, and 1m ticks detect a real
  stall in ~3m instead of ~15m. `SetFetchTimeout` / `SetMaxConsecutiveTimeouts`
  keep their signatures; their documented meaning becomes "liveness tick
  interval" and "no-progress ticks before recreate".

Pros: removes the churn driver H1 outright; fixes H2 without touching the
broker; zero API break; zero commit-semantics change; smallest blast radius.
Cons: rebalance cost on *real* membership change (deploys) remains — accepted,
that is inherent to the group protocol and rare.

### Approach B (rejected): per-topic group IDs (`groupId + "." + topic`)

Decouples every topic's membership, so no cross-topic rebalance coupling ever.
Rejected because a new group name has no committed offsets: with the default
`StartOffset = FirstOffset` (`config.go:18`), every consumer in the fleet
would **replay its topic's full retention** on the first deploy — for command
topics (wallet, saga) that is a correctness incident, and flipping to
`LastOffset` instead risks *dropping* messages produced during the deploy
window. A fleet-wide offset migration is far more dangerous than the bug.
Also unnecessary once A removes the churn source.

### Approach C (rejected for this task, recorded as follow-up): one multi-topic reader per service via `ReaderConfig.GroupTopics`

One group member per pod, one fetch session covering all the service's topics
— the strongest possible reduction of both rebalance surface and broker
fetch-session count. Rejected here because it rewrites the lib's core
topology (per-topic `Consumer`, per-topic handlers/Snapshot/maxInFlight would
all need a demux layer), which violates this task's "smallest change that
fixes the measured cause" bar and multiplies regression risk on the one
library every service vendors. If the findings show broker-side load still
matters after A, Approach C is the natural library-side half of the §9
cluster-infra follow-up.

## 4. Component Design

### 4.1 Reproduction & attribution harness

New file `libs/atlas-kafka/consumer/dwell_integration_test.go`, build-tagged
`integration` like the existing testcontainers tests (offsets pattern in
`libs/atlas-kafka/consumer/offsets_test.go`).

Topology (documented approximation of live): single-broker testcontainers
Kafka; **one consumer group with 15 idle topics + 1 active topic** (matches
atlas-saga-orchestrator's real fan-out, F3); a second group with a handful of
consumers to model cross-service coordinator sharing (PRD §9 Q4). Production
defaults except where a scenario says otherwise. The handler on the active
topic records receive-time; the publisher stamps send-time in the message —
publish→handler latency is measured end-to-end per message.

Scenarios (each is one test; thresholds asserted post-fix):

| # | Scenario | Models | Assertion (post-fix) |
|---|----------|--------|----------------------|
| S1 | Steady state: all 16 consumers up, publish M messages at intervals to the active topic | healthy live traffic | p99 publish→handler **< 1s** (PRD §8 target) |
| S2 | Churn: a "churn generator" closes and recreates one idle-topic reader every few seconds (simulating the legacy wedge cadence, compressed) while publishing to the active topic | H1 — pre-fix behavior reproduced deterministically without waiting 15m | pre-fix-equivalent config shows multi-second dwell (recorded in findings); post-fix steady loop (no self-churn) never enters this state on its own |
| S3 | Single forced recreate of the *active* topic's reader mid-stream (genuine-stall path) | recreate cost is bounded | delivery resumes; max dwell across the recreate **≤ 10s** (join + backoff budget from F8; exact bound confirmed from S3 measurements before being pinned in the test) |
| S4 | Control: deadline ticks at 2s cadence, recreate disabled, no churn | H3 refutation | p99 within S1's bound — ticks alone add no dwell |
| S5 | MaxWait A/B: idle fleet at `maxWait=50ms` vs `10s`, measure summed `Stats().Fetches` deltas per minute and S1 latency in both | H2 quantification | latency parity; fetch-request rate reduction reported in findings (extrapolated to 481 live partitions) |

S2 pre-fix numbers are captured once during investigation and written into
`findings.md`; the *committed* tests assert the post-fix bounds so the suite
stays green while still failing on regression (a reintroduced churn source
would break S1/S3).

### 4.2 Phase-timing instrumentation

Attribution counters on `Consumer`, surfaced through `Snapshot` (additive —
PRD §5): per-loop-iteration time spent in `FetchMessage` (last + max),
time from reader creation to first successful fetch (join/assignment cost),
cumulative time in recreate backoff, and handler-dispatch duration
(last + max). These are cheap monotonic-clock deltas around existing call
sites in `runFetchLoopSerial` / `runFetchLoopParallel` / `start`; no new
goroutines. The harness reads them via `Snapshot()` to attribute each
scenario's dwell to a phase; `findings.md` quotes them.

### 4.3 Idle-vs-stuck detection

- New optional interface in `manager.go`:

  ```go
  type StatsProvider interface{ Stats() kafka.ReaderStats }
  ```

  The fetch loop type-asserts the `KafkaReader` against it. `*kafka.Reader`
  satisfies it natively; existing test mocks that don't implement it fall back
  to legacy counting (every deadline tick counts toward the threshold), so no
  existing unit test breaks and mock-driven wedge tests keep working.
- On `DeadlineExceeded` with a `StatsProvider`: read `Stats()` (delta since
  last tick, F10 — safe because the lib owns the reader exclusively, F10).
  `delta.Fetches > 0 || delta.Dials > 0 || delta.Messages > 0` ⇒ **idle
  tick**: debug log, increment a new `idleTicks` counter, reset the
  no-progress count. Otherwise ⇒ **no-progress tick**: warn log naming it a
  stall-suspect, increment `noProgressTicks`; at `maxConsecutiveTimeouts`
  no-progress ticks return `errFetchWedged` (recreate path unchanged from
  there). The exact predicate is validated in S3/S4 and recorded in findings;
  if the harness shows a stall mode that still issues fetch attempts
  (e.g. error loops), non-deadline errors already exit the loop today
  (`manager.go:414`) and continue to.
- `Snapshot`/debug additions (additive to `Snapshot` and
  `debugAttributes` in `debug.go`): `IdleTicks`, `LastIdleTickAt`,
  `NoProgressTicks`, `LastNoProgressAt`, plus §4.2's phase timings.
  `ConsecutiveTimeouts`/`LastTimeoutAt` remain but now count only
  no-progress ticks — the wedge signal becomes alertable truth (PRD §4.3):
  `NoProgressTicks > 0` or `RecreateCount` rising is actionable;
  `IdleTicks` rising is noise-free normal.

### 4.4 Config default changes (rationale documented in `config.go`)

| Knob | Old | New | Rationale |
|------|-----|-----|-----------|
| `maxWait` | 50ms | 10s | F9: no latency cost with MinBytes=1; ~200× less idle broker load (H2); aligns with kafka-go default |
| `fetchTimeout` | 5m | 1m | now a liveness tick, not a recreate trigger; 3× faster genuine-stall detection at negligible cost |
| `maxConsecutiveTimeouts` | 3 | 3 (meaning refined) | counts **no-progress** ticks only |

All decorators keep their exact signatures and continue to override the
defaults (PRD §4.4). The parallel loop receives the identical idle-vs-stuck
change (same helper), keeping the two loops semantically aligned even though
production doesn't use it yet (F4).

### 4.5 What does NOT change

Commit semantics (serial in-order commit, parallel prefix-commit cursor),
at-least-once delivery, handler dispatch, header parsing, otel spans, the
manager/registration API, and the recreate-with-backoff mechanism for real
errors (`io.EOF`, broker errors) — untouched (PRD §4.5). No broker manifest
changes (PRD §2 non-goal).

## 5. Error Handling

- Genuine stall (no reader progress): unchanged escalation — recreate with
  capped backoff, `recordError`, `RecreateCount++` — but now only when truly
  stuck, and logged distinctly from idleness.
- Non-deadline fetch errors: unchanged (return from loop → recreate).
- Handler failure/panic: unchanged (`processMessage`/`safeHandle`).
- `Stats()` absent (mock readers): legacy per-tick counting — conservative,
  never less safe than today.

## 6. Testing Strategy

1. **Unit (no tag):** mock-reader tests for the new tick classification —
   idle tick resets the counter, no-progress ticks escalate to
   `errFetchWedged`, non-StatsProvider mocks keep legacy behavior; Snapshot
   field population; config default values and decorator overrides.
   Existing `manager_test.go` / `debug_test.go` / `offsets_test.go` must pass
   unmodified except where they assert the old idle-wedge cadence (those
   update to the new semantics deliberately, called out in the plan).
2. **Integration (`integration` tag):** the §4.1 scenario suite S1-S5.
3. **Repo invariants:** `go test -race ./...`, `go vet ./...` in
   `libs/atlas-kafka`; `tools/redis-key-guard.sh`; `docker buildx bake` for
   every Go service (shared-lib bump — CLAUDE.md build discipline).
4. **Deliverable:** `docs/tasks/task-136-consumer-fetch-wedge/findings.md`
   with the S1-S5 numbers, phase attribution, the client-vs-broker split, and
   the follow-up decision (§7).

## 7. Broker-Side Quantification & Follow-Up Gate (PRD §9 Q1)

The harness cannot replicate 481 partitions on 1.1 cores in CI, so the
findings quantify broker-side contribution as: (a) S5's measured idle
fetch-request rate at 50ms vs 10s, extrapolated to the live topic count;
(b) S1 latency delta between the two settings under the modeled fan-out.
Decision rule: if post-fix live observation (wedge logs gone, saga dwell
< 1s) still shows multi-second dwells, file the cluster-infra follow-up
(multi-broker / per-env topic reduction / Approach C) citing those numbers;
otherwise record "library fix sufficient" in findings and close §9 Q1.

## 8. Risks

- **R1: the 55s dwell has a contributor outside H1/H2.** Mitigation: the
  harness attributes phases (§4.2) before the fix is finalized; if S2 fails
  to reproduce multi-second dwell via churn, the investigation widens (e.g.
  coordinator contention, S5 second group) *before* committing to defaults.
  The fix ships only with a reproduced-then-eliminated dwell (PRD §2).
- **R2: removing recreate-on-idle un-fixes whatever motivated it (F5).**
  Mitigation: recreate survives for no-progress stalls and hard errors; S3
  proves the stall path still recovers.
- **R3: 10s MaxWait slows something unforeseen** (e.g. a code path that
  relied on frequent empty fetch returns). Mitigation: S1/S4/S5 latency
  assertions; ctx cancellation aborts `FetchMessage` immediately regardless
  of MaxWait (F6), so shutdown latency is unaffected.
- **R4: `Stats()` delta consumption conflicts with future metrics export.**
  Documented on the `StatsProvider` assertion: the lib owns the reader's
  stats stream; any future exporter must read the lib's Snapshot instead.

## 9. Acceptance Mapping (PRD §10)

| PRD criterion | Design element |
|---|---|
| Repro test, pre-fix dwell, post-fix < target | §4.1 S1-S3 |
| findings.md with dominant cause + client/broker split | §4.2, §6.4, §7 |
| Idle no longer logged/metered as wedge; stall separable | §4.3 |
| Latency target w/o wedge dependence | §3-A, §4.1 S1 |
| Semantics unchanged; existing tests pass | §4.5, §6.1 |
| Decorator API preserved; defaults documented | §4.4 |
| Build/vet/test/key-guard clean | §6.3 |
| Broker follow-up filed if material | §7 |
