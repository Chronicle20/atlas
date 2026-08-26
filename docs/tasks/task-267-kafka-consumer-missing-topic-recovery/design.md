# Kafka Consumer Recovery From a Late-Appearing Topic — Design

Version: v1
Status: Draft
Created: 2026-08-26
Input: `docs/tasks/task-267-kafka-consumer-missing-topic-recovery/prd.md` (approved)

---

## 1. Scope of this document

The PRD fixes the behaviour; this document fixes the *mechanism*, and it does
so against a decisive piece of evidence the PRD deliberately deferred to design
time (PRD §9 open questions 3 and 4): what kafka-go actually does when
`WatchPartitionChanges: true` meets a topic that does not exist.

The answer changes the shape of the fix. `WatchPartitionChanges: true` alone
does **not** close the outage; on its own it converts a silent outage into a
group-wide rebalance storm. That finding is established in §2 from
kafka-go v0.4.51 source (`libs/atlas-kafka/go.mod:9` pins v0.4.51), and it is
the reason this design is three parts rather than one.

---

## 2. Findings: what kafka-go v0.4.51 actually does

All line references are to
`$(go env GOMODCACHE)/github.com/segmentio/kafka-go@v0.4.51/`.

### 2.1 The leader tolerates a missing topic; the watcher does not

`assignTopicPartitions` (consumergroup.go:1010-1049) reads partitions for every
subscribed topic and then explicitly forgives the missing-topic case:

```go
partitions, err := conn.readPartitions(topics...)

// it's not a failure if the topic doesn't exist yet.  it results in no
// assignments for the topic.  this matches the behavior of the official
// clients: java, python, and librdkafka.
// a topic watcher can trigger a rebalance when the topic comes into being.
if err != nil && !errors.Is(err, UnknownTopicOrPartition) {
    return nil, err
}
```

This is exactly the incident: the group forms, the member joins, the topic
contributes no partitions, the member gets an empty assignment. kafka-go's own
comment names the intended remedy — "a topic watcher can trigger a rebalance
when the topic comes into being."

The watcher does not deliver on that promise. `Generation.partitionWatcher`
(consumergroup.go:500-552) does its *first* read outside the ticker loop:

```go
ops, err := g.conn.readPartitions(topic)
if err != nil {
    g.logError(...)
    return
}
oParts := len(ops)
for {
    select {
    case <-ticker.C:
        ops, err := g.conn.readPartitions(topic)
        switch {
        case err == nil, errors.Is(err, UnknownTopicOrPartition):
            ...
```

The ticker branch (consumergroup.go:527-529) treats `UnknownTopicOrPartition`
as a normal, non-fatal outcome. The startup read at consumergroup.go:512-518
does not — it returns on *any* error. That asymmetry is the bug.

### 2.2 `readPartitions` does return an error for a missing topic

The watcher's `conn` is a `timeoutCoordinator` wrapping a `*kafka.Conn`
obtained from `makeConnect` → `dialer.Dial` (consumergroup.go:878-894), and
`Dialer.DialContext` builds the `ConnConfig` with only `ClientID` and
`TransactionalID` (dialer.go:112-122) — so the coordinator connection's
`c.topic` is `""`.

`readTopicMetadatav1`/`readTopicMetadatav6` (conn.go:1030-1060) report a
per-topic error only when `c.topic == "" || t.TopicName == c.topic`. With
`c.topic == ""` the first disjunct holds, so a `UNKNOWN_TOPIC_OR_PARTITION`
(code 3) topic metadata entry is returned as `Error(3)` —
`UnknownTopicOrPartition`. The watcher's startup read therefore *does* fail for
a topic that does not exist.

**Answer to PRD open question 4:** the partition watch does not merely fail to
help on a zero-partition topic — its startup read errors and it gives up.

### 2.3 A watcher that returns kills the generation, and the rejoin has no backoff

`Generation.Start` (consumergroup.go:394-411) closes `g.done` as soon as *any*
Start'd function returns. The watcher is Start'd (consumergroup.go:846-850), so
its early return ends the generation immediately — before the generation is
even handed to `Next` in the common case, since the `cg.next <- &gen` send
(consumergroup.go:856-859) happens after the `gen.Start` calls.

`ConsumerGroup.run` (consumergroup.go:713-770) then sees `nextGeneration`
return a **nil** error — the generation "finished normally" — and takes the
`case err == nil: continue` arm. `JoinGroupBackoff` is applied only on the
error arm. So the rejoin is immediate.

