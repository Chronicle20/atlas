# Review: Task 4 — Parallel Go layer with ordered reporting (Layer 2, Go half)

Range: `bb4a034b0..90b0c2a05` (1 commit: `90b0c2a05`)
Files: `tools/verify.sh`, `tools/verify_test.sh`, `docs/verification.md`,
`docs/tasks/task-286-build-verify-concurrency/measurements.md`

## Scope confirmed

Matches the brief exactly: `GO_JOBS` default/validation block, `launch_go_layers`
/ `replay_go_layer`, the two-phase driver replacing the serial loop, 5 pool
assertions in `tools/verify_test.sh`, and the `docs/verification.md` /
`measurements.md` updates. No scope creep, no files touched beyond the brief's
list. `go_layer()` itself is untouched, as required (Task 7 owns its resource
budgets).

## Failure propagation — read by hand

`tools/verify.sh:186-210` (approx, by construct):

```sh
GO_LOG_DIR=""
launch_go_layers() {
    GO_LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/verify-go.XXXXXX")"
    trap 'rm -rf "$GO_LOG_DIR"' EXIT
    local i=0
    for mod in "${MODULES[@]}"; do
        while [ "$(jobs -rp | wc -l)" -ge "$GO_JOBS" ]; do wait -n; done
        (
            if go_layer "$mod" >"$GO_LOG_DIR/$i.log" 2>&1; then
                echo 0 >"$GO_LOG_DIR/$i.rc"
            else
                echo $? >"$GO_LOG_DIR/$i.rc"
            fi
        ) &
        i=$((i + 1))
    done
    wait
}

replay_go_layer() {
    cat "$GO_LOG_DIR/$1.log"
    return "$(cat "$GO_LOG_DIR/$1.rc")"
}
```

**PASS** — the aggregate `wait` at the end of `launch_go_layers` does not gate
anything on its own return value; each worker's real exit code is captured
independently inside its own subshell via the load-bearing `if`/`else` (the
comment correctly identifies why `cmd; rc=$?` would be wrong under `set -e`:
the subshell would abort before the `rc` file is written, silently losing the
failure). This sidesteps the classic "bash `wait` aggregate status" bug
entirely — there is no reliance on `wait`'s own exit status for pass/fail.

Traced by hand against the four required scenarios:
- **One worker fails**: its `.rc` file gets a non-zero code; `replay_go_layer`
  `return`s it; `step()`'s `if "$@"; then … else FAILED …` (`tools/verify.sh:108-113`)
  marks it FAILED and prints the `✗ … FAILED` line. Confirmed correct.
- **Several fail**: each writes its own `.rc` independently — no shared state,
  no clobbering. Confirmed correct.
- **The last worker fails**: `wait` (bare, at the end of `launch_go_layers`)
  blocks until every backgrounded subshell — including the last — has
  written its `.rc` file (the `if/else` in each subshell writes it before the
  subshell itself exits), so by the time `wait` returns every `.rc` file
  exists. Confirmed correct.
- **A worker writes to stderr but exits 0**: captured into the same
  `>"$GO_LOG_DIR/$i.log" 2>&1` stream, replayed as part of the pass output,
  rc=0 → PASSED. Confirmed correct (matches serial behaviour, where stderr
  chatter on a passing `go vet` has always been folded into stdout the same
  way).
- **A worker killed by a signal** (SIGKILL etc.): the `.log` file exists
  (created by the `>` redirect at subshell start) but the `.rc` file is never
  written. `replay_go_layer`'s `return "$(cat "$GO_LOG_DIR/$1.rc")"` then
  hits `cat: No such file, no output` → `return ""`. Verified directly:
  `bash -c 'set -euo pipefail; f(){ return "$(cat /nonexistent 2>/dev/null)"; }; f'`
  → `return: : numeric argument required`, exit code 2, and because
  `replay_go_layer` is only ever invoked through `step()`'s `if "$@"; then …`
  guard, `set -e` does not propagate past it — the module is marked FAILED
  (fails closed, not silently passed). Not the exact scenario the brief asked
  about, but confirmed to fail safe rather than fail open.

