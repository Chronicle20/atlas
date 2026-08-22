# Bug: cold-start runs fail on element-level retriable Kafka errors

Found by Task 8 Step 3 (manual acceptance against a throwaway
`apache/kafka:3.7.2` single-node KRaft broker, `localhost:19092`). Not caught
by the unit suites, which model these codes as *fatal* by design — see
`plan.md:438` and `plan.md:593`, where `kafka.LeaderNotAvailable` and
`kafka.NotLeaderForPartition` partition-level errors are asserted to produce a
hard error.

## Observed

Three consecutive runs against a fresh broker, identical environment
(6 topics, 2 override groups, one named `Atlas Channel [0]`):

Run 1 — exit 1, after `create`/`alter` both succeeded:

```
{"error":"end offset for topic \"event-character-status\" partition 0: [6] Not Leader For Partition: ...
end offset for topic \"event-config-environment-status\" partition 0: [6] Not Leader For Partition: ...
end offset for topic \"event-config-tenant-status\" partition 0: [6] Not Leader For Partition: ...
end offset for topic \"command-award-experience\" partition 0: [6] Not Leader For Partition: ...",
"level":"error","msg":"kafka precreate failed"}
```

Run 2 — exit 1, further along (end offsets now readable, seed reached):

```
{"error":"committing seed offset for group \"Atlas Channel [0]\" topic \"command-change-map\" partition 0: [16] Not Coordinator For Group: ...",
"level":"error","msg":"kafka precreate failed"}
```

Run 3 — exit 0, sub-second, fully correct (see "What is already right" below).

## Diagnosis

Both failures are **transient cluster-warmup states surfaced as
element-level errors inside an otherwise-successful response**, and both
bypass the retry the tool already has.

1. **`ListOffsets` / `NotLeaderForPartition` (6), `LeaderNotAvailable` (5).**
   `topics.Settle` polls `Metadata` until every topic is *visible with at
   least one partition* — but partition visibility is not leader election.
   Immediately after `CreateTopics`, a partition can appear in metadata with
   no elected leader, and `ListOffsets` for it returns code 6/5 per partition.
   `topics.EndOffsets` (`topics.go:239-271`) treats every non-nil
   `po.Error` as fatal and joins them, so the Job dies. There is **no retry
   at all** on this path — `WithCoordinatorRetry` deliberately does not wrap
   `ListOffsets` (documented at `kafkaops.go:57-63`), and the design's reason
   for that exclusion ("cannot produce these two codes") is correct for the
   *coordinator* codes but left the *leader* codes unhandled.

2. **`OffsetCommit` / `OffsetFetch` / `NotCoordinatorForGroup` (16),
   `GroupCoordinatorNotAvailable` (15).** `WithCoordinatorRetry` *is* applied
   to both calls, but its `fn` returns only the **transport-level** error
   (`groups.go:82-84`, `groups.go:203-211`). On a broker that is still
   electing the `__consumer_offsets` coordinator, the RPC succeeds at
   transport level and code 16 arrives as a **per-partition**
   `part.Error` — which the retry predicate never sees. `groups.Seed`
   (`groups.go:114-115`) then classifies it as `fatal`, and `groups.Verify`
   (`groups.go:237-238`) does the same. The retry machinery is present,
   tested, and simply not reached by the error it was built for.

Why the shell script this replaces never hit either: it drove the JVM
`kafka-topics.sh` / `kafka-consumer-groups.sh` CLIs, whose `AdminClient`
retries retriable error codes internally within `default.api.timeout.ms`.
The Go rewrite talks the admin protocol directly and inherited none of that.

## Why this matters

This is precisely the tool's reason to exist. It is a **sync-wave-0 Job
against a Kafka cluster that has just come up**, which is the exact window in
which leader election and coordinator election are still in flight. The
Kubernetes Job `backoffLimit` would eventually paper over it, but only after
burning failed pod runs on a cluster bring-up, and PRD NFR-1's steady-state
budget says nothing about a cold path that needs three attempts.

## What is already right (do not change)

Run 3 confirms the whole steady-state contract, verified against the broker
with the JVM CLI rather than from the tool's own logs:

- exit 0, wall clock well under 1s (NFR-1).
- 6 topics created; `cleanup.policy=compact` set on **exactly** the three
  config-projection topics (`event-config-tenant-status`,
  `event-config-service-status`, `event-config-environment-status`) and on no
  other Atlas topic. The fourth compacted topic in `kafka-configs.sh` output
  is `__consumer_offsets`, which Kafka compacts by default.
- Both override groups carry a committed offset on all 6 topics
  (`kafka-consumer-groups.sh --describe --all-groups`), including the group
  named `Atlas Channel [0]` — the spaces-and-brackets case the shell needed
  its `$(NF-1)` idiom for.
- Re-running against already-created topics reports `existing:6, created:0`
  and re-seeds idempotently.

## Fix

Retry the element-level retriable codes; leave every other code fatal.

- Generalize the retry loop in `kafkaops` to take a retriability predicate.
  Keep `WithCoordinatorRetry`'s exported signature and behaviour unchanged so
  its existing table test stays valid; add a sibling for the leader codes.
- Make the `fn` passed to the retry loop return a retriable **element-level**
  error so the loop drives it, rather than letting it escape as `fatal`. The
  whole RPC is re-issued, which is correct: all three calls are idempotent.
- Bound it with the existing `RetryConfig` budget. Do **not** widen the
  retriable set beyond codes 5, 6, 15, 16 — a diagnosable failure must not
  become a timeout (design §4 OQ-3).
- `kafka.UnknownMemberId` (25) keeps its current meaning in `Seed` (the
  commit race → skip). It is not retriable and must not be folded in.