**Answer to PRD open question 3:** `WatchPartitionChanges: true` alone does not
close the outage. Against a missing topic it produces
`join → sync → generation → watcher errors → generation ends → join …` with no
backoff. Because every Atlas topic consumer in a service shares one group ID,
22 members flapping would prevent the group from ever stabilising, which would
also strip the two *healthy* members of their assignments. That is strictly
worse than today's silent deafness.

The loop is not unbounded-fast: `nextGeneration` blocks on the unbuffered
`cg.next <- &gen` until our code calls `Next` (consumergroup.go:856-859), so
the storm is paced by `runGenerations`. That is the lever this design pulls in
§3.3.

### 2.4 Consequence for the design

`WatchPartitionChanges: true` is safe **only when we do not hold group
membership while the topic is absent**. So the fix must (a) not join while the
topic is missing, and (b) leave the group if the topic goes missing while
joined. The watch then becomes what the PRD wanted it to be: coverage for a
partition-count change on a topic that exists.

---

## 3. Architecture

Three cooperating parts, in `libs/atlas-kafka/consumer`:

| # | Part | Owns |
|---|---|---|
| 1 | **Partition-count seam** (`PartitionCountProducer`) | Answering "does this topic have partitions right now?" without a broker in tests. |
| 2 | **Pre-join topic-readiness gate** (`startGroupEngine`) | The cold-start case — the incident. Never join a group for a topic that does not exist. |
| 3 | **Zero-partition classification** (`runGenerations`) | The runtime case, and the observability the PRD asks for. Distinguish zero-partition from held-elsewhere; on zero-partition, exit the generation loop back to the gate. |

`WatchPartitionChanges: true` (FR-1.1) is enabled on top of all three, and is
load-bearing only for the partition-count *change* case (1→N) on an existing
topic; parts 2 and 3 carry the missing-topic case.

### 3.1 Part 1 — the partition-count seam

Resolves PRD open question 2 in favour of a producer function type, not an
extension of `Group`. `Group` is documented at `group.go:9-11` as a pure subset
of `*kafka.ConsumerGroup`; a metadata method has no counterpart there, and the
gate needs the lookup *before* a `Group` exists at all.

New in `group.go`:

```go
// ErrTopicNotFound reports that the broker knows no partitions for the topic:
// either UNKNOWN_TOPIC_OR_PARTITION, or a successful metadata response whose
// topic entry carries zero partitions. It is a definite negative — every other
// error is indeterminate (FR-2.5).
var ErrTopicNotFound = errors.New("topic not found or has no partitions")

// PartitionCountProducer reports the current partition count for a topic.
// Shaped as a producer function, like GroupProducer and
// PartitionReaderProducer, so both the pre-join gate and the empty-assignment
// classification can be scripted without a broker (FR-2.2).
type PartitionCountProducer func(ctx context.Context, brokers []string, topic string) (int, error)

func ConfigPartitionCountProducer(pcp PartitionCountProducer) ManagerConfig
```

Default implementation, `defaultPartitionCountProducer`, uses the modern client
rather than `Conn.ReadPartitions`:

```go
client := &kafka.Client{Addr: kafka.TCP(brokers...), Timeout: topicMetadataTimeout}
res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
```

Why `kafka.Client.Metadata` and not `kafka.Conn.ReadPartitions`:
`Conn.ReadPartitions` sends `topicMetadataRequestV6{… AllowAutoTopicCreation: true}`
(conn.go:984-986). A consumer must never create a topic as a side effect of
asking whether it exists. `Client.Metadata` builds `metadataAPI.Request` with
`TopicNames` only (metadata.go:40-44) and leaves `AllowAutoTopicCreation`
false. It also returns the per-topic error as a structured field
(`Topic.Error`, metadata.go:78-83) rather than collapsing the whole response
into one error, which is what lets us separate "no such topic" from "broker
unreachable".

Mapping:

| Metadata outcome | Result |
|---|---|
| topic present, `len(Partitions) >= 1` | `(n, nil)` |
| topic entry `Error` is `UnknownTopicOrPartition` | `(0, ErrTopicNotFound)` |
| topic present, `len(Partitions) == 0`, no error | `(0, ErrTopicNotFound)` |
| topic absent from the response entirely | `(0, ErrTopicNotFound)` |
| any other topic error, or transport/timeout error | `(0, err)` — **indeterminate** |

