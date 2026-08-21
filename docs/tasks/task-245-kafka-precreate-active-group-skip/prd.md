# Kafka Precreate — Skip Offset Seeding for Active Consumer Groups — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

The wave-0 `atlas-kafka-precreate` Job does two things for a sparse ephemeral
environment: it pre-creates every Kafka topic the environment will use, and
(task-232 Task 45, design §6.3, FR-4.9) it seeds committed end-of-log offsets
for every override consumer group named in `KAFKA_CONSUMER_GROUP`, so an
override deployment starts at the tip of the baseline's shared topics instead
of replaying their entire history.

The seeding pass is built on `kafka-consumer-groups.sh --reset-offsets
--to-latest --execute`, which Kafka accepts **only while the group is
inactive**. `deploy/k8s/base/kafka-precreate.sh`'s `seed_group` states this
assumption explicitly — "while the group is empty and therefore resettable.
Runs at sync-wave 0, before any Deployment starts (design §6.3)". That holds
on the *first* sync of an environment. It does not hold on any *subsequent*
sync of a live one: the Job carries
`argocd.argoproj.io/sync-options: Force=true,Replace=true`, so Argo CD deletes
and recreates it on every sync, while the environment's Deployments have been
running — and joined to those very groups — for hours.

The result is a false Degraded. Observed on `atlas-pr-1441` (PR #1441,
`task-244-listener-drain-socket-close`), whose branch touches no deployment
manifest and no Kafka code at all:

```
$ kubectl get application -A | grep 1441
argocd      atlas-pr-1441   Synced        Degraded

$ kubectl -n atlas-pr-1441 get jobs
atlas-kafka-precreate   Failed   0/1
Warning  BackoffLimitExceeded  Job has reached the specified backoff limit
```

Every Deployment in the namespace (`atlas-channel`, `atlas-configurations`,
`atlas-ingress`, `atlas-login`) was `1/1` Running and ~10h old. The
Application re-synced at `10:55:30Z` and failed at `11:01:07Z`. The cause,
reproduced live in that namespace:

```
$ kafka-consumer-groups.sh --describe --state \
    --group "Channel Service - 29047f69-1403-5c50-9cbb-deb8ce907296 [f8c5]"
GROUP                                    STATE    #MEMBERS
Channel Service - 29047f69-... [f8c5]    Stable   64

$ kafka-consumer-groups.sh --reset-offsets --to-latest --dry-run \
    --group "Channel Service - 29047f69-1403-5c50-9cbb-deb8ce907296 [f8c5]" --topic <any>
Error: Assignments can only be reset if the group
'Channel Service - 29047f69-1403-5c50-9cbb-deb8ce907296 [f8c5]' is inactive,
but the current state is Stable.
```

`set -euo pipefail` turns that non-zero exit into a Job failure;
`backoffLimit: 3` exhausts after three ~5-minute attempts (each re-running the
full 170-topic pre-create pass first); Argo CD's Job health check reports
Degraded for the whole Application.

This task makes the seeding pass **idempotent across re-syncs**: a group that
is already active is already initialized — which is the end state the pass
exists to reach — so it is skipped, not reset, and the Job stays green. Every
existing guarantee is preserved: a fresh environment still seeds every group
before any consumer starts, and main (`KAFKA_CONSUMER_GROUP` unset) is still
never touched.

## 2. Goals

Primary goals:

- A re-sync of a live sparse ephemeral environment leaves
  `atlas-kafka-precreate` **Complete**, and the Argo CD Application
  **Healthy**, with no operator intervention.
- A first sync of a fresh sparse environment behaves exactly as it does today:
  every override group is reset to end-of-log while inactive, and
  `verify_group_offsets` proves it before any Deployment (sync-wave 10)
  starts.
- The "skipped because active" condition is **observable in the Job log**,
  including which topics (if any) that group carries no committed offset on.
- The behaviour is covered by `atlas-kafka-precreate_test.sh` so it cannot
  silently regress.

Non-goals:

- **NG1.** Changing main / baseline behaviour. The `KAFKA_CONSUMER_GROUP`-unset
  early return (task-232 NG6) stays exactly as it is and stays the first thing
  the pass does. Main's groups carry real committed offsets and must never be
  reset.
- **NG2.** Converting the Job to an Argo CD `PreSync`/`Sync` hook, or otherwise
  changing when it runs. `Replace=true` at wave 0 stays; the Job re-running on
  every sync is desired (it is how a newly added topic gets created).
- **NG3.** Changing topic pre-creation (`precreate_topics`), the compacted
  config-projection topic handling, or the `xargs -P 16` concurrency.
