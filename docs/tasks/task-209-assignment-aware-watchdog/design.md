# Assignment-Aware Consumer Watchdog — Design

Task: task-209
Status: Proposed
Created: 2026-08-10
Inputs: [`prd.md`](prd.md), [`risks.md`](risks.md)

---

## 1. Problem restated in terms of the code

`Consumer.start` (`libs/atlas-kafka/consumer/manager.go:476`) owns one `*kafka.Reader`
built with `GroupID` set (`manager.go:161-167`). Inside kafka-go that reader constructs
its *own* `ConsumerGroup` with `Topics: []string{topic}` (`reader.go:717-733`), so a
service that registers 15 consumers under one group ID joins that group **15 times** —
one member per topic per pod. The range balancer then assigns each topic's partitions
only among the members that subscribed to that topic. With `NumPartitions: 1` and
`replicas: 2`, exactly one of the two members for each topic gets nothing.

The watchdog's health input is `readerMadeProgress` (`manager.go:65-72`), which reads
`Stats().Fetches | Dials | Messages`. An unassigned member issues no fetches, so every
`fetchTimeout` tick is classified no-progress (`manager.go:428-443`), and at
`maxConsecutiveTimeouts` the loop returns `errFetchWedged` → `start` closes the reader,
backs off, and rebuilds it (`manager.go:497-517`). Rebuilding re-joins the group, which
rebalances **every member of that group**, including the ones carrying hot gameplay
topics. That is the 4.7 s stall in trace `bd9b801a…`.

The missing fact is the partition assignment. `*kafka.Reader` never exposes it;
`ReaderStats.Partition` is stamped at construction, not live. `kafka.Generation`
exposes it directly as `Assignments map[string][]PartitionAssignment`
(`consumergroup.go:321`).

### 1.1 The second-order cause, and why it matters more than the first

Assignment-awareness alone stops the *unassigned* member from self-recreating. But the
recreate action itself is the real weapon: **any** reader rebuild on the current engine
is a group rejoin. A transient dial failure, an EOF, a retry exhaustion — each costs the
whole group a rebalance. Migrating to `ConsumerGroup` lets us scope recovery to a single
*partition reader*, which is a purely local operation with no broker-visible group
effect. This design treats that inversion — **recover in place, never by rejoining** —
as a co-equal goal with FR-2, because it is what drives the steady-state rebalance rate
to ~0 rather than merely reducing it.

## 2. Constraints that shape the design

| Constraint | Source | Consequence |
|---|---|---|
| Exported API frozen | FR-3.1 | New capability arrives as *additive* symbols only |
| No service source changes | FR-3.2 | Engine choice is env-driven, not call-site-driven |
| Legacy engine retained one release | FR-5.1 | Both engines coexist in one binary; shared state model |
| Identical offset semantics both ways | FR-5.3 | The commit-offset convention is a hard contract (§5) |
| `routine.Go` for every goroutine we spawn | CLAUDE.md item 6 | `gen.Start` is a library call, not a bare `go` — guard-clean |
| Existing test seam must survive or be migrated | FR-3.5 | Reuse `KafkaReader`/`ReaderProducer` for partition readers |

## 3. Architecture options considered

### Option A — one `ConsumerGroup` per `Consumer` (per topic) ✅ recommended

1:1 with today's topology: each `Consumer` owns a `ConsumerGroup` configured with
`Topics: []string{c.topic}` and the same `c.groupId`. Member count, group IDs, balancer
and offset keys are all bit-identical to today.

- **Pro** — Zero change to broker-visible group topology, so rollback (FR-5.3) is a
  restart, and a mixed fleet (some pods on `reader`, some on `consumergroup`) is
  well-defined. Preserves `AddConsumer` being callable at any time.
- **Pro** — Maps cleanly onto the existing per-topic `Consumer` struct and its
  per-topic `Snapshot`; the debug route keys by topic and stays correct.
- **Con** — Does not reduce member count (topics × pods). §9.3's zombie-member question
  is not addressed directly, only indirectly by removing the churn that creates them.

### Option B — one shared `ConsumerGroup` per group ID, subscribing to all topics

Collapse a service's N per-topic members into one member subscribing to all N topics;
fan generations out to the per-topic `Consumer` objects.

