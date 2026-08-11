# task-209 — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read out of source in
the worktree or out of the pinned kafka-go module — nothing is recalled from
memory. File:line citations are the authority; re-read them if a plan step
does not match what you find.

---

## 1. What this task is actually fixing

Every Kafka consumer in Atlas runs one `*kafka.Reader` with `GroupID` set
(`libs/atlas-kafka/consumer/manager.go:161-167`). Inside kafka-go that reader
constructs its **own** `ConsumerGroup` with `Topics: []string{topic}`
(`reader.go:717-733`), so a service registering N consumers under one group ID
joins that group N times — one member per topic per pod.

The watchdog decides liveness from `readerMadeProgress`
(`manager.go:65-72`), which reads `Stats().Fetches | Dials | Messages`. **An
unassigned group member issues no fetches**, so every `fetchTimeout` tick is
classified no-progress (`manager.go:428-443`) and at `maxConsecutiveTimeouts`
the loop returns `errFetchWedged` → the reader is closed and rebuilt
(`manager.go:497-517`). Rebuilding rejoins the group, which rebalances **every
member of that group**, including the ones holding hot gameplay topics.

Every topic in `atlas-main` is single-partition (237 of them) while services
run `replicas: 2`, so in every group exactly one member is always unassigned
and recreates itself every ~3 minutes, forever.

`*kafka.Reader` exposes no assignment accessor; `ReaderStats.Partition` is a
constant stamped at construction (`reader.go:688-691`), not a live assignment.
`kafka.Generation.Assignments` (`consumergroup.go:308-323`) is the missing
fact, and getting to it is why this is a migration rather than a patch.

Three prior tasks tuned this same watchdog — `task-016-kafka-consumer-selfheal`,
`task-039-kafka-fetch-deadline`, `task-136-consumer-fetch-wedge` — and
task-208's Part 3 was written and then reverted in favour of this task.
**task-209 is the sole owner of `libs/atlas-kafka/consumer/manager.go`**
(risks.md R5, verified: `git diff --stat main...task-208-command-idempotency --
libs/atlas-kafka` is empty).

## 2. kafka-go facts the design depends on (v0.4.51, verified)

Module path: `github.com/segmentio/kafka-go v0.4.51` (`libs/atlas-kafka/go.mod`).
Source read from the module cache; the line numbers below are from that copy.

