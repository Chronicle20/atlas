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
| `COMMAND_TOPIC_*`, `EVENT_TOPIC_*` | no | Every variable with one of these prefixes and a non-empty value names a topic to create. Three specific `EVENT_TOPIC_*` variables (the configuration-projection topics) are created with `cleanup.policy=compact`; every other topic is created plain. |
| `KAFKA_CONSUMER_GROUP` | no | Newline-delimited list of override consumer group IDs (one per line). Unset or empty means no groups to seed — the tool creates topics and exits, running exactly two RPCs. |

## Phases

1. **Discover** — scrape the process environment for topic-shaped
   variables and classify each into plain vs. compacted, and parse
   `KAFKA_CONSUMER_GROUP` into a group ID list.
2. **Create/Alter** — create the full topic union in one `CreateTopics`
   request, then apply `cleanup.policy=compact` to the compacted topics in
   one `IncrementalAlterConfigs` request. This phase always runs; topics
   are created whether or not there is a group to seed.
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
