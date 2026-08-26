# Review: Task 3 — the pre-join topic-readiness gate

Range reviewed: `c43189110..308c36e67` (single commit `308c36e67
"feat(atlas-kafka): gate consumer-group join on topic readiness"`).

Inputs: `.superpowers/sdd/plan/task-3-brief.md`,
`.superpowers/sdd/plan/task-3-report.md`,
`.superpowers/sdd/plan/review-c43189110..308c36e67.diff`.

Files touched: `libs/atlas-kafka/consumer/engine_group.go` (+74),
`libs/atlas-kafka/consumer/engine_group_test.go` (+186),
`libs/atlas-kafka/consumer/fakegroup_test.go` (+62). Matches the brief's
`### Files` list exactly; no unrelated file touched.

## Scope note (not a Task-3 defect)

The live worktree at review time carries **uncommitted** changes to
`libs/atlas-kafka/consumer/engine_group.go` (`errTopicMissing`,
`classifyEmptyAssignment`, an `errors.Is(err, errTopicMissing)` branch) —
evidently Task 4 work-in-progress. `go build ./...` against the dirty
worktree fails (`undefined: errTopicMissing`). This is **not** part of the
`c43189110..308c36e67` range and not a Task-3 defect: I verified the actual
commit `308c36e67` in an isolated `git worktree add -f` checkout, where it
builds, vets, and tests clean (see Verification below). Flagging only so the
controller doesn't mistake the dirty tree's build failure for a Task-3
regression.

## Requirement-by-requirement (brief + binding constraints)

1. **`wg.Add(1)` stays at the top, before the gate; gate `select`s on
   `ctx.Done()` on every wait.** Confirmed at `engine_group.go:17-19`
   (`wg.Add(1)` / `defer wg.Done()` precede the loop) and the gate call at
   `engine_group.go:31-33` sits after the top-of-loop `ctx.Err()` check and
   before `c.gp(...)`. Inside `awaitTopic` (`engine_group.go:73-121`) every
   blocking operation is bounded:
   - loop-top `ctx.Err() != nil` re-check (line 83-85) — catches cancellation
     that raced in before a lookup starts.
   - the lookup itself uses `lctx, cancel := context.WithTimeout(ctx,
     topicMetadataTimeout)` (line 87) — `lctx` is derived from the *parent*
     `ctx`, so a cancel during a blocking `c.pcp` call propagates through
     `kafka.Client.Metadata(ctx, ...)` (verified `defaultPartitionCountProducer`,
     `group.go:96-101`, passes the same `ctx` straight into the client call).
   - the backoff wait is `select { case <-ctx.Done(): return false; case
     <-time.After(wait): ... }` (lines 113-119) — reuses the shape of the
     existing loop's backoff/select block as required.
   PASS. `TestGateExitsOnContextCancel` (`engine_group_test.go:243-283`)
   pins this by cancelling mid-wait and asserting `wg.Wait()` returns within
   3s and `GroupProducer` was never called — I additionally rebuilt the
   commit with the gate call site stripped out (isolated scratch checkout,
   not the reviewed worktree) and confirmed this test, plus
   `TestGateDoesNotJoinUntilTopicExists` and
   `TestGroupEngineRecoversWhenTopicAppears`, genuinely FAIL without the
   fix — see Test honesty below.

2. **`false` return means cancellation; caller honours it.** `awaitTopic`
   returns `false` only from the two `ctx`-cancellation paths (line 84,
   line 116); every other path returns `true`. The call site
   (`engine_group.go:31-33`) does `if !c.awaitTopic(l, ctx) { return }` —
   returns rather than falling through to `c.gp(...)`. PASS.

3. **Nil `pcp` disables the gate entirely.** `awaitTopic` line 74-76: `if
   c.pcp == nil { return true }`, before any lookup or backoff state is
   touched. `newGroupConsumer` (test helper, pre-existing) builds via
   `newTestConsumer()` which never sets `pcp`, so it stays nil — matches
   the constraint that existing struct-literal `Consumer`s in
   `engine_group_test.go`, `idle_stuck_test.go`, `dwell_integration_test.go`,
   `state_test.go` are unaffected. Diff touches no existing test in those
   files (only appends). `TestNilPartitionCountProducerSkipsGate`
   (`engine_group_test.go:287-305`) pins this directly. PASS.

