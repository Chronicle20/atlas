# Compacted Config-Status Topics Never Actually Compact — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26
---

## 1. Overview

`atlas-kafka-precreate` classifies the three configuration-projection topics —
`EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`,
and `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` — as compacted
(`internal/discover/discover.go:25-29`) and applies `cleanup.policy=compact` to
them at creation time via `CreateTopics` `ConfigEntries` and to pre-existing
topics via a batched `IncrementalAlterConfigs` (`internal/topics/topics.go:53-128`).
That policy is correctly set on the broker. What the tool does **not** set is any
segment-rolling or compaction-lag configuration, and without it the policy is
inert: Kafka's log cleaner never touches the **active** segment, so a topic whose
segment never rolls is never compacted, no matter what its `cleanup.policy` says.

With broker defaults (`segment.bytes=1GB`, `segment.ms=7d`) and
`atlas-configurations` republishing the baseline environment record every 30s
(`environments/heartbeat.go:18-32`), the environment-status topic accumulates
~2880 records/day of ~270 bytes each — three orders of magnitude short of rolling
a 1 GB segment, and the 7-day timer rolls it only once a week. Observed on the
live cluster (`apache/kafka:4.1.1`, `kafka-broker-0`) on 2026-08-26:

```
EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main:0:36026   (end offset)
EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main:0:0       (log start offset)

$ ls -la /var/lib/kafka/data/kafka/EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main-0
-rw-rw-r-- 1 appuser root 10485760 Aug 26 14:49 00000000000000000000.index
-rw-rw-r-- 1 appuser root  9789992 Aug 26 14:50 00000000000000000000.log
-rw-rw-r-- 1 appuser root 10485756 Aug 26 14:49 00000000000000000000.timeindex

$ kafka-configs.sh --entity-type topics --entity-name ...-main --describe
cleanup.policy=compact sensitive=false synonyms={DYNAMIC_TOPIC_CONFIG:cleanup.policy=compact, DEFAULT_CONFIG:log.cleanup.policy=delete}
```

One segment, never rolled, log-start-offset still 0, 36026 records retaining
roughly three distinct keys.

The user-visible cost is startup latency. Every service wired with
`WithConfigProjection` replays the compacted log from `FirstOffset` under a fresh
per-process consumer group id on every container start
(`libs/atlas-service/projection.go:71-77`), and gates readiness on
`AwaitProjectionCatchUp`. Sparse PR environments consume the **baseline**
environment topic — `deploy/k8s/overlays/pr-sparse/kustomization.yaml:443` resolves
to `...-PLACEHOLDER_BASELINE_ENVIRONMENT`, i.e. `-main` — so every sparse PR env's
channel and login pods replay all 36k records before reporting ready. Measured on
`atlas-pr-1449`: `atlas-channel` streamed the backlog from 14:50:16.773 to
14:50:39.136 at roughly 5.5 ms/record and reached `1/1` at 5m07s of pod age.

This task makes compaction actually happen by giving the compacted topics the
segment-rolling and compaction-lag configuration the cleaner needs.

## 2. Goals

Primary goals:

- Make `cleanup.policy=compact` effective on the three config-status topics, so
  the retained log converges to roughly one record per key rather than growing
  without bound.
- Bound the uncompacted tail to a small, known window so projection catch-up —
  and therefore `atlas-channel` / `atlas-login` readiness in sparse PR
  environments — is fast and does not degrade as the cluster ages.
- Converge topics that already exist and are already overgrown, without operator
  intervention, on the next sync-wave-0 Job run.

Non-goals:

- Changing the cleanup policy, retention, or segment configuration of the plain
  (non-config-status) topic set. Those topics are event streams under DELETE
  cleanup and are out of scope.
- Changing the 30s cadence of `StartHeartbeat` in `atlas-configurations`. The
  cadence is load-bearing for the `StaleAfter` liveness check; compaction is the
  correct fix for the volume it produces.
- Changing the projection subscriber's replay model (per-process group id,
  `FirstOffset`, debug-level per-message logging) in `libs/atlas-service` or the
  per-service `configuration/projection` packages.
