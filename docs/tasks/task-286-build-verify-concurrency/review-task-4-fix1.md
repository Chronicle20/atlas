# Review — task-286 Task 4, fix round 1 (commit `55bfc577d`)

## Scope

Commit `55bfc577d` only (`90b0c2a05..55bfc577d`), judged solely against the one
blocking finding from `review-task-4.md`:

> none of the 5 pool assertions in `tools/verify_test.sh` drove a FAILING module
> through the Go worker pool, and the suite's `real_selected()` helper strips
> the ✓/✗ glyph before comparing — so a regression that turned a FAILED module
> into a reported PASS would pass the entire suite undetected.

No re-litigation of the pool implementation in `tools/verify.sh` — the
controller ruled that logic correct and out of scope for the fixer.

## What the commit does

`git show --stat 55bfc577d`: `tools/verify_test.sh | 31 +++++++++++++++++++++++++++++++` —
one file touched, 31 insertions, 0 deletions. **`tools/verify.sh` was not
touched** (confirmed both by `git show --stat` and by the report's own
`git status --short` note). This satisfies the "must not touch the pool
implementation" constraint.

The diff adds:
- `probe_broken_dir="$HERE/../services/zz-verify-probe-broken"` and a matching
  `rm -rf "$probe_broken_dir"` in `cleanup()` (`tools/verify_test.sh:105,110`).
- A new case (`tools/verify_test.sh:232-259`) that creates a real Go module
  under `services/zz-verify-probe-broken` with a syntactically invalid
  `main.go`, runs `"$VERIFY" --quick --base HEAD` for real (not `--facts`),
  and asserts two things against the **unstripped** raw output
  (`broken_out`, not passed through `real_selected()`):
  1. `assert_eq "a genuinely broken module makes the run exit non-zero" "1" "$broken_rc"`
  2. `assert_true "the broken module is reported FAILED, unstripped" \
     "$(printf '%s\n' "$broken_out" | grep 'FAILED' | grep 'zz-verify-probe-broken' ...)"`

Using the raw `broken_out` rather than `real_selected()` output is exactly the
right move — `real_selected()`'s glyph-stripping was precisely the blind spot
identified in round 1, and grepping for the literal `FAILED` label (which only
appears unstripped in `step()`'s output for a failing module, per the pool
implementation reviewed in round 1) directly targets the gap.

## Teeth verification (controller-supplied logs, not re-run)

`.superpowers/sdd/plan/logs/task4-fix-tests-clean.log`: clean run against
restored `verify.sh`, 42 assertions, both new assertions `ok`, suite reports
`verify_test.sh: all assertions passed`, `verify_test_rc=0`.

`.superpowers/sdd/plan/logs/task4-teeth-proof.log`: sabotage injects
`return 0` immediately before the `.rc`-file read in `replay_go_layer()`
(neutering per-worker failure propagation — precisely the regression class
round 1 worried about). Under sabotage:
- `ok   - a genuinely broken module makes the run exit non-zero` — **still
  passes** even with propagation disabled.
- `FAIL - the broken module is reported FAILED, unstripped (got '')` — this
  is the one assertion that actually catches the injected regression.
- Restore confirmed clean (`RESTORED: verify.sh diff=[]`).

So of the two new assertions, exactly one — "the broken module is reported
FAILED, unstripped" — has teeth against the specific regression class named
in the round-1 finding. That is sufficient: the round-1 finding was that
*no* assertion would catch a FAILED→reported-PASS regression; now one does,
directly and unambiguously (`FAIL` output confirms it fires, and the label
match is on the unstripped stream, closing the `real_selected()` blind spot).
**The finding is closed.**

## Non-blocking: comment overclaims what the exit-code assertion catches

`tools/verify_test.sh:232-236`:

```
# ... a regression in the pool's per-worker .rc propagation that silently
# turned a FAILED module into a reported PASS would not show up in the
# label-agreement assertions above — only in the module's own build failing
# and the overall run's exit status going non-zero. Assert both, on the
# unstripped output.
```

This reads as claiming both new assertions are independently diagnostic of
the propagation regression — i.e., that if `.rc` propagation broke, the run's
exit status would (correctly) flip from non-zero to zero, and the second
assertion would (correctly) flip from pass to fail, giving two independent
tripwires. The teeth-proof log contradicts the first half: under sabotage,
`broken_rc` stayed `1` and `assert_eq "... exit non-zero" "1" "$broken_rc"`
kept passing — the run remains non-zero for a reason unrelated to the
sabotaged `.rc` read (most likely some other gate in the same `--quick` run,
triggered by the same untracked probe module, still failing on its own
merits). That assertion is therefore not independently load-bearing for the
propagation-regression class the comment describes it as covering, and a
future reader relying on the comment could wrongly believe removing/weakening
the "reported FAILED, unstripped" assertion would still be caught by the
exit-code check. It would not, per the log evidence above.

This does not block closure of the round-1 finding (the other assertion
carries it alone, proven), but the comment should be corrected to say only
the "reported FAILED, unstripped" assertion is proven load-bearing against
`.rc`-propagation regressions, and that the exit-code assertion is a
sanity check on the broken-module fixture, not a second independent guard.

## Not evaluable

- Whether `broken_rc` is reliably `1` (vs. some other non-zero value) across
  environments was not independently re-run per the read-only/no-suite-run
  constraint; relying entirely on the supplied clean-run log, which shows it
  passing.

## Verdict rationale

The single blocking finding from round 1 is closed: the suite now drives a
genuinely FAILED module through `launch_go_layers`/`replay_go_layer` and
asserts, on unstripped output, that it is reported FAILED — proven to fail
under the exact sabotage the finding described, and to pass clean. The fixer
did not touch `tools/verify.sh`. The one documentation inaccuracy above
(comment overclaiming dual coverage) is a note for a future reader, not a
correctness defect in the assertion itself, so it is non-blocking.
