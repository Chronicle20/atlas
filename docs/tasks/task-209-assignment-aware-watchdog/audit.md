# Plan Audit — task-209-assignment-aware-watchdog

**Plan Path:** docs/tasks/task-209-assignment-aware-watchdog/plan.md
**Audit Date:** 2026-08-10
**Branch:** task-209-assignment-aware-watchdog
**Base Branch:** main

## Executive Summary

All 9 plan tasks were faithfully implemented, verified by reading the actual
code introduced by each commit (not just commit subject lines) and diffing it
against the plan's "Interfaces" sections. The plan's checkboxes were never
ticked, but the work is real: 17 commits, confined entirely to
`libs/atlas-kafka/consumer/`, `libs/atlas-kafka/README.md`, and
`docs/tasks/task-209-assignment-aware-watchdog/` — `git diff --stat
main...HEAD -- services/` is empty, satisfying FR-3.2. The Task 1 "pure move"
commit (`e463478d8`) is byte-for-byte identical code relocation (only a new
package-level doc comment was added at the top of the new file). FR-2.5
(no stall/wedge warnings while unassigned) is both structurally true (the
zero-assignment branch in `runGenerations` only logs at Debug and never calls
into `runPartition`/`handleFetchDeadline`) and covered by an explicit
logrus-hook-based test (`TestZeroAssignmentIsHealthyIdle`) and again by the S6
integration scenario. All 15 FR-3.1 frozen symbols exist with unchanged
signatures. One documented, well-justified deviation from the plan's literal
Task 5 interface signature was found (a later fix commit changed
`runPartitionFetchLoop`'s signature to avoid a `sync.WaitGroup` reuse panic);
this is marked DONE-with-note per the "same goal, different method" audit
rule. One small documentation inconsistency was found in `verification.md`
(a stale placeholder "Bake result" section contradicts the already-reported
Step 5 bake success above it) — cosmetic, not a functional gap. No unauthorized
scope expansion into the PRD §9 / plan "Open items carried forward" list was
found (no Option B shared-group code, no legacy-engine deletion, no
`WatchPartitionChanges: true`), and the `WatchPartitionChanges` opt-in
documentation obligation was fulfilled in the README. Builds/tests/guards were
not independently re-run in full per the dispatching instructions (goroutine-guard
was spot-checked and passed); their orchestrator-reported results are
recorded below as reported, not re-verified, except where independently
spot-checked.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Move legacy engine verbatim into `engine_reader.go` | DONE | `git show e463478d8` — `engine_reader.go` (+238) is character-identical to the removed `manager.go` block (−217, diff is exactly the 217 pre-existing lines plus one new 21-line package header comment, verified line-by-line). No signature, log line, or error-handling change. |
| 2 | Per-partition prefix-commit cursor | DONE | `libs/atlas-kafka/consumer/cursor.go` implements every symbol in the plan's Interfaces list (`inflight`, `cursor`, `newCursor`, `track`, `mark`, `len`, `advance`, `committedOffset`, `resumeOffset`, `reset`). `advance` commits `msg.Offset+1` (`cursor.go:81`, `pendingCommit = c.pending[i-1].offset + 1`) and never advances `committed` on a commit error (`cursor.go:97-99`). |
| 3 | `Group`/`Generation` seams and config builders | DONE | `libs/atlas-kafka/consumer/group.go` matches the plan's interfaces exactly: `Group`, `Generation`, `GroupProducer`, `PartitionReaderProducer`, `ConfigGroupProducer`, `ConfigPartitionReaderProducer`, `groupConfig()`, `partitionReaderConfig()`. `partitionReaderConfig` (`group.go:123-130`) builds a `kafka.ReaderConfig` literal with no `GroupID` field set at all — confirmed via `grep -n "GroupID:" libs/atlas-kafka/consumer/group.go` returning no match. `groupConfig()` sets `WatchPartitionChanges: false` explicitly (`group.go:115`). |
| 4 | Per-partition watchdog state and observability | DONE | `manager.go:295-362` — `legacyPartition = -1`, `partitionState` struct, `partitionStateFor`, `onAssignment` all match the plan's field/signature list exactly. `Snapshot()` (`manager.go:407-465`) aggregates `ConsecutiveTimeouts` by max, `IdleTicks`/`NoProgressTicks` by sum, timestamps by most-recent, and `AssignedPartitions` is built with `append([]int{}, ...)` so it is never nil. |
| 5 | Per-partition fetch loop with in-place recovery | DONE (documented deviation) | `partition.go` implements `runPartition`, `runPartitionFetchLoop`, `quiesce`, `drainTimeout=5s`, and `fakegroup_test.go` implements `fakeGroup`/`fakeGeneration` per the plan. **Deviation:** the plan's Task 5 Interfaces list specifies `runPartitionFetchLoop(..., wg *sync.WaitGroup, ...) error`; the shipped signature is `runPartitionFetchLoop(...) (*sync.WaitGroup, error)` — the function now allocates and returns its own per-attempt `WaitGroup` instead of receiving one. The code comment at `partition.go:119-124` explains why: a partition-lifetime `WaitGroup` reused across rebuild attempts could hit Go's documented-unsafe "Add concurrent with Wait" race if a straggler from a timed-out `quiesce` was still outstanding when a later attempt's `wg.Add` fired. This is commit `9148aedd1` ("bound serial-path drain and scope WaitGroup per rebuild attempt"), a self-identified fix during the branch's own review pass, not a silently dropped requirement. Same goal (bounded drain, FR-1.6) achieved by a safer mechanism. |
| 6 | Generation loop and assignment-aware liveness | DONE | `engine_group.go` — `startGroupEngine`, `runGenerations` match the plan's control flow exactly. The zero-assignment branch (`engine_group.go:77-87`) only logs at Debug and `continue`s — it never reaches `runPartition`/`handleFetchDeadline`, so the Warn-emitting code is structurally unreachable while unassigned (FR-2.5). `engine_group_test.go:35-80` (`TestZeroAssignmentIsHealthyIdle`) asserts via a `logrus/hooks/test` hook that no "wedged" or "stall suspect" log entries appear, plus `RecreateCount==0`, `NoProgressTicks==0`, and that `Next` is not called more than once (parks, does not spin). |
| 7 | Engine selection, default flip, rollback switch | DONE | `engine.go` matches the plan's code block verbatim: `EngineName`, `EngineConsumerGroup`/`EngineReader` constants, `engineEnvVar = "KAFKA_CONSUMER_ENGINE"`, `ConfigEngine`, `resolveEngine` (empty/unset → consumergroup default; unrecognised → Warn + default; explicit values pass through), and `(*Consumer).start` as the engine dispatcher. `GetManager` resolves the default at `manager.go:110` via `resolveEngine(logrus.StandardLogger())` before running configurators, so `ConfigEngine` can override it. All `GetManager(...)` calls in `manager_test.go` (21), `idle_stuck_test.go` (3), `timing_test.go` (1) and `debug_test.go` (3) are pinned with `ConfigEngine(EngineReader)` — every call site checked. |
| 8 | Integration scenarios — S6 and cross-engine round trip | DONE | `dwell_integration_test.go` — `dwellEngines` var present (line 44); `TestDwellS1..S4` each wrap their body in `for _, engine := range dwellEngines { t.Run(...) }`; `TestDwellS5` left unchanged (engine-agnostic, as specified); `TestDwellS6_MembersExceedPartitions` (line 504) and `TestCrossEngineOffsetRoundTrip` (line 609) both present with the exact assertions the plan specifies (`require.Zero(t, totalRecreates)`, `require.NotContains(... "wedged"/"stall suspect")` for S6; bidirectional `consumergroup-then-reader`/`reader-then-consumergroup` sub-tests for the round trip). |
| 9 | Documentation and full verification gate | DONE | `libs/atlas-kafka/README.md` gained the "Consumer engines" section verbatim from the plan, including the `WatchPartitionChanges` opt-in note. `docs/tasks/task-209-assignment-aware-watchdog/verification.md` created with the Step 2–6 evidence, S1–S6 + cross-engine numbers, and a "Post-deploy measurements still outstanding" section. One inconsistency found: see Skipped/Deferred Tasks below. |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (1 documented, justified signature deviation noted above; not a partial implementation of the requirement it serves — FR-1.6 is still fully met)

## Skipped / Deferred Tasks

None skipped. One minor documentation defect, not a task gap:

- **`verification.md` stale "Bake result" placeholder.** The file's Step 5
  section (lines 71-88) already reports `docker buildx bake all-go-services`
  exiting 0 with all 62 targets built. But a leftover section at the very end
  of the file (lines 168-172, "## Bake result") still reads `docker buildx
  bake all-go-services — result recorded below once the run completes.` —
  an unresolved placeholder that was never deleted after Step 5 was written.
  It does not contradict any acceptance criterion (the actual bake result is
  correctly recorded higher up and independently confirmed during this audit:
  `docker buildx bake all-go-services --print` enumerates exactly 62 targets
  in the `all-go-services` group, matching the verification.md claim), but it
  is sloppy and could confuse a future reader into thinking the bake was never
  completed. Low-severity cleanup item.

## Build & Test Results

Per the dispatching instructions, the full build/test/bake/integration suite
was **not** independently re-run in this audit (it was already run by the
orchestrating agent and its results are treated as confirmed per the
instructions). Spot-checks performed directly in this audit are marked
"independently confirmed":

| Check | Result | Source |
|---|---|---|
| `go test -race ./...` in `libs/atlas-kafka`, incl. `-count=5` on `consumer` | PASS (reported) | verification.md Step 2 — not re-run |
| 63/63 dependent modules build+vet+test clean | PASS (reported); **module count independently confirmed** (`grep -rl atlas-kafka --include=go.mod services libs \| xargs -n1 dirname \| wc -l` → 63) | verification.md Step 3 |
| `tools/redis-key-guard.sh` | PASS (reported) | verification.md Step 4 — not re-run |
| `tools/goroutine-guard.sh` | PASS — **independently re-run in this audit, exit 0** | spot-check |
| `tools/lint.sh --check` | PASS (reported) | verification.md Step 4 — not re-run |
| `docker buildx bake all-go-services`, 62/62 targets | PASS (reported); **target count independently confirmed** (`docker buildx bake all-go-services --print` → 62 targets in the `all-go-services` group) | verification.md Step 5 |
| Integration suite (`-tags integration`), S1-S6 + cross-engine round trip, both engines | PASS (reported) | verification.md — not re-run (requires Docker/testcontainers, ~10+ min); **test function names/count independently confirmed present in source** (7 top-level scenario functions: S1-S6 + `TestCrossEngineOffsetRoundTrip`, matching verification.md's scenario table) |
| `git diff --stat services/` empty (FR-3.2) | PASS — **independently confirmed** in this audit | `git diff --stat main...HEAD -- services/` → empty |
| `git status --short` clean at audit end | PASS — **independently confirmed** | only this new `audit.md` will be untracked after this write |

## Cross-Cutting Verdicts (audit brief items 1–9)

1. **Task 1 pure move:** CONFIRMED. `git show e463478d8` diff is exactly the
   217 pre-existing lines cut from `manager.go` pasted unmodified into the new
   `engine_reader.go`, plus one new 21-line package doc comment. No renamed
   variables, no changed error handling, no changed log lines, no changed
   signatures.
2. **FR-3.2 (`git diff --stat main...HEAD -- services/` empty):** CONFIRMED
   empty.
3. **FR-3.2 corollary (no `GroupID` on the partition reader):** CONFIRMED.
   `partitionReaderConfig` (`group.go:123-130`) never sets `GroupID`;
   `grep -n "GroupID:" libs/atlas-kafka/consumer/group.go` returns no match.
4. **FR-2.5 (no stall/wedge warn while unassigned):** CONFIRMED both
   structurally (`engine_group.go`'s zero-assignment branch only calls
   `l.Debugf` and `continue`s — the warn-emitting `handleFetchDeadline` path is
   only reachable via `runPartition`, which is only invoked by `gen.Start` when
   `len(parts) > 0`) and by test
   (`engine_group_test.go:TestZeroAssignmentIsHealthyIdle` asserts via a
   `logrus/hooks/test` hook that no "wedged"/"stall suspect" entries appear;
   `dwell_integration_test.go:TestDwellS6_MembersExceedPartitions` re-asserts
   the same at integration scope).
5. **FR-5.1/5.3 (env switch, `ConfigEngine`, identical offset semantics,
   cross-engine round trip):** CONFIRMED. `KAFKA_CONSUMER_ENGINE` defaults to
   `consumergroup` (`engine.go:resolveEngine`); `ConfigEngine(EngineReader)`
   wired as an additive `ManagerConfig`, resolved inside `GetManager`'s
   `once.Do` before configurators run. Both engines commit `msg.Offset+1`:
   legacy via `reader.CommitMessages` (kafka-go's own `+1` advance, untouched
   by the Task 1 move) and the new engine via `cursor.advance` →
   `pendingCommit = c.pending[i-1].offset + 1` (`cursor.go:81`). A
   cross-engine round-trip integration test exists
   (`TestCrossEngineOffsetRoundTrip`, both directions) and is reported PASS in
   `verification.md`.
6. **PRD §9 / plan "Open items carried forward":** CONFIRMED not silently
   implemented. No Option B (shared multi-topic `ConsumerGroup`) code found
   (`grep` for shared-group/multi-topic patterns in `consumer/*.go` returns
   nothing). `engine_reader.go` (the legacy engine) still exists, unremoved.
   `WatchPartitionChanges` is `false` everywhere it is set
   (`group.go:115`) and never `true` anywhere in non-test code. The
   `WatchPartitionChanges` opt-in documentation obligation (PRD §9.5 / design
   §10) is fulfilled: `libs/atlas-kafka/README.md`'s "Partition-count changes"
   section documents the manual opt-in path.
7. **`verification.md` numeric claims vs. repo state:** Module-count (63) and
   docker-bake target-count (62) independently re-derived and match. The 7
   integration test function names/count in `dwell_integration_test.go` match
   verification.md's scenario table (S1-S6 plus the cross-engine round trip).
   The one discrepancy found is the stale "Bake result" placeholder described
   above — cosmetic only, does not contradict any number.
8. **`maxInFlight` defaults to 1, serial path unchanged:** CONFIRMED.
   `manager.go:170-173` clamps `maxInFlight` to a minimum of 1 in
   `AddConsumer`. `runPartitionFetchLoop`'s serial branch
   (`partition.go:188-211`) fetches, dispatches exactly one handler, and waits
   for it to finish before fetching again — same one-at-a-time semantics as
   the legacy `runFetchLoopSerial`, now on a trackable goroutine so `quiesce`
   can bound it (see Task 5 deviation note above).
9. **FR-3.1 frozen exported symbols:** CONFIRMED, all 15 present with
   unchanged signatures: `Manager`, `GetManager`, `ResetInstance`,
   `ConfigReaderProducer`, `ManagerConfig`, `Manager.AddConsumer`,
   `Manager.AddConsumerAndRegister`, `Manager.RegisterHandler`,
   `Manager.RemoveHandler`, `Manager.Consumers`, `Consumer`,
   `Consumer.Snapshot`, `Config`, `NewConfig`, and all six `Set*` decorators
   (`SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`, `SetFetchTimeout`,
   `SetMaxConsecutiveTimeouts`, `SetMaxInFlight`) — grepped directly in
   `manager.go`/`config.go`.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the mandatory
  `superpowers:requesting-code-review` pass called for by plan Task 9 Step 9,
  and the two post-deploy live measurements the PRD names as outstanding —
  neither is a test-substitutable gate)

## Action Items

1. (Cosmetic, optional) Delete or update the stale "## Bake result" placeholder
   section at the end of `docs/tasks/task-209-assignment-aware-watchdog/verification.md`
   (lines 168-172) — it contradicts the already-reported Step 5 result above it.
2. Confirm plan Task 9 Step 9 (`superpowers:requesting-code-review` →
   `plan-adherence-reviewer` + `backend-guidelines-reviewer`) has run before
   opening the PR, per CLAUDE.md's "Code Review Before PR" rule — this audit
   covers plan adherence only, not the DOM-* backend-guidelines checklist.
3. Track the two PRD §10 post-deploy measurements
   (`atlas-main` recreate-rate-per-hour ~0, and a 10-minute play session with
   no multi-second attack→drop-visible stall) as follow-up verification after
   this branch ships — they cannot be satisfied pre-merge.
