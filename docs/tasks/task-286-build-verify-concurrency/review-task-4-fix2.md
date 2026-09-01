# Review: task-4 fix round 2 — restore go.work.sum after broken-module probe

Commit reviewed: `4fa12ad18` (range `8410746d2..4fa12ad18`)
File: `tools/verify_test.sh` (+31/-2, only file touched — confirmed via `git show --stat`)

## Summary

The commit folds a snapshot/restore of `go.work.sum` into the pre-existing
`cleanup()`/`trap cleanup EXIT` machinery in `tools/verify_test.sh`, guarding
the broken-module probe (the one real `verify.sh --quick` invocation in the
suite) so the toolchain-appended hash lines don't survive the test run. It
also corrects the comment above the two broken-module assertions to state
accurately that only the second assertion (FAILED-and-unstripped) guards the
`.rc`-propagation regression class.

Static review plus small isolated `bash` experiments (outside the repo, no
`verify.sh`/`verify_test.sh` invocation, no docker) were used to test claims
about trap/signal behavior that cannot be judged from the diff text alone.

## Findings

### 1. Trap correctness — PASS for pass/fail, gap for hard interrupt (BLOCKING)

`tools/verify_test.sh:112-124`:
```
gowork_sum="$HERE/../go.work.sum"
gowork_sum_backup="$HERE/zz-verify-probe-broken.go.work.sum.bak"
gowork_sum_backup_absent="$HERE/zz-verify-probe-broken.go.work.sum.absent"
cleanup() {
  ...
  if [ -f "$gowork_sum_backup" ]; then
    mv -f "$gowork_sum_backup" "$gowork_sum"
  elif [ -f "$gowork_sum_backup_absent" ]; then
    rm -f "$gowork_sum" "$gowork_sum_backup_absent"
  fi
}
trap cleanup EXIT
```

For a **clean pass or a clean fail** (the broken-module `$VERIFY` invocation
returning normally, whether rc=0 or rc=1), restore happens twice: once inline
right after the invocation (`tools/verify_test.sh:274-278`), and once more,
idempotently, via the `EXIT` trap at process exit. That path is solid — this
matches the controller-confirmed clean run (42/42, `go.work.sum` unmodified
after the suite).

For **interrupt**, the commit relies on `trap cleanup EXIT` alone — no `INT`
or `TERM` disposition is registered. The commit message explicitly claims
`go.work.sum` "comes back byte-identical whether the assertions pass, fail,
**or the suite is interrupted**." That specific claim is not reliably true.

Isolated reproduction (bash 5.2.37, this host):
```
cleanup() { echo "cleanup ran" >> log; }
trap cleanup EXIT
sleep 30                     # stand-in for the long-running $VERIFY --quick child
```
- `kill -INT $PID` / `kill -TERM $PID` sent directly to the script's own PID
  while the foreground child was running: cleanup fired in every trial (fast,
  within ~1s, not waiting for the 30s child).
- `timeout 3 ./script` (the closest analogue to how the implementer's own
  disclosed episode was invoked, and to any CI-imposed job timeout): cleanup
  fired in only **~4 of 5 trials**; the remaining trial exited with the `EXIT`
  trap never running (script terminated by the deferred `SIGTERM`, no log
  line written).
- Adding `trap cleanup EXIT INT TERM` to the same script made it **8 for 8**
  reliable under the identical `timeout 3 …` reproduction.

So: `EXIT`-trap-only is not "interrupt"-safe in this bash/host combination —
it is a genuine, reproducible (not one-off) race, not a theoretical concern.
The task's required-fix description explicitly calls for coverage
"unconditionally … covering pass, fail, and interrupt," and the commit
message asserts that property outright; the code does not fully deliver it.
This matters more here than for the file's *other*, pre-existing cleanup
targets (`probe_suite`, `probe_deploy`, the probe dirs) because a residual
dirty `go.work.sum` is not a leftover file — it silently poisons the very
signal (`fanout_reason=shared-lib:go.work.sum`, `modules_selected=91`) that
plan Task 9's acceptance case measures. A `timeout`-bounded CI run of this
suite, or a developer's Ctrl-C, can leave that poison behind roughly 1 time in
5 per this reproduction.

**Blocking recommendation:** `tools/verify_test.sh:126` — change
`trap cleanup EXIT` to `trap cleanup EXIT INT TERM` (there is no earlier trap
in the file to clobber — see finding 2 — so this is additive, not a chain).
This is a one-line, low-risk fix that measurably closes the gap (0/1 → 8/8 in
the reproduction above) and directly serves the stated purpose of the task.

### 2. No existing trap clobbered — PASS

