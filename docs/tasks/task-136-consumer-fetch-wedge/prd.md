# Kafka Consumer Fetch-Wedge — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-08
---

## 1. Overview

Every Atlas service consumes Kafka through the shared `libs/atlas-kafka`
consumer. Each consumer runs a fetch loop (`Consumer.runFetchLoopSerial` /
`runFetchLoopParallel` in `libs/atlas-kafka/consumer/manager.go`) that calls
`reader.FetchMessage(ctx)` under a per-call deadline `fetchTimeout` (default
**5 minutes**, `config.go`). Each `context.DeadlineExceeded` increments
`consecutiveTimeouts`; after `maxConsecutiveTimeouts` (default **3**) in a row
the loop logs `FetchMessage wedged: N consecutive timeouts … forcing reader
recreate`, returns `errFetchWedged`, and the outer loop recreates the reader
under an exponential backoff capped at 10s. A successful fetch resets the
counter.

On an idle topic this ticking is benign — no traffic for
`3 × fetchTimeout` naturally trips the recreate. The problem is the observed
**dwell time**: during task-102 live marketplace testing, a saga's
`award_currency_seller` command sat unconsumed for ~55 seconds on a
single-partition wallet topic while the same service processed the preceding
`award_currency_buyer` in ~184 ms. Cluster-wide, the fetch-wedge/recreate
signal fired ~20 times across the orchestrator, cashshop, and inventory
consumers in a 25-minute window. Earlier the same class of stall crash-looped
atlas-world/channel/character-factory (their config-projection catch-up
gate never received events in time).

Crucially, the ≤10s recreate backoff does not account for a ~55s dwell, so the
recreate window is not the (whole) cause — the real dwell-time source is
unknown and must be found. Candidates: the 5-minute idle `fetchTimeout`
interacting with kafka-go's group-consumer session/rebalance on reader
recreation; a single broker (`kafka-broker-0`, 1 replica, ~481 partitions
across env-suffixed topics, ~1.1 core) whose per-partition fetch latency grows
under fan-out; consumer-group rebalances stalling assignment; or per-partition
head-of-line blocking. The impact is real: slow saga steps, combined with the
saga terminal-state race (task-135), produced free-item marketplace purchases.

This task roots out the dwell-time cause and fixes it inside `libs/atlas-kafka`,
with a deterministic reproduction as the acceptance gate. Broker topology
changes (multi-broker, partition reduction) are called out as a separable
cluster-infra follow-up, not implemented here.

## 2. Goals

Primary goals:

- **Reproduce the dwell deterministically** in a test harness (testcontainers
  Kafka) that models the observed conditions — many idle topics plus an active
  topic, concurrent group consumers, single broker — and measures end-to-end
  publish→consume latency, so a fix can be proven rather than inferred from
  live logs.
- **Identify the root cause** of the multi-second-to-minute dwell between a
  message being produced and the shared consumer delivering it to a handler,
  distinguishing the contributing factors (idle-`fetchTimeout` interaction,
  reader-recreate/rebalance cost, broker fetch latency under partition fan-out,
  head-of-line blocking).
- **Fix the cause(s) that live in `libs/atlas-kafka`** — fetch-loop and Reader
  configuration and/or logic — so a produced message is delivered to its
  handler within a low, bounded latency under the reproduced conditions, and
  the wedge/recreate cycle no longer inflates delivery time on active topics.
- **Distinguish "idle tick" from "stuck"** so the benign no-traffic timeout is
  no longer logged/counted as a wedge, and a genuine stall is observable and
  alertable.

Non-goals:

- The saga terminal-state race (task-135) — that is the correctness net for the
  free-item outcome and is tracked separately.
- Replacing `segmentio/kafka-go` with a different client.
- Broker scaling / multi-broker / partition-count reduction / topic-topology
  redesign — **recommended** as a cluster-infra follow-up (§9), not implemented
  here. This task may reduce the broker's effective load only via consumer-side
  changes (e.g. fetch batching), not by changing the broker deployment.
- Any per-service consumer code change — the fix is in the shared library.