- **Pro** — Member count drops from topics × pods to pods (9 → 2 for
  `Monster Registry Service`). Rebalances get cheaper and the balancer can spread topics
  across pods so neither member is idle.
- **Con — disqualifying for this task** — `AddConsumer` currently starts a consumer the
  moment it is called. A shared group must know its full topic list *before* joining, so
  late registration forces a rejoin, and services would need a start barrier — a change
  visible at every call site, violating FR-3.2. It also makes a mixed-engine fleet
  ill-defined: one member subscribing to 15 topics next to 15 members subscribing to one
  each is a subscription topology the range balancer handles, but not one we can reason
  about during a staged rollout (risks.md §"Staged rollout").
- **Verdict** — Correct eventual destination; wrong first step. Option A is a strict
  prerequisite (it is what puts `ConsumerGroup` in our hands at all). Record as a
  follow-up task alongside PRD §9.2.

### Option C — keep `*kafka.Reader`, infer assignment out-of-band

Query the group's assignment via `kafka.Client.DescribeGroups` on a timer and gate the
watchdog on it.

- **Pro** — Small diff; could ship on the legacy engine today (risks.md R8's escape
  hatch).
- **Con** — Adds a broker round trip on a timer per consumer (~237 × 2 pods), violating
  NFR "assignment lookup MUST NOT add a broker round trip". Racy: the describe result is
  a snapshot from a different generation than the one the reader is in. And it fixes only
  FR-2, leaving every other recreate (EOF, dial failure) as a full group rejoin — §1.1.
- **Verdict** — Rejected as the design, retained as the documented emergency mitigation
  if the R8 window proves too long.

**Chosen: Option A.**

## 4. Target structure

New files under `libs/atlas-kafka/consumer/`; `manager.go` keeps the Manager, the
`Consumer` struct, the state recorders and `processMessage`.

```
consumer/
  manager.go        Manager, Consumer struct, recorders, processMessage, Snapshot  (trimmed)
  engine.go         engine selection (KAFKA_CONSUMER_ENGINE) + Consumer.start dispatch
  engine_reader.go  legacy path, MOVED VERBATIM from manager.go (no behaviour edits)
  engine_group.go   new path: ConsumerGroup lifecycle + generation loop
  partition.go      per-partition fetch loop, watchdog, prefix-commit cursor
  group.go          Group/Generation interfaces + kafka-go adapters + new seams
  config.go         unchanged public decorators; internal fields reused by both engines
  debug.go          three new attributes
```

Moving the legacy path **verbatim** into `engine_reader.go` in its own commit is
deliberate: it makes the review diff for the new engine a pure addition, and it keeps
`git blame` on the legacy path intact for the one release it survives.

### 4.1 Control flow — new engine

```go
func (c *Consumer) startGroupEngine(l logrus.FieldLogger, ctx context.Context) {
    backoff := newFetchBackoff()
    for attempt := 0; ; attempt++ {
        if ctx.Err() != nil { return }

        grp, err := c.gp(c.groupConfig())          // seam; see §6
        if err != nil { /* record, backoff, continue */ }

        err = c.runGenerations(l, ctx, grp)
        _ = grp.Close()

        if ctx.Err() != nil || errors.Is(err, context.Canceled) { return }
        c.recordError(err)
        wait := backoff.next()
        select { case <-ctx.Done(): return; case <-time.After(wait): c.recordBackoff(wait) }
    }
}

func (c *Consumer) runGenerations(l logrus.FieldLogger, ctx context.Context, grp Group) error {
    for {
        gen, err := grp.Next(ctx)                  // blocks until previous generation ends
        if err != nil { return err }               // ErrGroupClosed / ctx error

        parts := gen.Assignments()[c.topic]
        c.onAssignment(gen.ID(), partitionIDs(parts))   // FR-4.1, FR-4.2, FR-2.4

        if len(parts) == 0 {
            l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment "+
                "in generation %d; healthy-idle.", c.topic, c.groupId, gen.ID())
            continue                               // FR-2.1, FR-2.2, FR-2.5
        }
        for _, pa := range parts {
            pa := pa
            gen.Start(func(gctx context.Context) { c.runPartition(l, ctx, gctx, gen, pa) })
        }
    }
}
```

Three properties fall out of kafka-go's implementation and are load-bearing:

