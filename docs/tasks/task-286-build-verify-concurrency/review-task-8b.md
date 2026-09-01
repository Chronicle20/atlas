# Review: task-8b — make `tools/verify_test.sh` probes concurrency-safe

Commit under review: `24ddd77fc` (range `162a0f51c..24ddd77fc`).
Brief: `.superpowers/sdd/plan/task-8b-probe-fix-brief.md`
Report: `.superpowers/sdd/plan/task-8b-report.md`

Scope confirmed: exactly one file changed, `tools/verify_test.sh` (50
insertions / 13 deletions, per `git diff --stat 162a0f51c..24ddd77fc`). No
other file in the range. Matches the brief's stated scope constraint
("the ONLY file this task changes").

## Verdict

APPROVED_WITH_FINDINGS

## 1. Is the literal sweep complete?

`grep -n "zz-verify" tools/verify_test.sh` at HEAD shows every remaining
occurrence is a variable declaration or interpolation (`${probe_tag}`);
no bare `zz-verify-probe*` literal remains. In particular:

- `tools/verify_test.sh:389` — the FAILED-module grep — now reads
  `grep "zz-verify-probe-broken-${probe_tag}"`, matching the actual
  per-process directory name created at `tools/verify_test.sh:110`.

I proved this is load-bearing, not vacuous, by direct experiment (not by
inspection alone):

- Reverted line 389 in-place to the OLD literal (`grep
  'zz-verify-probe-broken'`, no tag) while leaving the rest of the file
  (including the tagged `probe_broken_dir`) untouched, then ran the full
  suite. Result: still 57/57 `ok`, including `ok - the broken module is
  reported FAILED, unstripped`. **The assertion passes either way**,
  because `grep` is a substring match and the un-tagged literal is a
  prefix of the tagged directory name (`zz-verify-probe-broken` is a
  substring of `zz-verify-probe-broken-3524981`). This means the sweep at
  this specific line, while correctly done, is not actually self-verifying
  — a future regression that silently dropped the `${probe_tag}` suffix
  here would not be caught by this test suite. This is a **non-blocking
  test-design weakness**, not a live defect (the literal was in fact
  updated correctly), but worth a one-line follow-up note (anchor the
  grep, e.g. with a trailing `-` or word boundary) if this file is touched
  again.
- Confirmed the assertion is not always-true: replacing the pattern with a
  string that cannot appear (`zz-totally-nonexistent-marker-xyz`) produced
  `FAIL - the broken module is reported FAILED, unstripped (got '')`, exit
  1, 56/57 ok — so the assertion genuinely exercises real output, it's
  only insensitive to the specific tag suffix.
- Restored the file after each experiment via direct diff against the
  pre-experiment backup; confirmed byte-identical, and `git status
  --porcelain` showed no residual diff on `tools/verify_test.sh` afterward.

**Disposition: PASS**, with the substring-match caveat noted above as
non-blocking.

## 2. Did the `ok` count and label set actually stay identical?

Derived independently, not taken from the report:

- Ran `bash tools/verify_test.sh` at HEAD (`24ddd77fc`): exit 0, `57` `ok`
  lines, `0` `FAIL` lines, tree clean afterward.
- Temporarily swapped `tools/verify_test.sh` for `git show
  162a0f51c:tools/verify_test.sh` (the pre-change version) in place, ran
  it: exit 0, `57` `ok`, `0` `FAIL`.
- Restored the post-change file (`cp` from a pre-swap backup); confirmed
  byte-identical to the committed version and `git status --porcelain`
  clean.
- Normalized both runs' `ok`/`FAIL` label lines (collapsing the one
  wall-clock-second label to `(Ns)`), sorted, and `cmp`'d: **`cmp` exit 0
  — the two label sets are byte-identical**, 57 lines each.

**Disposition: PASS.** The report's claim is independently confirmed, not
just trusted.

## 3. Was the scope expansion justified, and is the `flock` usage correct?

The report's own framing — that a pure path rename cannot make "no probes,
no bake gate" or "removing it deselects the LB port gate" pass under two
concurrent runs, because `bake_targets()` dedupes by SERVICE and the LB-port
gate fires on any `deploy/`-prefixed change, neither keyed on this run's own
probe leaf name — is correct on inspection of the surrounding code
(`tools/verify_test.sh:277-282` documents this precisely, and matches how
`bake_targets()`/the deploy-path gate are actually invoked via
`facts_selected`/`facts_key`, which this review did not need to re-derive
since the diff's own comments state the exact mechanism and the assertions'
existing logic — visible at `tools/verify_test.sh:289-303` — clearly
operates on category-level facts, not per-probe names). Given the brief's
own acceptance line required "two concurrent full runs both exit 0," which
is unreachable without addressing this, landing the fix in the same commit
(rather than a silent no-op "acceptance line descoped") is a reasonable,
disclosed judgment call, not scope creep smuggled in quietly.