4. **Only a definite negative keeps the gate closed; every other error is
   indeterminate and opens the gate.** The `switch` at
   `engine_group.go:91-100`:
   - `err != nil && !errors.Is(err, ErrTopicNotFound)` → Debug log, `return
     true` (join). Covers transport/timeout/any other error code.
   - `err == nil && count >= 1` → `return true` (join).
   - Anything else (`ErrTopicNotFound` with any count, or `err == nil &&
     count == 0`) falls through to the warn/backoff tail — i.e. "wait".
   This matches FR-2.5 exactly: an error is never treated as a zero count
   (the indeterminate branch returns immediately without touching
   `recordTopicMissing` or the warn path). `TestGateJoinsImmediatelyOnIndeterminateLookup`
   (line 309) scripts `context.DeadlineExceeded` and asserts immediate join,
   no warn log, `TopicMissingObservations == 0`. PASS.

5. **Log discipline.** Per-poll Debug at line 108
   (`"Topic [%s] is still absent; re-polling..."`), throttled Warn at line
   104 gated by `!waiting || now.Sub(lastWarn) >= topicMissingWarnInterval`
   (line 103) — `topicMissingWarnInterval` is Task 2's `1 * time.Minute`
   (`manager.go:325`), unmodified here. `recordTopicMissing()` (line 106)
   is called only on the Warn branch, so it counts *distinct* observations
   (one per warn-interval), not every poll — matches `recordTopicMissing`'s
   own doc comment (`manager.go:544-546`, Task 2, untouched).
   `TestGroupEngineRecoversWhenTopicAppears` asserts
   `TopicMissingObservations != 1` for a script that misses once then
   recovers — exactly one distinct observation. PASS.

6. **No readiness-probe or health-endpoint change; gate runs inside the
   per-consumer goroutine and never blocks HTTP server start.** The diff
   touches only `engine_group.go` (plus tests); nothing in `manager.go`'s
   `AddConsumer`/`routine.Go` call site changed (confirmed via
   `git diff --stat`: only the three listed files). `awaitTopic` runs
   entirely inside `startGroupEngine`, itself launched via `routine.Go` in
   `AddConsumer` (pre-existing, unmodified). PASS.

7. **`groupConfig()`'s `WatchPartitionChanges` stays `false`.** Confirmed
   unmodified at `group.go:189` (`WatchPartitionChanges: false`), and
   `group_test.go:39-44` still pins it. PASS.

8. **`runGenerations`' empty-assignment branch untouched.** Diff hunk for
   `engine_group.go` only *inserts* `awaitTopic` between `startGroupEngine`
   and `runGenerations`; no lines inside `runGenerations` are touched
   (confirmed from the diff — the second hunk's context lines for
   `runGenerations` are unchanged `@@` context, no `+`/`-` inside the
   function body). PASS.

9. **Produced interface exactly as specified.** `func (c *Consumer)
   awaitTopic(l logrus.FieldLogger, ctx context.Context) bool`
   (`engine_group.go:73`) — matches verbatim. Test helpers in
   `fakegroup_test.go`: `type counterResult struct { count int; err error
   }` (350-354), `func newFakePartitionCounter(results ...counterResult)
   *fakePartitionCounter` (366), `.produce` (375), `.set` (388),
   `.callCount` (395), `func logEntryContaining(hook *test.Hook, sub
   string) *logrus.Entry` (403) — all present with the exact signatures
   Task 4's brief will need. PASS.

10. **Reuses the shape of the existing backoff/select block.** `awaitTopic`
    builds its own `backoff := newFetchBackoff()` (line 78) and its wait
    tail (`wait := backoff.next(); select { <-ctx.Done() / <-time.After
    (wait): c.recordBackoff(wait) }`, lines 112-119) mirrors
    `startGroupEngine`'s own block at lines 46-52 structurally. This is a
    *separate* `fetchBackoff` instance from the outer loop's — intentional
    per the brief's literal code (not implementer deviation); noted as a
    design property, not a defect: a gate re-entry after a failed
    `c.gp`/`runGenerations` cycle restarts its own backoff at 500ms rather
    than continuing the outer loop's escalation. Non-blocking.

## Concurrency review (the stated highest-risk area)

Traced every blocking call in `awaitTopic` and its caller by hand:

- `ctx.Err()` checks (non-blocking, ×2: line 83, and the outer loop's own
  check before the gate call).