## 3. User Stories

- As a **service author**, I want a message produced to a topic my service
  consumes to be handled within a low, bounded latency regardless of how many
  other (idle) topics exist, so my sagas and event flows complete promptly.
- As an **operator**, I want the "wedge" signal to mean a real stall, not a
  routine idle tick, so the metric/log is actionable and not noise.
- As a **marketplace user**, I want a purchase saga to complete in seconds, not
  tens of seconds, so the client isn't left waiting (and the timeout race
  doesn't fire).
- As a **platform engineer**, I want a deterministic reproduction of the dwell
  so any future regression in consumer latency is caught by a test, not by a
  production incident.

## 4. Functional Requirements

### 4.1 Reproduction harness

- A test (build-tagged `integration`, consistent with the existing
  `libs/atlas-kafka/consumer/*_test.go` testcontainers pattern) MUST reproduce
  the dwell: spin up a single-broker Kafka, create N idle topics + 1 active
  topic (N chosen to approximate the live fan-out, documented), start group
  consumers via the shared `Consumer` on all of them, then publish to the
  active topic and assert the publish→handler latency.
- The harness MUST record and expose the publish→handler latency so before/after
  numbers are comparable, and MUST fail if that latency exceeds an agreed
  threshold (the target from §8) once the fix is in place.
- The harness MUST also exercise the reader-recreate path (force a wedge) and
  assert delivery latency across a recreate is bounded.

### 4.2 Root-cause instrumentation

- Add timing instrumentation (behind the existing `Snapshot`/debug surface or
  temporary during investigation) that attributes dwell to a phase: time in
  `FetchMessage`, time in reader recreation/backoff, time in group
  join/rebalance, time in handler dispatch. The investigation MUST produce a
  written root-cause finding (a `findings.md` in the task folder) that names the
  dominant contributor with evidence from the harness.

### 4.3 Idle-vs-stuck distinction

- A no-traffic fetch deadline on a healthy consumer MUST NOT be logged or
  metered as a wedge. Reserve the wedge signal (and reader recreate) for a
  consumer that is actually failing to make progress, not one that is merely
  idle.
- If reader recreation remains a defensive mechanism, its trigger MUST be
  distinguishable in logs/metrics from routine idleness, and its cost (the
  ≤10s backoff + any rebalance) MUST NOT be paid while messages are actually
  available.

### 4.4 The fix

- Change `libs/atlas-kafka/consumer` configuration and/or fetch-loop logic so
  that, under the reproduced conditions, a produced message reaches its handler
  within the §8 latency target, with no dependence on the wedge/recreate cycle
  for active topics.
- Candidate levers to evaluate (design decides which apply): `fetchTimeout` /
  `MaxWait` values and their interaction; `MinBytes`/`MaxBytes`/`QueueCapacity`
  Reader tuning; whether the per-call `FetchMessage` deadline is the right model
  vs a long-lived fetch with a liveness check; group session/heartbeat/rebalance
  timeouts; and whether the recreate-on-idle behavior should exist at all.
- Any config change MUST preserve the existing decorator API
  (`SetMaxWait`, `SetFetchTimeout`, `SetMaxConsecutiveTimeouts`, etc.) and keep
  per-consumer overrides working; new defaults are chosen with documented
  rationale.

### 4.5 No behavioral regressions

- At-least-once delivery, in-order commit (serial loop), and the parallel loop's
  prefix-commit cursor semantics MUST be unchanged. Existing
  `libs/atlas-kafka/consumer` tests (unit + integration) MUST pass. The redis
  key-guard and other repo invariants MUST remain clean.

## 5. API Surface

No REST or Kafka wire-contract changes. Internal library surface:

- The `Config` decorator API is preserved; default values may change (documented).
- The `Snapshot` struct may gain fields for the idle-vs-stuck distinction and
  phase timing (additive; the `/debug/consumers` route continues to serialize
  it).
- No new external endpoints.

## 6. Data Model