**Lock file is correctly shared, not per-process.** `shared_state_lock=
"${TMPDIR:-/tmp}/atlas-verify-test-shared-state.lock"` (line 121) is a
FIXED path — no `${probe_tag}` — so all concurrent instances of
`verify_test.sh` on the same machine contend on the same file, which is
what makes it an actual mutex rather than a no-op. Confirmed by grep: no
`probe_tag` interpolation anywhere near `shared_state_lock`.

**fd choice does not collide with existing locking.** `tools/verify_test.sh`
uses fd 8 (`exec 8>"$shared_state_lock"`). `tools/tidy-all-go.sh` uses fd 9.
`tools/lib/build-slot.sh` uses fd 200. Confirmed by `grep -n "exec [0-9]>"
tools/*.sh tools/lib/*.sh`: only these three fd numbers appear, all
distinct. No fd collision with Task 6's build-slot broker or Task 8's
`tidy-all-go.sh`.

**No self-deadlock via the RECURSION guard.** `tools/verify.sh` runs every
changed `tools/*_test.sh`, including `verify_test.sh` itself, so a
flock-protected top-level run's own subprocess invocation of `$VERIFY
--quick` (inside the broken-module block, still holding fd 8) could in
principle re-invoke `verify_test.sh` recursively. Traced this: the
`ATLAS_VERIFY_TEST_INNER` guard (`tools/verify_test.sh:60-66`) makes any
nested invocation run ONLY the `structural()` assertions and `exit 0`
**before** reaching `probe_tag`, `shared_state_lock`, or any `flock` call
(all declared later, from line 104 on). A nested child therefore never
attempts to acquire fd 8, so it cannot deadlock against its own ancestor
holding the same lock. Verified this is real, not just inspected: while
running my own experiments above, a genuinely separate top-level
`verify_test.sh` instance (PID visible in litter file names, e.g.
`services/zz-verify-probe-broken-3501077/`, `tools/zz-verify-probe-
3501077_test.sh`) was running concurrently — almost certainly the
background `tools/verify.sh --quick --base 162a0f51c` gate's own
shell-tooling-guard step treating `verify_test.sh` as one of its discovered
test suites (a genuine, uncoordinated top-level run, not a recursive
child). All of my own runs and that concurrent run completed with exit 0
and left the tree clean — a live demonstration of the two-top-level-
instances scenario the brief describes, not a synthetic one.

**Every acquire has a matching release on the success path; no early exit
inside a critical section.** Confirmed by `grep -n flock`: three
`flock 8` / `flock -u 8` pairs (lines 192/199, 283/303, 354/384), and by
reading each critical section's body (`tools/verify_test.sh:192-199,
270-303, 347-389`): none contains `exit`, `return`, or any other early-out
between acquire and release; `set -uo pipefail` (no `-e`) means a failing
assertion never short-circuits the section. On a signal (INT/TERM) or an
uncaught EXIT mid-section, the `trap cleanup EXIT INT TERM` runs `cleanup()`
(which does not explicitly `flock -u 8`), but process exit closes fd 8,
which releases the kernel-held flock automatically — no permanent lock
leak from a killed run. (`SIGKILL` still leaves the lock held only for the
lifetime of the killed process — inherent to any `flock`-based scheme, not
a regression this diff introduces.)

**Minimum needed / no unprotected 4th shared state found.** Read the whole
diff for any other filesystem or `go.work.sum`-adjacent write outside the
three locked sections; found none — the remaining probe operations
(`probe_suite`, `probe_deploy`, `probe_ban_dir`/`probe_account_dir`,
`probe_jobs0_err`) are all genuinely per-process (leaf name carries
`${probe_tag}`) and correctly left unlocked.

**Disposition: PASS.** The `flock` usage is correct, matches the existing
`tools/tidy-all-go.sh` precedent, does not collide with fd 8/9/200
conventions, and cannot deadlock against `tools/verify.sh`'s own build-slot
locking or against itself via the recursion guard.

## 4. Cleanup and litter

- `cleanup()` (`tools/verify_test.sh:139-145`) removes every per-process
  probe path, including the newly-tagged `probe_jobs0_err`
  (`tools/verify_test.sh:111`, previously a fixed, untagged literal
  declared inline at its use site — the report's self-described "seventh,
  previously-untagged fixed name" — confirmed present and now tagged).
- `trap cleanup EXIT INT TERM` (line 145) is unchanged in structure from
  before the diff; it already referenced the (now-tagged) variables.
- Empirically verified clean-tree behavior three times: a solo run, a
  four-experiment sequence (each restored), and while a genuinely separate
  concurrent top-level run (PID 3501077) was executing — `git status
  --porcelain` showed only pre-existing, unrelated untracked doc files in
  every check, no stray probe files, no modified tracked files beyond a
  transient, self-cleaning mid-flight snapshot of the other run's own
  litter (observed once, gone by the next check — the other run's own
  trap cleanup firing, not evidence of a leak).
- Litter-glob risk: `probe_suite` keeps the `zz-` prefix and `_test.sh`
  suffix per the brief's constraint, so a crashed run's stray file would
  still match `tools/*_test.sh` and could in principle be picked up by
  the shell-tooling guard — but this is unchanged from before the diff
  (same risk class, same mitigation: the trap). The PID tag does not
  reduce or increase this specific risk; it only prevents two
  simultaneously-running suites from fighting over the identical
  filename. No shell-guard-glob-specific regression found.

**Disposition: PASS.**

## Non-blocking findings

1. `tools/verify_test.sh:389` — the FAILED-module grep matches by
   substring, so it does not actually distinguish a correctly-tagged
   probe name from the old, untagged literal (proven by reverting it and
   observing the suite still pass). Not a live bug — the literal is
   correctly tagged as landed — but a latent test-design gap: a future
   accidental de-tagging of this one line would go undetected by the
   suite itself. Consider anchoring the grep (e.g. append a trailing
   non-digit boundary) if this file is touched again.
2. The cross-process race the diff's `flock` protects against is
   specifically "another `verify_test.sh` instance" (including one nested
   inside a live `tools/verify.sh` gate's own test-suite discovery, which
   is the scenario the brief's motivating text describes). It does **not**
   protect `go.work.sum` against a wholly unrelated, non-`verify_test.sh`
   Go toolchain invocation (e.g. a developer's own `go build ./...`)
   mutating `go.work.sum` at the same instant. That is a pre-existing,
   general Go-workspace hazard outside this defect's stated scope
   (verify.sh/verify_test.sh interaction) and would exist with or without
   this diff; flagged for completeness, not as a defect in this change.

## Not evaluable

None — the full diff (63 lines) was read, and every claim in the report
was independently reproduced rather than trusted.

## Evidence log (commands run, in order)

- `git diff --stat 162a0f51c..24ddd77fc` — confirms single-file scope.
- `git diff 162a0f51c..24ddd77fc -- tools/verify_test.sh` — full hunk read.
- `grep -n "zz-verify" tools/verify_test.sh` — literal sweep completeness.
- Scratch experiment: reverted `tools/verify_test.sh:389`'s literal to the
  pre-change form, ran the full suite, restored, diffed against backup to
  confirm exact restoration.
- Scratch experiment: replaced the same literal with a non-matching
  string, ran the full suite, confirmed a genuine `FAIL`, restored.
- Baseline run at `24ddd77fc`: 57 ok / 0 FAIL.
- Swapped in `git show 162a0f51c:tools/verify_test.sh`, ran it: 57 ok / 0
  FAIL; restored; `cmp` of normalized, sorted label sets: identical.
- `grep -n "exec [0-9]>" tools/*.sh tools/lib/*.sh` — fd collision check.
- `grep -n "flock\|exec [0-9]" tools/lib/build-slot.sh tools/tidy-all-go.sh`
  — precedent and fd convention check.
- `grep -n flock tools/verify_test.sh` plus manual read of each of the
  three critical sections — acquire/release pairing and early-exit check.
- `sed -n '55,95p' tools/verify_test.sh` — `ATLAS_VERIFY_TEST_INNER` guard
  read, confirming nested invocations exit before reaching any `flock`.
- `tools/shell-guard.sh --require-shellcheck` — passed, 83 scripts OK.
- `git status --porcelain` re-checked after every experiment; clean at
  the end (only pre-existing, unrelated untracked docs).
