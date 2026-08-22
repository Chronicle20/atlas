# Kafka Precreate Go Tool — Design

Version: v1
Status: Draft
Created: 2026-08-21
Input: [prd.md](prd.md) (approved), [spike-findings.md](spike-findings.md)

---

## 1. Scope of this document

The PRD fixes *what* the tool must do; every behavioural requirement there is a
faithful port of `deploy/k8s/base/kafka-precreate.sh`. This document fixes *how*:
the module layout, the request pipeline, the seams that make the logic testable
without a broker, the build/deploy wiring, and the resolutions of PRD §9's five
open questions.

Two things changed the shape of the design relative to what the PRD assumed, and
both are recorded in §4:

- `kafka-go` v0.4.51 ships **`IncrementalAlterConfigs`** alongside the legacy
  full-replace `AlterConfigs`. That settles OQ-1 in the safe direction.
- The `kafka-go` transport **serves `Metadata` requests from a TTL cache**
  (`transport.go:352-374`, default TTL 6s). A `ListOffsets` issued immediately
  after `CreateTopics` routes off that cache, so a freshly created topic can
  route to `Broker{ID: -1}`. This is a hazard the shell version never had,
  because the JVM CLI opened a fresh admin client per invocation. §3.3 handles it.

---

## 2. Module and package layout

### 2.1 Placement — OQ-4 resolved: flat, no `atlas.com/` nesting

```
services/atlas-kafka-precreate/
├── Dockerfile
├── README.md
├── go.mod                     module atlas.com/kafka-precreate
├── go.sum
├── main.go                    wiring + exit codes only
└── internal/
    ├── discover/              env scrape, classification, group parsing (pure)
    │   ├── discover.go
    │   └── discover_test.go
    ├── topics/                CreateTopics + IncrementalAlterConfigs
    │   ├── topics.go
    │   └── topics_test.go
    ├── groups/                describe / seed / verify
    │   ├── groups.go
    │   └── groups_test.go
    └── kafkaops/              the narrow admin interface + retry wrapper
        ├── ops.go
        └── ops_test.go
```

The `services/atlas-<name>/atlas.com/<name>/` nesting exists for one reason: the
shared root `Dockerfile` discovers the module with
`ls -d services/${SERVICE}/atlas.com/*/ | head -1` (`Dockerfile:88`, `:104`).
This image does not use the shared Dockerfile — it has no `libs/atlas-*`
dependency, so the shared Dockerfile's twenty-two `COPY libs/…` layers and its
synthesized `go.work` are pure cost, and its runtime stage (`alpine`,
`EXPOSE 8080`, nine placeholder data dirs, root user) is the opposite of NFR-9.
`services/atlas-pr-bootstrap/` is the precedent for a flat support image with its
own Dockerfile, and this follows it.

Nothing in the toolchain requires the nesting for a non-`go-service`:

- `tools/verify.sh:180-183` (`all_modules`) finds *any* `go.mod` under
  `services/` or `libs/`, so the go build/vet/`-race` layer picks this module up
  regardless of depth.
- `tools/service-registration-guard.sh` only globs
  `services/{name}/atlas.com/*/main.go` for entries whose `type == "go-service"`
  (`:119`), which this is not.

### 2.2 Module dependencies

`github.com/segmentio/kafka-go v0.4.51` (already required by 65 modules in this
repo, so the version is not a new decision) and `github.com/sirupsen/logrus`
(NFR-7). Nothing else — no `libs/atlas-*`, per PRD §2 non-goals. Keeping the
dependency set to two is what lets the Dockerfile use the service directory as
its build context.

`go.work` gains `./services/atlas-kafka-precreate`. **This makes every
`tools/verify.sh` run on this branch a full 86-module fan-out**
(`verify.sh:186-189`, `fanout_paths` matches `^(go\.work|libs/)`). That is
expected and unavoidable — a new module must be in the workspace. While
iterating, pass `--base <last-gated-commit>` so the change set is the increment,
not the accumulated branch; the final flagless gate run is the full fan-out by
design.

### 2.3 Package responsibilities

