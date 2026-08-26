# Kafka Consumer Recovery From a Late-Appearing Topic — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26
---

## 1. Overview

A `libs/atlas-kafka` consumer running the `consumergroup` engine that joins its
group before its subscribed topic exists is permanently deaf. kafka-go hands a
group member subscribed to a zero-partition topic an empty assignment;
`libs/atlas-kafka/consumer/group.go:115` pins `WatchPartitionChanges: false`, so
the member is never told the topic later gained a partition, and
`libs/atlas-kafka/consumer/engine_group.go:71-86` classifies the empty
assignment as *healthy-idle*, logs it at debug, and `continue`s into a `Next`
that parks until the generation ends. Nothing ends that generation. The consumer
never fetches a message again until the pod restarts.

This was observed in `atlas-pr-1449` (ATLAS_ENV `1450`) on 2026-08-26. The
service pods came up at ~18:02:52; the `atlas-kafka-precreate` Job for that
environment reconciled its 170 topics at 18:12:41, ten minutes later. 22 of ~24
topic consumers in the `Saga Orchestrator Service [1450]` group logged
"holds no partition assignment in generation 1; healthy-idle" and never logged
an assignment. The only two that did hold partition 0 were the two
configuration-projection topics that already existed at join time.
`atlas-character-factory` accepted `POST /api/factory/characters/from-preset`,
returned 202 with a transaction id, and produced onto `COMMAND_TOPIC_SAGA-1450`
successfully — the topic's log-end offset reached 4 with nothing consumed.
`atlas-saga-orchestrator` logged nothing for any of those transactions. A
`kubectl rollout restart` at 18:15:29 fixed it instantly: the new pod took
partition 0 and drained all four backlogged sagas at 18:15:30. The reported
symptom ("I cannot create a character via the ui") was the most visible instance
of a whole-environment outage — every saga-driven flow in that environment was
equally dead.

The *healthy-idle* classification is correct for the case it was designed for:
Atlas topics are single-partition and services run `replicas: 2`, so exactly one
member of every group legitimately holds nothing and must not tick, warn, or
recreate — recreating is what rejoins the group and rebalances every other topic
in it. The defect is that the branch cannot distinguish that case from "this
topic has zero partitions," so it silently absorbs a permanent outage with no
warn, no metric, and no self-heal. This task makes the two cases distinguishable
and makes the second one recover on its own.

## 2. Goals

Primary goals:

- A consumer-group member whose subscribed topic gains its first partition after
  the member joined acquires that partition and begins consuming, without a pod
  restart and without operator intervention.
- The zero-partition case is distinguishable from the legitimate
  another-member-holds-it case in both logs and the consumer debug snapshot, so
  an environment in this state can be triaged without reading pod logs.
- The legitimate healthy-idle case keeps its existing behaviour exactly: no
  tick, no warn, no recreate, no group rejoin.
- `handleCreateFromPreset` in `atlas-character-factory` logs its errors, so a
  future preset-creation failure is visible.

Non-goals:

- Changing the Argo sync-wave deploy ordering. The gate already exists
  (`deploy/k8s/base/atlas-kafka-precreate.yaml:38` at wave 0; every Deployment
  patched to wave 10 at `deploy/k8s/base/kustomization.yaml:89-102`) and is
  correct for a sync that rolls the Deployments. It structurally cannot cover a
  re-sync that recreates topics under already-running pods, which is what this
  library fix addresses. Any deploy-side hardening is a separate task.
- Auditing whether `atlas-main` can hit the same window on a cold start. The
  library fix makes the question moot for recovery purposes; the investigation
  is deliberately left out of this task's scope.
- Changing the `reader` (legacy) engine. It is the rollback target and is
  frozen.