`topicMetadataTimeout = 5 * time.Second`, a named constant with the
justification comment FR-2.6 requires; it also bounds the `Client.Timeout`.

**Nil-seam default is "skip".** `Consumer.pcp` is populated from
`Manager.pcp` in `AddConsumer`, exactly like `gp`/`prp`
(`manager.go:190-191`). A `Consumer` built as a struct literal — which is how
every existing test in `engine_group_test.go`, `idle_stuck_test.go`,
`dwell_integration_test.go` and `group_test.go` builds one — has `pcp == nil`,
and a nil `pcp` means *gate disabled, classification indeterminate*. That is
what keeps the existing suite behaviourally unchanged without editing it, and
it is the same posture FR-2.5 mandates for a failed lookup.

### 3.2 Part 2 — the pre-join topic-readiness gate

In `startGroupEngine`, immediately before `c.gp(c.groupConfig())`:

```go
if !c.awaitTopic(l, ctx) {   // false == parent context cancelled
    return
}
grp, err := c.gp(c.groupConfig())
```

`awaitTopic` (new, `engine_group.go`):

- returns `true` immediately when `c.pcp == nil` (seam absent) or when the
  lookup returns `count >= 1`;
- returns `true` immediately when the lookup returns an **indeterminate** error
  — a broker outage must not hold a consumer out of its group, and FR-2.5
  forbids treating an error as zero;
- on `ErrTopicNotFound`: records the observation (§3.4), logs, sleeps a
  bounded backoff, and re-polls, until the topic appears or `ctx` is done.

Backoff reuses `newFetchBackoff()` (`manager.go:596-614`, 500 ms doubling to a
10 s cap). The ten-minute incident window costs ~60 metadata reads per
consumer — trivial next to the rebalance storm it replaces. Wait time is
attributed through the existing `recordBackoff` so it shows up in
`totalBackoffNs`.

Log cadence: **warn** on entering the wait, then **debug** per poll, with the
warn re-emitted every `topicMissingWarnInterval = 1 * time.Minute` while still
waiting, and an **info** on the transition to "topic appeared, joining". A
ten-minute outage produces ten warns per consumer rather than six hundred.

This is the piece that actually fixes the incident: those 22 consumers would
never have joined a group deaf. They would have waited out
`atlas-kafka-precreate`, joined at 18:12:41, taken partition 0, and drained the
backlog — with no membership churn at all during the window, because a
non-member cannot destabilise a group.

**FR-3.4 holds.** `AddConsumer` launches `con.start` on `routine.Go`
(`manager.go:199`) and returns; the gate runs entirely inside that goroutine.
It touches no readiness probe and blocks no HTTP server start. `wg.Add(1)`
already happens at the top of `startGroupEngine`, before the gate, so shutdown
still joins the goroutine correctly — hence the gate's `select` on `ctx.Done()`
is mandatory, not optional.

### 3.3 Part 3 — zero-partition classification in `runGenerations`

The empty-assignment branch (`engine_group.go:71-86`) splits three ways:

```go
if len(parts) == 0 {
    switch c.classifyEmptyAssignment(ctx) {
    case emptyBecauseTopicMissing:
        c.recordTopicMissing()
        l.Warnf("Consumer for topic [%s] (group [%s]) holds no partition assignment in "+
            "generation %d because the topic does not exist or has no partitions; "+
            "leaving the group until it appears.", c.topic, c.groupId, gen.ID())
        return errTopicMissing
    default: // emptyHealthyIdle and emptyIndeterminate
        l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle.",
            c.topic, c.groupId, gen.ID())
        continue
    }
}
```

`classifyEmptyAssignment` calls `c.pcp` once, under a
`context.WithTimeout(ctx, topicMetadataTimeout)`. Nil seam, or any error other
than `ErrTopicNotFound`, yields `emptyIndeterminate`. FR-2.6's "at most once
per generation" is structural: the branch is reached once per generation and
then either `continue`s into a parking `Next` or returns. There is no poll loop
here — the polling lives in the gate, which is a *non-member* state.

