# Review: Task 4 — Zero-partition classification in `runGenerations`

Range reviewed: `308c36e67..dc17e62d9`
Files touched (confirmed via `git diff --stat` and a repo-wide diff excluding the two named files, which returned empty): `libs/atlas-kafka/consumer/engine_group.go`, `libs/atlas-kafka/consumer/engine_group_test.go`.

## Scope confirmation

The diff matches the brief exactly: production changes confined to `engine_group.go` (the `errTopicMissing` sentinel, `emptyAssignmentClass` type/constants, `classifyEmptyAssignment`, the three-way empty-assignment switch in `runGenerations`, and the sentinel arm in `startGroupEngine`), and test changes confined to an append to `engine_group_test.go` plus an added `"strings"` import. No other file in the module changed in this range.

## Requirement-by-requirement

1. **Produced interface matches spec exactly.**
   - `var errTopicMissing = errors.New(...)` — unexported sentinel. `engine_group.go:157`.
   - `type emptyAssignmentClass int` with `emptyHealthyIdle`, `emptyBecauseTopicMissing`, `emptyIndeterminate` — `engine_group.go:160-171`.
   - `func (c *Consumer) classifyEmptyAssignment(ctx context.Context) emptyAssignmentClass` — `engine_group.go:177-190`.
   PASS.

