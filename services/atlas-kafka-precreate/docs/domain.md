## Kafka topic and consumer-group offset precreation

### Responsibility

Discover the set of Kafka topics and override consumer groups an Atlas
environment declares through process environment variables, ensure every
declared topic exists on the cluster with the correct cleanup configuration,
and — for any declared override consumer group that is safe to seed — commit
end-of-log offsets so a first (or re-)sync does not replay the full retention
window.

### Core Models

- `discover.Topics` — `Plain []string` and `Compact []string`, each sorted
  and de-duplicated, never nil. `Union()` returns the sorted, de-duplicated
  merge of both.
- `topics.EnsureResult` — `Created int`, `Existing int`: tally of topics
  created new versus already present, as reported by `topics.Ensure`.
- `topics.SettleConfig` — `Poll`, `Ceiling`, `Sleep`, `Now`: poll cadence and
  ceiling `topics.Settle` applies while waiting for topic visibility;
  `Poll` defaults to 250ms, `Ceiling` to 30s.
- `groups.SeedResult` — `Seeded []string`, `Skipped []string`,
  `States map[string]string`: outcome of `groups.Seed` per group, in input
  order. `WasSkipped(group)` reports whether `group` is in `Skipped`.
- `groups.VerifyReport` — `Group string`, `Total int`, `Missing []string`:
  outcome of `groups.Verify` for one group — the count of (topic, partition)
  pairs checked and the sorted topic names with at least one uncommitted
  partition.
- `kafkaops.RetryConfig` — `Base`, `Max`, `Budget`, `Sleep`, `Now`: bounded
  exponential backoff parameters. `kafkaops.DefaultRetryConfig()` returns
  250ms base, 2s per-attempt cap, 60s total budget.
- `kafkaops.AdminClient` — the interface of `*kafka.Client` operations the
  tool depends on: `CreateTopics`, `IncrementalAlterConfigs`, `Metadata`,
  `ListOffsets`, `DescribeGroups`, `OffsetCommit`, `OffsetFetch`.

### Invariants

- A topic-shaped environment variable is one whose name has prefix
  `COMMAND_TOPIC_` or `EVENT_TOPIC_` and whose value is non-empty; every
  other variable is ignored by discovery.
- Exactly three variable names are classified compacted:
  `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`,
  `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`,
  `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS`. Every other topic-shaped
  variable is classified plain.
- `discover.Topics.Plain` and `discover.Topics.Compact` are disjoint: a value
  classified compact is removed from the plain set if also present there.
- `discover.Groups` splits its input on `\n`, drops only empty lines (after
  trimming a trailing `\r`), does not trim interior whitespace, and
  preserves input order.
- `discover.StateIsSeedable` is an allowlist: only `"Empty"`, `"Dead"`, or
  `""` (absent/unknown) are seedable; every other state, including one not
  otherwise recognized, is not seedable.
- Every compacted topic is created with, and unconditionally converged to
  (on every run, not only for topics created that run), the same four-entry
  configuration: `cleanup.policy=compact`, `max.compaction.lag.ms=600000`,
  `segment.ms=600000`, `min.cleanable.dirty.ratio=0.01`.
- `topics.Ensure` performs no pre-flight existence check; `CreateTopics` is
  issued once for the full topic union, and a per-topic
  `kafka.TopicAlreadyExists` error is counted as `Existing` rather than
  treated as fatal. Any other per-topic error is fatal.
- `topics.Settle` returns immediately with an empty map if given no topic
  names, and otherwise polls `Metadata` until every requested topic name is
  present with at least one partition, or the configured ceiling elapses.
- `groups.Seed` returns a zero `SeedResult` immediately, without issuing any
  Kafka call, when given an empty group ID list.
- `groups.Seed` commits offsets as a non-member: `GenerationID -1`,
  `MemberID ""`.
- A group is skipped by `groups.Seed` if its probed state is not seedable,
  or if the offset commit itself returns `kafka.UnknownMemberId` on any
  partition (the commit-race case); a group's `Seeded` and `Skipped`
  memberships are mutually exclusive.