| Package | Owns | Depends on |
|---|---|---|
| `discover` | `os.Environ()` scrape, compaction classification, `KAFKA_CONSUMER_GROUP` parsing. **Zero I/O, zero Kafka types.** | stdlib |
| `topics` | Building and issuing the one `CreateTopics` and the one `IncrementalAlterConfigs`; per-topic error triage. | `kafkaops` |
| `groups` | State probe → seed → verify, and the skipped-group set that couples them. | `kafkaops` |
| `kafkaops` | The `AdminClient` interface (§5.1), the concrete `*kafka.Client` adapter, and the coordinator retry wrapper. | `kafka-go` |
| `main` | Read env, construct client, run the four phases in order, map failures to exit codes. | all of the above |

The split follows the one that already exists in the shell script's function
boundaries (`precreate_topics` / `group_state` + `state_is_seedable` /
`seed_override_offsets` / `verify_group_offsets`), so a reviewer comparing old
against new reads them side by side.

---

## 3. The run pipeline

`main` runs five phases in strict order under a single
`context.WithTimeout(ctx, 240*time.Second)` — comfortably inside the Job's
`activeDeadlineSeconds: 300` (NFR-4), so the tool's own deadline fires first and
produces a named error rather than an opaque pod kill.

### Phase A — Discover (pure, no I/O)

`discover.FromEnviron(os.Environ())` returns:

```go
type Topics struct {
    Plain    []string  // sorted, de-duplicated
    Compact  []string  // sorted, de-duplicated
}
func (t Topics) Union() []string
```

Rules, in order (FR-1.1 → FR-1.5):

1. Select `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` by name prefix.
2. Drop empty values.
3. The three `EVENT_TOPIC_CONFIGURATION_{TENANT,SERVICE,ENVIRONMENT}_STATUS`
   variables contribute to `Compact`; everything else to `Plain`.
4. Subtract `Compact` from `Plain` — compaction wins on a collision (FR-1.5).
5. Sort both for deterministic logs and deterministic tests.

Note what step 4 replaces: the shell needed `sort -u | comm -23`, and PR #1463
had to pin `LC_ALL=C` because topic names mix `_` and `-`, exactly where locale
collation and byte order disagree. In Go this is a `map` difference and the
locale failure class does not exist.

`BOOTSTRAP_SERVERS` absent or empty ⇒ fatal before any Kafka call (FR-1.6).

`discover.Groups(os.Getenv("KAFKA_CONSUMER_GROUP"))` splits on `\n`, trims `\r`
(the value arrives from a YAML block scalar), drops empty lines, and **does not
trim interior whitespace** — `Channel Service - <uuid> [pr-123]` must survive
byte-for-byte (FR-3.2). Empty input returns an empty slice, which is the NG6
signal every later phase keys off.

### Phase B — Create topics (one request)

One `CreateTopics` carrying **both** lists (FR-2.1, FR-2.2):

```go
kafka.TopicConfig{
    Topic:             name,
    NumPartitions:     1,
    ReplicationFactor: 1,
    ConfigEntries:     nil,  // or {{"cleanup.policy", "compact"}} for Compact
}
```

`CreateTopicsResponse.Errors` is a `map[string]error` keyed by topic name.
Triage per entry:

- `nil` ⇒ created.
- `errors.Is(err, kafka.TopicAlreadyExists)` ⇒ success (FR-2.3). Spike Q1
  confirmed the code and that `errors.Is` matches it.
- anything else ⇒ collect, then fail the Job naming **every** offending topic
  and its typed error (FR-2.4, NFR-8). Collect-then-fail rather than fail-fast:
  one run should report all the broken topics, not just the first.

No list-then-diff (FR-2.5). `defaultCreateTopicsTimeout` in `kafka-go` is 2s
(`client.go:14`); the request carries an explicit longer per-request timeout so a
170-topic create on a cold cluster is not cut short.

### Phase C — Metadata settle (new; not in the PRD)

Before any offset work, poll `client.Metadata(ctx, &MetadataRequest{Topics:
union})` until every union topic is present with at least one partition, or the
budget expires.

