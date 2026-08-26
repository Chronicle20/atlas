# atlas-kafka
Module which provides uniform kafka operations.

## Consumer engines

`consumer` ships two implementations, selected at process start by
`KAFKA_CONSUMER_ENGINE`:

| Value | Engine | Notes |
|---|---|---|
| unset / `consumergroup` | `kafka.ConsumerGroup` + `Generation` | Default. Partition assignment is read directly from the generation; a consumer holding zero partitions is healthy-idle and never recreates; a stalled partition rebuilds only its own reader, with no group rejoin. |
| `reader` | legacy `*kafka.Reader` with `GroupID` | Rollback path, retained for one release (task-209 FR-5.1). |

Both engines use the same consumer-group IDs and commit the same offset value
for the same delivered message — `msg.Offset + 1` — so switching between them
is a pod restart. No topic, offset, consumer-group or database migration is
involved, and it is safe in either direction.

`GET /api/debug/consumers` reports `engine`, `assignedPartitions`,
`generationId` and `lastAssignmentAt` per consumer. An empty
`assignedPartitions` on the `consumergroup` engine means healthy-idle, not
stuck: with single-partition topics and two replicas, one member of every
group is expected to hold nothing.

`recreateCount` means different things per engine. On `reader` it counts group
rejoins (each rebalances every member of the group); on `consumergroup` it
counts local partition-reader rebuilds. Do not compare the number across a
rollback.

### Missing topics and partition-count changes

The `consumergroup` engine **will not join a consumer group for a topic that
does not exist.** Before each join it asks the cluster for the topic's
partition count; while the answer is a definite "no such topic, or no
partitions" it waits — logging a warn on entry and then once a minute — and
joins as soon as the topic appears. If the lookup cannot answer (broker
unreachable, timeout, any other error) the consumer joins immediately, exactly
as it always did: a broker blip must never hold a consumer out of its group.

This exists because of the 2026-08-26 `atlas-pr-1449` incident. kafka-go's
group leader forgives a missing topic when computing assignments and hands the
member an empty assignment, expecting its own partition watcher to trigger a
rebalance once the topic appears. The watcher does not deliver: its startup
read returns on *any* error including `UnknownTopicOrPartition`
(`consumergroup.go:512-518`) — the very error its ticker branch tolerates
(`consumergroup.go:527-529`) — so the goroutine gives up and the member stays
permanently deaf. Twenty-two consumers in one pod raced `atlas-kafka-precreate`
and lost; they were silently deaf for the process's lifetime. That upstream
asymmetry is the reason the pre-join wait exists; a future kafka-go bump that
fixes it should revisit the gate.

The same logic covers the runtime case. When a generation gives this member no
partitions, it asks once whether the topic still has any. If it definitively
does not, the consumer logs a warn and **leaves the group** until it comes
back, rather than holding a doomed membership. If it does — the normal case,
since every atlas-main topic has one partition and services run `replicas: 2` —
that is healthy-idle, logged at debug, and the member parks awaiting the next
rebalance.

`WatchPartitionChanges` is `true` on the `consumergroup` engine, so a topic
repartitioned from 1 to N partitions is picked up within
`PartitionWatchInterval` (kafka-go default, 5s) with no restart. The legacy
`reader` engine leaves it at kafka-go's `false`. **Do not enable this flag
without the two guards above**: against a missing topic, the watcher's failing
startup read ends each generation and kafka-go rejoins with no backoff, which
is a group-wide rebalance storm, strictly worse than the silent deafness it was
meant to fix.

`GET /api/debug/consumers` reports `topicMissingObservations` and
`lastTopicMissingAt`. The counter is monotonic and is **superseded, not
cleared**: after recovery it keeps the history, which is exactly the
post-mortem evidence the incident lacked. A consumer is currently healthy iff
`assignedPartitions` is non-empty, or `lastAssignmentAt` is after
`lastTopicMissingAt` — the same read an operator already performs for
`idleTicks`.