- `groups.Verify` returns `nil, nil` immediately when given an empty group
  ID list.
- For a group present in `seeded.Skipped`, a missing committed offset is
  recorded in `VerifyReport.Missing` but does not cause `Verify` to return
  an error. For a group not in `seeded.Skipped`, the first topic with a
  missing or negative committed offset causes `Verify` to return an error
  naming the group and topic.
- A top-level `OffsetFetchResponse.Error` is always fatal to `Verify`,
  regardless of whether the group was seeded or skipped.
- `kafkaops.WithCoordinatorRetry` retries only on `kafka.NotCoordinatorForGroup`
  or `kafka.GroupCoordinatorNotAvailable`; every other error is returned on
  the first attempt.
- `kafkaops.WithLeaderRetry` retries only on `kafka.NotLeaderForPartition` or
  `kafka.LeaderNotAvailable`; every other error is returned on the first
  attempt.
- Both retry helpers stop retrying, returning an error, once
  `now().Sub(start)+backoff > cfg.Budget`, and also return immediately if
  `ctx.Err()` is non-nil before an attempt.

### State Transitions

The tool runs five phases in sequence from `main.run()`:

1. **Discover** — `discover.FromEnviron` scrapes `os.Environ()` into a
   `discover.Topics`; `discover.Groups` parses `KAFKA_CONSUMER_GROUP` into a
   group ID list.
2. **Create/Alter** — `topics.Ensure` creates the topic union and applies the
   compacted configuration to the compacted set. This phase always runs. If
   `groupIDs` is empty, `main.run()` returns after this phase.
3. **Settle** — `topics.Settle` polls `Metadata` for topic visibility, then
   `topics.EndOffsets` reads end-of-log offsets for every (topic, partition)
   pair.
4. **Seed** — `groups.Seed` probes each group's state via `DescribeGroups`
   and, for a seedable state, commits end-of-log offsets via
   `OffsetCommit`.
5. **Verify** — `groups.Verify` fetches committed offsets via `OffsetFetch`
   and checks that every (topic, partition) in the union carries one.

A consumer group's observed state, as returned by `groups.probeState`
(`DescribeGroups`), determines eligibility for phase 4 through
`discover.StateIsSeedable`.

### Processors

- `discover.FromEnviron(environ []string) Topics` — classifies topic-shaped
  environment variables into plain and compacted sets.
- `discover.Groups(raw string) []string` — parses the newline-delimited
  consumer group list.
- `discover.StateIsSeedable(state string) bool` — reports whether a group
  state is safe to seed.
- `topics.Ensure(ctx, client, addr, discover.Topics) (EnsureResult, error)` —
  creates the topic union and applies the compacted topic configuration.
- `topics.Settle(ctx, client, addr, names []string, SettleConfig) (map[string][]int, error)` —
  polls for topic visibility and returns sorted partition IDs per topic.
- `topics.EndOffsets(ctx, client, addr, partitions map[string][]int, RetryConfig) (map[string]map[int]int64, error)` —
  reads end-of-log offsets per (topic, partition).
- `topics.CompactConfigNames() []string` — returns the names of the configs
  applied to every compacted topic, in declaration order.
- `groups.Seed(ctx, client, addr, groupIDs []string, partitions map[string][]int, offsets map[string]map[int]int64, RetryConfig) (SeedResult, error)` —
  seeds end-of-log offsets for every seedable group.
- `groups.Verify(ctx, client, addr, groupIDs []string, partitions map[string][]int, SeedResult, RetryConfig) ([]VerifyReport, error)` —
  checks that every group's committed offsets cover the requested
  (topic, partition) union.
- `kafkaops.WithCoordinatorRetry(ctx, RetryConfig, fn func() error) error` —
  bounded retry for group-coordinator RPCs (`DescribeGroups`,
  `OffsetCommit`, `OffsetFetch`).
- `kafkaops.WithLeaderRetry(ctx, RetryConfig, fn func() error) error` —
  bounded retry for the partition-leader case on `ListOffsets`.