2. **Classification logic — only a definite negative is topic-missing.**
   `engine_group.go:180-190`:
   ```go
   switch {
   case errors.Is(err, ErrTopicNotFound):
       return emptyBecauseTopicMissing
   case err != nil:
       return emptyIndeterminate
   case count >= 1:
       return emptyHealthyIdle
   default:
       return emptyBecauseTopicMissing
   }
   ```
   `ErrTopicNotFound` is checked first and exclusively maps to topic-missing; every other error (`err != nil`) maps to indeterminate before any zero-count check is reached, so an error can never be read as a zero count. A *successful* zero-count lookup (`count == 0, err == nil`) correctly falls to the `default` arm as topic-missing (this is the "successful metadata exchange with zero partitions" case named in the plan's global constraints, not a violation of "only ErrTopicNotFound"). PASS.

3. **Nil `pcp` classifies as indeterminate, never topic-missing.**
   `engine_group.go:178-179`: `if c.pcp == nil { return emptyIndeterminate }`, checked first. Confirmed pre-existing `TestZeroAssignmentIsHealthyIdle` (nil `pcp`, unmodified) still passes — ran `go test ./consumer/ -run TestZeroAssignmentIsHealthyIdle -v` → PASS. PASS.

4. **Healthy-idle debug wording is byte-for-byte preserved (FR-2.4).**
   Diffed the pre-change and post-change lines directly:
   - Pre: `l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle.", c.topic, c.groupId, gen.ID())`
   - Post (moved unaltered into the `default` arm, `engine_group.go:233-234`): identical string, identical arguments.
   Confirmed via `git diff` hunk — the only change to that block is its indentation/relocation inside the new `switch`, not its content. PASS (per controller ruling, this preservation is contract, not duplication — not flagged as a defect).

5. **`errTopicMissing` sentinel arm in `startGroupEngine` matches the controller's binding ruling** (backoff applied, no `recordError`, no Error log).
   `engine_group.go:47-61`:
   ```go
   if errors.Is(err, errTopicMissing) {
       wait := backoff.next()
       select {
       case <-ctx.Done():
           l.Infof("Topic consumer stopped during backoff.")
           return
       case <-time.After(wait):
           c.recordBackoff(wait)
       }
       continue
   }
   c.recordError(err)
   l.WithError(err).Errorf("Consumer group for topic [%s] exited; rejoining after backoff.", c.topic)
   ```
   The sentinel arm is checked and returns via `continue` before ever reaching `c.recordError(err)` or the `Errorf` call. `c.recordBackoff(wait)` is invoked (attributing into `totalBackoffNs`), matching the ruling precisely. The `ctx.Done()` select is present, satisfying the "must select on ctx.Done() on every wait" global constraint. PASS.

6. **`grp.Close()` still runs on the topic-missing path.**
   `engine_group.go:37-43`: `runGenerations`'s return value (including `errTopicMissing`) is assigned to `err` inside the `if err == nil { ... }` block that unconditionally calls `grp.Close()` afterward, regardless of what `err` now holds. So the member's `Close()` — the actual "leave the group" call — happens before the sentinel arm's backoff. Correct: this is what makes "leaving the group" real rather than just semantic. PASS (not explicitly named as a check item in the brief, but load-bearing for the stated behaviour; verified because it directly determines whether the fix does what it claims).

7. **`groupConfig()`'s `WatchPartitionChanges` still false.**
   `grep -n WatchPartitionChanges libs/atlas-kafka/consumer/*.go` → `group.go:189: WatchPartitionChanges: false,` — unchanged, and not touched by this diff. PASS.

8. **`engine_reader.go` untouched.**
   `git diff 308c36e67..dc17e62d9 -- libs/atlas-kafka/consumer/engine_reader.go` → empty. PASS.

9. **Test helper reuse — no second fake introduced.**
   `newFakePartitionCounter`, `counterResult`, `silentLogger`, `hasLogContaining`, `logEntryContaining` are all defined in `fakegroup_test.go` (Task 3's helpers, untouched in this diff) and reused as-is by the four new tests. No new fake type was added. PASS.

10. **Only `engine_group_test.go` was appended to; no existing test files edited.**
    Confirmed by diff stat: only `engine_group_test.go` (test) and `engine_group.go` (production) changed. `idle_stuck_test.go`, `dwell_integration_test.go`, `state_test.go`, `group_test.go`, `fakegroup_test.go` are untouched. PASS.

## The implementer's one deviation: `l.SetLevel(logrus.DebugLevel)`

Judged **correct and necessary**, not a scope violation.

- `test.NewNullLogger()` (aliased `silentLogger()`, `fakegroup_test.go:218`) constructs a `logrus.New()` logger, which defaults to `InfoLevel`. Without raising the level, `Debugf` calls are dropped before ever reaching the hook, so `hasLogContaining(hook, "holds no partition assignment in generation 4; healthy-idle.")` would never see the line — the `waitFor` in both `TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist` and `TestEmptyAssignmentIndeterminateLookupIsHealthyIdle` would simply time out, i.e. the brief's literal snippet as given does not compile-pass; it hangs and fails, it does not silently pass.
- This is exactly the same fix already applied in sibling test files for the identical reason: `idle_stuck_test.go:70` and `dwell_integration_test.go:511` both call `l.SetLevel(logrus.DebugLevel)` immediately after `silentLogger()`/`test.NewNullLogger()` before asserting on a Debug-level line. Confirmed via direct grep of both files.
- The addition changes nothing about the assertions, the log wording, or the classification logic under test — it only makes the existing hook observe the Debug level it is already supposed to assert against. Ran the tests myself (`go test ./consumer/ -run 'TestEmptyAssignment|TestSteadyPartitionCount|TestZeroAssignmentIsHealthyIdle' -v`) — all 5 pass.

Verdict on the deviation: not a defect. It is a minimal, convention-following correction of an omission in the brief's literal snippet, consistent with established repo pattern.

## Evidence run

```
cd libs/atlas-kafka && go build ./... && go vet ./consumer/... && \
  go test ./consumer/ -run 'TestEmptyAssignment|TestSteadyPartitionCount|TestZeroAssignmentIsHealthyIdle' -v
```
```
=== RUN   TestZeroAssignmentIsHealthyIdle
--- PASS: TestZeroAssignmentIsHealthyIdle (0.31s)
=== RUN   TestEmptyAssignmentWarnsWhenTopicMissing
--- PASS: TestEmptyAssignmentWarnsWhenTopicMissing (0.01s)
=== RUN   TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist
--- PASS: TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist (0.31s)
=== RUN   TestEmptyAssignmentIndeterminateLookupIsHealthyIdle
--- PASS: TestEmptyAssignmentIndeterminateLookupIsHealthyIdle (0.31s)
=== RUN   TestSteadyPartitionCountCausesNoRejoin
--- PASS: TestSteadyPartitionCountCausesNoRejoin (0.31s)
PASS
```
Full `-race ./consumer/` run not re-executed independently beyond a cached confirmation (`go test -race ./consumer/` → `ok ... (cached)`); the implementer's report carries a fresh `-race` run with output, per the task instructions not to re-run the full suite.

## Findings

None blocking. None non-blocking beyond what is discussed above (the `SetLevel` deviation, judged correct).

## Not evaluable

None. The full review surface (both changed files, plus the reused test-helper contracts in `fakegroup_test.go` and the `groupConfig`/`engine_reader.go` guard files) was directly inspected and verified against the binding constraints.

## Verdicts

- **Spec compliance:** PASS. Every item in the brief's produced interface, classification rules, sentinel-arm ruling, debug-wording contract, nil-`pcp` handling, and test-helper-reuse constraint is implemented exactly as specified.
- **Task quality:** PASS. Code is well-commented, the one deviation from the brief's literal test snippet is justified and matches existing repo convention, and the diff is minimal and scoped to exactly the two named files.

## Overall verdict

APPROVED