| Fact | Where | Why it matters |
|---|---|---|
| `Next` blocks until the previous generation ends | `consumergroup.go:701-713` receives on the unbuffered `cg.next`, fed only after `<-gen.done` in `nextGeneration` (`consumergroup.go:855-869`) | The zero-assignment `continue` **parks**, it does not spin. |
| Heartbeats are per-generation and unconditional | `gen.heartbeatLoop(...)` at `consumergroup.go:840`, before the assignment is used | An unassigned member that starts nothing still heartbeats and stays eligible — FR-2.2 is satisfied by doing nothing. |
| **A `gen.Start` function returning ENDS the generation** | `consumergroup.go:387-405` — `close(g.done)` fires as soon as the first fn exits | `runPartition` must not return except when the generation or process is shutting down. This is the design §1.1 inversion. |
| `gen.close()` waits for every `Start`'d goroutine | `consumergroup.go:344-373` (`joined` chan) | A slow drain stalls the whole group's rebalance → `drainTimeout` must stay under `defaultRebalanceTimeout` (30 s, `consumergroup.go:47`). 5 s chosen, matching `defaultTimeout` (`consumergroup.go:63`). |
| `PartitionAssignment.Offset` is already "next offset to read" | `consumergroup.go:264-274`; built by `makeAssignments` from `fetchOffsets` (`consumergroup.go:1146-1204`), falling back to `config.StartOffset` | Feed it to `SetOffset` verbatim, sentinels included. |
| `ConsumerGroupConfig.StartOffset` must be `FirstOffset` or `LastOffset` | `Validate`, `consumergroup.go:240-247` | `Config.startOffset` defaults to `kafka.FirstOffset` (`config.go:36`), so it maps straight across. |
| `*kafka.Reader` commits `msg.Offset + 1` | `reader.go:1529`, and the `r.offset = m.message.Offset + 1` advance at `reader.go:846` | **The offset contract.** Both engines must write this exact value or a rollback replays or skips. |
| `SetOffset` is rejected on a group reader | `reader.go` `SetOffset` → `errNotAvailableWithGroup` (`reader.go:36`) | Partition readers must have **no** `GroupID`. |
| `ReaderConfig.Validate` rejects `Partition` + `GroupID` together | `reader.go:545-547` | Same conclusion, enforced at construction. |
| A non-group reader defaults to `offset: FirstOffset` | `reader.go:700-716` (`offset: FirstOffset` in the struct literal) | `SetOffset(FirstOffset)` is a harmless no-op; positioning is still explicit. |
| `CommitMessages` on a non-group reader returns `errOnlyAvailableWithGroup` | `reader.go` `CommitMessages` first statement | Partition readers must never commit through the reader — commits go through `Generation.CommitOffsets`. The test double asserts this by returning an error. |
| `WatchPartitionChanges` is `false` today | `manager.go:161-167` never sets it; `reader.go:731-732` forwards that `false` into the internal `ConsumerGroupConfig` | Partition-count changes are **already** not watched. Matching `false` is a no-regression, not a new limitation. PRD §9.5 resolved; risks.md R4 drops to informational. |

## 3. The library as it stands

`libs/atlas-kafka/consumer/` — 3624 lines including tests.

| File | Lines | Role |
|---|---|---|
| `manager.go` | 735 | `Manager`, `Consumer`, state recorders, `Snapshot`, the fetch loops, `processMessage` |
| `config.go` | 129 | `Config` + the `Set*` decorators; defaults documented against task-136 |
| `debug.go` | 107 | `GET /debug/consumers` JSON:API route, mounted by ~20 services |
| `header.go` | 66 | `SpanHeaderParser`, `TenantHeaderParser` |
| `offsets.go` | 80 | `ReadEndOffsets` / `ReadReplayableEndOffsets` — unrelated to this task |
| `manager_test.go` | 1436 | The bulk of the pin |
| `dwell_integration_test.go` | 411 | task-136 testcontainers harness, S1–S5, `-tags integration` |
| `idle_stuck_test.go` | 233 | Idle-vs-stuck classification |
| `debug_test.go` | 181 | Debug-route JSON shape |
| `timing_test.go` | 99 | Phase-timing snapshot fields; also holds `snapshotForTopic` |

Test package layout: `config_test.go` is **internal** (`package consumer`);
every other test file is **external** (`package consumer_test`). New test files
in the plan are internal, because they touch unexported types (`cursor`,
`partitionState`, `resolveEngine`).

Key seams already in place and deliberately **frozen** (FR-3.5):
`KafkaReader`, `MessageReader`, `MessageCommitter`, `Closer`, `StatsProvider`,
`ReaderProducer`, `ConfigReaderProducer` (`manager.go:28-83`). Every existing
mock is written against `KafkaReader`, which is why the new per-partition
reader is also a `KafkaReader` — roughly 1400 lines of `manager_test.go`
survive untouched.

## 4. Decisions made while planning (and why)

These go beyond what design.md spells out. Each is a judgment call an
implementer would otherwise have to make blind.

**`legacyPartition = -1`.** Design §7 moves watchdog counters to a
per-partition map. Rather than keep both a scalar set (for the legacy engine)
and a map (for the new one), the legacy engine keys the same map with `-1`.
Real partition ids are non-negative (`ReaderConfig.Validate`,
`reader.go:533-535`), so there is no collision, and a single-entry map
aggregates in `Snapshot` — max / sum / most-recent over one element — to
exactly the pre-task scalar values. That is what keeps `idle_stuck_test.go`,
`timing_test.go` and `debug_test.go` passing **unedited**, which is the
FR-4.4 gate.