This exists because `connPool.roundTrip` answers `meta.Request` **from the
transport's cached cluster state** (`transport.go:352-362`) rather than from the
broker, and that cache refreshes on `Transport.MetadataTTL` (default 6s,
`transport.go:205-210`). `ListOffsets` is split by `kafka-go` into one request
per topic-partition and routed to each partition leader looked up in that same
cached state (`protocol/listoffsets/listoffsets.go:37-50`, `:52-93`); a topic
missing from the cache resolves to `Broker{ID: -1}`. Without this phase, a
first-sync run could create 170 topics and then fail to route offsets for them.

Mitigations, both applied:

- `Transport.MetadataTTL = 1 * time.Second`, so the background refresh picks up
  the new topics quickly.
- The settle loop: 250ms poll, 30s ceiling. On timeout, fail naming the topics
  that never appeared.

The metadata response is also the source of **partition IDs** — see OQ-2 in §4.

Phase C is skipped entirely when there are no groups to seed (NG6): nothing
downstream needs partition data. This keeps the `main` environment's run to
exactly two RPCs.

### Phase D — Seed override group offsets

Skipped wholesale, with a log line, when `Groups` is empty (FR-3.1, NG6). This is
the first thing the phase checks, ahead of any Kafka call, so `main` is provably
never touched.

Per group:

1. **Probe** — `DescribeGroups{GroupIDs: []string{g}}`, read `GroupState`
   (FR-3.3). A transport error, a per-group `Error`, or an absent group all
   collapse to the empty string, matching the shell's "any failure yields `""`".
2. **Classify** — `discover.StateIsSeedable(state)`: allowlist of `Empty`,
   `Dead`, `""` (FR-3.4). Everything else is active and skipped. Pure function,
   exhaustively table-tested.
3. **Seed** — for a seedable group: one `OffsetCommit` carrying every
   `(topic, partition)` in the union with the end-of-log offset from Phase E's
   `ListOffsets`, `GenerationID: -1`, `MemberID: ""` (FR-3.5, FR-3.7). Spike Q2
   confirmed this against a real broker with a spaces-and-brackets group name.
4. **Race triage** — `OffsetCommitResponse.Topics[t][p].Error` matching
   `errors.Is(err, kafka.UnknownMemberID)` ⇒ non-fatal skip; record the group in
   the skipped set and move on (FR-3.6). Spike Q3 confirmed code 25 is what a
   live group returns. Any other per-partition error is fatal, naming group,
   topic, partition, and error.

Ordering note: `ListOffsets` for the whole union is issued **once**, before the
per-group loop, and its result is reused for every group. End-of-log is a
property of the topic, not of the group, so re-listing per group would multiply
requests for identical answers.

