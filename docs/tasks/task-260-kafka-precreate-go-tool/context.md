# Context — task-260 kafka-precreate Go tool

Companion to [plan.md](plan.md). What an executor needs that the plan's task
bodies do not repeat.

## What this replaces

`deploy/k8s/base/kafka-precreate.sh` (436 lines of bash) running in
`apache/kafka:3.7.2`, driving `kafka-topics.sh` / `kafka-configs.sh` /
`kafka-consumer-groups.sh`. Every CLI invocation is a JVM cold start
(~1-1.5s) performing exactly one operation; with ~170 topics that is paid
~170 times per run. Read the script before Task 3 or Task 4 — it is the
behavioural contract being ported, and its comments carry the *why* for
every rule the plan restates as a requirement.

Function → package mapping, so a reviewer can read old against new:

| shell | Go |
|---|---|
| `precreate_topics` env scrape (lines 50-72) | `discover.FromEnviron` |
| `precreate_topics` create + alter (lines 78-113) | `topics.Ensure` |
| `group_state` (lines 143-151) | the probe inside `groups.Seed` |
| `state_is_seedable` (lines 168-173) | `discover.StateIsSeedable` |
| `seed_group` (lines 216-241) | the commit inside `groups.Seed` |
| `seed_override_offsets` (lines 245-300) | `groups.Seed` |
| `verify_group_offsets` (lines 322-405) | `groups.Verify` |
| `main` (lines 407-411) | `main.run()` |

Nothing in the shell corresponds to `topics.Settle` or
`kafkaops.WithCoordinatorRetry`. Both exist because a direct protocol client
faces problems the JVM CLI handled internally — see "Two things the port
adds" below.

## Key files

**New module** — `services/atlas-kafka-precreate/`, module
`atlas.com/kafka-precreate`. Flat, no `atlas.com/<name>` nesting: that
nesting exists only because the shared root `Dockerfile` discovers the module
with `ls -d services/${SERVICE}/atlas.com/*/ | head -1` (`Dockerfile:88`,
`:104`), and this image does not use the shared Dockerfile.
`services/atlas-pr-bootstrap/` is the precedent for a flat support image.

**Read before editing:**

- `deploy/k8s/base/kafka-precreate.sh` — the port source (deleted in Task 6).
- `deploy/k8s/base/atlas-kafka-precreate.yaml` — the Job. Its header comment
  is the only surviving prose on why wave 0 exists; preserve and update it.
- `.github/workflows/pr-validation.yml:1091-1095` — the JSON-6902 patch that
  injects `KAFKA_CONSUMER_GROUP`. Two constraints follow (plan Task 6 Step 3).
- `tools/service-registration-guard.sh:119` (go-service `main.go` glob) and
  `:254-256` (the bake-name reverse check).
- `tools/verify.sh:180-189` (module discovery + the `go.work` fan-out
  predicate) and `:293-302` (bake selects `go-service` only).
- `services/atlas-pr-bootstrap/Dockerfile` — the flat-support-image shape.

**kafka-go v0.4.51 source** — `go list -m -f '{{.Dir}}' github.com/segmentio/kafka-go`.
Every struct field name in the plan was read from there, not remembered.

## Decisions the plan encodes

**`IncrementalAlterConfigs`, not `AlterConfigs`** (design OQ-1). The legacy
API is full-replace for topic configs; sending only `cleanup.policy` would
reset every other topic-level override to broker default. One request carries
all three compacted topics.

**Partitions enumerated from metadata, not assumed to be 0** (design OQ-2).
`topics.Settle` already fetches metadata for routing reasons, so the
partition list is free, and it is the faithful port —
`kafka-consumer-groups.sh --reset-offsets --to-latest` resets *all*
partitions.