1. **`Next` blocks until the current generation ends** (`consumergroup.go:701` receiving
   on the unbuffered `cg.next`, fed only after `<-gen.done` in
   `nextGeneration`/`consumergroup.go:855-869`). So the `continue` in the zero-assignment
   branch is not a spin — it parks.
2. **Heartbeats are per generation, not per assignment.** `gen.heartbeatLoop` is started
   unconditionally in `nextGeneration` (`consumergroup.go:840`). An unassigned member that
   starts nothing still heartbeats and stays eligible for the next assignment — FR-2.2 is
   satisfied by doing nothing, which is exactly the property we want.
3. **`gen.Start`'s function exiting ends the generation** (`consumergroup.go:387-405`).
   This is the inversion in §1.1 stated as a rule: **`runPartition` must not return
   except on `gctx.Done()` / `ctx.Done()`.** Every recoverable error is handled in place.

### 4.2 Control flow — per partition

```go
func (c *Consumer) runPartition(l, ctx, gctx, gen, pa) {
    st := c.partitionState(pa.ID)                  // per-partition watchdog counters
    defer c.releasePartition(pa.ID)

    cur := newCursor(pa.Offset)                    // next offset to read; see §5
    for {
        if gctx.Err() != nil || ctx.Err() != nil {
            c.drain(l, gen, pa.ID, cur)            // FR-1.6
            return
        }
        rd := c.prp(c.partitionReaderConfig(pa.ID), cur.next())   // seam; see §6
        err := c.runPartitionFetchLoop(l, ctx, gctx, gen, pa.ID, rd, cur, st)
        _ = rd.Close()

        if gctx.Err() != nil || ctx.Err() != nil { c.drain(...); return }

        // Any other error — EOF, wedge, dial failure — rebuilds THIS reader only.
        c.recordError(err)
        c.onPartitionReaderRebuilt(pa.ID)          // increments RecreateCount
        wait := st.backoff.next()
        select { case <-gctx.Done(): return; case <-ctx.Done(): return; case <-time.After(wait): c.recordBackoff(wait) }
    }
}
```

`runPartitionFetchLoop` is the existing `runFetchLoopSerial` / `runFetchLoopParallel`
logic with three substitutions:

- the loop's cancel signal is `gctx` (generation-scoped) rather than only `ctx`;
- `reader.CommitMessages(ctx, msg)` becomes `cur.commit(gen, topic, partition, msg)` (§5);
- `handleFetchDeadline` takes the per-partition state `st` instead of the Consumer-level
  scalars.

`handleFetchDeadline` and `readerMadeProgress` themselves are **unchanged** — a
partition reader (no `GroupID`) still reports `Fetches`/`Dials`/`Messages`, so the
idle-vs-stuck classification from task-136 carries over intact. It is now only ever
reached by a reader that *holds an assignment*, which is the whole point.

## 5. The offset contract (highest-risk surface — risks.md R1)

This is the one place where a one-character error causes silent gameplay data loss, so
the convention is pinned to source rather than to intuition.

**Read side.** `PartitionAssignment.Offset` is the committed offset from
`offsetFetch`, or `config.StartOffset` when nothing is committed
(`consumergroup.go:1168-1173`, `1182-1204`). It is therefore already "the next offset to
read", not "the last offset consumed". A partition reader is positioned with
`SetOffset(pa.Offset)` verbatim — including the `FirstOffset`/`LastOffset` sentinels,
which `SetOffset` accepts (`reader.go` `SetOffset`, guarded only against group readers).
`ConsumerGroupConfig.StartOffset` is set from `c.startOffset`, satisfying FR-1.4.

**Write side.** `*kafka.Reader` commits `msg.Offset + 1` (`reader.go:1529`, and the
`r.offset = m.message.Offset + 1` advance at `reader.go:846`). The new engine commits
the identical value:

```go
gen.CommitOffsets(map[string]map[int]int64{ c.topic: { partition: msg.Offset + 1 } })
```

Both engines therefore write the same number for the same delivered message, which is
precisely what makes FR-5.3 (rollback in either direction, no replay, no loss) true
rather than aspirational. **This `+ 1` gets its own named unit test asserting the exact
committed value, and a round-trip integration test that consumes N messages on one
engine and resumes on the other.**