Per-group outcome logs and a `seeded=N skipped=M` summary (FR-3.8), plus the
shell's "all groups already active — re-sync no-op" line when `seeded == 0 &&
skipped > 0`.

### Phase E — Verify

Skipped when `Groups` is empty (FR-4.5). Otherwise, one `OffsetFetch` per group
over the full union:

- Group **seeded** this run, any union `(topic, partition)` with
  `CommittedOffset < 0` ⇒ **fatal**, exit non-zero (FR-4.2). Argo's health check
  on the Job carries the signal.
- Group **skipped**: report only. Zero missing ⇒ an info line. Otherwise a WARN
  naming at most 10 topics plus `(+N more)` (FR-4.3, FR-4.4). The Job stays
  green: a skipped group has live consumers, which is the end state this gate
  exists to establish, and re-proving it against the full union would fail the
  Job the first time a topic is added to a live environment.

`OffsetFetchPartition.CommittedOffset` is `-1` for "no committed offset", which
is the typed replacement for the shell's `awk`-extracted `"-"` sentinel — and,
with it, the `$(NF-7)` / `$(NF-5)` column arithmetic that existed solely because
group names contain spaces.

---

## 4. Open questions resolved

### OQ-1 — `AlterConfigs` batching and semantics

**Use `IncrementalAlterConfigs`, one request, all compacted topics.**

`kafka-go` exposes both APIs. The legacy `Client.AlterConfigs`
(`alterconfigs.go:64`) maps to Kafka's `AlterConfigs`, which is **full-replace**
for topic configs: sending only `cleanup.policy` would reset every other
topic-level override to broker default. That is exactly the clobber the PRD
flagged.

`Client.IncrementalAlterConfigs` (`incrementalalterconfigs.go:77`) maps to
Kafka's `IncrementalAlterConfigs`, which applies per-key operations. Using
`ConfigOperationSet` on `cleanup.policy=compact` leaves every other key untouched
— the exact semantics of the shell's
`kafka-configs.sh --alter --add-config cleanup.policy=compact`.

Batching: `IncrementalAlterConfigsRequest.Resources` is a slice, and all three
topics go in one request (FR-2.6). `protocol/incrementalalterconfigs`'s
`Request.Broker` routes to the controller (`:38`); the "at most one broker"
restriction in that method applies to `ResourceTypeBroker` resources, not topics,
so three topic resources in one request are fine.

Per-resource errors come back in `Resources[i].Error`. `nil` ⇒ applied; anything
else ⇒ fatal naming the topic. The call is unconditional over the compacted set —
newly created and pre-existing alike — because a topic created before this policy
existed is exactly the case FR-2.6 is for, and setting a config that is already
set is a no-op.

### OQ-2 — `ListOffsets` partition scope

**Enumerate partitions from metadata; do not assume partition 0.**

Two reasons, and the cost is zero:

1. Phase C already fetches metadata for routing reasons, so the partition list is
   in hand.
2. It is the *faithful* port. `kafka-consumer-groups.sh --reset-offsets
   --to-latest` resets **all** partitions of each named topic. Assuming partition
   0 would be a behaviour change, not a simplification, for any topic created
   out-of-band with more than one partition.

`ListOffsetsRequest.Topics` is `map[string][]OffsetRequest`; the union's every
`(topic, partition)` is submitted with `kafka.LastOffsetOf(p)`. `kafka-go` splits
this into one wire request per topic-partition
(`protocol/listoffsets/listoffsets.go:52`) and merges the responses. That is
O(topics × partitions) round trips — but over a **single pooled connection set**,
with no process starts, so ~170 sub-millisecond RPCs, not ~170 JVM cold starts.
NFR-2's O(1)-with-respect-to-topic-count constraint is written against the
*topic-creation* pass (FR-2.1), which is genuinely one request; the offset pass
was never O(1) in the shell version either.

A `PartitionOffsets.Error` on any partition is fatal, naming topic and partition.

### OQ-3 — Retry budget shape

**Exponential backoff, 250ms base, ×2, capped at 2s per attempt; 60s total
budget per retried call; whole-run deadline 240s.**

`kafkaops.withCoordinatorRetry` wraps only the three group-coordinator calls
(`DescribeGroups`, `OffsetCommit`, `OffsetFetch`) and retries only on
`errors.Is(err, kafka.NotCoordinatorForGroup)` or
`errors.Is(err, kafka.GroupCoordinatorNotAvailable)` (FR-5.1). Every other error
returns immediately — a retry loop that swallows unrelated errors turns a
diagnosable failure into a timeout.

The budget arithmetic (FR-5.2, NFR-4): worst case is one group exhausting 60s
while every other group succeeds first; the whole-run 240s deadline bounds the
total regardless, and both sit inside `activeDeadlineSeconds: 300`. The observed
spike condition — coordinator not yet elected on a brand-new cluster — resolves
in seconds, so 60s is roughly an order of magnitude of headroom.

The retry is *not* applied to `CreateTopics`, `Metadata`, or
`IncrementalAlterConfigs`: those route to the controller, not the group
coordinator, and cannot produce these two codes.

### OQ-4 — Module placement

Resolved in §2.1: flat, `services/atlas-kafka-precreate/`, following
`atlas-pr-bootstrap`. The `atlas.com/<name>` nesting is a requirement of the
shared root `Dockerfile`, which this image does not use.

### OQ-5 — `atlas-env` availability at wave 0

**Confirmed, no change.** All three overlays (`main`, `pr`, `pr-sparse`) generate
an `atlas-env` ConfigMap, and the base Job already consumes it via `envFrom`. The
tool changes the container image, not the env wiring.

---

## 5. Testability

### 5.1 The seam

```go
// kafkaops.AdminClient is the entire Kafka surface this tool uses.
type AdminClient interface {
    CreateTopics(context.Context, *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)
    IncrementalAlterConfigs(context.Context, *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error)
    Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error)
    ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
    DescribeGroups(context.Context, *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)
    OffsetCommit(context.Context, *kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)
    OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
}
```

`*kafka.Client` satisfies this as-is — no adapter methods, no re-declared types.
The interface exists so `topics` and `groups` can be driven by a hand-written
stub that returns canned responses, including typed `kafka.Error` values
(`kafka.TopicAlreadyExists`, `kafka.UnknownMemberID`,
`kafka.NotCoordinatorForGroup`), which is what FR-7.4 requires. Per CLAUDE.md, no
`*_testhelpers.go`: the stub is a `_test.go`-local struct with function fields,
constructed with the project's builder idiom where it needs more than two knobs.

### 5.2 Test inventory (FR-7.1 → FR-7.4)

| Unit | Cases |
|---|---|
| `discover.FromEnviron` | prefix selection; non-matching vars ignored; empty value skipped (FR-1.2); duplicate values collapsed (FR-1.3); the three config-status vars land in `Compact` (FR-1.4); a topic named by both a config-status var and another var stays compacted (FR-1.5); `_`-vs-`-` ordering determinism |
| `discover.Groups` | unset ⇒ empty (NG6, FR-7.2); single group; multi-line; blank lines dropped; `\r\n`; names with spaces and brackets round-trip byte-for-byte (FR-7.3); a name with a leading `-` |
| `discover.StateIsSeedable` | `Empty`, `Dead`, `""` ⇒ true; `Stable`, `PreparingRebalance`, `CompletingRebalance`, and an invented future state ⇒ false (FR-7.2) |
| `topics.Ensure` | one `CreateTopics` call for N topics, asserted by call count (acceptance criterion 2); compacted topics carry `cleanup.policy=compact`; all-`TopicAlreadyExists` response ⇒ success (FR-7.4); a mixed response with one `InvalidTopicException` ⇒ error naming that topic; `IncrementalAlterConfigs` issued once with all compacted resources and `ConfigOperationSet` |
| `groups.Seed` | seedable group ⇒ commit issued with `GenerationID: -1` / `MemberID: ""`; active state ⇒ no commit issued at all; `UnknownMemberID` on commit ⇒ non-fatal, group lands in the skipped set (FR-3.6); another commit error ⇒ fatal |
| `groups.Verify` | seeded group missing an offset ⇒ error (FR-4.2); skipped group missing offsets ⇒ no error, WARN bounded to 10 names + remainder (FR-4.3/4.4); `CommittedOffset == -1` is the missing sentinel |
| `kafkaops.withCoordinatorRetry` | retries on both coordinator codes and eventually succeeds; gives up at the budget; does **not** retry an unrelated error |

Everything above runs without a broker. The end-to-end run against a real KRaft
broker stays a manual acceptance step (PRD acceptance criteria 14) — this repo
has no Kafka test harness and adding one is not in scope.

---

## 6. Build and deploy wiring

### 6.1 Dockerfile — `services/atlas-kafka-precreate/Dockerfile`

Two stages. Build: `golang:${GO_VERSION}-alpine${ALPINE_VERSION}`, `CGO_ENABLED=0
go build -trimpath -ldflags="-s -w"`. Runtime: `scratch`, plus
`/etc/ssl/certs/ca-certificates.crt` copied from the build stage, `USER 65532`,
`ENTRYPOINT ["/atlas-kafka-precreate"]` (NFR-9). No shell, no Kafka CLI, no JRE.

Build context is `services/atlas-kafka-precreate` (like `atlas-pr-bootstrap`),
which is possible only because the module depends on no `libs/atlas-*`.

### 6.2 `docker-bake.hcl`

```hcl
target "atlas-kafka-precreate" {
  context    = "services/atlas-kafka-precreate"
  dockerfile = "Dockerfile"
  tags       = ["atlas-kafka-precreate:${ATLAS_IMAGE_TAG}"]
}