**Retry wraps only the three coordinator calls** (design OQ-3):
`DescribeGroups`, `OffsetCommit`, `OffsetFetch`, and only on
`NotCoordinatorForGroup` / `GroupCoordinatorNotAvailable`. 250ms base, ×2,
capped at 2s, 60s budget, inside a 240s whole-run deadline, inside the Job's
`activeDeadlineSeconds: 300`.

**No list-then-diff** (FR-2.5). Per-topic `TopicAlreadyExists` makes the
create idempotent on its own. The diff is exactly where PR #1463's
locale-collation bug lived (topic names mix `_` and `-`, where locale
collation and byte order disagree); in Go it is a map difference and the
failure class does not exist.

**Decisions this plan makes that the design left open:**

- A top-level `OffsetFetchResponse.Error` is fatal for *both* seeded and
  skipped groups. The design only covers per-partition results. Rationale: an
  RPC-level failure is not the same as a missing offset, and silently
  WARN-ing on an authorization failure would hide a real misconfiguration.
- A topic-level metadata `Error` during `topics.Settle` is a not-yet-present
  signal, not a fatal — the loop keeps polling until the ceiling. A
  freshly-created topic legitimately reports `UnknownTopicOrPartition` for a
  moment.
- `discover.Groups` drops only *empty* lines. A line of spaces is a valid
  group name and survives; the plan's test table pins this.
- Per-request timeouts come from `kafka.Client.Timeout` (60s), since
  `CreateTopicsRequest` has no timeout field — kafka-go derives the wire
  timeout from `Client.Timeout` and the context deadline
  (`client.go:125-139`), halving it internally.

## Two things the port adds

**Metadata settle (`topics.Settle`).** kafka-go's transport serves `Metadata`
from a TTL cache (`transport.go:352-374`, default 6s). `ListOffsets` is split
per topic-partition and routed to leaders looked up in that same cache
(`protocol/listoffsets/listoffsets.go:37-93`), so a topic missing from the
cache resolves to `Broker{ID: -1}`. Without the settle loop, a first-sync run
could create 170 topics and then fail to route offsets for them. Mitigated
twice: `Transport.MetadataTTL = 1s` plus the 250ms/30s poll loop. The shell
version never had this hazard because the JVM CLI opened a fresh admin client
per invocation.

**Coordinator retry.** Spike Q4: the first group request against a
freshly-created cluster returned `Not Coordinator For Group` (16) because
`__consumer_offsets` had not settled. Real production requirement — the Job
runs at wave 0 against an environment that may be brand new.

## Sizing notes

Eight tasks. Tasks 1-5 are one package each plus its tests; each stays under
five files and one module. Tasks 6 and 7 are the cutover, deliberately split:
Task 6 is base + build registration (6 files), Task 7 is the three overlays
(3 files). A single combined cutover task would have touched ten files and is
exactly the shape that produces a `PARTIAL` hand-back.

Nothing here is deliberately oversized.

Two things a reviewer should check that no gate can:

1. **`tools/verify.sh` never builds this image.** Its bake step filters to
   `type == "go-service"`. Task 5 Step 4 and Task 8 Step 2 build it by hand;
   both are mandatory, not belt-and-braces.
2. **The manual broker acceptance run** (Task 8 Step 3). This repo has no
   Kafka test harness and adding one is out of scope, so PRD acceptance
   criteria 14 is a human step. The wall-clock number belongs in the PR
   description.

## Rebase note

This branch's base predates PR #1463, which landed CPU/memory limits and
cross-namespace pod anti-affinity on the Job (plus the list-then-diff and
`-P 16` → `-P 4` changes to the shell script, which this task deletes
wholesale). If #1463 is on `main` at rebase time, **keep** the limits and the
anti-affinity — they are orthogonal to the image swap and remain a useful
ceiling even though the Go tool sits far below them. The shell-script
mitigations rebase away with the script.

The `go.work` change makes every `tools/verify.sh` run on this branch a full
86-module fan-out (`verify.sh:186-189`). Expected and unavoidable for a new
module; use `--base <last-gated-commit>` while iterating, flagless once at the
end.
