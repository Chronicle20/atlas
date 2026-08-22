# Spike Findings — kafka-go admin APIs

Date: 2026-08-21
Library: `github.com/segmentio/kafka-go v0.4.51` (already required by 65 modules in this repo)
Environment: single-node KRaft broker, `apache/kafka:3.7.2`, throwaway container

These were run before the PRD was written, to de-risk the parts of the port that
are not mechanical. The spike program itself is throwaway and was not kept.

---

## Q1 — Does one `CreateTopics` request create N topics, with per-topic errors?

**CONFIRMED.**

First run created all three topics (`err=<nil>` for each). An identical second
request returned, per topic:

```
run2 spike_topic_a       err=[36] Topic Already Exists: ... errors.Is(TopicAlreadyExists)=true
run2 spike_topic_b-main  err=[36] Topic Already Exists: ... errors.Is(TopicAlreadyExists)=true
run2 spike_topic_c       err=[36] Topic Already Exists: ... errors.Is(TopicAlreadyExists)=true
```

`CreateTopicsResponse.Errors` is a `map[string]error` keyed by topic name, and the
errors carry Kafka error codes testable with `errors.Is`.

**Consequence for the design:** the list-then-diff pass is unnecessary. The shell
version needs `--list` + `sort` + `comm` (and, after a bug found during PR #1463,
an `LC_ALL=C` pin, because topic names mix `_` and `-` — exactly where locale
collation and byte order disagree). None of that machinery is needed in Go: send
everything, tolerate `TopicAlreadyExists` per topic. An entire class of bug
disappears rather than being defended against.

## Q2 — Does `OffsetCommit` with `GenerationID: -1` / `MemberID: ""` work on an Empty group?

**CONFIRMED.**

Against a group with five records in the log and no members:

```
== Q2: commit as non-member to EMPTY group (end offset 5) ==
  empty-group commit spike_topic_a  p0 err=<nil>
  => Q2 fetched back spike_topic_a  p0 committed=5 err=<nil>
```

The group name used was `"Spike Service - probe2 [pr-999]"` — spaces and brackets,
matching real Atlas group names.

**Consequence for the design:** this is the replacement for
`kafka-consumer-groups.sh --reset-offsets --to-latest --execute`. It also means
the shell script's `awk` column-anchoring (`$(NF-1)`, `$(NF-7)`, `$(NF-5)`),
which exists solely because group names contain spaces and shift fixed columns
in CLI output, has no analogue in Go — responses are structured.

## Q3 — Is that commit refused on an ACTIVE group, with a typed error?

**CONFIRMED.**

After joining a live `kafka.Reader` to the same group and waiting for state
`Stable`:

```
=> Q3 active-group commit spike_topic_a p0 err=[25] Unknown Member ID: the member id is not in the current generation (typed=kafka.Error)
```

**Consequence for the design:** the script currently classifies this case by
string-matching the CLI's `"Assignments can only be reset if the group"` message
— a hack its own comments apologize for. In Go it is a typed `kafka.Error`.

**Nuance worth carrying into design:** the code is `UnknownMemberID` (25), not a
literal "group is active" code. Because the tool always sends `MemberID: ""`,
this error on a non-empty group unambiguously means active — but the intent is
only explicit because of the `DescribeGroups` state pre-check. That pre-check
should be **kept**, not replaced by error handling alone.

`DescribeGroups` returns `GroupState` as a plain string field, so the state
allowlist (`Empty`/`Dead`/unknown ⇒ seedable) ports directly.

## Q4 — (unplanned) Coordinator availability at wave 0

**NEW FINDING — not anticipated in the design.**

The very first group request against the freshly-created cluster returned:

```
empty-group commit spike_topic_a p0 err=[16] Not Coordinator For Group
```

`__consumer_offsets` and the group coordinator had not settled yet. Adding a
bounded retry on `NotCoordinatorForGroup` / `GroupCoordinatorNotAvailable` made
the run deterministic.

Routing itself is **not** the problem: `protocol/offsetcommit` and
`protocol/offsetfetch` both implement `protocol.GroupMessage`, and
`connPool.sendRequest` in `transport.go` automatically resolves the coordinator
via `FindCoordinator` for those. The transient is broker-side readiness, not
client-side addressing.

**Consequence for the design:** this is a real production requirement, not a test
artifact. The Job runs at sync-wave 0 against an environment that may be brand
new. The shell version never had to handle it because the JVM CLI retries
internally; a direct protocol client must do so explicitly. See PRD FR-5.1.

---

## Not covered by the spike

Carried into the PRD as open questions (§9):

- `AlterConfigs` batching and whether topic-config alteration is incremental or
  full-replace (a full-replace would clobber unrelated topic config).
- Whether to assume partition 0 for `ListOffsets`/`OffsetCommit` or enumerate
  partitions from metadata.
- The specific retry backoff and ceiling for Q4's requirement.