- Changing the deploy overlays. The configuration is applied programmatically by
  the tool; no `kustomization.yaml` or ConfigMap change is expected.
- Adding new topic-classification categories. The compacted set stays exactly the
  three variables in `compactVars`.

## 3. User Stories

- As a developer opening a PR, I want my sparse PR environment's channel and login
  pods to become ready promptly, so that I am not waiting minutes on config
  projection replay before I can test my change.
- As an operator, I want the config-status topics to stay small indefinitely, so
  that disk usage and every consumer's cold-start cost do not grow linearly with
  cluster uptime.
- As an operator, I want the existing overgrown `-main` topics to be repaired by
  the normal sync-wave-0 Job run, so that I do not have to run manual
  `kafka-configs.sh` commands against a production broker.
- As a developer reading `atlas-kafka-precreate`, I want the compaction
  configuration and the reason each knob exists to be documented at the point of
  use, so that the next person does not re-introduce an inert `cleanup.policy`.

## 4. Functional Requirements

### 4.1 Compaction configuration

- **FR-1.1** Every topic in `Topics.Compact` MUST be created with, and converged
  to, the following topic-level configuration in addition to the existing
  `cleanup.policy=compact`:

  | Config | Value | Purpose |
  | --- | --- | --- |
  | `max.compaction.lag.ms` | `600000` (10 min) | Upper bound on how long a record may remain uncompacted. Since KIP-354 the cleaner force-rolls the active segment to honor this, which is the specific mechanism the current configuration lacks. |
  | `segment.ms` | `600000` (10 min) | Time-based segment roll, independent of the cleaner. Belt-and-braces: guarantees a rollable segment even if the KIP-354 force-roll path does not apply. |
  | `min.cleanable.dirty.ratio` | `0.01` | Lets the cleaner select a segment whose dirty fraction is small. Without it a rolled segment of near-duplicate records can still sit below the 0.5 default threshold. |

- **FR-1.2** The values MUST be declared as named constants in
  `internal/topics/topics.go` alongside the existing `compactCleanupPolicy`, not
  as inline string literals at the call sites.

- **FR-1.3** The configuration MUST be applied in **both** paths that already
  carry `cleanup.policy`:
  1. the `kafka.TopicConfig.ConfigEntries` of the `CreateTopics` request, so a
     newly created topic is correct from its first record; and
  2. the `IncrementalAlterConfigs` resource list, so a topic created before this
     change converges.

  The two paths MUST derive from a single shared declaration of the config set —
  a divergence between them is exactly the defect class this task fixes.

- **FR-1.4** The `IncrementalAlterConfigs` request MUST continue to use
  `ConfigOperationSet` on the incremental API, never the legacy full-replace
  `AlterConfigs`, preserving the existing rationale at
  `internal/topics/topics.go:87-100`: full-replace would reset unrelated
  topic-level overrides an operator set by hand.

- **FR-1.5** The alter path MUST continue to run unconditionally over the whole
  compacted set rather than only over topics `CreateTopics` just created. This is
  the mechanism by which the already-overgrown `-main` topics self-heal
  (see FR-3.1).

- **FR-1.6** All four configs (including `cleanup.policy`) for all compacted
  topics MUST still be applied in a single `IncrementalAlterConfigs` request, as
  today. The request carries one resource per topic; the added configs extend
  each resource's `Configs` list, not the resource count.

- **FR-1.7** The plain topic set MUST be unaffected. A topic in `Topics.Plain`
  continues to be created with only `NumPartitions` and `ReplicationFactor` and
  MUST NOT appear in the alter request.

### 4.2 Error handling

- **FR-2.1** A per-resource error in the `IncrementalAlterConfigs` response MUST
  remain fatal and MUST name the topic, as today
  (`internal/topics/topics.go:120-127`). Adding configs does not soften this.

- **FR-2.2** If the broker rejects one of the newly added config keys — for
  example an older broker that does not know `max.compaction.lag.ms` — the Job
  MUST fail loudly with the broker's error rather than silently continuing with a
  partially applied config. A compacted topic that is silently left uncompacted
  is the bug being fixed; it must not be reachable through a swallowed error.

