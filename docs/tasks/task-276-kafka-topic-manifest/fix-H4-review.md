# Review: fix-H4 — report go.work drift under `--facts` instead of aborting it

Commit under review: `84d3d597e`
Brief: `.superpowers/sdd/plan/fix-H4-brief.md`
Report: `.superpowers/sdd/plan/fix-H4-report.md`
Diagnosis: `docs/tasks/task-276-kafka-topic-manifest/h4-diagnosis.md`

## Scope confirmed

Exactly the three files the brief named: `tools/lib/go-work.sh`,
`tools/verify.sh`, `tools/verify_test.sh` (`git show --stat 84d3d597e`). No
other file touched. Matches the brief's stated scope.

## Findings

### 1. Fatal path on a real run — PASS

`tools/lib/go-work.sh:83-101` (`check_workspace_drift()`): the stderr
`ERROR —` message block (three `echo`/`printf` lines, byte-identical text)
and the `return 1` contract on drift are unchanged. The only addition is
`printf '%s\n' "$dropped"` (line 99) inserted *before* `return 1`, i.e. an
addition to stdout, not a change to the fatal contract.

`tools/verify.sh:258-278` (`all_modules()`): drift is captured via
`drift="$(check_workspace_drift ...)" || drift_rc=$?` (line 272), then:
```
if [ "$drift_rc" -ne 0 ]; then
    if [ "$FACTS" -eq 1 ]; then
        printf '%s\n' "$drift" >> "$WORKSPACE_DRIFT_FILE"
    else
        exit 1
    fi
fi
```
On a real run (`FACTS=0`), the `else` branch runs `exit 1` unconditionally on
drift — same effective behavior as the pre-fix `check_workspace_drift ... ||
exit 1` one-liner. Confirmed by the implementer's manual verification quoted
in the report (real `--quick` run with a probe present: `EXIT=1`, stderr
names `.../services/atlas-ban/zz-verify-probe-repro`).

`_changed_modules_out="$(changed_modules | sort -u)" || exit 1` at
`tools/verify.sh:460` is still unconditional at script top level, unchanged
from before this fix — the brief's "rejected alternative" (deferring this
call) was correctly *not* implemented.

### 2. No exemptions — PASS

Grep for `zz-verify-probe` and `probe` inside `go-work.sh` and the non-test
parts of `verify.sh`: none. `check_workspace_drift()`'s comparison logic
(`comm -23` between `found` and `workspace`, `tools/lib/go-work.sh:92-93`) is
untouched — no path-based carve-out was added anywhere. The only new
conditional is on `$FACTS` in the *caller* (`verify.sh`), not on the path
being checked. Matches the brief's "the check must stay general."

### 3. The three bake-target assertions — PASS, unchanged

`git diff 84d3d597e~1 84d3d597e -- tools/verify_test.sh | grep '^-'` shows
exactly one removed line: the `for k in ...` fact-key list
(`fanout_reason modules_selected guard_suites ...`), replaced by the same
list with `workspace_drift` inserted. No line inside the bake-target block
(`tools/verify_test.sh:356-374`, formerly lines 355-374) was touched — the
new drift-assertion block is appended *after* that block's own
`flock -u 8` (confirmed at `tools/verify_test.sh:377` onward). The
broken-probe assertion (`grep -F "zz-verify-probe-broken-${probe_tag}"`,
line 493) is likewise absent from the diff.

### 4. Fact block contract — PASS

`workspace_drift=$(printf '%s\n' "$workspace_drift_rel" | sed '/^$/d' |
paste -sd, -)` (`tools/verify.sh:1028`). Verified `paste -sd, -` behavior
directly:
```
$ printf 'a\nb\n' | sed '/^$/d' | paste -sd, -
a,b
$ printf '' | sed '/^$/d' | paste -sd, -
<empty>
```
Multiple dropped modules join with `,` (not a newline or space), and zero
drift produces an empty string — both are single `key=value` lines, matching
`verify_test.sh`'s `grep "^${k}="` assertion (line 517) and the new
`grep '^workspace_drift=$'` no-drift assertion (line 397). No `csv()` helper
reuse here (that helper defaults empty to `"none"`); the brief explicitly
asked for an empty value, and the implementation matches it literally.

Dedup: `WORKSPACE_DRIFT_FILE` can receive multiple appends because
`all_modules()` is called from more than one call site under `--facts`
(`tools/verify.sh:406` via `changed_modules()`, and potentially again via
`libs_changed_module_dirs()` at line 312 when the FACTS block computes
`fanout_reason` for the shared-lib-closure case, line ~1011). The read side
(`sort -u "$WORKSPACE_DRIFT_FILE"`, line 990) absorbs any duplicate lines
from repeat calls, so a double-append cannot double-list the same module or
break the single-line contract.

### 5. `modules_selected` in the drift case — PASS

After the `--facts` branch appends to `WORKSPACE_DRIFT_FILE` and does *not*
exit, `all_modules()` falls through to `result="$(comm -12 ...)"`
(`tools/verify.sh:279`) — the normal intersection against `workspace`,
excluding the drifted (non-workspace) module(s) exactly as it would if there
were no drift. `MODULES` is populated from that at the top-level
`changed_modules()` call, so `modules_selected=${#MODULES[@]}` (line 1022)
reflects the real, non-drifted module count rather than an empty or
truncated value. Implementer's manual repro confirms
`modules_selected=0` alongside `workspace_drift=services/atlas-ban/...` in
the reproduced no-real-modules-changed-but-drift-present case, consistent
with this code path.

### Side-effect on `lint.sh` / `analyzer-guard.sh` callers — checked, no regression

`check_workspace_drift()` is shared (`tools/lint.sh:178`,
`tools/lib/analyzer-guard.sh:170`); both call it directly with
`|| exit 1` / `|| return 1`, not via a nested command substitution of their
own. The new stdout `printf` therefore becomes part of whatever those
scripts' own enclosing `"$(...)"` capture holds (e.g. `tools/lint.sh:205`,
`modules="$(discover_modules)" || exit 1`), but the `|| exit 1` right after
that capture means the captured value is never read — the process exits
immediately on drift, identically to before this fix. Neither file is part
of the diff; this is confirmed only to rule out an unintended regression
from the shared library change, not evaluated further (out of scope).

## Not evaluable

- `tools/verify_test.sh`'s own execution (the new drift block, the
  unchanged bake-target block, and the full 500+-line suite) was not run —
  explicitly withheld per the task instructions ("the controller is running
  `tools/verify.sh` right now... do not run `verify_test.sh`"). All findings
  above are from reading the diff and the implementer's quoted manual-run
  output, not from an independent execution.

## Verdict rationale

All five weighted concerns pass with direct `file:line` evidence, no
exemption was introduced, the fatal path is intact, the three regression-proof
bake-target assertions are untouched, and the fact-block contract holds in
both the drift and no-drift cases. No blocking findings.