**Why `return errTopicMissing` and not `continue`.** §2.3: with the watch
enabled, a topic that disappears under a joined member makes kafka-go end each
generation the instant the watcher's startup read fails, and `run` rejoins with
no backoff. Because that loop is paced by our `Next` calls, returning is what
bounds it: `startGroupEngine` closes the group (leaving it), and the next
iteration of its loop hits the gate, which waits the topic out as a
non-member. One extra rejoin cycle, then out — instead of an unbounded storm.

`errTopicMissing` is a sentinel handled explicitly in `startGroupEngine`'s
post-`runGenerations` block: it is **not** passed to `recordError` and **not**
logged at Error with "exited; rejoining after backoff", because it is not an
error — it is a deliberate, already-warned withdrawal. Control falls through to
the top of the loop and into the gate.

**FR-2.4 is preserved exactly.** The held-elsewhere case reaches the same
`l.Debugf` with the same wording and the same `continue`. The one *added* cost
on that path is a single metadata read per generation on the idle replica, on a
rebalance — bounded, off the message path, and not observable in behaviour.

### 3.4 Part 4 — snapshot and debug surface

Resolves PRD open question 1 in favour of the **counter + timestamp pair**,
consistent with `IdleTicks`/`LastIdleTickAt` and
`NoProgressTicks`/`LastNoProgressAt`:

- `Consumer`: `topicMissingObservations int`, `lastTopicMissingAt time.Time`,
  under the existing `c.mu`.
- `recordTopicMissing()` — increments and stamps, mirroring
  `recordIdleTick` (`manager.go:506-513`). Called by both the gate (per warn,
  not per poll, so the counter reads as "distinct observations", not "poll
  count") and the classification branch.
- `Snapshot`: `TopicMissingObservations int`, `LastTopicMissingAt time.Time`.
- `debugAttributes` / `snapshotToAttributes` (`debug.go:74-97,105-135`):
  `topicMissingObservations`, `lastTopicMissingAt` — lowerCamelCase, matching
  the file.

**FR-3.3 supersession semantics.** The counter is monotonic and keeps its
history; it is *superseded*, not cleared. A consumer is currently healthy iff
`assignedPartitions` is non-empty, or `lastAssignmentAt` is after
`lastTopicMissingAt`. This is documented in `README.md` alongside the existing
`recreateCount` caveat, and it is the same read an operator already performs
for `idleTicks`. Choosing the counter over a self-clearing boolean keeps the
post-mortem signal: after the pod recovers, `topicMissingObservations: 10`
with a `lastTopicMissingAt` from 18:03 is exactly the evidence that was missing
from this incident.

### 3.5 Part 5 — `groupConfig()` and the README

`groupConfig()` sets `WatchPartitionChanges: true` (FR-1.1) and leaves
`PartitionWatchInterval` at kafka-go's default (FR-1.5 permits this;
`config.PartitionWatchInterval == 0` is defaulted to `defaultPartitionWatchTime`
at consumergroup.go:203-205). ID, Brokers, Topics and StartOffset are untouched
(FR-1.4), so `Validate()` (FR-1.3) and the `reader` rollback path are
unaffected. The `reader` engine keeps the flag false — it never builds a
`ConsumerGroupConfig` at all.

`TestGroupConfigMirrorsTodaysTopology` (`group_test.go:37-42`) inverts, and its
comment is rewritten to state the deliberate divergence: the legacy engine
inherits kafka-go's `ReaderConfig` default and cannot be changed without
touching the frozen rollback path, while the `consumergroup` engine enables the
watch as one of three parts of the task-267 fix — and is only safe doing so
because parts 2 and 3 keep it out of the group whenever the topic is absent.

The README's "Partition-count changes" section is rewritten to describe the new
behaviour, the 2026-08-26 `atlas-pr-1449` incident, the kafka-go
startup-read asymmetry from §2.1, why the watch is not the whole fix, and the
new snapshot fields' supersession rule.

### 3.6 Part 6 — character-factory logging

`handleCreateFromPreset` (`services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:57-70`)
gains one line before `w.WriteHeader(statusCode)`, matching its sibling at
line 104:

```go
d.Logger().WithError(err).Error("Error creating character from preset.")
```

The message differs from the seed handler's "Error creating character from
seed." so a log search separates them (FR-4.2). `categorizePresetError` and the
status mapping are untouched (FR-4.3). Independent of the library change and
carried on the same branch.

---

## 4. Alternatives considered and rejected

