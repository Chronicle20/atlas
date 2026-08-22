# Review — task-8 fix (cold-start retriable Kafka errors)

Commit reviewed: `fbff7c891` (range `df0f420d1..fbff7c891`, single commit).
Requirements: `.superpowers/sdd/plan/task-8-fix-brief.md`,
`docs/tasks/task-260-kafka-precreate-go-tool/bug-cold-start-retriable-errors.md`,
`.superpowers/sdd/plan/task-8-fix-report.md`.

## Scope

`git diff --stat df0f420d1..fbff7c891` touches exactly the 8 files the brief
named: `kafkaops/ops.go`, `kafkaops/ops_test.go`, `topics/topics.go`,
`topics/topics_test.go`, `groups/groups.go`, `groups/groups_test.go`,
`main.go`, `README.md`. No scope creep. The implementer report's filename
correction (`kafkaops.go` → the real `ops.go`) is accurate and immaterial.

## 1. Retriable set — exactly codes 5, 6, 15, 16?

`internal/kafkaops/ops.go`:

```go
func isCoordinatorError(err error) bool {
	return errors.Is(err, kafka.NotCoordinatorForGroup) || errors.Is(err, kafka.GroupCoordinatorNotAvailable)
}

func isLeaderError(err error) bool {
	return errors.Is(err, kafka.NotLeaderForPartition) || errors.Is(err, kafka.LeaderNotAvailable)
}
```

`isCoordinatorError` is untouched by this commit (pre-existing). `isLeaderError`
is new and checks exactly codes 6 and 5. `firstLeaderError`
(`internal/topics/topics.go:293-311`) gates on the same two `errors.Is` checks
before surfacing an error from the retry `fn`; every other `po.Error` is left
untouched. `firstCoordinatorError` (`groups.go:146-161`) and
`firstFetchCoordinatorError` (`groups.go:305-320`) gate on exactly
`NotCoordinatorForGroup`/`GroupCoordinatorNotAvailable` (16/15).
**PASS** — no widening beyond 5/6/15/16, confirmed by reading every gate, not
just the report's claim.

## 2. `kafka.UnknownMemberId` (25) — still "commit race → skip" in `Seed`, provably not retriable?

`firstCoordinatorError` (`groups.go:146-161`) checks only
`NotCoordinatorForGroup`/`GroupCoordinatorNotAvailable`; `UnknownMemberId` is
excluded by construction — it falls through, `firstCoordinatorError` returns
`nil` for a response whose only error is code 25, `WithCoordinatorRetry`
sees no retriable error and returns immediately, and the pre-existing
post-loop classifier at `groups.go:104-125` runs exactly as before: code 25
sets `commitRace = true` → group appended to `Skipped`.

Mutation test performed (not merely inspected): widened
`firstCoordinatorError` to also match `kafka.UnknownMemberId` and reran
`go test ./internal/groups/...`. Result: two tests fail —
`TestSeed/commit_race_is_a_non-fatal_skip` (pre-existing) and
`TestSeed_CommitPartitionUnknownMemberIdNotRetried` (new). Reverted the
mutation; suite is green again. **PASS**, and the suite provably catches a
regression on this exact axis.

## 3. Are the retried RPCs genuinely idempotent as re-issued?

- `ListOffsets` (`topics.go:248-256`): re-issues the same `req` built once
  before the retry loop (`LastOffsetOf` per partition) — a pure read, safe
  to repeat.
- `OffsetCommit` (`groups.go:77-89`, unchanged by this commit): built fresh
  each retry with `GenerationID: -1, MemberID: ""` — a fixed non-member
  identity, not incrementing state, so re-committing the same offsets is a
  no-op past the first success. Confirmed these two fields are literal
  constants at the call site, not derived from prior state.
- `OffsetFetch` (`groups.go:229-238`, unchanged by this commit): a pure read.

All three match the brief's idempotence claim. **PASS**.

## 4. Is `WithCoordinatorRetry`'s exported behaviour unchanged, and is its table test unmodified?

Read the diff directly (not the report):
`git diff df0f420d1..fbff7c891 -- .../kafkaops/ops.go` shows
`WithCoordinatorRetry` reduced to a one-line call into the new unexported
`withRetry(ctx, cfg, retriable, fn)`, with `isCoordinatorError` passed as the
predicate — the loop body (backoff computation, `now().Sub(start)+backoff >
budget` check, `sleep(backoff)`) is moved verbatim into `withRetry`, not
altered. The one behavioural change is the budget-exhaustion error text:
`"coordinator retry budget of %s exhausted"` → `"retry budget of %s
exhausted"`. The implementer report flags this; confirmed both the existing
`TestWithCoordinatorRetry_GivesUpAtBudget` and the new
`TestWithLeaderRetry_GivesUpAtBudget` assert via `errors.Is`/substring on
topic+partition, not the literal budget-exhaustion prefix — `go test
./internal/kafkaops/...` passes, and grepping the test file for the old
string returns nothing. Not a hidden regression.