**Cursor.** Per partition, `cursor` holds `pending []*inflight` and `committed int64`.

- `maxInFlight == 1` (default): commit immediately after a successful `processMessage`;
  the cursor is a single value. Bit-equivalent to today's serial path.
- `maxInFlight > 1`: the prefix-commit walk from `runFetchLoopParallel`
  (`manager.go:601-617`) moves into `cursor.advance`, keyed per partition. This is
  *stricter* than today, where one cursor spanned every partition the group reader held —
  FR-1.5 is satisfied and slightly strengthened.
- **On commit failure**: log at Warn, leave `committed` where it is, keep the entries in
  `pending`. The next successful commit re-attempts the same high-water mark. The cursor
  never advances on a failed commit — FR-1.3.

**Generation teardown (FR-1.6).** `drain` stops fetching, waits for in-flight handlers up
to `drainTimeout`, then makes one final `advance`. `drainTimeout` must stay well below
kafka-go's `defaultRebalanceTimeout` of 30 s (`consumergroup.go:47`) or a slow drain
stalls the group's rebalance — the exact failure `gen.Start`'s doc warns about. **Chosen:
5 s**, matching `defaultTimeout` (`consumergroup.go:63`). Handlers still running at the
deadline are abandoned uncommitted → redelivered next generation. That is at-least-once
and is the correct side to err on (risks.md R1: "loss is worse than duplication").

## 6. Seams and testability (FR-3.5)

`ReaderProducer`, `KafkaReader`, `MessageReader`, `MessageCommitter`, `Closer`,
`StatsProvider` and `ConfigReaderProducer` are **unchanged**. The new engine's
per-partition reader is a `KafkaReader`, so every existing mock (`ChannelMockReader`,
`manager_test.go:73`) works against the new fetch loop with no edits. This is the single
biggest reason to reuse rather than replace the seam: ~1400 lines of `manager_test.go`
survive.

Two additive seams:

```go
// group.go — interfaces mirroring the kafka-go shapes we depend on.
type Group interface {
    Next(ctx context.Context) (Generation, error)
    Close() error
}

type Generation interface {
    ID() int32
    Assignments() map[string][]kafka.PartitionAssignment
    Start(fn func(ctx context.Context))
    CommitOffsets(offsets map[string]map[int]int64) error
}

type GroupProducer func(cfg kafka.ConsumerGroupConfig) (Group, error)

// A partition reader needs positioning, which KafkaReader deliberately does not
// expose. The offset is a producer argument rather than an interface method, so
// KafkaReader stays frozen.
type PartitionReaderProducer func(cfg kafka.ReaderConfig, offset int64) KafkaReader

func ConfigGroupProducer(gp GroupProducer) ManagerConfig            // additive
func ConfigPartitionReaderProducer(p PartitionReaderProducer) ManagerConfig  // additive
```

Defaults wrap `kafka.NewConsumerGroup` and `kafka.NewReader` + `SetOffset`. A thin
adapter satisfies `Generation` over `*kafka.Generation` (its `ID`/`Assignments` are
fields, so the adapter is method-wrappers over an embedded pointer).

The fake group used in unit tests drives generations by hand: `Next` returns scripted
generations from a channel, `Start` runs the function on a `routine.Go` with a
controllable done-channel, `CommitOffsets` records into a slice. That is what makes
FR-2.1 testable without a broker — the scripted generation simply assigns zero
partitions.

## 7. State model and observability

Watchdog counters move from Consumer-level scalars to a per-partition map
`map[int]*partitionState` guarded by `c.mu`, holding
`consecutiveTimeouts, idleTicks, noProgressTicks, lastTimeoutAt, lastIdleTickAt, lastNoProgressAt, backoff`.

The reason is correctness, not tidiness: with a scalar, a 2-partition topic where one
partition wedges and the other flows would have its wedge masked by the healthy
partition's resets. On the 1-partition topics that make up all of `atlas-main`, the
per-partition map has exactly one entry and behaves identically to today.

`Snapshot` keeps every existing field with its existing meaning (FR-4.4) by aggregating:

| Field | Aggregation across partitions |
|---|---|
| `ConsecutiveTimeouts` | max |
| `IdleTicks`, `NoProgressTicks` | sum |
| `LastTimeoutAt`, `LastIdleTickAt`, `LastNoProgressAt`, `LastFetchAt` | most recent |
| `RecreateCount` | sum of partition-reader rebuilds |