- **NG4.** Changing how CI resolves and injects the group list
  (`.github/workflows/pr-validation.yml`'s `PRECREATE_GROUPS_BLOCK`, the
  `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` anchor in
  `deploy/k8s/overlays/pr-sparse/kustomization.yaml`, or the rendered-manifest
  assertions that follow them).
- **NG5.** Seeding a group's *true* subscribed-topic subset. The full-union
  seeding decision (task-232 Task 45 review round 1, finding 3) stands
  unchanged.
- **NG6.** Any change to Go service code. This task touches shell, YAML
  comments, tests, and docs only.

## 3. User Stories

- As an Atlas developer with an open PR, I want a routine Argo CD re-sync of my
  ephemeral environment to leave it Healthy, so that a Degraded badge always
  means something is actually wrong with my branch.
- As an Atlas developer, I do not want to spend triage time discovering that a
  Degraded environment is caused by infrastructure re-entrancy rather than by
  my change.
- As an operator reading `atlas-kafka-precreate` Job logs, I want to see
  explicitly which groups were seeded, which were skipped because they were
  already active, and which topics a skipped group has no committed offset on,
  so that I can tell "healthy no-op" from "something needs attention" without
  querying Kafka by hand.
- As a maintainer of `kafka-precreate.sh`, I want the active-group path
  asserted by the checked-in test script, so a later edit to `seed_group`
  cannot reintroduce the Degraded-on-resync failure.

## 4. Functional Requirements

### 4.1 Group state classification

- **FR-1.1** Before attempting to reset a group's offsets,
  `seed_override_offsets` MUST query that group's state via
  `kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --group
  "<group>" --describe --state`, using the `$KAFKA_BIN` full path (the
  `apache/kafka:3.7.2` image's `PATH` does not include the Kafka CLI tools —
  a bare name exits 127 and, under `set -euo pipefail`, crash-loops the Job).
- **FR-1.2** A group whose state is `Empty` or `Dead`, or that does not exist
  yet, is **seedable**: `seed_group` is called for it exactly as today.
- **FR-1.3** A group in any other state (`Stable`, `PreparingRebalance`,
  `CompletingRebalance`, or any state Kafka may add) is **active**: it MUST be
  skipped, and the skip MUST NOT fail the Job.
- **FR-1.4** The state query MUST tolerate its own failure (unreachable
  coordinator, unparseable output) by treating the group as **seedable** and
  falling through to `seed_group` — whose own error classification (FR-2)
  then governs. Never let the state probe itself become a new failure mode.
- **FR-1.5** Group names contain spaces and brackets (e.g. `Channel Service -
  <uuid> [f8c5]`), which shift every fixed column in `--describe --state`
  output. Parsing MUST anchor from the END of the line (the `NF`-relative
  idiom already used and documented in `verify_group_offsets`), never from a
  fixed forward column offset.

### 4.2 Reset-error classification (race handling)

- **FR-2.1** The window between the FR-1.1 state probe and the
  `--reset-offsets --execute` call is real: a self-heal or a rescheduled pod
  can join a group in between. `seed_group` MUST therefore ALSO classify the
  reset's own failure.
