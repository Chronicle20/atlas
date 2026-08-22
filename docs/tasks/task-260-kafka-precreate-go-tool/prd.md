# Kafka Precreate Go Tool — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

`atlas-kafka-precreate` is the Argo sync-wave 0 Job that pre-creates every Kafka topic an environment will use, and seeds committed offsets for override consumer groups, before any Deployment starts. Every Deployment sits at sync-wave 10 behind it. It exists because Atlas services rely on Kafka topic auto-create on first publish and many subscribe to the same topic concurrently at startup — without pre-creation the result is a "consumer fetch wedged: exceeded consecutive timeouts" stampede.

Today it is a bash script (`deploy/k8s/base/kafka-precreate.sh`) running in the `apache/kafka:3.7.2` image, driving the Kafka CLI tools. That design has a structural cost: every `kafka-topics.sh` / `kafka-consumer-groups.sh` invocation is a full JVM cold start — classload, JIT warmup, admin-client bootstrap, and a full cluster-metadata fetch — that performs exactly one operation. The RPC itself is a rounding error inside ~1–1.5s of near-100% CPU. With ~170 topics the script pays that cost ~170 times per run, and on 2026-08-21 three concurrent copies (`atlas-main`, `atlas-pr-1434`, `atlas-pr-1450`) took 6.5 of node `eos`'s 8 cores and held it at 99% for minutes at a time.

PR #1463 landed an interim mitigation in the shell script (list once and create only the diff, `-P 16` → `-P 4`, CPU/memory caps, cross-namespace pod anti-affinity), which reduced a steady-state run from ~4 minutes to 6 seconds. This task removes the root cause instead: replace the shell script and the JVM CLI with a small Go binary using `segmentio/kafka-go`'s batch admin APIs, where the entire topic pass is a **single** request over a **single** connection.

A spike against a real KRaft broker has already validated the three protocol behaviours this design depends on, plus one it did not anticipate (see §4.5 and §9).

## 2. Goals

Primary goals:

- Replace all JVM CLI invocations in the precreate path with in-process `segmentio/kafka-go` admin calls.
- Reduce the topic-creation pass from ~170 sequential JVM cold starts to one `CreateTopics` request.
- Replace the script's CLI-output string matching and `awk` column arithmetic with typed Kafka errors and structured responses.
- Preserve today's externally observable behaviour exactly: same topics created, same cleanup policies, same seeding semantics, same skip rules, same pass/fail conditions for wave 0.
- Keep the tool small and dependency-light — it is a support image, not a service.

Non-goals:

- **Changing how the topic set is discovered.** The `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` environment scrape is ported faithfully. Replacing it with a declarative, code-derived manifest is tracked separately in issue #1464 and is explicitly out of scope here.
- Changing consumer-group naming or the `libs/atlas-kafka/consumergroup` resolver.
- Adding admin wrappers to `libs/atlas-kafka`. The tool uses `kafka-go` directly; wrapping admin APIs in the shared lib for one consumer would be scope creep.
- Changing the sync-wave 0 Job's position, its `atlas-env` ConfigMap wiring, or the ordering guarantee Deployments depend on.
- Altering partition count or replication factor (both remain 1, as today).

## 3. User Stories

- As a **platform operator**, I want the precreate Job to consume negligible CPU so that spinning up a PR environment does not degrade the game services sharing that node.
- As a **developer opening a PR**, I want the wave-0 Job to finish in under a second on a re-sync so that my ephemeral environment is not gated behind minutes of topic reconciliation.
- As an **on-call engineer**, I want a wave-0 failure to name the exact topic or consumer group that failed, with a typed error, rather than requiring me to reverse-engineer a parsed CLI message.
- As a **maintainer**, I want the seeding logic covered by Go unit tests so that a change to the skip rules is caught by CI rather than in a live environment.

## 4. Functional Requirements

### 4.1 Topic discovery (faithful port)