**A. `WatchPartitionChanges: true` alone (the PRD's minimal reading).**
Rejected on the §2 evidence. It replaces a silent single-consumer outage with a
group-wide rebalance storm that also breaks the members that *were* working.

**B. Patch or vendor kafka-go's `partitionWatcher`** so its startup read
tolerates `UnknownTopicOrPartition` like its ticker read does. This is the
correct upstream fix and is worth filing, but vendoring a fork of kafka-go for
14 services is a much larger blast radius than a gate in our own engine, and it
still leaves the member joined-but-deaf between the topic appearing and the
next 5 s tick. Not chosen; the upstream asymmetry is documented in the README
so a future kafka-go bump can revisit the gate.

**C. Throttle in the empty-assignment branch (sleep, then `continue`) instead
of leaving the group.** Works — §2.3 shows our `Next` paces kafka-go's rejoin
loop — but it keeps a doomed member in the group and forces one full group
rebalance per backoff interval for the whole outage window. Leaving is
strictly cheaper and reuses the gate we need anyway.

**D. Extend the `Group` interface with a metadata method** (PRD open question
2's second option). Rejected: it breaks the "pure subset of
`*kafka.ConsumerGroup`" invariant at `group.go:9-11`, forces every existing
`fakeGroup` to grow a method, and cannot serve the pre-join gate, which runs
before a `Group` exists.

**E. A self-clearing boolean for the snapshot field** (PRD open question 1's
second option). Rejected: it destroys the post-mortem signal, which is the
main thing this incident lacked, and it breaks the naming symmetry with the
two existing counter/timestamp pairs.

**F. Readiness-probe integration** — hold the pod `NotReady` until every topic
exists. Explicitly forbidden by FR-3.4, and it would have turned a ten-minute
degradation into a ten-minute `CrashLoopBackoff`/rollout stall.

---

## 5. Test strategy

All broker-free, on the existing `fakegroup_test.go` harness plus a scripted
`PartitionCountProducer`. New helper: `fakePartitionCounter`, a scriptable
`PartitionCountProducer` recording call count and returning a queued sequence
of `(count, error)` results, so a test can drive 0 → 0 → 1.

| Acceptance criterion | Test |
|---|---|
| `WatchPartitionChanges: true`, `Validate()` passes, comment rewritten | `TestGroupConfigMirrorsTodaysTopology` (inverted) |
| join → empty on zero-partition topic → topic gains partition 0 → consuming, no restart | `TestGroupEngineRecoversWhenTopicAppears` — full `startGroupEngine` run with a counter scripted `ErrTopicNotFound, ErrTopicNotFound, 1` and a `fakeGroup` whose first generation assigns partition 0; asserts a fetch happens |
| zero-partition logs **warn** naming topic, group, generation | `TestEmptyAssignmentWarnsWhenTopicMissing` via `hasLogContaining` on the hook, asserting `logrus.WarnLevel` |
| held-elsewhere still logs **debug**, no tick/recreate/rejoin | `TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist` — counter returns `(1, nil)`; asserts the debug wording, `recreateCount == 0`, `idleTicks == 0`, and `fakeGroup.nextCalls()` advances by exactly one (parked in `Next`, not rejoined) |
| failed lookup falls through to healthy-idle (FR-2.5) | `TestEmptyAssignmentIndeterminateLookupIsHealthyIdle` — counter returns a transport error; asserts debug wording and that `runGenerations` did **not** return |
| steady partition count triggers no additional rebalance | `TestSteadyPartitionCountCausesNoRejoin` — counter always `(1, nil)` across two scripted generations; asserts `nextCalls()` matches the scripted generation count exactly and `recreateCount == 0`. (kafka-go's own watcher goroutine is not under test here — it is upstream-owned and broker-bound; what we own and therefore assert is that *our* engine adds no rejoin.) |
| snapshot exposes the fields; debug renders lowerCamelCase | extend `debug_test.go`'s existing attribute assertions |
| snapshot reflects the observation and its post-recovery value | `TestSnapshotTopicMissingSupersededByAssignment` — after recovery, `topicMissingObservations` retains its count and `lastAssignmentAt` is after `lastTopicMissingAt` |
| gate does not join while topic missing | `TestGateDoesNotJoinUntilTopicExists` — asserts the `GroupProducer` is not called until the counter returns `>= 1` |
| gate exits on context cancel | `TestGateExitsOnContextCancel` — asserts `wg` is released and no group is created |
| nil seam is inert | `TestNilPartitionCountProducerSkipsGate` — pins the §3.1 default that keeps the existing suite green |
| preset handler logs at Error with a preset-specific message; status mapping unchanged | handler test in `services/atlas-character-factory/.../factory` using a `test.Hook`, plus a table test over `categorizePresetError` |
| `tools/verify.sh` (flagless) exits 0 | Phase-4 gate |

The recovery test (`TestGroupEngineRecoversWhenTopicAppears`) is the one that
would have caught the incident, and it is written against `startGroupEngine` —
the whole engine — not against `runGenerations` alone, because the gate is
where recovery now lives.

---

## 6. Risks

| Risk | Mitigation |
|---|---|
| Every consumer now issues metadata reads at startup and on each idle rebalance. | One read, 5 s timeout, off the message path. A healthy service issues one per consumer at boot. |
| A metadata blip is misread as "topic gone" and the consumer leaves the group. | Only `ErrTopicNotFound` — a definite negative from a *successful* metadata exchange, or an explicit `UnknownTopicOrPartition` — triggers the leave. Every other error is indeterminate and falls through (FR-2.5). |
| `WatchPartitionChanges: true` causes rebalances on a repartition. | Intended, and Atlas never repartitions in steady state. The dangerous interaction (missing topic) is neutralised by §3.2/§3.3. |
| The gate delays a consumer's first message in a slow-broker environment. | The gate only waits on a definite `ErrTopicNotFound`; a slow or unreachable broker is indeterminate and joins immediately, as today. |
| Existing tests build `Consumer` literals and would hit a nil seam. | Nil `pcp` is defined as "gate disabled, classification indeterminate" (§3.1); the existing suite is behaviourally unchanged. |
| `errTopicMissing` leaks into `lastError` / the Error log. | Handled as an explicit sentinel case in `startGroupEngine`, ahead of `recordError`; covered by `TestEmptyAssignmentWarnsWhenTopicMissing` asserting `lastError == ""`. |

---

## 7. PRD open questions — resolution

1. **Snapshot field semantics** → counter + timestamp pair
   (`TopicMissingObservations` / `LastTopicMissingAt`), superseded rather than
   cleared, rule documented in the README. §3.4.
2. **Seam shape** → `PartitionCountProducer` function type; `Group` stays a
   pure subset of `*kafka.ConsumerGroup`. §3.1, §4-D.
3. **Does the watch alone close the outage?** → **No.** It ends the generation
   immediately and rejoins with no backoff. §2.1-2.3.
4. **Does the watch fire at all on a zero-partition topic?** → **No.** Its
   startup read returns `UnknownTopicOrPartition` and the watcher gives up —
   the very error its own ticker branch tolerates. §2.1-2.2.

Nothing remains open. No external blocker.

---

## 8. Files touched

| File | Change |
|---|---|
| `libs/atlas-kafka/consumer/group.go` | `WatchPartitionChanges: true`; `ErrTopicNotFound`, `PartitionCountProducer`, `ConfigPartitionCountProducer`, `defaultPartitionCountProducer`, `topicMetadataTimeout` |
| `libs/atlas-kafka/consumer/engine_group.go` | `awaitTopic` gate; `classifyEmptyAssignment`; three-way empty-assignment branch; `errTopicMissing` sentinel handling |
| `libs/atlas-kafka/consumer/manager.go` | `Manager.pcp`, default wiring in `GetManager`, `Consumer.pcp` populated in `AddConsumer`; `topicMissingObservations`/`lastTopicMissingAt`; `recordTopicMissing`; `Snapshot` fields; `topicMissingWarnInterval` |
| `libs/atlas-kafka/consumer/debug.go` | Two new attributes |
| `libs/atlas-kafka/consumer/group_test.go` | Invert `WatchPartitionChanges` assertion + rewrite comment |
| `libs/atlas-kafka/consumer/fakegroup_test.go` | `fakePartitionCounter` helper |
| `libs/atlas-kafka/consumer/engine_group_test.go`, `debug_test.go` | New tests per §5 |
| `libs/atlas-kafka/README.md` | Rewrite "Partition-count changes"; document the new fields |
| `services/atlas-character-factory/.../factory/resource.go` | One log line |
| `services/atlas-character-factory/.../factory/*_test.go` | Handler log + status-mapping test |

No service outside `atlas-character-factory` changes; every service inherits the
library fix and must still build and test clean.