- **FR-2.2** A `--reset-offsets --execute` failure whose output identifies the
  group as non-inactive (Kafka's `Assignments can only be reset if the group
  '<name>' is inactive, but the current state is <state>.`) MUST be treated
  identically to an FR-1.3 skip: logged, not fatal.
- **FR-2.3** Any OTHER `--reset-offsets` failure MUST remain fatal, exactly as
  today. Broker-unreachable, authorization, and malformed-argument failures
  must still fail the Job — this task narrows the fatal set by exactly one
  well-identified condition, and must not degrade into a blanket
  `|| true`.
- **FR-2.4** `seed_group` MUST capture the command's combined output to make
  FR-2.2 classification possible, while keeping the success path's output
  suppressed as today (the current `>/dev/null`).

### 4.3 Verification symmetry

- **FR-3.1** `verify_group_offsets` MUST skip any group that
  `seed_override_offsets` skipped. An active group is by definition joined by a
  live consumer, which is the state the activation gate (FR-5.3 of task-232)
  exists to establish; re-proving it against the full topic union would
  reintroduce a Job failure the first time a topic is added to a live
  environment.
- **FR-3.2** The set of skipped groups MUST be recorded by
  `seed_override_offsets` in a form `verify_group_offsets` can consume — the
  two functions already share global state (`$topics`, `$compact_topics`) by
  the same convention.
- **FR-3.3** For every group that WAS seeded, `verify_group_offsets` keeps its
  current behaviour unchanged: a committed offset (anything but `-`) is
  required on every topic in the union, and a missing one exits 1 and fails the
  Job.
- **FR-3.4** The `KAFKA_CONSUMER_GROUP`-unset early return in
  `verify_group_offsets` stays, unchanged and symmetric with
  `seed_override_offsets` (NG1).

### 4.4 Observability

- **FR-4.1** Each skipped group MUST produce a log line naming the group and
  the state that caused the skip, e.g.
  `skipping group 'Channel Service - <uuid> [f8c5]': already active (Stable) — offsets already initialized`.
- **FR-4.2** For each skipped group, the pass MUST report which topics in the
  union that group carries **no committed offset** on — the topics that a live
  group can no longer be seeded for. This is a WARNING, not a failure: the Job
  stays green (an unseeded topic falls back to the consumer's own
  `auto.offset.reset`, and failing here would wedge an environment on a benign
  mid-life topic addition).
- **FR-4.3** The FR-4.2 warning MUST name the topics (or, above a small
  threshold, a count plus a bounded sample) rather than only a count — the log
  is the only diagnostic surface for this condition.
- **FR-4.4** When a skipped group carries a committed offset on every topic in
  the union — the common re-sync case — the pass MUST say so explicitly, so
  "skipped, fully seeded" is distinguishable at a glance from "skipped, missing
  offsets".
- **FR-4.5** The existing summary lines (`seeding end-of-log offsets for
  override consumer groups`, `override consumer group offsets seeded`,
  `verifying…`, `verified`) stay, and the terminal summary SHOULD carry the
  seeded-vs-skipped counts.

### 4.5 Idempotence contract

- **FR-5.1** Running `kafka-precreate.sh` twice in a row against the same live
  environment MUST exit 0 both times.
- **FR-5.2** The second run MUST NOT change any committed offset of any active
  group. This is the safety property behind the whole task: an override
  environment that has been consuming for hours must not be silently
  fast-forwarded past unprocessed messages by a routine Argo CD re-sync.
- **FR-5.3** A fresh environment's first run MUST still reset every named group
  to end-of-log. Regression coverage for this is the existing
  `atlas-kafka-precreate_test.sh` seed assertions, which must continue to pass
  unchanged.

## 5. API Surface

No HTTP or Kafka API surface changes. This task modifies one shell script and
its test harness.

Shell function contracts (the de-facto internal API of
`deploy/k8s/base/kafka-precreate.sh`, sourced directly by
`atlas-kafka-precreate_test.sh`):

| Function | Change |
|---|---|
| `precreate_topics` | Unchanged (NG3). |
| `seed_group <group> <topic...>` | Return contract widens: a reset refused because the group is active is a **non-fatal, distinguishable** outcome (FR-2.2), not a script-fatal error. Signature unchanged, so the existing multi-topic test assertions keep working. |
| `group_is_active <group>` (new, name TBD at design) | Returns the group's state / a seedable-vs-active verdict per FR-1. Must be independently sourceable and testable, like `seed_group`. |
| `seed_override_offsets` | Skips active groups (FR-1.3), records the skip set (FR-3.2), emits FR-4 logging. `KAFKA_CONSUMER_GROUP`-unset early return unchanged. |
| `verify_group_offsets` | Skips groups in the skip set (FR-3.1); otherwise unchanged (FR-3.3). |

The executed-vs-sourced probe (`if ! (return 0 2>/dev/null); then main "$@"; fi`)
stays, so the test script can keep loading the definitions without a live
Kafka.

## 6. Data Model

No database entities, no migrations, no tenant-scoped rows. The only persistent
state involved is Kafka's `__consumer_offsets` — and the entire point of this
change is to stop writing to it for groups that already have live members
(FR-5.2).

## 7. Service Impact

| Path | Change |
|---|---|
| `deploy/k8s/base/kafka-precreate.sh` | The change. New state probe, skip path, error classification, logging. |
| `deploy/k8s/base/atlas-kafka-precreate_test.sh` | New assertions (§10). |
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Header comment only — the "while each group is still empty and therefore resettable" claim becomes conditional and must be corrected in place. No spec change (`backoffLimit`, `Replace=true`, sync-wave 0 all stay). |
| `docs/runbooks/sparse-environments.md` | Document the re-sync semantics and how to read the new log lines; note that a live group cannot be re-seeded and what that means for a topic added mid-life. |
| `deploy/k8s/overlays/pr-sparse/kustomization.yaml` | Comment only, if the `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` commentary asserts the now-conditional invariant. No functional change (NG4). |

No Go module is touched, so no service rebuild is implied. `tools/verify.sh`
must still pass flagless.

## 8. Non-Functional Requirements

- **Performance.** The state probe adds one `kafka-consumer-groups.sh --describe
  --state` JVM cold start (~1–1.5s) per named group — single-digit groups per
  sparse environment, so ≤ a few seconds against a Job whose topic pass already
  runs for minutes. It MUST NOT reintroduce a per-(group, topic) invocation:
  the O(groups) collapse from task-232 Task 45 review round 1 finding 2 stands.
  Net effect on the re-sync path is a large *saving* — today's failure burns
  three full ~5-minute attempts.
- **Correctness / safety.** FR-5.2 is the hard safety property: no active
  group's offsets may be mutated. The design must make it structurally true
  (skip before reset), not merely probable.
- **Failure isolation.** FR-2.3 — the fatal set narrows by exactly one
  identified condition. No blanket `|| true`, no `set +e` region wider than the
  single classified call.
- **Portability.** `kafka-precreate.sh` runs under `bash` in
  `apache/kafka:3.7.2` but is *sourced* by a `#!/usr/bin/env sh` test script;
  new code must not break that dual use. All Kafka CLI invocations use
  `$KAFKA_BIN` full paths.
- **Multi-tenancy.** Not applicable — this is environment-scoped
  infrastructure, below the tenant layer.
- **Observability.** FR-4. The Job log is the only surface; every non-obvious
  branch must announce itself.

## 9. Open Questions

- **OQ-1.** Should the FR-4.2 "topics with no committed offset on a skipped
  group" report be bounded (first N topics + count) to keep the log readable
  when a group legitimately subscribes to a small subset of a 170-topic union?
  Note the union is deliberately a superset (NG5), so on a *seeded* group every
  topic has an offset — this only lists genuinely unseedable ones. Resolve at
  design.
- **OQ-2.** Is `--describe --state` output stable enough across the Kafka
  versions this Job may run against to parse a state token, or should the
  classification key off `#MEMBERS > 0` instead? Both come from the same
  command; design should pick one and justify it against the actual
  `apache/kafka:3.7.2` output.
- **OQ-3.** Does any *other* environment shape reach `seed_override_offsets`
  with a pre-existing active group — e.g. a sparse environment whose baseline
  was switched, or a re-created override sharing a previous env's `ATLAS_ENV`
  suffix? If so, the same skip covers it, but the runbook wording should say so.
- **OQ-4.** Should the Job emit a distinct terminal log line when *every* named
  group was skipped (the pure re-sync case), to make "this run did nothing, by
  design" unambiguous in Argo CD's Job log view? Leaning yes; confirm at design.

## 10. Acceptance Criteria

- [ ] `seed_override_offsets` probes each group's state before resetting it and
      skips any group that is not `Empty`/`Dead`/absent (FR-1.1–FR-1.5).
- [ ] `seed_group`'s reset failure is classified: the "group is inactive"
      refusal is non-fatal and distinguishable; every other failure remains
      fatal (FR-2.1–FR-2.4).
- [ ] `verify_group_offsets` skips exactly the groups
      `seed_override_offsets` skipped, and is otherwise unchanged
      (FR-3.1–FR-3.4).
- [ ] The `KAFKA_CONSUMER_GROUP`-unset early return is untouched in both
      functions, and the existing NG6 assertion in
      `atlas-kafka-precreate_test.sh` still passes (NG1).
- [ ] Job log distinguishes, per group: seeded / skipped-and-fully-seeded /
      skipped-with-unseeded-topics, naming the group, its state, and the
      unseeded topics (FR-4.1–FR-4.5).
- [ ] `atlas-kafka-precreate_test.sh` gains an assertion that `seed_group`
      against a group with a live member does NOT fail the script and does NOT
      change that group's committed offset (FR-5.2). It must SKIP cleanly when
      `BOOTSTRAP_SERVERS` is unset, matching the existing convention.
- [ ] `atlas-kafka-precreate_test.sh` gains an assertion that a **second**
      full `main` run against an already-seeded, active-group environment exits
      0 (FR-5.1), or an equivalent function-level assertion if driving `main`
      twice is impractical in the harness.
- [ ] The existing seed assertions (single-topic, multi-topic) pass unchanged
      (FR-5.3).
- [ ] `deploy/k8s/base/atlas-kafka-precreate.yaml`'s header comment no longer
      asserts unconditionally that groups are empty when the pass runs.
- [ ] `docs/runbooks/sparse-environments.md` documents the re-sync semantics and
      the new log lines.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Live confirmation: an Argo CD re-sync of a running sparse environment
      leaves `atlas-kafka-precreate` `Complete` and the Application `Healthy`.
      `atlas-pr-1441` is the reproduction case on record; the branch's own
      ephemeral environment is an equally valid subject once it has been up
      long enough for its groups to reach `Stable`.