- **FR-2.3** Applying a config that is already set to the same value MUST remain a
  no-op on the broker and MUST NOT be treated as an error, preserving the existing
  idempotence of a repeated Job run.

### 4.3 Remediation of existing topics

- **FR-3.1** No new remediation phase, flag, or operator step is introduced. The
  existing unconditional alter (FR-1.5) is the remediation: the next sync-wave-0
  Job run sets the new configs on the pre-existing `-main` topics, `segment.ms`
  then rolls the 9.8 MB active segment, and the cleaner compacts it under the
  lowered dirty ratio.

- **FR-3.2** Design MUST verify this self-healing claim against the live topic
  rather than assume it, since the topic's single segment has a first-message
  timestamp older than the new `segment.ms` and the roll-on-append behavior for
  that case is the load-bearing assumption. If verification shows the segment does
  not roll, the design MUST surface that and propose a remediation step rather
  than silently shipping a fix that leaves `-main` broken.

### 4.4 Documentation

- **FR-4.1** The `services/atlas-kafka-precreate/README.md` environment-variable
  table entry for `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` and the "Create/Alter" phase
  description MUST be updated to state the full compacted-topic configuration, not
  just `cleanup.policy=compact`.

- **FR-4.2** The comment block on the compacted config set MUST explain *why*
  each knob exists — specifically that `cleanup.policy` alone is inert because the
  cleaner never processes the active segment. A future reader must not be able to
  conclude the extra knobs are redundant.

## 5. API Surface

No HTTP or JSON:API surface. `atlas-kafka-precreate` is a Job with no server.

The only external contract that changes is the Kafka admin protocol request
bodies:

- `CreateTopics` — each compacted topic's `ConfigEntries` grows from one entry to
  four. Request count unchanged (one).
- `IncrementalAlterConfigs` — each compacted topic's resource `Configs` list grows
  from one entry to four. Request count and resource count unchanged.

No new RPC is issued; the tool's two-RPC no-groups short circuit
(README "Phases", step 2) is preserved.

## 6. Data Model

No database, entity, or migration. The tool holds no persistent state.

The in-memory model changes only insofar as FR-1.3 requires a single shared
declaration of the compacted-topic config set feeding both the create and alter
paths. Whether that is a `[]kafka.ConfigEntry` with a projection to
`[]kafka.IncrementalAlterConfigsRequestConfig`, or an internal slice of
`(name, value)` pairs projected into both, is a design decision.

`discover.Topics` is unchanged: the `Plain` / `Compact` split already carries all
the classification this task needs.

## 7. Service Impact

| Service | Change |
| --- | --- |
| `services/atlas-kafka-precreate` | `internal/topics/topics.go`: new config constants, shared config-set declaration, both request builders extended. `internal/topics/topics_test.go`: assertions extended. `README.md`: documentation. |
| `deploy/k8s/**` | None expected. The configuration is applied programmatically; no overlay, ConfigMap, or Job manifest change. |
| `atlas-configurations` | None. The heartbeat is explicitly out of scope (§2). |
| `libs/atlas-service`, per-service `configuration/projection` | None. Consumers benefit from a shorter log; no code change. |

Runtime effect on already-deployed environments: the next `atlas-kafka-precreate`
Job run in each namespace converges that namespace's compacted topics. Sparse PR
envs point at the baseline topic, so the `atlas-main` Job run is what repairs the
topic the PR envs read.

## 8. Non-Functional Requirements

- **NFR-1 (Performance).** After the change, the retained record count on
  `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main` MUST be bounded by roughly
  one record per key plus one 10-minute window of heartbeats (~20 records at the
  30s cadence), rather than growing unbounded. Projection catch-up on a cold
  container start should complete in well under a second of broker read time,
  against the observed minutes today.
