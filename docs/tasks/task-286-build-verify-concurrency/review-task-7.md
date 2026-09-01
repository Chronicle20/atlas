# Review — Task 7: verify.sh adopts build slots, per-slot budgets, and a capacity preflight

Commit range: `7b26b2d79..7af8a679b` (single commit `7af8a679b`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git diff --stat 7b26b2d79..7af8a679b`:

```
.../measurements.md   | 65 +++++++++++++++
docs/verification.md  | 34 ++++++++
tools/verify.sh       | 95 ++++++++++++++++++++--
tools/verify_test.sh  | 61 ++++++++++++++
4 files changed, 246 insertions(+), 9 deletions(-)
```

Matches the brief's Files list exactly (`tools/verify.sh`,
`tools/verify_test.sh`, `docs/verification.md`) plus the task's own
`measurements.md`, which the brief's acceptance section requires. No files
outside this set were touched — `tools/lib/build-slot.sh` and
`tools/with-build-slot.sh` (Task 6) are consumed read-only, as the report
claims and the diff confirms.

## Findings

### 1. Preflight fails closed — PASS, independently verified

Ran the three acceptance-mandatory invocations myself (not trusting the
report's transcript):

```
$ ATLAS_MIN_FREE_MB=99999999 tools/verify.sh --base HEAD --no-docker --no-ui
...
✗ preflight (capacity)
1 check(s) FAILED — the branch is not ready.
EXIT=1

$ tools/verify.sh --base HEAD --no-docker --no-ui
...
✓ preflight (capacity)
All checks passed, but docker bake was skipped — not a pre-PR pass.
EXIT=0

$ tools/verify.sh --quick --base HEAD --no-docker --no-ui   (clean retry, see note below)
− preflight (capacity) (--quick)
EXIT=0
```

Fails closed, is not permanently on, and is off the `--quick` path — all
three confirmed by direct execution, matching `tools/verify.sh:169-172,132-159`.

Note: a first attempt at the `--quick` run raced against the concurrently
running gate mentioned in the dispatch (`tools/verify.sh --quick --base
7b26b2d79` in this same worktree) and picked up a stale
`services/zz-verify-probe-broken` fixture the other process's own
`verify_test.sh` probe had transiently created (both processes share the
same fixed, non-`$$` probe paths — see `tools/verify_test.sh:97-104`). That
is contention between two independent gate invocations sharing this
worktree's fixed probe filenames, not a Task 7 defect; a clean retry passed.

### 2. `awk` field index for free RAM — PASS, confirmed on this host

```
$ free -m
               total        used        free      shared  buff/cache   available
Mem:           32053       13438        2998        4473       20541       18615
```

"available" is the last (7th) field on the `Mem:` line, matching
`awk '/^Mem:/ {print $NF}'` at `tools/verify.sh:139`. The report's own
`free -m` transcript shows the same 7-field shape and the same conclusion —
the implementer genuinely ran it rather than assuming the layout.

### 3. Slot granularity — PASS

```
$ grep -n 'acquire_build_slot\|with-build-slot.sh' tools/verify.sh
362:            if acquire_build_slot "go test -race"; then
469:            ./tools/with-build-slot.sh "bake" -- \
```

Exactly two occurrences, at `tools/verify.sh:362` (around the whole
`launch_go_layers` call, `verify.sh:360-372`) and `:469` (around the bake
`step`, `verify.sh:466-470`). Neither is inside `go_layer()` (the per-worker
subshell, `verify.sh:294-309`) or on the guard/lint/`--facts` paths. Verified
by breaking this on a scratch copy: inserting a bogus
`acquire_build_slot "per-worker-bug"` at the top of `go_layer()` moves the
file-wide count from 2 to 3, which the new structural assertion
(`tools/verify_test.sh:244-249`, "appears exactly twice total") would catch.

### 4. `GO_POOL_SLOT_OK` guard — judged on merit, correct

`tools/verify.sh:359-380`. On a failed `acquire_build_slot`, the code does
`FAILED+=("build slot (go test -race)"); GO_POOL_SLOT_OK=0` and then skips
the per-module `replay_go_layer` loop entirely (guarded by
`[ "$FACTS" -eq 1 ] || [ "$GO_POOL_SLOT_OK" -eq 1 ]`). This is a direct
`FAILED` push without a `PASSED`/`step()` counterpart, which is not a new
pattern — the same style is used for the bake-target-resolution failure at
`tools/verify.sh:452-453`. A failed slot acquisition correctly surfaces as a
`FAILED` entry (`${#FAILED[@]} -gt 0` at the summary, `verify.sh:868-869`)
and the run's exit code is non-zero — it does not get swallowed into a false
pass. Under `--facts`, `acquire_build_slot` is never called (guarded by
`[ "$FACTS" -eq 0 ]`), so `GO_POOL_SLOT_OK` stays at its initial `1` and the
per-module labels are always `SELECTED` under `--facts` — correct, since
`--facts` predicts a normal run's selection, and slot-acquisition failure is
an abnormal-host condition outside what `--facts` claims to model.

### 5. `--facts`/real agreement loop — PASS, unaffected

The existing agreement loop (`tools/verify_test.sh:142-159`) only compares
`--quick` runs, and the new slot-acquire code path is gated on
`[ "$QUICK" -eq 0 ]`, so it is never exercised by that loop — consistent
with the brief's Step 1 table, which scopes the agreement requirement to
"unchanged". The new `preflight (capacity)` assertions
(`tools/verify_test.sh:191-198`) check `facts_selected` alone against the
label's presence/absence, not against a real run, which is appropriate since
running the real (non-quick) preflight in this test would compete for a
build slot pointlessly.

### 6. `verify_test.sh` count — PASS, confirmed by name-set diff (not just count)

```
$ git show 7b26b2d79:tools/verify_test.sh | grep -c '^assert_true\|^assert_eq'
25
$ grep -c '^assert_true\|^assert_eq' tools/verify_test.sh
40
```

40 − 25 = 15 assert *call sites* added (the report's "57 ok" is the number
of assertion *executions*, since two of the baseline call sites loop over
`for args in ...` twice each). Diffing the sorted assertion-label sets
(not just counts) confirms exactly 15 new, distinct labels added and zero
removed or renamed — see the name-set diff below (base 23 unique labels →
new 38, all 15 additions are the ones listed in the report's TDD section,
nothing else moved):

```
0a1
> acquire_build_slot/with-build-slot.sh appears exactly twice total (bake + Go pool)
3a5,6
> an un-tuned host is reported, not assumed (names Host tuning)
> a starved run fails rather than proceeding (non-zero exit)
9a13
> go build and go test take distinct -p budgets
11a16,17
> go_layer exports GOMAXPROCS
> go vet takes no -p (not a valid go vet flag)
16a23,25
> no slot acquisition on the guard, lint, --facts, or summary paths
> preflight is NOT selected under --quick
> preflight is selected on a full run
17a27
> the bake (docker layer) holds exactly one slot reference
20a31,32
> the Go pool (go-modules layer) holds exactly one slot reference
> the preflight message names the free-RAM shortfall
21a34,36
> the same command without the override exits 0 (preflight is not permanently on)
> the starved run's summary reports a preflight failure
> the un-tuned-host report points at docs/verification.md
```

### 7. New tests are load-bearing — spot-checked

On a scratch copy (never the tracked file):

- Added `-p "${ATLAS_GO_P:-6}"` to the `go vet` call inside `go_layer()` →
  the "go vet takes no -p" assertion's detection grep (`grep -F 'go vet -p'`)
  fires, which would flip that assertion to FAIL. Confirmed.
- Inserted a bogus `acquire_build_slot` call at the top of `go_layer()`
  (simulating per-worker slotting) → the file-wide `acquire_build_slot|
  with-build-slot\.sh` count goes from 2 to 3, which would flip the
  "appears exactly twice total" assertion to FAIL. Confirmed.

Both breaks were reverted immediately (scratch copies under `/tmp`, never
the tracked `tools/verify.sh`); `git status --porcelain -- tools/` shows no
drift.

### 8. `go_layer` budgets — PASS

`tools/verify.sh:296-308`:
```
export GOMAXPROCS="${ATLAS_GOMAXPROCS:-6}"
go build -p "${ATLAS_GO_P:-6}" ./... && go vet ./... || exit 1
...
go test -p "${ATLAS_GO_TEST_P:-2}" -race ./... || exit 1
```
Matches the brief's Step 3 snippet verbatim, `go vet` correctly gets no
`-p`. Applied only inside `go_layer()`, not elsewhere.

### 9. fd 9 reservation — no conflict

`grep -n '9>\|>&9\|<&9\|exec 9' tools/verify.sh` → no matches. The build-slot
broker itself (`tools/lib/build-slot.sh:35-38`, Task 6, consumed read-only
here) uses a fixed fd 200 for its lock and explicitly documents fd 9 as
reserved for Task 8's module-cache lock. Task 7 introduces no new fd usage.

### 10. `shellcheck -S error` — PASS

`shellcheck -S error tools/verify.sh` → exit 0, no findings.

### 11. Docs — PASS

`docs/verification.md` gained a "What's slotted" paragraph naming the two
slotted phases and the deliberately-unslotted ones, a new "Capacity
preflight" subsection with the threshold table and env-override names, and
a per-slot-budget paragraph under "The Go layer" — all match the
implementation exactly (thresholds, defaults, env var names, `-p` values).

### 12. Four-worktree deferral — correctly recorded, not fabricated

`measurements.md`'s "## Layer 3 — concurrency" section explicitly states
that only one worktree existed at dispatch time, that the controller
resolved not to create additional worktrees for this measurement, and
defers acceptance criterion 4 to the branch-end pass — with no fabricated
`dmesg`/OOM evidence. This matches the controller's ruling stated in this
review's dispatch and is not being treated as a finding against the
implementer.

## Not evaluable

- **Full `verify_test.sh` suite re-run in this environment — root cause
  confirmed, attributed away from Task 7.** I ran the brief's
  individually-mandated invocations directly (see Finding 1) and
  independently confirmed the assertion arithmetic via a name-set diff
  (Finding 6) and via targeted scratch-copy breaks (Finding 7). I also
  attempted a full `./tools/verify_test.sh` re-run in the background; it
  completed with 3 FAILs (`skipped gate count agrees (--quick --base HEAD
  --no-ui)`: want 27 got 23; `the real run selects it too`: got `''`;
  `--facts lists it under guard_suites`: got `''` — all three are
  **pre-existing** assertions, none of the 15 new ones). I traced the cause
  directly rather than assuming: this worktree's `go.work.sum` is currently
  dirty (`M go.work.sum`, +21 lines — see below), and `changed_modules()`'s
  `fanout_paths()` (`tools/verify.sh:237-263`, unmodified by this diff)
  treats any change under `go.work*` as a shared-lib change that fans a run
  out to every module in the monorepo. I confirmed this directly: a
  hand-run `real_skipped_count --quick --base HEAD --no-ui` against the
  current (dirty-`go.work.sum`) tree exceeded a 2-minute timeout building
  every module, where the same invocation is fast against a clean tree. So
  the 3 FAILs are a consequence of the dirty `go.work.sum` in this working
  tree (see next bullet) changing `changed_modules()`'s output between the
  suite's internal sub-invocations — a pre-existing property of
  `changed_modules()`'s fan-out logic reacting to *this session's* dirty
  tree state, not a defect in the Task 7 diff. The report's own
  clean-environment run (`env -u ATLAS_MIN_FREE_MB -u ATLAS_MIN_TMP_MB
  ./tools/verify_test.sh`, 57 ok / 0 FAIL, run before this dirtiness
  accumulated) remains the authoritative evidence for full-suite behavior
  on a clean tree; I was not able to restore a clean tree to re-confirm it
  myself without violating this review's read-only constraint.
- `go.work.sum` shows as modified (`M go.work.sum`, 21 insertions) at time
  of writing this review, and is the proximate cause of the item above. Per
  the dispatch's operating note, this is the known EXIT-trap interaction
  from running `go build`/`go vet` locally (my own `--no-docker --no-ui`
  invocations during this review, and/or the concurrent gate) — not left by
  Task 7's diff itself (the commit does not touch `go.work.sum`) and not
  committed or reverted by me.
- Acceptance criterion 4 (four concurrent worktrees) — deferred by
  controller ruling, not evaluated, not treated as a finding (see Finding 12).

## Verdict rationale

No blocking defects found. The highest-risk surface — fail-closed behavior
of the preflight, and correct slot granularity — was independently
re-verified by direct execution and by breaking a scratch copy, not just by
reading the implementer's transcript. The one design decision beyond the
brief's literal snippet (`GO_POOL_SLOT_OK`) is correct on inspection: a
failed slot acquisition surfaces as a `FAILED` entry and a non-zero exit,
not a silently-skipped pass. The 3 full-suite FAILs observed during this
review were traced to this review session's own dirty `go.work.sum`
interacting with pre-existing (not Task-7-added) fan-out logic, not to the
diff under review.