No persistent data model changes. All state is the in-memory per-`Consumer`
observable fields (`aliveSince`, `lastFetchAt`, `consecutiveTimeouts`,
`recreateCount`, …) plus any additive phase-timing fields. No `tenant_id`
scoping applies (this is the transport layer, below tenant context).

## 7. Service Impact

- **`libs/atlas-kafka/consumer`** — the only code that changes: fetch-loop
  configuration/logic, the wedge-vs-idle distinction, instrumentation, and tests.
- **Every Atlas service** — no source change; each inherits the improved
  consumer behavior on its next build. Because the module is shared, the change
  requires rebuilding and re-baking all Go services that vendor it (the
  standard shared-lib bump — CLAUDE.md build/bake discipline applies).
- **`deploy/k8s` kafka manifests** — untouched by this task (topology changes
  are the §9 follow-up), but the findings should quantify how much of the dwell
  is broker-side to justify that follow-up.

## 8. Non-Functional Requirements

- **Latency target (primary, testable):** under the reproduced conditions, a
  message produced to an actively-consumed topic is delivered to its handler in
  **under ~1 second** (design confirms the exact number and the fan-out the
  harness models). The ~55s and even multi-second dwells observed must not
  recur in the harness.
- **Determinism:** the fix is validated by the harness (§4.1), not by live-log
  observation alone.
- **Observability:** wedge/idle/recreate are cleanly separable in logs and in
  the `Snapshot`/debug surface; a real stall is alertable.
- **Correctness preserved:** at-least-once + commit ordering unchanged (§4.5).
- **Resource:** the fix must not materially increase broker load (e.g. tight
  polling) — if fetch batching or wait tuning is used, quantify the broker-side
  effect in the findings.
- **Backward compatibility:** existing per-consumer decorator overrides keep
  working; default changes are documented with rationale.

## 9. Open Questions

- How much of the ~55s dwell is broker-side (single broker, ~481 partitions,
  ~1.1 core) vs client-side (fetch-loop/rebalance)? The findings must quantify
  this to decide whether the cluster-infra broker follow-up is required or the
  library fix alone suffices. If broker-side dominates, file the follow-up:
  multi-broker and/or reducing per-env topic multiplication (~150 topics × 3
  env suffixes on one broker).
- Does the serial fetch loop's per-call 5-minute `FetchMessage` deadline
  interact badly with kafka-go's consumer-group session management (does a
  deadline-cancel drop the group session and force a rejoin/rebalance, whose
  cost is the real dwell)? This is the leading hypothesis to test first.
- Are any services using the parallel loop, and does it exhibit the same dwell,
  or is the issue specific to the serial loop?
- Is the observed dwell correlated with consumer-group rebalances triggered by
  many consumers (one per topic per service) sharing few group coordinators?

## 10. Acceptance Criteria

- [ ] A build-tagged integration test reproduces the pre-fix dwell (asserts a
      high publish→handler latency under the modeled fan-out) and, with the fix,
      asserts the latency is under the §8 target.
- [ ] `findings.md` in the task folder names the dominant root cause with
      evidence from the harness (phase-timing attribution), and states how much
      is client-side vs broker-side.
- [ ] Routine idle fetch-timeouts on a healthy consumer are no longer logged or
      metered as "wedged"; a genuine stall still is, and the two are separable
      in the `Snapshot`/debug surface.
- [ ] Under the reproduced conditions, a produced message reaches its handler
      within the §8 latency target with no dependence on the wedge/recreate
      cycle for active topics.
- [ ] At-least-once delivery and commit-ordering semantics are unchanged;
      existing `libs/atlas-kafka/consumer` unit + integration tests pass.
- [ ] The `Config` decorator API is preserved; any default changes are
      documented with rationale in the config file and findings.
- [ ] `go build`, `go vet`, `go test ./...` (and the `integration`-tagged suite)
      clean in `libs/atlas-kafka`; `tools/redis-key-guard.sh` clean.
- [ ] If the findings show broker topology is a material contributor, a
      cluster-infra follow-up task is filed (multi-broker / partition reduction)
      with the quantified justification — but this task does not change the
      broker deployment.
