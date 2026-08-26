# Review: Task 1 — the partition-count seam (libs/atlas-kafka)

Range reviewed: `ac4e576c4..708e396ef` (single commit `708e396ef`).
Files touched: `libs/atlas-kafka/consumer/group.go`, `consumer/manager.go`,
`consumer/group_test.go`. Matches the brief's `### Files` list exactly —
confirmed via `git show --stat 708e396ef`.

## Spec compliance — PASS

Checked every item in the "Produces" interface list against the diff, symbol
for symbol:

- `var ErrTopicNotFound error` — `group.go`, `errors.New("topic not found or has no partitions")`. `errors.New` returns `error`, matches the declared type exactly.
- `const topicMetadataTimeout = 5 * time.Second` — `group.go`, present verbatim.
- `type PartitionCountProducer func(ctx context.Context, brokers []string, topic string) (int, error)` — `group.go`, exact signature.
- `func ConfigPartitionCountProducer(pcp PartitionCountProducer) ManagerConfig` — `group.go`, follows the `ConfigGroupProducer`/`ConfigPartitionReaderProducer` shape at `group.go:33-54` exactly (`//goland:noinspection` comment included, same closure-over-`m.pcp` pattern).
- `func partitionCountFromMetadata(topic string, res *kafka.MetadataResponse) (int, error)` — `group.go`, pure function, no I/O.
- `func defaultPartitionCountProducer(ctx context.Context, brokers []string, topic string) (int, error)` — `group.go`.
- `Manager.pcp` / `Consumer.pcp`, both `PartitionCountProducer`, wired `Manager` → `Consumer` in `AddConsumer` — `manager.go:88` (`Manager.pcp` field), `manager.go:113` (`pcp: defaultPartitionCountProducer` in `GetManager`), `manager.go:194` (`pcp: m.pcp` in the `AddConsumer` `&Consumer{...}` literal), `manager.go:261` (`Consumer.pcp` field with the exact brief-specified comment).

## Binding constraints — all verified

1. **Only a definite negative produces `ErrTopicNotFound`.** Read
   `partitionCountFromMetadata` by hand against the FR-2.5 table:
   - `res == nil` → `ErrTopicNotFound` (not in the brief's table but a
     reasonable defensive case, doesn't violate the rule — nil metadata is
     itself the strongest form of "topic not found").
   - Topic entry present with `t.Error != nil` and
     `errors.Is(t.Error, kafka.UnknownTopicOrPartition)` → `ErrTopicNotFound`.
   - Topic entry present with any *other* `t.Error` → `return 0, t.Error`,
     unchanged, indeterminate. Verified `kafka.Error` is a bare `int` type
     (`error.go:12`, `kafka-go@v0.4.51`) with no custom `Unwrap`/`Is` method,
     so `errors.Is(t.Error, kafka.UnknownTopicOrPartition)` correctly reduces
     to `==` comparison — no false positives/negatives possible here.
   - Topic entry present, no error, `len(t.Partitions) == 0` →
     `ErrTopicNotFound`.
   - Topic entry absent from `res.Topics` entirely (loop falls through) →
     `ErrTopicNotFound`.
   - No transport/timeout path can reach `partitionCountFromMetadata` at all:
     `defaultPartitionCountProducer` returns `(0, err)` directly from
     `client.Metadata`'s own error before ever calling the mapping function —
     so a transport error is never even routed through the ErrTopicNotFound
     classification. Correct per FR-2.5.
   - `TestPartitionCountFromMetadata` (`group_test.go:136-160`) pins all seven
     cases from the brief's table, including the negative assertion (line
     ~158) that an indeterminate error is never collapsed into
     `ErrTopicNotFound`.

2. **Nil seam means inert.** `Consumer.pcp` is a new field, only ever set via
   the `AddConsumer` struct literal (`m.pcp` propagation) or a test's
   `ConfigPartitionCountProducer`. No struct-literal `Consumer` in
   `engine_group_test.go`, `idle_stuck_test.go`, `dwell_integration_test.go`,
   `state_test.go` was touched (confirmed: commit touches only the three
   named files) — so `pcp` is the zero value (`nil`) there, matching the
   contract. Only the two tests named in the brief
   (`TestManagerDefaultProducersArePresent`, `TestConfigProducersOverrideDefaults`)
   were edited, exactly as scoped.

3. **`kafka.Client.Metadata`, never `kafka.Conn.ReadPartitions`.**
   `defaultPartitionCountProducer` (`group.go:83-90`) uses
   `(&kafka.Client{...}).Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})`.
   `grep -rn "ReadPartitions" libs/atlas-kafka/consumer/` shows the only
   occurrence is pre-existing, unrelated code in `offsets.go:60` — not
   touched by this diff, not part of this seam.

4. **`groupConfig()`'s `WatchPartitionChanges` still `false`.** Confirmed at
   `group.go:189`, unchanged by this diff (`git diff --stat` shows zero
   changes to that function's line range).

5. **`engine_group.go` untouched.** `git diff --stat ac4e576c4..708e396ef -- .../engine_group.go` returns empty.

## Correctness / build

- `git show 708e396ef:libs/atlas-kafka/consumer/{group,manager,group_test}.go`
  extracted and run through `gofmt -l` independently of the working tree
  (which currently carries *later*, uncommitted task work) — all three
  gofmt-clean at the exact commit under review. No formatting regression from
  this task.
- Report's `go build ./... && go test ./consumer/` output (module root
  `libs/atlas-kafka`) is accepted per instructions not to re-run the suite;
  the diff read supports that result — no compile-breaking signature
  mismatches found.

## Test honesty

- `TestPartitionCountFromMetadata` exercises `partitionCountFromMetadata`
  directly with hand-built `*kafka.MetadataResponse` values; it cannot pass
  without the new function existing and behaving per the FR-2.5 boundary. Not
  vacuous.
- `TestManagerDefaultProducersArePresent`'s new assertion and
  `TestConfigProducersOverrideDefaults`'s new `pcpCalled` seam both require
  `Manager.pcp` and `ConfigPartitionCountProducer` to exist and to actually be
  invoked — the added assertions would fail to compile, then fail at
  runtime, without the implementation. Not vacuous.

## Non-blocking notes

- The working tree at review time carries additional *uncommitted* changes to
  `manager.go` (a `topicMissingObservations`/`lastTopicMissingAt` field pair,
  `topicMissingWarnInterval`, and `Snapshot` additions) that are visibly
  later-task work already staged in-progress on top of this commit. None of
  it is part of `708e396ef`; noted here only so a `gofmt -l` run against the
  live working tree is not mistaken for a defect in this task's commit (it
  isn't — the committed blobs are independently gofmt-clean, verified above).
  This is an observation about worktree state, not a finding against Task 1.

## Not evaluable

- None. The full review surface (the three touched files, the `kafka-go`
  `Client.Metadata`/`Error`/`Topic` contracts the new code depends on) was
  available and read.

## Verdict

APPROVED. Spec compliance: full match, symbol-for-symbol, against the
brief's binding interface list and all named constraints. Task quality: the
mapping function correctly implements the FR-2.5 definite-negative boundary,
the nil-seam invariant is preserved by construction (no off-limits file
touched), and the new/modified tests are non-vacuous.