**`ConfigEngine` as an additive `ManagerConfig`.** Design §8 resolves the
engine inside `GetManager`'s `once.Do` from env and notes `t.Setenv` +
`ResetInstance()` as the test pattern. That works, but ~30 existing tests need
pinning and process env is shared state across a package's tests. An explicit
`ConfigEngine(EngineReader)` on the `GetManager` call is one line per test,
races with nothing, and doubles as a supported override for an embedder. Env
remains the production selector; `ConfigEngine` wins when both are set.

**Both engines share the cursor.** With `maxInFlight == 1` the prefix walk
commits exactly the one tracked message, and a failed handler blocks the
cursor — behaviourally identical to today's serial path
(`manager.go:570-574`). Unifying them means one commit surface to test rather
than two.

**`context.AfterFunc` instead of a merge goroutine.** `runPartition` must
cancel on *either* the process context or the generation context, and the
generation context (kafka-go's `genCtx`, `consumergroup.go:276-301`) is not
derived from ours. `pctx, cancel := context.WithCancel(ctx)` plus
`stop := context.AfterFunc(gctx, cancel)` merges them with no goroutine of our
own — which also keeps `tools/goroutine-guard.sh` quiet.

**Quiesce before choosing a resume offset.** On a partition-reader rebuild the
loop waits for in-flight handlers and advances the cursor *before* computing
`cur.resumeOffset(pa.Offset)`, then `cur.reset()`s the queue. Without this, a
rebuild would re-fetch messages whose handlers are still running and process
them twice. It is still at-least-once — the drain is bounded at 5 s — but the
common case does not duplicate.

**`cursor` uses two mutexes.** `mu` guards the queue and offset marks; `cmu`
serializes the commit network call so two concurrent `advance` calls from
handler goroutines cannot write offsets out of order. They are never held
together across the call.

## 5. Where each requirement lands

| Requirement | Task |
|---|---|
| FR-1.1 ConsumerGroup + `Next` | 6 |
| FR-1.2 one reader per assigned partition | 5, 6 |
| FR-1.3 commit failures never advance the cursor | 2 (cursor), 5 (wiring) |
| FR-1.4 `startOffset` honoured | 3 (`groupConfig`), 5 (`pa.Offset` → `SetOffset`) |
| FR-1.5 prefix-commit cursor, per partition | 2 |
| FR-1.6 generation teardown does not drop work | 5 (`quiesce`) |
| FR-2.1 zero assignments = healthy-idle | 6 |
| FR-2.2 unassigned member stays in the group | 6 (structural — kafka-go heartbeats per generation) |
| FR-2.3 assigned + no progress still recovers | 5 |
| FR-2.4 unassigned→assigned resets state | 4 (`onAssignment`), 6 (end to end) |
| FR-2.5 no stall/wedge warns while unassigned | 6 (structural — the warn path is only reachable from `runPartition`) |
| FR-3.1 exported API frozen | 3, 7 (additive symbols only) |
| FR-3.2 no service source changes | 7 step 8, 9 step 6 (`git diff --stat services/`) |
| FR-3.3 header parsers unchanged | `processMessage` (`manager.go:684`) is untouched and shared |
| FR-3.4 group-ID resolution unchanged | `libs/atlas-kafka/consumergroup/resolver.go` not touched |
| FR-3.5 test seam survives | 3 (`KafkaReader` reused verbatim) |
| FR-4.1 new Snapshot fields | 4, 7 |
| FR-4.2 assignment changes logged at Info | 6 |
| FR-4.3 healthy-idle logged at Debug | 6 |
| FR-4.4 existing Snapshot fields keep their meaning | 4 step 9 (existing tests unedited) |
| FR-5.1 env-selectable engine | 7 |
| FR-5.2 rollback is a restart | 7, 8 |
| FR-5.3 identical offset semantics both ways | 2, 5, 8 (cross-engine round-trip) |

## 6. Verification gates for this branch

`libs/atlas-kafka` is consumed by **63 modules** under `services/` and `libs/`
(`grep -rl atlas-kafka --include=go.mod services libs | wc -l`), so
verification is cluster-wide, not module-local. Per CLAUDE.md:

1. `go test -race ./...` in `libs/atlas-kafka` and every dependent module.
2. `go test -race -tags integration ./consumer/...` — dwell S1–S6 plus the
   cross-engine round-trip. **Requires Docker** (testcontainers,
   `confluentinc/cp-kafka:7.6.0`).
3. `go vet ./...`.
4. `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`
   from the repo root. The goroutine guard sweeps `./...` in every module —
   **test files included** — so test fakes must use `routine.Go`.
   `tools/lint.sh --check` needs nvm on PATH or it false-fails on node.
5. `docker buildx bake all-go-services` from the worktree root.
6. Code review via `superpowers:requesting-code-review` before the PR, with
   reviewer subagents pinned to the cheaper model.
7. `git diff --stat services/` empty.

Guards that do **not** apply (nothing in their scope changes on this branch):
service-registration, template-opcode-order, template-duplicate-binding,
template-movement-types, skill-job-id, buff-duration.

## 7. Risks to keep in view while implementing

From [`risks.md`](risks.md), ranked:

- **R1 — offset-commit regression → silent message loss.** Critical. The
  cursor never advances on a failed commit; on any ambiguity prefer
  redelivery. Lost drops and lost EXP are invisible in logs.
- **R2 — duplicate delivery amplified during generation churn.** task-208's
  idempotency guard covers only atlas-inventory, atlas-cashshop and
  atlas-storage. Landing 208 first is the cheapest safety net but is no longer
  blocking.
- **R3 — `maxInFlight` cursor mis-ported.** Default stays 1. The
  `maxInFlight > 1` cursor tests are not optional.
- **R6 — latency regression.** The dwell S1 test asserts only `p99 < 1s`
  (`dwell_integration_test.go:228`); 22.0 ms / 87.1 ms are the *measured*
  task-136 S1 numbers the PRD says the new engine must meet or beat. Record
  the actual per-engine p99/max and compare — do not silently accept a
  regression, and do not retune a threshold to make a run pass.
- **R7 — the new engine is itself wedge-prone in some unforeseen way.** This
  is what `KAFKA_CONSUMER_ENGINE=reader` and the staged rollout exist for.

**Staged rollout** (risks.md): ephemeral/PR env → one low-traffic service on
`atlas-main` (e.g. `atlas-chalkboards`) → one hot-path service
(`atlas-monsters`, the traced stall) → the remainder. Both engines coexist in
one cluster because group and offset semantics are identical.

**Rollback triggers** — any one is sufficient: consumer lag on a hot topic
sustained > 5 s for more than 2 minutes; any evidence of message loss;
recreate rate materially worse than the 19–246/hour/service baseline; delivery
p99 above the task-136 S1 ceiling.

## 8. Post-deploy measurements no test can substitute

Both are PRD §10 acceptance criteria and belong in `verification.md` as
outstanding until observed on `atlas-main`:

- `count_over_time({namespace="atlas-main"} |= "Recreated reader for topic" [1h])`
  ~0 for all services. Baseline: 19–246/hour/service; wedge/recreate events
  were observed across ~25 services in a single 60-second window.
- Attack→drop-visible latency with no multi-second outliers over a 10-minute
  play session. Baseline: trace `bd9b801a…` on 2026-08-10 — a 4.7 s stall on
  the attack→death path and a further 4.2 s before atlas-quest credited the
  kill, 9.4 s end to end.