**Ordered reporting** — confirmed by construct: the replay loop is a plain
sequential `for mod in "${MODULES[@]}"` over `i = 0..n-1`, `cat`ing
`$GO_LOG_DIR/$i.log` in that fixed order regardless of actual completion
order. A reader cannot tell concurrency happened, and a failure is
unambiguously attributed to its module label. Confirmed by the implementer's
manual deliberately-broken-module test (report, acceptance criterion 3) and
independently by code reading — no path exists for the pool's log-cat loop to
interleave or reorder.

**`--facts` contract preserved**: when `FACTS=1`, `step()` returns at
`tools/verify.sh:103-106` before evaluating `"$@"`, so `replay_go_layer` is
never actually invoked in that path even though it's passed as an argument —
confirmed by reading `step()`'s early return. `launch_go_layers` is
correspondingly skipped by the `if [ "$FACTS" -eq 0 ]; then … fi` guard
(`tools/verify.sh:220-222` by construct), written as the brief's required
`if` form rather than the `set -e`-hazardous bare `&&`. Confirmed correct.

## GO_JOBS knob (cross-task contract with Task 7)

`tools/verify.sh` (near line 26-31, confirmed present):
```sh
GO_JOBS="${ATLAS_VERIFY_GO_JOBS:-4}"
case "$GO_JOBS" in
    ''|*[!0-9]*|0) echo "verify.sh: ATLAS_VERIFY_GO_JOBS must be a positive integer (got '$GO_JOBS')" >&2; exit 2 ;;
esac
```
One clearly identifiable default (`4`), one override variable
(`ATLAS_VERIFY_GO_JOBS`), validated at startup with `exit 2` and a message
naming the variable. `launch_go_layers` is a standalone function, not inlined
into the driver, so Task 7 can wrap its call site
(`if [ "$FACTS" -eq 0 ]; then launch_go_layers; fi`) in a build-slot
acquire/release without restructuring. Both satisfied.

## Non-blocking observation: `jobs -rp` / `wait -n` under `set -e`

`while [ "$(jobs -rp | wc -l)" -ge "$GO_JOBS" ]; do wait -n; done` — `wait -n`
with zero background jobs returns 127 (verified:
`bash -c 'set -euo pipefail; wait -n'` → exit 127, no output), and a failing
statement in a `while` loop *body* (not its condition) is not exempt from
`set -e`. In principle a race between the `jobs -rp` snapshot and the `wait -n`
call could abort the whole script. In practice this is unreachable here: the
`while` condition can only be true when `jobs -rp` observed at least one
*running* background job, and bash does not remove a job table entry when it
exits — it marks it `Done` and leaves it for the next `wait`/`jobs` to reap —
so `wait -n` always has something to reap when the loop body runs. This
pattern is verbatim from the brief (not an implementer deviation), is a
widely-used bounded-parallelism idiom, and I found no realistic path to the
127 case. Flagging for awareness only, not blocking.

## Blocking: no automated regression test for failure attribution through the pool

The 5 new pool assertions (`tools/verify_test.sh:284-307`) are:
1. gate labels job-count invariant (1 vs 4)
2. module count job-count invariant (1 vs 4)
3. `GO_JOBS=0` rejected, exit 2, stderr names the variable
4. `GO_JOBS=x` (non-numeric) rejected, exit 2
5. structural grep for the `mktemp`/`trap` lines

None of these exercises a *failing* module through the pool. Assertions 1-2
run against the working tree's actual (currently all-passing) module set, and
critically, `real_selected()` (`tools/verify_test.sh:75-80`) strips the
`✓`/`✗` prefix before comparing labels:
```sh
real_selected() {
  "$VERIFY" "$@" 2>/dev/null | ... | sed -n 's/^  [✓✗] *//p' | ...
}
```
So assertion 1 ("gate labels are job-count invariant") would still pass
byte-for-byte even if `replay_go_layer`'s exit-code plumbing were silently
broken such that every module reported `✓` regardless of its real result —
the label text is identical either way, only the leading glyph (discarded by
the `sed`) would differ. None of assertions 2-5 touch pass/fail attribution
either.