group "all-services" {
  targets = concat(go_services, ["atlas-ui", "atlas-pr-bootstrap", "atlas-kafka-precreate"])
}
```

**Guard interaction to get right.**
`tools/service-registration-guard.sh:254-256` scans `docker-bake.hcl` with
`^\s*"(atlas-[a-z0-9-]+)",?\s*$` and fails on any match that is not a
`go-service`. That regex matches a line consisting solely of a quoted name — i.e.
the `go_services = [...]` entries. It does **not** match `target "…" {` (trailing
brace) nor the single-line `concat(…)` (extra content). Adding the target in the
form above is therefore safe; adding the name on its own line inside a multi-line
list would fail the guard.

Second thing to know: `tools/verify.sh`'s bake step selects only
`type == "go-service"` entries (`verify.sh:293-302`), so **`verify.sh` never
builds this target**. The image must be built by hand at least once
(`docker buildx bake atlas-kafka-precreate`) before the PR; CI's `cideps`
enrichment includes any `services.json` entry with `docker_image` set regardless
of type, which is precisely why the bake target has to exist.

### 6.3 `.github/config/services.json`

```json
{
  "name": "atlas-kafka-precreate",
  "type": "support-image",
  "path": "services/atlas-kafka-precreate",
  "docker_image": "ghcr.io/chronicle20/atlas-kafka-precreate/atlas-kafka-precreate",
  "docker_context": "services/atlas-kafka-precreate"
}
```

Mirrors the `atlas-pr-bootstrap` entry exactly in shape.

### 6.4 The Job manifest — `deploy/k8s/base/atlas-kafka-precreate.yaml`

Changes: image → the new one, `command`/script volume/volumeMount removed,
`activeDeadlineSeconds: 300` added (NFR-4). Retained unchanged: sync-wave 0, the
`Force=true,Replace=true` sync-option, `envFrom: atlas-env`, `backoffLimit: 3`,
`ttlSecondsAfterFinished: 600`, `restartPolicy: OnFailure`, and the container
name `kafka-precreate`.

**Two hard constraints on this manifest, both from
`.github/workflows/pr-validation.yml:1091-1095`:**

1. The container must stay at **index 0** — the JSON-6902 patch targets
   `/spec/template/spec/containers/0/env`.
2. The container must keep **`envFrom` only, with no `env:` key**. The patch uses
   `op: add` on the whole `env` path, which *creates* the key. If the base
   manifest grows an `env:` key, that patch silently **replaces** it, and
   `KAFKA_CONSUMER_GROUP` becomes the container's only env var. The workflow's own
   comment says so; this design inherits the constraint rather than testing it.

If PR #1463's mitigations (CPU/memory limits, cross-namespace pod anti-affinity)
have landed on `main` by the time this branch rebases, they are retained as-is —
they are orthogonal to the image swap, and the limits stay a useful ceiling even
though NFR-3 expects the tool to sit far below them. This branch's base predates
#1463, so the rebase is where that reconciliation happens.

### 6.5 Deletions and comment repair (FR-6.1 → FR-6.3)

- Delete `deploy/k8s/base/kafka-precreate.sh` and
  `deploy/k8s/base/atlas-kafka-precreate_test.sh`.
- Remove the `atlas-kafka-precreate-script` `configMapGenerator` entry from
  `deploy/k8s/base/kustomization.yaml:87-92`, including its comment.
- Repair the two comments that name the deleted script:
  `deploy/k8s/base/atlas-kafka-precreate.yaml`'s header block, and
  `deploy/k8s/overlays/pr-sparse/kustomization.yaml:274`
  (`kafka-precreate.sh:176`) — plus the equivalent anchor comment in
  `deploy/k8s/overlays/pr/kustomization.yaml` if it carries the same reference.
  A comment pointing at a deleted file is how the next reader loses an hour.

The Job manifest's long header comment is the only surviving prose explanation of
*why* wave 0 exists, why `Force=true,Replace=true` is there, and why an active
group is skipped rather than reset. It must be preserved and updated in place —
not deleted along with the script it references. The parts that describe CLI
mechanics move into Go doc comments on `groups.Seed` and `groups.Verify`.

---

## 7. Logging and exit codes

`logrus` with `WithFields`, structured rather than interpolated (NFR-7):

```
INFO  phase=create  plain=167 compact=3 created=170 existing=0
INFO  phase=alter   topics=3 policy=compact
INFO  phase=seed    group="Channel Service - … [pr-123]" outcome=seeded partitions=170
INFO  phase=seed    group="World Service [pr-123]"       outcome=skipped state=Stable
INFO  phase=seed    seeded=1 skipped=1
WARN  phase=verify  group="World Service [pr-123]" missing=12 of=170 topics="a, b, …" more=2
INFO  phase=verify  ok
```

Exit codes: `0` success — including all-already-exists, all-skipped, and the NG6
no-op run. Non-zero on any of: `BOOTSTRAP_SERVERS` unset (FR-1.6), a non-tolerated
per-topic create error (FR-2.4), an `IncrementalAlterConfigs` resource error, a
metadata settle timeout, a non-`UnknownMemberID` commit error, a seeded group
failing verification (FR-4.2), or the run deadline. Every fatal path names the
topic or group and the underlying typed error (NFR-8). The Job's
`backoffLimit: 3` means a transient failure still gets three retries at the pod
level.

---

## 8. Alternatives considered

**Port the script to `rpk` instead of Go.** `atlas-pr-bootstrap` already ships
`rpk`, which is a single static Go binary with no JVM — so it removes the cold
start too. Rejected: it keeps the whole class of defect this task exists to
delete. Group names with spaces still need column-anchored parsing of CLI output,
`TopicAlreadyExists` is still a string match, and the tool would be pinned to
`rpk`'s `--format json` schema (which `atlas-pr-bootstrap` already treats as a
stability boundary with checked-in fixtures). The PRD's typed-error requirements
(FR-2.3, FR-3.6) are not satisfiable through any CLI.

**Add admin wrappers to `libs/atlas-kafka`.** Rejected by PRD §2 non-goals, and
the design agrees: a shared-lib API with exactly one consumer is scope creep,
and it would put every service's `verify.sh` run behind a lib change.

**Keep the list-then-diff pass as a belt-and-braces check.** Rejected. Spike Q1
proved per-topic `TopicAlreadyExists` makes the create idempotent on its own
(FR-2.5), and the diff is precisely where PR #1463's locale-collation bug lived.
Keeping it would preserve the bug class for no benefit.

**Skip the metadata settle and assume partition 0.** Rejected — see OQ-2. It
would be both a behaviour change against `--to-latest` and a live routing hazard
against the transport's metadata cache.

**Replace the env scrape with a code-derived topic manifest.** Explicitly out of
scope (PRD §2, issue #1464). The scrape ports faithfully; changing the discovery
mechanism in the same change as the transport rewrite would make a regression
impossible to attribute.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| Metadata cache staleness makes offset routing fail on a first sync | Phase C settle loop + `MetadataTTL: 1s`; fails with named topics rather than an opaque routing error |
| `verify.sh` never builds the new bake target | Build it by hand before the PR; §6.2 records why the gate does not cover it |
| A future edit adds `env:` to the Job container, silently breaking CI's group patch | §6.4 constraint recorded here and repeated as a comment in the manifest |
| `go.work` change makes every gate run a full 86-module fan-out | Expected; use `--base` while iterating, flagless run once at the end |
| Behaviour drift from the shell version goes unnoticed | The test inventory (§5.2) is written as a per-FR port of the shell's assertions, not as fresh coverage |
| `IncrementalAlterConfigs` unsupported on an older broker | Deployed brokers are `apache/kafka:3.7.2` (KRaft); the API has existed since Kafka 2.3. A per-resource error is fatal and named, so a mismatch surfaces immediately rather than silently no-op'ing |

---

## 10. Requirement traceability

| PRD | Where |
|---|---|
| FR-1.1 → FR-1.6 | §3 Phase A |
| FR-2.1 → FR-2.5 | §3 Phase B |
| FR-2.6 | §4 OQ-1 |
| FR-3.1 → FR-3.8 | §3 Phase D |
| FR-4.1 → FR-4.5 | §3 Phase E |
| FR-5.1, FR-5.2 | §4 OQ-3 |
| FR-6.1 → FR-6.4 | §6.4, §6.5 |
| FR-7.1 → FR-7.4 | §5 |
| NFR-1 → NFR-3 | §3 (one create request; pooled connections; no process starts) |
| NFR-4 → NFR-6 | §3 (240s deadline), §6.4, §3 Phase D step 2 |
| NFR-7, NFR-8 | §7 |
| NFR-9 | §6.1 |
| NFR-10 | n/a per PRD §6 |
| OQ-1 → OQ-5 | §4 |