- **NFR-2 (Segment churn).** A 10-minute `segment.ms` implies up to 144 segment
  rolls per day per compacted topic. With three compacted topics per environment
  this is a small, bounded number of files; design should confirm it is acceptable
  against the broker's open-file and index-allocation behavior, noting that each
  rolled segment preallocates `segment.index.bytes` (10 MB by default on this
  cluster, per the observed `.index`/`.timeindex` sizes).
- **NFR-3 (Idempotence).** Repeated Job runs MUST remain safe and side-effect-free
  beyond converging config, preserving the tool's existing "create whether or not
  there is a group to seed" contract.
- **NFR-4 (Multi-tenancy).** Topic names are already environment-scoped by the
  overlay-substituted suffix; this task introduces no tenant-scoped state and no
  new cross-environment coupling. It does not change the existing fact that sparse
  PR envs read the baseline environment's topic.
- **NFR-5 (Observability).** The Job's existing summary logging MUST report the
  compacted-topic count as it does today. Design may extend the log line to name
  the applied config set; it MUST NOT reduce what is logged.
- **NFR-6 (Testability).** Per FR-7.1 of task-260, the config set MUST be
  assertable without a live broker — the fake `AdminClient` in
  `internal/kafkaops` already captures request bodies.

## 9. Open Questions

1. **KIP-354 force-roll on `apache/kafka:4.1.1`.** The claim that
   `max.compaction.lag.ms` force-rolls the active segment needs confirmation
   against this broker version's actual behavior. `segment.ms` (FR-1.1) is the
   fallback if it does not, but design should establish which mechanism is
   actually doing the work rather than shipping both and assuming.
2. **Self-healing of the existing 9.8 MB segment (FR-3.2).** Whether a segment
   whose first-message timestamp predates the new `segment.ms` rolls on the next
   append, or only after further records accumulate, determines how quickly the
   `-main` topic actually shrinks. Verifiable directly on the live cluster.
3. **Index preallocation cost.** The observed segment directory shows 10 MB
   `.index` and 10 MB `.timeindex` files for a single segment. If each roll
   preallocates that, 144 rolls/day per topic has a disk-footprint implication
   worth checking before settling on 10 minutes; a design finding here may argue
   for also lowering `segment.index.bytes` on the compacted set, or for a longer
   window.
4. **Other environments' Job cadence.** How soon each namespace's sync-wave-0 Job
   re-runs after this lands — and therefore when each environment's topics
   converge — is an ArgoCD sync question, not a code question, but it determines
   when the observed symptom disappears.

## 10. Acceptance Criteria

- [ ] `internal/topics/topics.go` declares `max.compaction.lag.ms`,
      `segment.ms`, and `min.cleanable.dirty.ratio` as named constants with the
      values in FR-1.1.
- [ ] Both the `CreateTopics` `ConfigEntries` path and the
      `IncrementalAlterConfigs` path carry all four configs, derived from one
      shared declaration (FR-1.3).
- [ ] The alter path still uses `ConfigOperationSet` on the incremental API and
      still runs over the entire compacted set unconditionally (FR-1.4, FR-1.5).
- [ ] Plain topics carry no `ConfigEntries` and appear in no alter resource
      (FR-1.7).
- [ ] A per-resource alter error remains fatal and names the topic (FR-2.1,
      FR-2.2).
- [ ] Table-driven tests assert the exact config set on both request bodies via
      the existing fake `AdminClient`, including the negative case that a plain
      topic carries none of it.
- [ ] `go build ./...` and `go test ./...` pass in
      `services/atlas-kafka-precreate`.
- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] `README.md` describes the full compacted-topic configuration, and the
      in-code comment explains why `cleanup.policy` alone is inert (FR-4.1,
      FR-4.2).
- [ ] Design has answered OQ-1 and OQ-2 with evidence from the live broker, and
      recorded whether the existing `-main` topics self-heal (FR-3.2).
- [ ] Post-deploy verification on the live cluster: after the `atlas-main`
      sync-wave-0 Job re-runs,
      `kafka-configs.sh --entity-type topics --entity-name EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --describe`
      shows all four configs, and
      `kafka-get-offsets.sh --time -2` reports a log-start-offset greater than 0 —
      the direct signal that compaction has run at least once.
