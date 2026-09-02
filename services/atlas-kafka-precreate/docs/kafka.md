## Topics Consumed

None. This service does not consume Kafka messages.

## Topics Produced

None. This service does not produce Kafka messages. It targets, as the
objects of Kafka Admin protocol operations, the topic names discovered from
`COMMAND_TOPIC_*` and `EVENT_TOPIC_*` environment variables (see
`docs/domain.md`).

## Message Types

The service issues the following `github.com/segmentio/kafka-go` Admin
protocol request/response types, via the `kafkaops.AdminClient` interface
(satisfied by `*kafka.Client`):

- `CreateTopicsRequest` / `CreateTopicsResponse`
- `IncrementalAlterConfigsRequest` / `IncrementalAlterConfigsResponse`
- `MetadataRequest` / `MetadataResponse`
- `ListOffsetsRequest` / `ListOffsetsResponse`
- `DescribeGroupsRequest` / `DescribeGroupsResponse`
- `OffsetCommitRequest` / `OffsetCommitResponse`
- `OffsetFetchRequest` / `OffsetFetchResponse`

`CreateTopicsRequest` entries for a compacted topic carry `ConfigEntries`
naming `cleanup.policy`, `max.compaction.lag.ms`, `segment.ms`, and
`min.cleanable.dirty.ratio`; `IncrementalAlterConfigsRequest` resources for
the compacted set carry the same four config names, each with
`ConfigOperation: kafka.ConfigOperationSet`.

`OffsetCommitRequest` is issued with `GenerationID: -1` and `MemberID: ""`.

## Transaction Semantics

None. No Kafka transactions (`kafka.Transaction`) are used.
