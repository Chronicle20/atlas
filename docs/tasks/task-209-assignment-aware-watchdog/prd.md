# Assignment-Aware Consumer Watchdog — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-10

---

## 1. Overview

Every Kafka consumer in Atlas is driven by `libs/atlas-kafka/consumer`, which wraps
`*kafka.Reader` and runs a liveness watchdog: if `FetchMessage` returns nothing for
`maxConsecutiveTimeouts` (3) consecutive `fetchTimeout` (1m) ticks, the reader is
declared wedged and recreated. The watchdog decides "is this reader alive?" from
`readerMadeProgress` (`libs/atlas-kafka/consumer/manager.go:65`), which inspects
`Stats().Fetches | Dials | Messages`.

That proxy cannot distinguish a **stalled** reader from an **unassigned** one. A
consumer-group member that holds no partition assignment issues no fetches at all,
so it is permanently classified as wedged. Every topic in `atlas-main` is
single-partition (237 of them) while services run `replicas: 2`, so in every group
exactly one member is always unassigned — and recreates itself every ~3 minutes,
forever. Each recreate rejoins the group and forces a rebalance.

The blast radius is larger than one topic, because a consumer group spans every
topic a service consumes. `Monster Registry Service` covers `COMMAND_TOPIC_MONSTER`,
`COMMAND_TOPIC_MONSTER_MOVEMENT`, `EVENT_TOPIC_CHARACTER_BUFF_STATUS` and
`EVENT_TOPIC_MAP_STATUS` with 9 registered members for 2 pods. A rejoin triggered by
an unassigned reader on a *cold* topic therefore pauses consumption on the *hot*
gameplay topics in the same group.

This is observable in production. Trace `bd9b801a…` on 2026-08-10: atlas-channel
emitted a damage command at 13:46:28.36; `atlas-monsters` logged
`FetchMessage wedged … [COMMAND_TOPIC_MONSTER-main] (group [Monster Registry Service]);
forcing reader recreate` at 13:46:33.455; the kill event surfaced at 13:46:33.3 —
a **4.7 s** stall on the attack→death path, followed by a second **4.2 s** stall
before atlas-quest credited the kill (9.4 s end to end). Consumer lag measured
immediately afterwards was 0 on every monster-status group, confirming a transient
rebalance stall rather than a backlog. In one 60-second window, wedge/recreate
events were observed across ~25 services (recreate attempt counters at 442, 447,
449, 469).

Four prior tasks have tuned this watchdog — `task-016-kafka-consumer-selfheal`,
`task-039-kafka-fetch-deadline`, `task-136-consumer-fetch-wedge`, and task-208's
Part 3 — each compensating for the same missing fact: **the consumer cannot observe
its own partition assignment**. `*kafka.Reader` exposes no assignment accessor, and
`ReaderStats.Partition` is a constant stamped at construction (`reader.go:717`), not
a live assignment. This task removes the gap at its source by migrating the consumer
onto kafka-go's lower-level `ConsumerGroup` API, where `Generation.Assignments`
(`consumergroup.go:321`) makes the assignment directly observable.

## 2. Goals

Primary goals:

- Migrate `libs/atlas-kafka/consumer` from `*kafka.Reader` to `kafka.ConsumerGroup` +
  `Generation`, so partition assignment is a first-class, directly-read input.
- Never recreate a consumer that holds zero partition assignments; classify it as
  healthy-idle.
- Preserve genuine wedge recovery: an *assigned* reader making no progress must still
  be recovered within a bounded window.
- Preserve the public API of `libs/atlas-kafka/consumer` so no service source changes.
- Reduce steady-state consumer-group rebalance rate on an idle `atlas-main` to
  approximately zero.

Non-goals:

- Producer-side latency (`BatchTimeout: 50ms`, the one-message-at-a-time loop in
  `libs/atlas-kafka/producer/producer.go:61`). Tracked separately.
- Changing topic partition counts, the producer `Balancer`, or service replica counts.
- Splitting consumer groups per topic to reduce rebalance blast radius (§9).
- Zombie group-member reaping / session-timeout tuning (§9).
- Command idempotency — task-208 Parts 1–2 own that and remain necessary regardless.

## 3. User Stories

- As a **player**, I want monster damage, death and drops to resolve without
  multi-second freezes, so combat feels responsive.
- As an **operator**, I want consumer recreates to indicate a real fault, so
  `Recreated reader for topic (attempt N)` is a signal rather than background noise.
- As a **service developer**, I want to add a consumer without knowing anything about
  partition assignment, so the existing `AddConsumer`/`RegisterHandler` API keeps working
  unchanged.
- As an **operator**, I want to see a consumer's current partition assignment and
  generation in its debug snapshot, so I can tell "idle" from "stuck" without reading
  broker state by hand.

## 4. Functional Requirements

### FR-1 — ConsumerGroup-backed fetch loop