- `c.pcp(lctx, c.brokers, c.topic)` — bounded by `context.WithTimeout(ctx,
  topicMetadataTimeout)` (5s, `group.go:69`), and `lctx` derives from the
  real `ctx`, so parent cancellation cuts this short too, not just the
  timeout.
- `select { <-ctx.Done() / <-time.After(wait) }` — the only place a "wait"
  actually elapses; both arms guarded, `ctx.Done()` always wins a race
  against `time.After`. `recordBackoff` executes only on the `time.After`
  arm, correctly not on cancellation.

No unguarded `time.Sleep`, no unguarded channel receive, no lock held across
a blocking call — `awaitTopic` takes no lock at all; `recordTopicMissing`
and `recordBackoff` (Task 1/2 code, unmodified) take and release `c.mu`
internally, not held across the poll or the wait.

`TestGateExitsOnContextCancel` closes the loop on this empirically: cancel
mid-wait, assert `wg.Wait()` returns within 3s. Passed at the reviewed
commit (see Verification).

## Test honesty

Ran the module test suite is out of scope per instructions (implementer's
`-race` evidence is trusted), but I independently re-derived it in an
isolated `git worktree add -f 308c36e67` checkout (not the reviewed
worktree — no files in the review surface were modified):

- `go build ./...`, `go vet ./...` — clean at `308c36e67`.
- `go test -race ./consumer/...` — `ok  ...consumer  9.192s`.
- `go test ./consumer/ -run 'TestGate|TestGroupEngineRecovers|TestNilPartitionCountProducer' -v`
  — all 5 new tests PASS, plus pre-existing `TestGate*` tests in the same
  package (unrelated env-gate tests) unaffected.
- **RED check** (the implementer's report admits it skipped a literal
  red/green cycle): in a second throwaway checkout of `308c36e67` I removed
  only the four-line gate call site (`if !c.awaitTopic(l, ctx) { return
  }`) — not committed, not part of the reviewed tree — and reran the same
  five tests. Result: `TestGateDoesNotJoinUntilTopicExists`,
  `TestGroupEngineRecoversWhenTopicAppears`, and `TestGateExitsOnContextCancel`
  genuinely FAIL (the first two timeout/assert-fail, the cancel test times
  out at 3s exactly as its own failure message predicts).
  `TestNilPartitionCountProducerSkipsGate` and
  `TestGateJoinsImmediatelyOnIndeterminateLookup` still pass without the
  call site, which is expected and correct — both scripts describe
  "join immediately," which is also the *pre-fix* behaviour, so they pin a
  non-regression rather than the fix itself. The three tests that matter
  for the fix are not vacuous. PASS (test honesty).

## Non-blocking notes

- The doc-comment on `awaitTopic` cites specific kafka-go line ranges
  (`consumergroup.go:1010-1049`, `consumergroup.go:512-518`). These are
  reproduced verbatim from the brief (confirmed against
  `task-3-brief.md`), not the implementer's own claim, and Task 1/2's
  grounding for the same kafka-go behaviour was presumably reviewed
  earlier in this plan. Out of this task's surface to re-verify against
  the actual `kafka-go` vendor source; noted only so a reader knows the
  citation's provenance is "copied from the brief," not independently
  re-checked here.
- `awaitTopic`'s inner `fetchBackoff` is independent of the outer loop's
  (see item 10 above) — intentional, not a defect, flagged only for
  visibility.

## Not evaluable

None. The full review surface (the single commit's diff, plus the Task
1/2 seams it consumes — `ErrTopicNotFound`, `PartitionCountProducer`,
`topicMetadataTimeout`, `recordTopicMissing`, `topicMissingWarnInterval`,
`defaultPartitionCountProducer`/`partitionCountFromMetadata`) was directly
inspectable and was inspected.

## Verdicts

- **Spec compliance: APPROVED.** Every FR/constraint bound to Task 3 is
  satisfied and traced to `file:line`; the produced interface matches the
  brief exactly; nothing outside Task 3's `### Files` list was touched;
  Task 4/5's owned code (`runGenerations`' empty-assignment branch,
  `WatchPartitionChanges`) is untouched.
- **Task quality: APPROVED.** Concurrency trace is clean (every blocking
  op selects on `ctx.Done()` or is bounded by a context derived from the
  real `ctx`), the new tests are not vacuous (3 of 5 independently
  reproduced as failing pre-fix), log/counter discipline matches Task 2's
  contract, gofmt clean, `go vet` clean, `-race` clean at the reviewed
  commit.

No blocking findings.