`RecreateCount`'s *meaning* narrows: on the legacy engine it counted group rejoins; on
the new engine it counts partition-reader rebuilds, which are local. The dwell S3
assertion ("forced recreate is bounded") remains valid — it asserts boundedness, not a
mechanism. This narrowing is called out in the field's doc comment because an operator
comparing the number across a rollback would otherwise misread it.

New fields (FR-4.1) plus one beyond the PRD:

```go
AssignedPartitions []int      // sorted; empty slice, never nil, when healthy-idle
GenerationID       int32
LastAssignmentAt   time.Time
Engine             string     // "consumergroup" | "reader"
```

`Engine` is not in FR-4.1 but is the first thing an operator needs during a staged
rollout or a rollback (risks.md §"Staged rollout" runs both engines in one cluster) —
without it, "which engine is this pod on?" is answerable only from the pod's env. It is
additive and costs nothing. All four are mirrored into `debugAttributes`
(`debug.go:56-79`) with the existing lowerCamel JSON naming.

Logging (FR-4.2/4.3): assignment changes log at **Info** with topic, group, generation
and partition list; entering healthy-idle logs at **Debug**. No `stall suspect` or
`wedged` Warn can be emitted while unassigned, because the code path that emits them is
only reachable from `runPartition`, which only exists when `len(parts) > 0` — FR-2.5 is
structural, not a conditional we could forget to check.

## 8. Engine selection and rollback (FR-5)

```go
const engineEnvVar = "KAFKA_CONSUMER_ENGINE"   // "consumergroup" (default) | "reader"
```