- Repartitioning any topic, or changing any topic's partition count.
- `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`. The originally-reported "unsuffixed
  topic" is a false alarm: all three overlays suffix it
  (`deploy/k8s/overlays/pr/kustomization.yaml:323`,
  `main/kustomization.yaml:203`, `pr-sparse/kustomization.yaml:489`), and the
  unsuffixed literal at
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go:118`
  is the env-*var name*, not the topic value — the same shape as every other
  topic in that service.

## 3. User Stories

- As an Atlas operator, I want a service whose topics were created after it
  started to begin consuming on its own, so that a PR environment is not silently
  dead until someone notices and restarts a pod.
- As an Atlas operator triaging a wedged environment, I want
  `GET /api/debug/consumers` to tell me *why* a consumer holds no partitions, so
  I can distinguish "this replica is the idle one" from "this topic does not
  exist."
- As an Atlas developer, I want a preset character-creation failure to appear in
  `atlas-character-factory`'s logs, so a 4xx/5xx from that endpoint is
  diagnosable.
- As an Atlas developer relying on the existing two-replica topology, I want the
  idle replica of every group to keep behaving exactly as it does today, so this
  fix does not introduce rebalance churn.

## 4. Functional Requirements

### 4.1 Partition-change watching

- **FR-1.1** `Consumer.groupConfig()` (`libs/atlas-kafka/consumer/group.go:107-116`)
  MUST set `WatchPartitionChanges: true` for the `consumergroup` engine.
- **FR-1.2** The change MUST be recorded as a deliberate decision, not a silent
  edit. Specifically:
  - `libs/atlas-kafka/consumer/group_test.go:37-42`
    (`TestGroupConfigMirrorsTodaysTopology`) currently asserts the flag stays
    `false` "to match the legacy engine". That assertion MUST be inverted, and
    its comment rewritten to state why the two engines now deliberately differ.
  - The "Partition-count changes" section of `libs/atlas-kafka/README.md`
    (lines ~30-40) currently states that neither engine watches, and that
    enabling the flag is "a deliberate opt-in, not something to enable as a side
    effect of another change." That section MUST be rewritten to describe the new
    behaviour, the incident that motivated it, and the fact that the `reader`
    engine still does not watch.
- **FR-1.3** The resulting config MUST still pass `kafka.ConsumerGroupConfig.Validate()`.
- **FR-1.4** Group ID, broker list, subscribed-topic list, and `StartOffset`
  MUST be unchanged, so broker-visible group topology and the rollback path to
  the `reader` engine are unaffected.
- **FR-1.5** `PartitionWatchInterval` MAY be left at kafka-go's default (5s). If
  it is set explicitly, the value MUST be a named constant with a comment
  justifying it.

### 4.2 Zero-partition classification

- **FR-2.1** On entering the empty-assignment branch of `runGenerations`
  (`libs/atlas-kafka/consumer/engine_group.go:71-86`), the consumer MUST
  determine whether the subscribed topic currently has zero partitions, as
  distinct from having partitions all held by other members.
- **FR-2.2** The partition-count lookup MUST go through an injectable seam (in
  the style of the existing `GroupProducer` / `PartitionReaderProducer`
  producers in `libs/atlas-kafka/consumer/group.go:33-54`) so the two cases can
  be scripted in tests without a broker, consistent with the design that made
  `Group`/`Generation` interfaces.
- **FR-2.3** When the topic has zero partitions, the consumer MUST log at
  **warn**, naming the topic, the group, and the generation, and stating that
  the topic does not exist or has no partitions.
- **FR-2.4** When the topic has one or more partitions and this member holds
  none, behaviour MUST be byte-for-byte what it is today: a **debug** log with
  the existing healthy-idle wording, then `continue`. No tick, no warn, no
  recreate, no rejoin.
- **FR-2.5** A failure of the partition-count lookup itself (broker unreachable,
  metadata timeout) MUST NOT be reported as zero partitions and MUST NOT crash
  or exit the generation loop. It MUST be treated as indeterminate: log at debug
  and fall through to the existing healthy-idle behaviour, leaving recovery to
  FR-1.1's partition watch.
- **FR-2.6** The lookup MUST be bounded by a timeout and MUST NOT be issued more
  than once per generation — the branch is entered once per generation and parks
  in `Next`, so this is a natural bound, but it MUST NOT be moved into a poll
  loop.

### 4.3 Observability

- **FR-3.1** `Snapshot` (`libs/atlas-kafka/consumer/`) MUST gain a field
  recording that the consumer observed its topic with zero partitions, and when.
  Naming and shape follow the existing `IdleTicks` / `LastIdleTickAt` and
  `NoProgressTicks` / `LastNoProgressAt` pairs.
- **FR-3.2** `debugAttributes` and `snapshotToAttributes` in
  `libs/atlas-kafka/consumer/debug.go:74-97,105-135` MUST expose the new field(s)
  in the `GET /api/debug/consumers` JSON:API response, with lowerCamelCase JSON
  tags matching the file's existing convention.
- **FR-3.3** The field MUST be cleared or superseded once the consumer receives a
  non-empty assignment, so a recovered consumer does not read as still-broken.
  The exact semantics (counter that keeps its history vs. boolean that clears)
  are a design decision; whichever is chosen MUST be documented in the README
  alongside the existing `recreateCount` caveat.
- **FR-3.4** The consumer MUST NOT gain a readiness-probe or health-endpoint
  dependency on this state. A slow `atlas-kafka-precreate` must not hold every
  pod `NotReady`.

### 4.4 Character-factory error logging

- **FR-4.1** `handleCreateFromPreset`
  (`services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:57-70`)
  MUST log the error before writing the status code, matching its sibling
  `handleCreateCharacter` at line 104
  (`d.Logger().WithError(err).Error(...)`).
- **FR-4.2** The log message MUST distinguish preset creation from seed creation
  ("Error creating character from preset."), so the two handlers are separable in
  a log search.
- **FR-4.3** The mapped status code from `categorizePresetError`
  (`resource.go:33-53`) MUST be unchanged. This requirement is a logging fix
  only; no status-code or response-body semantics change.

## 5. API Surface

No new or modified service endpoints.

One modified debug response: `GET /api/debug/consumers` (served by
`Manager.DebugHandler()`, `libs/atlas-kafka/consumer/debug.go:15-56`) gains
attribute(s) per FR-3.2. The response remains a JSON:API document of
`type: "consumers"` resources keyed by topic; the change is purely additive, so
any existing consumer of that endpoint is unaffected.

Illustrative shape (exact names decided at design time):

```json
{
  "data": [
    {
      "type": "consumers",
      "id": "COMMAND_TOPIC_SAGA-1450",
      "attributes": {
        "topic": "COMMAND_TOPIC_SAGA-1450",
        "groupId": "Saga Orchestrator Service [1450]",
        "assignedPartitions": [],
        "generationId": 1,
        "engine": "consumergroup",
        "topicMissingObservations": 1,
        "lastTopicMissingAt": "2026-08-26T18:02:58.233Z"
      }
    }
  ]
}
```

`POST /api/factory/characters/from-preset` is unchanged on the wire. FR-4.1 adds
a server-side log line only.

## 6. Data Model

No persisted entities, no schema migration, no tenant-scoped data. All new state
is in-process, per-`Consumer`, and lives only in the existing `Snapshot`
structure.

## 7. Service Impact

| Component | Change |
| --- | --- |
| `libs/atlas-kafka/consumer/group.go` | `WatchPartitionChanges: true` (FR-1.1); new injectable partition-count producer seam (FR-2.2). |
| `libs/atlas-kafka/consumer/group_test.go` | Invert the `WatchPartitionChanges` assertion and rewrite its comment (FR-1.2). |
| `libs/atlas-kafka/consumer/engine_group.go` | Split the empty-assignment branch into zero-partition (warn) and healthy-idle (debug, unchanged) (FR-2.1–2.6). |
| `libs/atlas-kafka/consumer/debug.go` | Expose the new snapshot field(s) (FR-3.2). |
| `libs/atlas-kafka` snapshot/state source | New field(s) and their recording path (FR-3.1, FR-3.3). |
| `libs/atlas-kafka/README.md` | Rewrite "Partition-count changes"; document the new snapshot field (FR-1.2, FR-3.3). |
| `services/atlas-character-factory/.../factory/resource.go` | Log the error in `handleCreateFromPreset` (FR-4.1–4.3). |
| **Every service consuming `libs/atlas-kafka`** | Inherits the fix through the shared library. No per-service code change, but every service's `go.work`-resolved build is affected and must still build and test clean. |

## 8. Non-Functional Requirements

- **Performance.** The partition-count lookup is one metadata read per
  generation per consumer, only on the empty-assignment path. A group with N
  topic consumers where the environment is healthy issues at most one such read
  per idle consumer per rebalance. `PartitionWatchInterval` adds a background
  metadata poll per consumer-group member; the default is 5s.
- **Rebalance churn.** `WatchPartitionChanges: true` makes a partition-count
  change trigger a rebalance. Atlas topics are single-partition and are never
  repartitioned in steady state, so in a healthy environment the watch should
  fire only on topic creation. This must be argued explicitly at design time and
  covered by a test that a steady partition count triggers no rebalance.
- **Backward compatibility.** No group ID, offset, or topic migration. Switching
  back to `KAFKA_CONSUMER_ENGINE=reader` (`libs/atlas-kafka/consumer/engine.go:24-26`)
  remains a pod restart with no state migration; the `reader` engine keeps
  `WatchPartitionChanges` false.
- **Observability.** Warn-level only for the genuinely broken case. The
  healthy-idle case stays at debug — Atlas runs `replicas: 2` against
  single-partition topics, so a warn there would fire on one member of every
  group in every environment permanently, which is why it is debug today.
- **Multi-tenancy.** None of this is tenant-scoped; the consumer layer is below
  tenant context. The debug endpoint is already documented as tenant-agnostic and
  internal-network-only (`libs/atlas-kafka/consumer/debug.go:10-14`), and stays
  so.
- **Testing.** New behaviour is covered without a broker, using the existing
  `fakegroup_test.go` scripted-generation harness plus the new partition-count
  seam. Both branches — zero-partition and partitions-held-elsewhere — need a
  test asserting the log level and the snapshot field, and there must be a test
  that a consumer parked on a zero-partition topic acquires the partition once
  the count changes.

## 9. Open Questions

1. **Snapshot field semantics (FR-3.3):** monotonic counter with a `LastAt`
   timestamp (matching `IdleTicks`/`NoProgressTicks`, preserves history across a
   recovery) versus a boolean that clears on the next non-empty assignment
   (reads unambiguously "broken right now"). Design decision; the counter form is
   more consistent with the file's existing pairs.
2. **Partition-count seam shape (FR-2.2):** a `PartitionCountProducer` function
   type mirroring the existing producers, versus extending the `Group` interface
   with a metadata method. The former keeps the `Group` interface a pure subset
   of `*kafka.ConsumerGroup` as documented at `group.go:9-11`.
3. **Does `WatchPartitionChanges: true` alone already close the outage?** If
   kafka-go's partition watch reliably rebalances a member subscribed to a topic
   that goes 0→1 partitions, FR-2.x becomes observability rather than self-heal.
   This must be established empirically at design time by reading kafka-go's
   watch implementation, not assumed — and it changes whether FR-2.3's warn needs
   an accompanying recovery action.
4. **Does the partition watch fire at all on a topic with zero partitions?** A
   watcher that reads partition count via metadata for a nonexistent topic may
   error rather than report zero; if it errors and gives up, the watch does not
   help and FR-2.x carries the whole fix. Same rule: read kafka-go, do not assume.

## 10. Acceptance Criteria

- [ ] `Consumer.groupConfig()` returns `WatchPartitionChanges: true`, and
      `cfg.Validate()` passes.
- [ ] `TestGroupConfigMirrorsTodaysTopology` asserts `true` with a rewritten
      comment explaining the deliberate divergence from the `reader` engine.
- [ ] `libs/atlas-kafka/README.md`'s "Partition-count changes" section describes
      the new behaviour and cites the 2026-08-26 `atlas-pr-1449` incident.
- [ ] A test drives a consumer through: join → empty assignment on a
      zero-partition topic → topic gains partition 0 → consumer holds partition 0
      and fetches, with no pod restart and no manual intervention.
- [ ] A test asserts the zero-partition case logs at **warn** naming topic,
      group and generation.
- [ ] A test asserts the partitions-exist-but-held-elsewhere case still logs at
      **debug** with the existing healthy-idle wording, and performs no tick, no
      recreate and no rejoin.
- [ ] A test asserts a failed partition-count lookup falls through to
      healthy-idle rather than being reported as zero partitions (FR-2.5).
- [ ] A test asserts a steady partition count produces no additional rebalance.
- [ ] `Snapshot` exposes the zero-partition field(s), and
      `GET /api/debug/consumers` renders them with lowerCamelCase tags.
- [ ] A test asserts the debug snapshot field reflects the zero-partition
      observation and its post-recovery value per the FR-3.3 decision.
- [ ] `handleCreateFromPreset` logs `WithError(err)` at Error level with a
      preset-specific message, and `categorizePresetError`'s status mapping is
      unchanged (covered by an existing or new handler test).
- [ ] No readiness-probe or health-endpoint change (FR-3.4).
- [ ] The `reader` engine path is untouched.
- [ ] `tools/verify.sh` (flagless) exits 0.