The only test of the failure path in this task was the implementer's manual,
unrecorded, already-reverted acceptance-criterion-3 run (broken
`services/atlas-kafka-precreate/main.go`, confirmed `FAILED` + correct
attribution, then reverted) — it exists only in the task report's prose, not
in the committed test suite. A future edit to `launch_go_layers` or
`replay_go_layer` that reintroduces exactly the bug this task's own comment
warns against (`cmd; rc=$?` losing the failure under `set -e`) would pass all
37 assertions in `tools/verify_test.sh` today, including all 5 new pool
assertions.

This is squarely the class of defect this review was asked to weight most
heavily ("a gate that silently passes when a module fails is the worst
possible defect here"), and it is a gap in test honesty (review priority 4)
rather than a defect in the shipped code itself — the code is correct as
verified above. It is feasible to close with the same probe pattern already
used elsewhere in this file (e.g. the bake-target probes at
`tools/verify_test.sh:177-191`): create an untracked, deliberately-broken Go
module (or module with a syntax error), assert the real run reports it
`FAILED` and names it, then clean up. Recommend adding this before merge, or
in an immediate follow-up commit to this same task before Task 7 builds on
top of an unguarded `launch_go_layers`.

## Test suite / gate re-run (this session)

- `./tools/verify_test.sh` — re-ran directly: **37/37 assertions pass**,
  including all 5 new pool assertions and the pre-existing structural,
  `--facts` agreement, and Task 3 bake-selection assertions (all unweakened,
  all still present, all still passing).
- `./tools/shell-guard.sh --require-shellcheck` — re-ran directly: `76
  script(s) OK (syntax + shellcheck -S error)`.
- `git status --porcelain` — clean except pre-existing review artifacts from
  earlier tasks in this same review session; no leftover probe files from
  this task's test run.
- Did not re-run the flagless `tools/verify.sh` bake, nor a fresh
  deliberately-broken-module run (out of scope per instructions — a separate
  `--quick --base bb4a034b0` run is in flight elsewhere and its verdict is
  not mine to produce).

## Doc updates

`docs/verification.md` `## The Go layer` — new paragraph accurately describes
the pool, the default (`4`), the override variable, the log-and-replay
mechanism, and the `0`/non-numeric rejection. Matches the implementation.

`measurements.md` `## Layer 2 — parallelism (Go half)` — records a fair
warm/warm comparison (35.3s pooled vs 115.2s serial, `GO_JOBS=4`, 91 modules)
with an explicit "what was/was not measured" section (excludes `-race`, the
log-replay overhead itself, and any figure from a full flagless
`tools/verify.sh` run). Honest about scope; no fabricated numbers.

## Not evaluable

- Live behaviour under actual host CPU contention (the flaky-timeout /
  `pkill` incident) — already adjudicated by the controller as not this
  review's concern, and not re-litigated here.
- Task 7's build-slot wrapping of `launch_go_layers` — not yet written,
  correctly out of scope for this task.

## Findings summary

Blocking (1):
- `tools/verify_test.sh:278-307` — the 5 new pool assertions never exercise a
  failing module through the pool, and `real_selected()`'s label comparison
  (`tools/verify_test.sh:75-80`) discards the `✓`/`✗` glyph, so a regression
  that silently turns a FAILED module into a reported PASS through
  `launch_go_layers`/`replay_go_layer` would not be caught by this test
  suite. Recommend adding an automated failing-module probe (same pattern as
  the existing bake-target probes) before this is relied on by Task 7.

Non-blocking (1):
- `tools/verify.sh` (`jobs -rp`/`wait -n` throttle loop) — a theoretical
  `set -e`-abort race if `wait -n` were ever called with zero background
  jobs; traced by hand and found unreachable given how bash reaps job-table
  entries, and the pattern is verbatim from the brief. Flagged for awareness,
  not blocking.