- **FR-1.1** The tool MUST enumerate its own environment via `os.Environ()` and select every variable whose name begins with `COMMAND_TOPIC_` or `EVENT_TOPIC_`. This mirrors the script's `compgen -e | grep -E '^(COMMAND|EVENT)_TOPIC_'`.
- **FR-1.2** A selected variable with an empty value MUST be skipped, not treated as a topic named `""`.
- **FR-1.3** The resulting topic names MUST be de-duplicated. Multiple variables commonly resolve to the same topic.
- **FR-1.4** The three config-projection variables `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, and `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` MUST be classified as **compacted** topics rather than plain ones.
- **FR-1.5** A topic named by both a config-status variable and any other variable MUST remain classified as compacted. Compaction wins.
- **FR-1.6** `BOOTSTRAP_SERVERS` MUST be required. Its absence is a fatal startup error, matching the script's `: "${BOOTSTRAP_SERVERS:?}"`.

*Rationale for compaction (preserved from the script): config-projection consumers replay these topics from first-offset at every boot to rebuild tenant/service config state, and the outbox never re-emits a `(topic, key)` it already delivered. Under the default DELETE cleanup, retention empties the topic ~7 days after the last config change and every later projection boot has nothing to replay. Events are keyed, so compaction retains the latest snapshot per key forever.*

### 4.2 Topic creation

- **FR-2.1** All plain topics MUST be created in a **single** `CreateTopics` request, with `NumPartitions: 1` and `ReplicationFactor: 1`.
- **FR-2.2** All compacted topics MUST be created with `ConfigEntries` carrying `cleanup.policy=compact` at creation time. They MAY be included in the same `CreateTopics` request as the plain topics.
- **FR-2.3** A per-topic `TopicAlreadyExists` error in the response MUST be treated as success. The tool MUST discriminate it with `errors.Is(err, kafka.TopicAlreadyExists)`, never by string matching.
- **FR-2.4** Any other per-topic error MUST fail the Job with a message naming the topic and the error.
- **FR-2.5** The tool MUST NOT perform a list-then-diff pass. Per-topic error discrimination makes the create idempotent on its own, which removes the sorting, the `comm`, and the entire locale-collation failure class the shell version had to defend against.
- **FR-2.6** For every compacted topic — whether newly created or pre-existing — the tool MUST issue an `AlterConfigs` call adding `cleanup.policy=compact`, to converge topics created before this policy existed. This SHOULD be a single batched request.

### 4.3 Override consumer-group offset seeding

- **FR-3.1** If `KAFKA_CONSUMER_GROUP` is unset or empty, the tool MUST skip seeding and verification entirely and log that it did so. This is the `main` environment, whose groups carry real committed offsets that must never be reset.
- **FR-3.2** `KAFKA_CONSUMER_GROUP` MUST be parsed as a **newline-delimited** list. Group names contain spaces and brackets (e.g. `Channel Service - <uuid> [pr-123]`), so a space-delimited parse would be wrong.
- **FR-3.3** For each group, the tool MUST probe its state via `DescribeGroups` and read the `GroupState` field.
- **FR-3.4** A group whose state is `Empty`, `Dead`, or unknown/unreadable MUST be treated as seedable. This is an **allowlist**: any other state (including a state a future Kafka version introduces) is treated as active and skipped. Skipping never mutates a committed offset; that is the safe direction.
- **FR-3.5** For a seedable group, the tool MUST determine end-of-log offsets via `ListOffsets` and commit them via `OffsetCommit` with `GenerationID: -1` and `MemberID: ""`.
- **FR-3.6** An `OffsetCommit` refused because the group became active between the probe and the commit MUST be treated as a non-fatal skip, not a failure. The refusal surfaces as a typed `kafka.Error` (`UnknownMemberID`, code 25) and MUST be discriminated as such, replacing the script's match on the CLI's `"Assignments can only be reset if the group"` message.
- **FR-3.7** Seeding MUST cover the full union of plain and compacted topics, matching current behaviour. A group carrying an offset on a topic it never reads is inert and is reclaimed with the group at teardown.
- **FR-3.8** The tool MUST log a per-group outcome (seeded / skipped-with-state) and a run summary of seeded and skipped counts.

### 4.4 Verification gate

- **FR-4.1** After seeding, for each group in `KAFKA_CONSUMER_GROUP`, the tool MUST verify committed offsets via `OffsetFetch`.
- **FR-4.2** For a group that was **seeded** this run, a missing committed offset on any union topic MUST fail the Job (exit non-zero), so Argo's health check on the Job carries the signal.
- **FR-4.3** For a group that was **skipped** because it was already active, a missing committed offset MUST NOT fail the Job. It MUST be reported as a WARN naming the affected topics, because the group already has live consumers — which is the end state this gate exists to establish — and re-proving it against the full union would fail the Job the first time a topic is added to a live environment.
- **FR-4.4** WARN output for missing topics MUST be bounded (at most 10 named, plus a remainder count), matching current behaviour.
- **FR-4.5** Verification MUST be skipped entirely when `KAFKA_CONSUMER_GROUP` is unset, symmetric with FR-3.1.

### 4.5 Coordinator resilience

- **FR-5.1** The tool MUST retry group-coordinator requests that fail with `NotCoordinatorForGroup` or `GroupCoordinatorNotAvailable`, with a bounded retry budget.
- **FR-5.2** The retry budget MUST be bounded such that the total run stays comfortably inside the Job's `activeDeadlineSeconds` (§8).

*This requirement comes from the spike, not from the existing script. At sync-wave 0 the cluster may be brand new, and the first group request can land while `__consumer_offsets` and the group coordinator are still settling. The shell version never had to handle this because the JVM CLI retried internally; a direct protocol client must do it explicitly.*

### 4.6 Cutover

- **FR-6.1** `deploy/k8s/base/kafka-precreate.sh` MUST be deleted.
- **FR-6.2** `deploy/k8s/base/atlas-kafka-precreate_test.sh` MUST be deleted, with its behavioural assertions ported to Go table-driven tests (§4.7).
- **FR-6.3** The `atlas-kafka-precreate-script` `configMapGenerator` entry MUST be removed from `deploy/k8s/base/kustomization.yaml`, along with the Job's script volume and volumeMount.
- **FR-6.4** The Job MUST invoke the new image directly with no shell wrapper.

### 4.7 Testing

- **FR-7.1** Environment scraping, compaction classification, and the seedable-state allowlist MUST be pure functions with table-driven unit tests, independent of a live broker.
- **FR-7.2** The assertions from `atlas-kafka-precreate_test.sh` MUST be carried over — specifically: seeding is skipped when `KAFKA_CONSUMER_GROUP` is unset (NG6), and the state allowlist accepts `Empty`/`Dead`/unknown while rejecting every active state.
- **FR-7.3** Group-name handling MUST be tested with names containing spaces and brackets.
- **FR-7.4** Per-topic `TopicAlreadyExists` tolerance MUST be tested against a fake/stub response, asserting the run still succeeds.

## 5. API Surface

No HTTP or JSON:API surface. The tool is a batch binary with no server, no REST layer, and no inbound interface.

Its contract is:

**Inputs (environment, injected by the `atlas-env` ConfigMap via `envFrom`):**

| Variable | Required | Meaning |
|---|---|---|
| `BOOTSTRAP_SERVERS` | yes | Kafka bootstrap address |
| `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` | yes (≥1) | Topic names to ensure |
| `KAFKA_CONSUMER_GROUP` | no | Newline-delimited override consumer groups to seed. Unset ⇒ `main`, seeding skipped |

**Output:** logrus-formatted log lines on stdout/stderr.

**Exit codes:** `0` success (including all-already-exists and all-skipped runs); non-zero on any fatal condition per FR-1.6, FR-2.4, FR-4.2.

## 6. Data Model

No database, no persisted entities, no migrations. The tool holds only in-memory values for the duration of one run:

- a de-duplicated set of plain topic names,
- a de-duplicated set of compacted topic names,
- the parsed list of override consumer group names,
- the set of groups skipped as already-active (consumed by the verification gate to choose hard-fail vs WARN).

`tenant_id` scoping does not apply: this operates on Kafka cluster metadata for a whole environment, below the tenancy layer. Multi-tenancy is expressed in the topic and group *names* supplied by the environment, which the tool treats as opaque.

## 7. Service Impact

| Area | Change |
|---|---|
| **`services/atlas-kafka-precreate/`** (new) | New Go module implementing the tool. Registered as `"type": "support-image"` following the `atlas-pr-bootstrap` precedent — it is not a `go-service` and MUST NOT inherit the service scaffold (no REST, no DB, no tenancy middleware, no health endpoint). |
| **`.github/config/services.json`** | New entry: `name: atlas-kafka-precreate`, `type: support-image`, `path`, `docker_image`, `docker_context`. |
| **`docker-bake.hcl`** | New explicit target alongside `atlas-ui` and `atlas-pr-bootstrap`, and inclusion in the `all-services` group. Required because `cideps` `EnrichDockerServices` includes any `services.json` entry with `docker_image` set regardless of type — a missing bake target makes CI's per-shard `bake <names…>` fail the first time a PR touches this path. |
| **`deploy/k8s/base/atlas-kafka-precreate.yaml`** | Image and command change; script volume/volumeMount removed; `activeDeadlineSeconds` added. Sync-wave, `envFrom`, resource caps, and anti-affinity from PR #1463 are retained. |
| **`deploy/k8s/base/kustomization.yaml`** | `atlas-kafka-precreate-script` configMapGenerator entry removed. |
| **`deploy/k8s/base/kafka-precreate.sh`, `atlas-kafka-precreate_test.sh`** | Deleted. |
| **Game services** | None. |
| **`libs/atlas-kafka`** | None. |

## 8. Non-Functional Requirements

**Performance**
- **NFR-1** A steady-state run (all topics present, all groups active) MUST complete in under 5 seconds wall-clock. Target is well under 1 second; the shell version's post-#1463 best case is ~6 seconds.
- **NFR-2** The topic pass MUST issue O(1) Kafka requests with respect to topic count, not O(n).
- **NFR-3** Peak CPU MUST stay within the Job's existing `2`-core limit with large headroom; the design intent is that this Job stops being visible in node CPU accounting at all.

**Reliability**
- **NFR-4** The Job MUST gain `activeDeadlineSeconds: 300`, matching the precedent set by `wave0-create-dbs.yaml`. This was deliberately omitted in PR #1463 because a slow first sync could plausibly exceed it under the shell implementation; a single batch RPC removes that risk, so the deadline becomes safe and converts a hung wave 0 from an indefinite block into a visible failure.
- **NFR-5** The tool MUST remain idempotent across Argo re-syncs. The Job carries `Force=true,Replace=true` and is recreated on every sync.
- **NFR-6** The tool MUST NOT fast-forward a live environment past unprocessed messages. This is the core safety property; FR-3.4 and FR-3.6 exist to guarantee it.

**Observability**
- **NFR-7** Logging MUST use `logrus`, consistent with every Atlas Go service, so Job output lands in the same observability pipeline. Per-topic and per-group outcomes SHOULD be structured fields rather than interpolated strings.
- **NFR-8** Every fatal exit MUST name the specific topic or group and the underlying typed error.

**Security**
- **NFR-9** The image SHOULD be minimal (scratch or distroless) with a non-root user, and MUST NOT ship a shell or the Kafka CLI tooling.

**Multi-tenancy**
- **NFR-10** Not applicable at the tool level; see §6.

## 9. Open Questions

1. **Compacted-topic `AlterConfigs` batching.** FR-2.6 requires converging pre-existing topics to `cleanup.policy=compact`. Whether `kafka-go`'s `AlterConfigs` accepts all three topics in one request, and whether it is incremental or full-replace for topic configs, was not covered by the spike. A full-replace semantic would clobber other topic-level config and must be checked during design.
2. **`ListOffsets` partition scope.** All topics are created with a single partition, so seeding partition 0 is correct today. Should the tool defensively enumerate partitions via metadata instead of assuming partition 0, in case a topic was created out-of-band with more?
3. **Retry budget shape.** FR-5.1 requires bounded retry; the specific backoff and ceiling are a design decision, constrained by NFR-4's 300s deadline.
4. **Go module placement within the service directory.** Existing Go services nest as `services/atlas-<n>/atlas.com/<n>`. Whether this support image follows that nesting or uses a flatter layout (it is not a `go-service` and the root `Dockerfile`'s `SERVICE` arg convention may not apply) is a design decision.
5. **`atlas-env` availability.** The tool depends on `atlas-env` being present at wave 0, which is already true for the shell version. No change, but worth confirming no overlay omits it.

## 10. Acceptance Criteria

- [ ] `services/atlas-kafka-precreate/` exists as a Go module with no REST, DB, or tenancy scaffolding.
- [ ] The topic pass issues exactly one `CreateTopics` request regardless of topic count, verified by test or by log evidence.
- [ ] Per-topic `TopicAlreadyExists` is tolerated via `errors.Is`; no string matching on Kafka output anywhere in the tool.
- [ ] Compacted topics are created with `cleanup.policy=compact`, and pre-existing ones are converged via `AlterConfigs`.
- [ ] `KAFKA_CONSUMER_GROUP` unset ⇒ seeding and verification both skipped, logged, exit 0 (NG6 preserved).
- [ ] Newline-delimited group names containing spaces and brackets are parsed and used correctly.
- [ ] An `Empty`/`Dead`/unknown group is seeded; every other state is skipped with its state logged.
- [ ] A group that becomes active mid-run is skipped non-fatally via typed `UnknownMemberID`, not a string match.
- [ ] The verification gate hard-fails for seeded groups and WARNs (bounded to 10 names + count) for skipped ones.
- [ ] Coordinator errors (`NotCoordinatorForGroup`, `GroupCoordinatorNotAvailable`) are retried within a bounded budget.
- [ ] `kafka-precreate.sh` and `atlas-kafka-precreate_test.sh` are deleted; the configMapGenerator entry, volume, and volumeMount are removed.
- [ ] The Job runs the new image with no shell and carries `activeDeadlineSeconds: 300`.
- [ ] `services.json` and `docker-bake.hcl` entries exist; all four overlays (`main`, `pr`, `pr-cleanup`, `pr-sparse`) render.
- [ ] Table-driven unit tests cover env scraping, compaction classification, the state allowlist, group-name parsing, and already-exists tolerance.
- [ ] A steady-state run against a real broker completes in under 5 seconds with exit 0.
- [ ] `tools/verify.sh` (flagless) exits 0.