`git diff --stat` on `ops_test.go` shows `1 file changed, 141 insertions(+)`
— **zero deletions**, i.e. a pure append; `TestWithCoordinatorRetry` and its
two satellite tests are byte-for-byte unmodified, exactly as the brief
required. **PASS**.

## 5. Are the new tests non-vacuous?

Every new/updated case asserts a call count, not just the final result:

- `kafkaops`: `TestWithLeaderRetry` table asserts `calls` per case (1 for
  no-retry paths, 2 for the two retried codes) plus `expectBackoff`.
- `topics`: `partition leader error retries then succeeds` asserts
  `len(stub.listCalls) != 2` and `len(clock.slept) != 1`; `non-retriable
  partition error stays fatal on first call` asserts `len(stub.listCalls) !=
  1`; the budget-exhaustion case asserts the error names both topic and
  partition and `errors.Is(err, kafka.LeaderNotAvailable)`.
- `groups`: `TestSeed_CommitPartitionErrorIsRetried` asserts
  `len(stub.commitCalls) != 2`; `TestSeed_CommitPartitionUnknownMemberIdNotRetried`
  asserts `len(stub.commitCalls) != 1`; `TestVerify_FetchPartitionErrorIsRetried`
  asserts `len(stub.fetchCalls) != 2`.

Mutation-tested two load-bearing branches directly (not merely read):

1. **`groups.firstCoordinatorError`** widened to also match
   `kafka.UnknownMemberId` → `go test ./internal/groups/...` fails 2 tests
   (`TestSeed/commit_race_is_a_non-fatal_skip`,
   `TestSeed_CommitPartitionUnknownMemberIdNotRetried`). Caught.
2. **`topics.EndOffsets`**'s retry `fn` short-circuited to `return nil`
   instead of `return firstLeaderError(...)` → `go test
   ./internal/topics/...` fails `TestEndOffsets/partition_leader_error_retries_then_succeeds`
   with the raw `NotLeaderForPartition` error escaping unretried. Caught.

Both mutations reverted; `go build ./... && go test ./...` clean afterward,
working tree confirmed clean (`git status --short` shows only the two
pre-existing untracked triage docs, not part of this commit). **PASS** — the
suite is provably non-vacuous on both axes asked about.

## 6. `Verify`'s stale test → `NotCoordinatorForGroup`; `NotLeaderForPartition`-on-`OffsetFetch` still fatal — does code match the claim?

`groups.go:229-238`: the retry `fn` for `OffsetFetch` calls
`firstFetchCoordinatorError(resp)`, which gates only on codes 15/16
(`groups.go:305-320`). A `NotLeaderForPartition` (6) per-partition error is
therefore invisible to the retry loop and returns from `WithCoordinatorRetry`
unretried. Post-loop, `groups.go:265-268` treats *any* non-nil `part.Error`
(no code filter) as fatal: `return nil, fmt.Errorf("fetching committed offset
for group %q topic %q partition %d: %w", ...)`. The updated test
`"non-coordinator partition-level fetch error stays fatal"`
(`groups_test.go:555-568`) exercises exactly this: a `NotLeaderForPartition`
error on `OffsetFetch`, asserting `wantErr: "a"` (fatal, not retried). This
matches the controller's stated judgment exactly — leader codes are not
retried on `OffsetFetch`, and the code path proves it structurally (the gate
function simply doesn't recognize code 6), not just by the one test case.
**PASS**.

The new `TestVerify_FetchPartitionErrorIsRetried` correctly uses
`kafka.NotCoordinatorForGroup` (16) instead of literally reusing the stale
plan's `NotLeaderForPartition`, which is the only choice consistent with
"do not widen beyond the four named codes" — reusing the old code would have
required `OffsetFetch` to retry a leader code, which requirement 1 forbids.
The implementer's judgment call is correct and the diff proves it, not just
the report's narration.

## Other observations (non-blocking)

- `firstLeaderError` and `firstFetchCoordinatorError`/`firstCoordinatorError`
  rebuild a `map[int]...` from `resp.Topics[topic]` per topic on every retry
  attempt — O(partitions) per call, negligible at this tool's scale (single
  Job run, small topic/group counts). Not a concern.
- README changes (`README.md` diff) accurately describe the new retry
  behaviour in phases 3-5 and match the code; optional per the brief, done
  well.
- `main.go`'s `EndOffsets` call site correctly passes
  `kafkaops.DefaultRetryConfig()`, matching the new parameter.

## Not evaluable

None. The retry logic, its two new predicates, both call sites, and the
full new/updated test coverage were all read and, where the review's own
judgment questions demanded it, mutation-tested directly rather than taken
on the report's word. `main.go`'s broader flow (topic creation, settle,
group discovery) was not re-reviewed — outside this commit's diff and not
touched by it.

## Verdict

APPROVED. All six focus questions confirmed against the diff and, for the
two most safety-critical claims (`UnknownMemberId` non-retriability, and the
leader-retry surfacing actually driving the loop), confirmed by mutation
testing rather than inspection alone. `WithCoordinatorRetry`'s table test is
provably byte-for-byte unmodified. No widening beyond codes 5/6/15/16
anywhere in the diff.