- FR-1.1 The consumer MUST obtain its group membership via `kafka.NewConsumerGroup`
  and consume generations via `ConsumerGroup.Next(ctx)`.
- FR-1.2 For each generation, the consumer MUST read its own assignment from
  `Generation.Assignments[topic]` and start one partition reader per assigned
  partition, torn down when the generation ends.
- FR-1.3 Offsets MUST be committed via `Generation.CommitOffsets`, preserving current
  at-least-once semantics. Commit failures MUST be logged and MUST NOT silently
  advance the cursor.
- FR-1.4 `startOffset` (default `kafka.FirstOffset`) MUST be honoured for partitions
  with no committed offset.
- FR-1.5 The `SetMaxInFlight` prefix-commit cursor MUST be preserved: with
  `maxInFlight > 1`, only the highest *contiguously*-completed offset is committed,
  per partition. Default remains 1 (serial).
- FR-1.6 Generation transitions MUST NOT drop in-flight messages without either
  completing or leaving them uncommitted for redelivery.

### FR-2 — Assignment-aware liveness

- FR-2.1 A consumer whose current generation assigns it **zero partitions** for its
  topic MUST be classified `healthy-idle`. It MUST NOT increment no-progress ticks
  and MUST NOT be recreated.
- FR-2.2 An unassigned consumer MUST remain a live group member (it must keep
  heartbeating and stay eligible for future assignment) — it must not leave the group.
- FR-2.3 A consumer holding ≥1 partition that makes no progress for
  `maxConsecutiveTimeouts` consecutive `fetchTimeout` ticks MUST still be recovered,
  preserving the task-136 behaviour.
- FR-2.4 On transition from unassigned to assigned, no-progress state MUST reset.
- FR-2.5 A consumer MUST NOT log `stall suspect` or `wedged` warnings while unassigned.

### FR-3 — Public API compatibility

- FR-3.1 The following exported symbols MUST retain their current signatures and
  semantics: `Manager`, `GetManager`, `ResetInstance`, `ConfigReaderProducer`,
  `ManagerConfig`, `Manager.AddConsumer`, `Manager.AddConsumerAndRegister`,
  `Manager.RegisterHandler`, `Manager.RemoveHandler`, `Manager.Consumers`,
  `Consumer`, `Consumer.Snapshot`, `Config`, `NewConfig`, and every `Set*` decorator
  (`SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`, `SetFetchTimeout`,
  `SetMaxConsecutiveTimeouts`, `SetMaxInFlight`).
- FR-3.2 No service under `services/` may require source changes. A service's
  consumer wiring is unchanged.
- FR-3.3 Header parsers (`SpanHeaderParser`, `TenantHeaderParser`) MUST continue to
  run per message before the handler, preserving trace and tenant context propagation.
- FR-3.4 Group ID resolution via `libs/atlas-kafka/consumergroup.Resolve` is unchanged.
- FR-3.5 `ConfigReaderProducer` / the `KafkaReader` seam MUST remain usable for test
  injection, or be replaced by an equivalent documented seam with all existing tests
  migrated.

### FR-4 — Observability

- FR-4.1 `Snapshot` MUST gain: `AssignedPartitions []int`, `GenerationID int32`, and
  `LastAssignmentAt time.Time`.
- FR-4.2 Assignment changes MUST be logged at Info, including topic, group, generation
  and the assigned partition list.
- FR-4.3 A consumer entering healthy-idle MUST log at Debug (not Warn) — this state is
  expected, not exceptional.
- FR-4.4 Existing `Snapshot` fields MUST retain their meaning so the task-136 dwell
  harness assertions remain valid.

### FR-5 — Rollback safety

- FR-5.1 The implementation MUST be selectable at runtime by env var
  (`KAFKA_CONSUMER_ENGINE=consumergroup|reader`), defaulting to the new engine, with
  the legacy `*kafka.Reader` path retained for one release.
- FR-5.2 Switching engines MUST require only a pod restart — no topic, offset, or
  group-state migration.
- FR-5.3 Both engines MUST use the same group IDs and offset-commit semantics, so a
  rollback resumes from committed offsets without replay or loss beyond normal
  at-least-once behaviour.

## 5. API Surface

No HTTP/REST surface changes. This is a Go library-internal migration; the consumer
package's exported Go API is the contract and is frozen by FR-3.

The debug consumer-state endpoint exposed via `Consumer.Snapshot()` gains the three
fields in FR-4.1. Existing fields are additive-compatible — consumers of the snapshot
JSON must not break.

## 6. Data Model

No database entities, no migrations, no tenant-scoped tables. Kafka group and offset
state is unchanged: same group IDs (FR-3.4), same `__consumer_offsets` semantics
(FR-5.3).

## 7. Service Impact