Resolved once per `Manager` construction (inside `GetManager`'s `once.Do`, alongside the
existing producer defaults) and stamped onto each `Consumer` at `AddConsumer` time, so
every consumer in a process runs the same engine and `Snapshot.Engine` is stable.
Resolution mirrors `consumergroup.Resolve`'s style: unset/empty → default; unrecognised
value → **Warn once and use the default** (fail-soft; a typo in a deployment env var must
not take a service's consumers offline).

Reading it inside `once.Do` keeps it testable — `t.Setenv` + `ResetInstance()`
(`manager.go:96`) is the pattern the existing tests already use.

Rollback is a pod restart with the env var flipped, with no state migration, because §5
pins both engines to the same committed-offset convention and Option A keeps group IDs
and member topology identical.

## 9. Testing strategy

Layered, cheapest first. Nothing here claims rebalance behaviour from a unit test.

**Unit (no broker) — new `engine_group_test.go`, `partition_test.go`:**
- Zero-assignment generation: assert `RecreateCount == 0`, no `stall suspect`/`wedged`
  log entries (via the existing `logrus/hooks/test` hook), `AssignedPartitions` empty,
  and that the consumer parks rather than spins. **This is the FR-2.1 acceptance test.**
- Unassigned → assigned transition resets no-progress state (FR-2.4).
- Assigned partition with a stalled reader still returns wedge and rebuilds *that reader*
  after `maxConsecutiveTimeouts` ticks — and the generation does **not** end (FR-2.3 plus
  the §1.1 inversion, asserted by checking `Next` was not called again).
- Commit value is exactly `msg.Offset + 1` (§5).
- Commit failure does not advance the cursor; the next commit retries the same offset.
- Prefix-commit ordering and failed-message-blocks-cursor, at `maxInFlight > 1`, per
  partition — ported from `manager_test.go:1144` and `:1231`.
- Drain on generation end: in-flight completions commit; abandoned ones do not.

**Existing suite:** `manager_test.go`, `idle_stuck_test.go`, `timing_test.go`,
`debug_test.go` become table-driven over both engines where the mock seam allows, so the
legacy path keeps its coverage for the release it survives and the new path inherits it.

**Integration (`-tags integration`, testcontainers):** new **S6** in
`dwell_integration_test.go` — create a 1-partition topic, register **two** managers
(two `ResetInstance`-separated `Manager`s, or two processes' worth of consumers) under
one group ID so members outnumber partitions, run a compressed-tick window, assert
`totalRecreates == 0` and that the assigned member keeps delivering. Existing S1–S5 must
pass unchanged, with S1 gated at p99 ≤ 22 ms / max ≤ 87 ms (risks.md R6).

Plus a **cross-engine offset round-trip**: consume N messages on `consumergroup`, stop,
restart the same group on `reader`, assert no replay and no gap; then the reverse. This
is the executable form of FR-5.3.

**Live (post-deploy):** the two PRD §10 measurements — recreate-count/hour per service
from Loki, and an attack→drop-visible trace with no multi-second gap. Neither is
substitutable by a test.

## 10. Open questions from the PRD — resolved

**§9.5 `WatchPartitionChanges` — RESOLVED.** Today's `kafka.ReaderConfig`
(`manager.go:161-167`) never sets `WatchPartitionChanges`, so it is `false`, and
`reader.go:731-732` forwards that `false` into the internal `ConsumerGroupConfig`.
**Partition-count changes are already not watched.** The new engine sets `false` to
match, so there is no regression and risks.md **R4 drops to informational**. Enabling it
(`PartitionWatchInterval` default 5 s, `consumergroup.go:59`) is now a one-line opt-in
should we ever repartition — worth a note in the lib README, not a behaviour change
smuggled into a migration.

**§9.2 group blast radius — narrowed.** Option A does not change it, but §1.1 removes
the dominant *trigger*: after this task a rebalance happens on pod start/stop and genuine
membership loss, not on every transient reader error. Option B (§3) is the structural fix
and is the natural follow-up task; it is unblocked by this one.

**§9.3 zombie members — unchanged, measurable after deploy.** The hypothesis that 9
members for 2 pods is rejoin churn residue is testable directly once churn stops; this
design adds `Snapshot.GenerationID`, which makes it observable per pod rather than only
from broker state.

**§9.4 legacy engine removal — proposed.** Remove `engine_reader.go` and the
`KAFKA_CONSUMER_ENGINE` switch in the release *after* the one in which every service has
run `consumergroup` through at least two full deploy cycles with recreate rate ~0. That
is a follow-up task, not a dangling TODO in this branch.

## 11. Requirements traceability

| Requirement | Where satisfied |
|---|---|
| FR-1.1 | §4.1 `startGroupEngine` / `runGenerations` |
| FR-1.2 | §4.1 `gen.Assignments()[c.topic]`, one `gen.Start` per partition |
| FR-1.3 | §5 write side; cursor never advances on commit failure |
| FR-1.4 | §5 read side; `ConsumerGroupConfig.StartOffset` from `c.startOffset` |
| FR-1.5 | §5 cursor, per partition |
| FR-1.6 | §5 `drain`, 5 s bound |
| FR-2.1 / 2.2 | §4.1 zero-assignment branch; heartbeat is generation-scoped |
| FR-2.3 | §4.2 in-place partition-reader rebuild on wedge |
| FR-2.4 | §7 `onAssignment` resets per-partition state |
| FR-2.5 | §7 — structural: the warn path is unreachable while unassigned |
| FR-3.1 / 3.2 | §6 additive symbols only; §8 env-driven engine |
| FR-3.3 | `processMessage` (`manager.go:684`) untouched, shared by both engines |
| FR-3.4 | Option A keeps group IDs; `consumergroup.Resolve` not touched |
| FR-3.5 | §6 `KafkaReader`/`ReaderProducer` frozen and reused |
| FR-4.1 / 4.2 / 4.3 / 4.4 | §7 |
| FR-5.1 / 5.2 / 5.3 | §8, §5 |

## 12. Verification gates for this branch

Per CLAUDE.md "Build & Verification" — `libs/atlas-kafka` is consumed by every Go
service, so verification is cluster-wide, not module-local:

1. `go test -race ./...` in `libs/atlas-kafka` **and** every dependent module.
2. `go test -race -tags integration ./consumer/...` (dwell S1–S6).
3. `go vet ./...`.
4. `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` — all
   clean from the repo root. The goroutine guard is the one to watch: `gen.Start` is a
   method call, not a bare `go`, so it passes, but any hand-rolled goroutine in the
   partition loop must go through `routine.Go`.
5. `docker buildx bake all-go-services` from the worktree root.
6. Code review (`superpowers:requesting-code-review` → `backend-guidelines-reviewer`)
   before the PR.
7. `git diff --stat services/` empty — the machine-checkable form of FR-3.2.
