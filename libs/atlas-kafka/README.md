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

### Partition-count changes

Neither engine watches for partition-count changes: today's `ReaderConfig`
never sets `WatchPartitionChanges`, and kafka-go forwards that `false` into
its internal `ConsumerGroupConfig`, so `groupConfig()` matches it deliberately.
If a topic is ever repartitioned, set
`ConsumerGroupConfig.WatchPartitionChanges = true` in
`consumer/group.go`'s `groupConfig()` (kafka-go polls every
`PartitionWatchInterval`, default 5 s). That is a deliberate opt-in, not
something to enable as a side effect of another change.