| Area | Change |
|---|---|
| `libs/atlas-kafka/consumer` | Rewritten fetch/commit core; assignment-aware watchdog; new Snapshot fields |
| `libs/atlas-kafka/consumergroup` | None (group-ID resolution unchanged) |
| `libs/atlas-kafka/producer` | None |
| `services/*` (all Go services) | **No source changes.** All consume the lib via `go.work`; every service must be rebuilt and re-verified |

Because `libs/atlas-kafka` is consumed by every Go service, verification is
cluster-wide: `docker buildx bake all-go-services` from the worktree root, not a
single-service build.

## 8. Non-Functional Requirements

**Performance**
- Steady-state recreate rate on an idle `atlas-main` MUST be ~0/hour, down from the
  measured 19–246/hour/service.
- Assignment lookup MUST NOT add a broker round trip on the message path; it is read
  from the in-process `Generation`.
- Message-delivery latency MUST NOT regress: task-136 S1 measured p99 22.0 ms /
  max 87.1 ms publish→handler. The new engine MUST meet or beat this.

**Reliability**
- At-least-once delivery is preserved. Handlers remain responsible for idempotency
  (task-208 Parts 1–2).
- A genuinely stalled assigned consumer MUST still recover within
  `maxConsecutiveTimeouts × fetchTimeout` (~3 min at defaults).

**Observability**
- Assignment and generation visible in `Snapshot` (FR-4.1).
- `atlas_*` Prometheus metrics are currently **not scraped** in `atlas-main` (verified:
  zero `atlas_*` series). This task does not fix scraping, but MUST NOT add
  metrics-only observability that would be invisible — operator-facing signals go to
  logs and `Snapshot`.

**Multi-tenancy**
- Tenant context propagation via `TenantHeaderParser` is unchanged (FR-3.3).

**Security**
- No new external surface; no credential or authz changes.

## 9. Open Questions

1. **Coordination with task-208.** Its Part 3 rewrites `readerMadeProgress` with a
   persistent backoff (15-min cap, "5× reduction in churn"). Working assumption:
   **209 supersedes 208 Part 3**, and 208 keeps Parts 1–2 (idempotency), which are
   needed regardless. Both branches edit `libs/atlas-kafka/consumer/manager.go`;
   merge order must be agreed before either lands.
2. **Group blast radius.** One group spans N topics, so any rejoin stalls all of them.
   Out of scope here; C makes a per-topic group split tractable as a follow-up. Worth
   its own task?
3. **Zombie members.** 9 members for 2 pods (`Monster Registry Service`), 13 for
   `Consumables Service`. Expected to shrink once rejoin churn stops — needs
   confirmation post-deploy, and a separate task if it persists.
4. **Legacy engine removal.** FR-5.1 keeps the `*kafka.Reader` path for one release.
   Which release removes it?
5. **`WatchPartitionChanges`.** `ReaderConfig` sets partition-watch behaviour today;
   confirm the equivalent `ConsumerGroupConfig` setting so a partition-count increase
   is still picked up without restart.

## 10. Acceptance Criteria

- [ ] `libs/atlas-kafka/consumer` consumes via `kafka.ConsumerGroup` + `Generation`;
      no `*kafka.Reader` with `GroupID` set remains on the new path.
- [ ] A consumer with zero assigned partitions never logs `stall suspect`/`wedged` and
      never increments `RecreateCount`, verified by an integration scenario.
- [ ] A new scenario in `libs/atlas-kafka/consumer/dwell_integration_test.go` (S6)
      provisions a group with more members than partitions and asserts
      `totalRecreates == 0` over a compressed-tick run.
- [ ] Existing dwell scenarios S1–S5 still pass, with S1 p99 ≤ 22 ms and max ≤ 87 ms.
- [ ] An assigned consumer whose fetches genuinely stop is still recreated within
      `maxConsecutiveTimeouts × fetchTimeout` (regression test).
- [ ] `Snapshot` exposes `AssignedPartitions`, `GenerationID`, `LastAssignmentAt`.
- [ ] Every exported symbol in FR-3.1 retains its signature; `git diff --stat services/`
      is empty.
- [ ] `KAFKA_CONSUMER_ENGINE=reader` restores the legacy path; both engines resume from
      the same committed offsets across a restart.
- [ ] `go test -race ./...` clean in `libs/atlas-kafka` and every module that depends on it.
- [ ] `go vet ./...` clean; `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
      `tools/goroutine-guard.sh` all clean.
- [ ] `docker buildx bake all-go-services` succeeds from the worktree root.
- [ ] Post-deploy on `atlas-main`: `count_over_time({namespace="atlas-main"} |= "Recreated reader for topic" [1h])`
      is ~0 for all services (baseline: 19–246/hour/service).
- [ ] Post-deploy: attack→drop-visible latency shows no multi-second outliers over a
      10-minute play session (baseline: 4.7 s and 4.2 s stalls in trace `bd9b801a…`).