`grep -n trap tools/verify_test.sh` shows exactly one `trap` statement in the
whole file (line 126, this commit's target). No prior trap registration
exists elsewhere to be overwritten.

### 3. Absent-file case — PASS

Before the broken-module run (`tools/verify_test.sh:254-258`):
```
rm -f "$gowork_sum_backup" "$gowork_sum_backup_absent"
if [ -f "$gowork_sum" ]; then
  cp -p "$gowork_sum" "$gowork_sum_backup"
else
  : > "$gowork_sum_backup_absent"
fi
```
and both the inline restore (`276-280`) and `cleanup()` (`117-121`) check the
sentinel first (`.bak`) and fall back to the sentinel-driven remove (`.absent`
→ `rm -f "$gowork_sum" "$gowork_sum_backup_absent"`), so a not-previously-
existing `go.work.sum` is removed rather than replaced with a stray empty/
partial backup. The stale-run self-heal at start-of-file (`cleanup` is called
once, unconditionally, right after `trap cleanup EXIT` at line 127) also
covers the case where a prior crashed run left a `.bak`/`.absent` sentinel
behind, mirroring the file's existing "fixed names, not `$$`" convention
documented at `tools/verify_test.sh:97-98`.

### 4. Backup file placement / auto-discovery collision — PASS

Backup and sentinel are `$HERE/zz-verify-probe-broken.go.work.sum.bak` and
`...absent` (`tools/verify_test.sh:113-114`). Checked against the two
discovery mechanisms named in the brief:
- `tools/shell-guard.sh:46`: `find tools -name '*.sh' -type f` — neither
  filename ends in `.sh`, no match.
- `tools/verify.sh:171`: `find tools -name '*_test.sh' -type f` (auto-run
  suite discovery) — neither filename ends in `_test.sh`, no match.

No collision with either mechanism, and no new stray-`zz-verify-probe*`
litter class introduced (both files are actively created and removed within
the same cleanup path checked in finding 3).

### 5. Assertions intact — PASS

Diffed `8410746d2:tools/verify_test.sh` against the current version for the
two broken-module assertions (`assert_eq "a genuinely broken module makes the
run exit non-zero" "1" "$broken_rc"` and `assert_true "the broken module is
reported FAILED, unstripped" …`): byte-identical to the pre-fix version. Both
still run against the same real `"$VERIFY" --quick --base HEAD` invocation on
the same broken module. Neither assertion was weakened, relaxed, or removed.

### 6. Corrected comment — PASS

New comment (`tools/verify_test.sh:247-253`):
> "Of the two assertions below, only the second — the broken module still
> showing up as FAILED on the unstripped output — actually guards that
> regression class; a sabotaged `.rc` read can still leave the overall exit
> status non-zero for unrelated reasons, so the first assertion alone would
> not catch it."

This accurately narrows the claim to the FAILED/unstripped assertion, matches
the controller's teeth-proof finding (`broken_rc` staying `1` even with the
`.rc` read sabotaged), and does not overclaim that the `assert_eq` on
`broken_rc` also guards the regression.

### 7. Scope — PASS

`git show --stat 4fa12ad18` shows only `tools/verify_test.sh | 33 +++--`.
`tools/verify.sh` is untouched.

## Implementer's self-disclosed concern — assessment

The implementer ran an ad-hoc, unsupported `tools/verify_test.sh --facts`
under an external `60s` `timeout`; the script does not parse `--facts` as a
top-level flag (only internal helper functions like `facts_selected()` take
`"$@"` from their own call sites), so the invocation ran the full 42-assertion
suite regardless, got killed at the 60s mark, and left `go.work.sum` plus
stray probe files dirty, later restored by hand (controller-confirmed clean
now).

This is **not purely an artifact of a malformed probe** — the malformed flag
is a red herring (it changed nothing about execution), but the *external
60s `timeout`* landing squarely inside the long-running broken-module
`$VERIFY --quick` child is exactly the "interrupt" scenario finding 1
reproduces and shows is unreliable under `EXIT`-trap-only. The controller's
own required run (`timeout 2400 tools/verify_test.sh`, generous headroom, no
premature kill) passed clean specifically because it was never actually
interrupted mid-run — it is not evidence that the interrupt path is solid,
only that the pass/fail path is. Treat the disclosed episode as a real,
reproducible signal that finding 1's gap exists, not as noise from the
`--facts` typo.

## Not evaluable

- Behavior across other bash versions/OSes than this host's `bash 5.2.37` on
  Linux was not tested; the reproduction is host-specific evidence, not a
  portability guarantee. Reported here rather than assumed either way.

## Verdict rationale

The commit correctly satisfies the letter of "restore go.work.sum on pass and
fail," fixes the absent-file case, avoids litter/discovery collisions, keeps
both assertions intact, and corrects the comment as required. It does not
satisfy the "interrupt" leg of the stated requirement, and the commit
message's own wording overclaims that it does — a claim I found to be false
under direct, repeated reproduction using a mechanism (`timeout`) materially
similar to how the implementer's own disclosed episode failed. Given the
specific, named stakes (contaminating the Task 9 fan-out baseline), and that
the fix is a one-line, low-risk addition, this is blocking rather than a note.

---

```text
verdict: CHANGES_REQUIRED
artifact: docs/tasks/task-286-build-verify-concurrency/review-task-4-fix2.md
scope_confirmed: reviewed commit 4fa12ad18 (tools/verify_test.sh, +31/-2) against the task-4 fix-report requirements; no other files touched by the commit, none reviewed beyond it
blocking: 1
  - tools/verify_test.sh:126 — `trap cleanup EXIT` does not reliably restore go.work.sum on SIGINT/SIGTERM interrupt (reproduced ~1-in-5 failure under `timeout`-delivered SIGTERM to a script blocked on a foreground child, matching the implementer's own disclosed episode); the commit message's claim that restore covers "interrupted" runs is not accurate as implemented. Add `INT TERM` to the trap (`trap cleanup EXIT INT TERM`) — no existing trap to clobber, reproduction went 8/8 reliable with that change.
non_blocking: 0
not_evaluable: 1
```
