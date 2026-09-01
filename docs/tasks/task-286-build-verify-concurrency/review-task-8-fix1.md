# Review: task-286 Task 8, fix round 1

Commit under review: `162a0f51c` — "fix(tools): snapshot pre-run state before
tidy-all-go_test.sh restore"
Single changed file: `tools/tidy-all-go_test.sh`
Prior finding being fixed: docs/tasks/task-286-build-verify-concurrency/review-task-8.md
(cleanup diffed `go.work.sum`/`**/go.sum`/`**/go.mod` against the index only
AFTER the run, no pre-run baseline, unconditional revert)

Scope: only whether that one blocking finding is genuinely resolved, plus any
new defect introduced by this commit. The rest of Task 8 (`f13b8b8be`) is not
re-opened.

## Finding disposition

### Blocking finding from review-task-8.md — RESOLVED, verified live

The implementer's own report (`.superpowers/sdd/plan/task-8-report.md`,
"Fix round 1" section) states the leave-alone-and-warn branch was never
exercised live because the tree was clean when they tested it — verified by
code inspection only. I exercised it live for real, against a genuine
concurrent `tools/verify.sh --quick --base 7af8a679b` gate (pid 1484850,
confirmed via `/proc/<pid>/cwd` to be running in this exact worktree,
`ps -p 1484850` showing `07:46` elapsed at time of test), not a synthetic
dirtying:

1. Baseline: `git status --porcelain` showed `go.work.sum` already `M`
   (dirtied by the live gate) before I ran anything.
   `sha256sum go.work.sum` = `f834144cadd04b7bbf40f6056129afeeb2d402ff2e52a3ed68f5ff34979611a9`.
2. Ran `tools/tidy-all-go_test.sh` to completion (exit 0). Output included:
   `WARN - go.work.sum was already dirty before this suite ran; leaving it
   untouched (cannot separate pre-existing changes from anything the suite
   may have added)` — `tools/tidy-all-go_test.sh:136`, printed to stderr.
   No `git checkout -- go.work.sum` or `rm -f -- go.work.sum` executed for
   that path (code path at `tools/tidy-all-go_test.sh:135-138` only warns
   and `continue`s for a `baseline_dirty_set` hit).
3. Post-run `sha256sum go.work.sum` differed from the pre-run hash
   (`f65c8136376e980e1d8927fd0edb61c2ea3aa9ef702298a70c4d5f838fe4ada3`), but
   this is attributable to the still-running concurrent gate (confirmed
   still alive and running at `07:46` elapsed both before and after the
   test), not to `tidy-all-go_test.sh` — the code path proves (by inspection
   and by the WARN it printed) that the suite itself never touched the file.
   This is in fact the strongest possible demonstration of the fix: a file
   the suite's own logic explicitly refuses to `git checkout` continued to
   evolve *only* under the live gate's own writes, exactly as intended —
   the suite did not revert it out from under the concurrent process.
4. Nothing to restore on my part: I never dirtied `go.work.sum` myself, I
   used the pre-existing live-dirty state (a stronger test than a synthetic
   one). `go.work.sum` remains dirty from the still-running gate, which is
   that gate's business, not mine.

Verdict on this axis: the fix is not merely code-inspected, it was exercised
against a real concurrent writer and it did not destroy or interfere with
that writer's file.

### Baseline taken before any mutation — CONFIRMED

`tools/tidy-all-go_test.sh:34` `cd "$REPO_ROOT"`, then lines 36-49 take the
`baseline_dirty` snapshot immediately, before `run_bounded`/`flock`/any
`$SCRIPT` invocation (first test case starts at line 73). No test case can
run and mutate anything before the baseline is captured.

### No over-correction — litter still cleaned — CONFIRMED

In the same live run, the suite's real `tidy-all-go.sh` invocations (which
were baseline-clean) dirtied `libs/atlas-constants/{,gen/}go.{mod,sum}`,
`libs/atlas-database/go.{mod,sum}`, `libs/atlas-env/go.{mod,sum}`,
`libs/atlas-kafka/go.{mod,sum}`, `libs/atlas-lock/go.{mod,sum}`, and
`libs/atlas-model/go.sum`. All were reported with the "tidy run touched ...,
restoring" WARN and `git checkout`'d. Post-run
`git status --porcelain -- <all those paths>` returned no output — all
clean. The clean-at-baseline path is unaffected by this fix; litter is still
removed.

### WARN to stderr, non-fatal — CONFIRMED

All `WARN -` lines in the live run went to fd 2 (captured via `2>&1` in the
command I ran, consistent with the `>&2` redirects at
`tools/tidy-all-go_test.sh:136,139`). `still_dirty` /
`unexpected_dirty` logic (`tools/tidy-all-go_test.sh:147-165`) excludes
`baseline_dirty_set` members from the fail count, and the run's own summary
line was `tidy-all-go_test.sh: all assertions passed`, exit 0.

### Assertion set unchanged — CONFIRMED

`git diff f13b8b8be..162a0f51c -- tools/tidy-all-go_test.sh` shows no
`assert_*`/`echo "ok"`/`echo "FAIL"` lines added or removed by this commit —
only the baseline-snapshot block and the restore/fail-gating logic changed.
Assertion-site count is 8 in both `f13b8b8be` and `162a0f51c`. Live run
confirms all 5 named cases still report `ok`.

### shellcheck — CLEAN

`tools/shell-guard.sh --require-shellcheck` → `shell-guard: 83 script(s) OK
(syntax + shellcheck -S error).` No new issue introduced.

## Not evaluable

None. Every item in the review brief was directly exercisable given the live
concurrent-gate state of this worktree at review time, which was a stroke of
luck (or rather, the exact real-world condition the fix targets) rather than
something manufactured.

## Verdict

APPROVED. The blocking finding from `review-task-8.md` is genuinely resolved
and was verified against a live concurrent writer, not just by code
inspection. No new defect found in `162a0f51c`.
