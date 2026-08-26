# atlas-kafka-precreate

A sync-wave-0 Kubernetes Job (`deploy/k8s/base/atlas-kafka-precreate.yaml`)
that pre-creates every Kafka topic an Atlas environment declares before any
other service or Job starts, and seeds committed offsets for any override
consumer group so a first (or re-)sync never replays the full retention
window.

It replaces `deploy/k8s/base/kafka-precreate.sh`, which ran the whole pass
through the `apache/kafka:3.7.2` image's JVM CLI — one JVM cold start per
topic. This tool does the same work as a single Go binary talking directly
to the Kafka admin protocol: one `CreateTopics` request and one
`IncrementalAlterConfigs` request cover the entire topic set, regardless of
how many topics an environment declares.

## Environment variables

| Variable | Required | Meaning |
| --- | --- | --- |
| `BOOTSTRAP_SERVERS` | yes | Comma-separated `host:port` list of Kafka brokers. Empty or unset fails the Job immediately, before any Kafka client is constructed. |
| `COMMAND_TOPIC_*`, `EVENT_TOPIC_*` | no | Every variable with one of these prefixes and a non-empty value names a topic to create. Three specific `EVENT_TOPIC_*` variables (the configuration-projection topics) are created and converged with the compacted topic configuration — `cleanup.policy=compact`, `max.compaction.lag.ms=600000`, `segment.ms=600000`, `min.cleanable.dirty.ratio=0.01` (see [Compacted topics](#compacted-topics)); every other topic is created plain. |
| `KAFKA_CONSUMER_GROUP` | no | Newline-delimited list of override consumer group IDs (one per line). Unset or empty means no groups to seed — the tool creates topics and exits, running exactly two RPCs. |

## Phases

1. **Discover** — scrape the process environment for topic-shaped
   variables and classify each into plain vs. compacted, and parse
   `KAFKA_CONSUMER_GROUP` into a group ID list.
2. **Create/Alter** — create the full topic union in one `CreateTopics`
   request, then apply the compacted topic configuration to the compacted
   topics in one `IncrementalAlterConfigs` request. This phase always runs;
   topics are created whether or not there is a group to seed. The alter
   runs over the whole compacted set on every run, not just the topics this
   run created, so a topic created by an earlier version of the tool
   converges to the current configuration. `cleanup.policy=compact` on its
   own is inert — the log cleaner never touches a partition's active
   segment, so a topic whose segment never rolls is never cleaned;
   `max.compaction.lag.ms` is what lowers the segment-roll deadline and
   therefore what makes the policy mean anything.
3. **Settle** — poll `Metadata` until every created topic is visible in the
   client's own cached cluster view, then read end-of-log offsets for every
   (topic, partition) pair. Skipped when `KAFKA_CONSUMER_GROUP` is unset.
   Topic visibility is not leader election: immediately after `CreateTopics`
   on a cold cluster, `ListOffsets` for a just-created partition can return
   `NotLeaderForPartition`/`LeaderNotAvailable` until a leader is elected.
   That read is retried, bounded by the same backoff budget as phases 4-5.
4. **Seed** — for each override group currently in a seedable state
   (`Empty`, `Dead`, or absent — never a group with an active member),
   commit end-of-log offsets as a non-member. A group already active is
   left untouched and reported as skipped. On a cluster where the
   `__consumer_offsets` coordinator is still being elected, a commit can
   return `NotCoordinatorForGroup`/`GroupCoordinatorNotAvailable` for a
   partition inside an otherwise-successful response; that is retried the
   same as a transport-level coordinator error, since the commit is
   idempotent.
5. **Verify** — for each group, confirm every (topic, partition) in the
   union carries a committed offset. A seeded group missing an offset is a
   fatal error; a skipped (already-active) group missing an offset is a
   warning, since a live consumer already owns that group's progress. The
   same per-partition coordinator-election retry as phase 4 applies to the
   `OffsetFetch` read.

## Compacted topics

The three configuration-projection topics are compacted: each carries the
latest record per key, which is exactly the projection replay model. They
are created with, and converged to, this configuration:

| Config | Value | Purpose |
| --- | --- | --- |
| `cleanup.policy` | `compact` | Retain the latest record per key instead of deleting by age. |
| `max.compaction.lag.ms` | `600000` (10 min) | Upper bound on how long a record may remain uncompacted. For a compacted topic the broker's effective segment-roll deadline is `min(segment.ms, max.compaction.lag.ms)`, so this lowers the roll deadline to 10 minutes — and the roll is what hands the cleaner a non-active segment to work on. |
| `segment.ms` | `600000` (10 min) | The same deadline on the policy-independent knob. |
| `min.cleanable.dirty.ratio` | `0.01` | Lets the cleaner select a segment whose dirty fraction is below the 0.5 default. |

Which knob is doing the work, stated plainly so nobody deletes the wrong
one: **`max.compaction.lag.ms` is load-bearing.** Verified against
`apache/kafka:4.1.1`, a topic carrying `cleanup.policy=compact` and
`max.compaction.lag.ms` alone — no `segment.ms` — rolled its segment on the
next append past the lag and was compacted from 200 records over 3 keys
down to 3. `segment.ms` at the same value is exactly redundant given the
`min()` rule; it is set so the roll bound survives someone changing
`cleanup.policy`, and so the cadence is readable from a `--describe`
without knowing that rule. `min.cleanable.dirty.ratio` was not isolated by
that experiment and the forced-cleaning path bypasses the ratio check
anyway — it is cheap steady-state insurance, not a load-bearing knob.

The roll is triggered **on append**, not by a timer: a quiescent compacted
topic does not roll and does not churn segments. Index and timeindex files
are preallocated for the active segment only and trimmed to their real size
on roll, so the steady-state footprint is one active segment's preallocated
pair plus a compacted tail of kilobytes.

### Verifying compaction on a live broker

Log-start-offset is **not** the signal — compaction rewrites a segment in
place, keeping surviving records at their original offsets, so a fully
compacted partition can still report a log start offset of `0`. The correct
signals, in decreasing sharpness:

1. `grep <topic> /var/lib/kafka/data/kafka/cleaner-offset-checkpoint` returns
   a line — the direct statement that the cleaner has processed the partition.
2. The partition directory shows more than one `.log`, with the base-0 `.log`
   far smaller than the uncompacted total.
3. A `--from-beginning` consume returns on the order of the key count rather
   than the record count.

## Exit codes

- `0` — every phase that ran completed successfully (including the
  no-groups-to-seed short circuit after phase 2, and a skipped group logged
  as a warning in phase 5).
- `1` — any phase returned an error: `BOOTSTRAP_SERVERS` unset, a Kafka RPC
  failure, the tool's own 240-second deadline expiring, or a seeded group
  failing the phase-5 verification gate. The error is logged as a single
  structured JSON line (`logrus.JSONFormatter`, matching the rest of the
  Atlas Go services) before the process exits.

## Building the image

`tools/verify.sh`'s bake step only builds targets of type `go-service`, and
this tool is not one, so its image is never built by the gate path. Build it
by hand before opening a PR that changes this directory:

```sh
docker build -t atlas-kafka-precreate:local services/atlas-kafka-precreate
```
