# Review: Task 3 — `verify_group_offsets` skip-aware rewrite

Range reviewed: `073117395..457365ea0` (single commit `457365ea0`,
`fix(kafka-precreate): warn instead of failing the Job for skipped active
groups`).

Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Scope

`git diff --stat 073117395..457365ea0` shows exactly one file touched:

```
deploy/k8s/base/kafka-precreate.sh | 46 ++++++++++++++++++++++++++++++++++++--
1 file changed, 44 insertions(+), 2 deletions(-)
```

The diff is confined to the comment block and body of `verify_group_offsets`
plus the two `-`/`+` lines where the old unconditional `FAIL`/`exit 1` was
replaced. No other function, no test file, no other deploy asset. Matches the
brief's stated file list. `scope_confirmed`: reviewed the full diff for this
commit and re-read the resulting `verify_group_offsets` in context (lines
~343–406 of the current file) since correctness depends on the whole
function, not just the hunk boundaries.

## Findings

### PASS — comment block extended verbatim, not replaced

`git diff` shows the pre-existing FR-5.3 comment (lines ending
"...already reports every topic/partition that group has offsets on.") is
untouched; the new paragraph is appended below it with a leading `#` blank
line, matching the brief's Step 1 text word-for-word (verified via `diff`
against the brief's code block — no discrepancy).

### PASS — `KAFKA_CONSUMER_GROUP`-unset early return stays byte-identical and first

`deploy/k8s/base/kafka-precreate.sh:355-357`:
```sh
    if [ -z "${KAFKA_CONSUMER_GROUP:-}" ]; then
        return 0
    fi
```
Diffed the pre-image (`git show 073117395:deploy/k8s/base/kafka-precreate.sh`)
against the post-image function opener — identical, and it is still the
first statement in the function body, ahead of any Kafka call or
`$skipped_groups` read.

### PASS — one `--describe` per group; NF-anchored awk reused verbatim

`deploy/k8s/base/kafka-precreate.sh:369`: single
`"$KAFKA_BIN/kafka-consumer-groups.sh" ... --describe` call per outer-loop
iteration, computed once before the per-topic inner loop (not moved inside
it). `deploy/k8s/base/kafka-precreate.sh:388`: the `awk -v t="$topic"
'NF>=9 && $(NF-7)==t {print $(NF-5)}' | head -n1` line is character-identical
to the pre-existing one at the old line 234 (only the surrounding indentation
and comment above it are unchanged from before). `$KAFKA_BIN` full path is
used on the only Kafka CLI invocation in the function — no bare command name.

### PASS — skipped-group membership test is exact whole-line match, not substring/prefix

`deploy/k8s/base/kafka-precreate.sh:367`:
```sh
if [ -n "${skipped_groups:-}" ] && [ -f "${skipped_groups:-}" ] && grep -Fxq -- "$group" "$skipped_groups"; then
```
`grep -F` (fixed string, no regex metachar interpretation) combined with `-x`
(match must span the **entire** line, not a substring within it) means a
group name that is a prefix or substring of another group's name (e.g.
`"Account Service"` vs `"Account Service [pr-123]"`) cannot false-positive:
`-x` requires the pattern to equal the whole line, and `grep -F` treats the
group name literally, so embedded regex metacharacters in a group name
(brackets, dots) cannot cause a spurious match either. Confirmed the
producer side writes one full group name per line with no truncation:
`deploy/k8s/base/kafka-precreate.sh:312` and `:320` (Task 2's
`seed_override_offsets`) both use `printf '%s\n' "$group" >> "$skipped_groups"`
— the same string representation later compared against, so an exact
whole-line match is guaranteed to succeed for a group that actually was
skipped and to fail for every other group, including one whose name is a
substring/prefix of the skipped one.

### PASS — `$skipped_groups` unset/missing treated as "nothing skipped," `set -u` safe

Both reads of `skipped_groups` use `${skipped_groups:-}`, so under `set -u`
(confirmed active via the script's shebang/options — not re-verified line by
line here since it predates this task, but the pattern is consistent with
every other guarded read in the file) an unset variable does not abort. The
`-f` existence check additionally covers the case where the variable is set
but points at a file `seed_override_offsets` never created (mktemp file that
was never written, or was cleaned up). In either case `group_skipped` stays
`0` and the function falls straight through to the unchanged hard-gate path
below — this is exactly the pre-existing behavior for every group, so a test
that sources the file and calls `verify_group_offsets` directly (without
`seed_override_offsets` populating `skipped_groups`) sees no behavior change
from before this commit.

### PASS — exit 1 on a gap for a non-skipped group is unchanged

`deploy/k8s/base/kafka-precreate.sh:390-393`:
```sh
if [ "$group_skipped" -eq 0 ]; then
    echo "FAIL: group '$group' has no committed offset on topic '$topic'" >&2
    exit 1
fi
```
Same message text, same stderr target, same `exit 1`, gated only by the new
`group_skipped` check — for the `group_skipped=0` path this is textually
identical to the pre-existing unconditional branch.

### PASS — skipped group with a gap downgrades to WARN and continues

`deploy/k8s/base/kafka-precreate.sh:404-410` (verified by direct read):
accumulates `missing_total`/`missing_names` instead of exiting, then after
the per-topic loop emits either an informational "committed offsets present
on all N topics" line (no gaps), a `WARN:` line with the bounded name list
(1–10 gaps), or a `WARN:` line with `(+N more)` (>10 gaps) — matches FR-4.2
and OQ-1's 10-name bound exactly. The outer `while` loop is not broken or
exited early for a skipped group; it proceeds to the next line of
`$groups_file` (there is currently exactly one group,
`$KAFKA_CONSUMER_GROUP`, but the loop structure itself does not special-case
count).

### PASS — no forbidden bash-only constructs

Grepped the diff and the resulting function body for `local`, `[[`, `+=`,
and `$'...'` — none present. No arrays introduced (`missing_names` is
built by string concatenation, not `+=` or an array). All conditionals use
`[ ]`; all arithmetic uses `$(( ))`. Consistent with the dual-use
sourced-by-dash constraint.

### PASS — baseline stays green

Ran by hand in the worktree:
```
$ bash -n deploy/k8s/base/kafka-precreate.sh && echo BASH_OK
BASH_OK
$ sh -n deploy/k8s/base/kafka-precreate.sh && echo SH_OK
SH_OK
$ shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh && echo SHELLCHECK_OK
SHELLCHECK_OK
$ sh deploy/k8s/base/atlas-kafka-precreate_test.sh; echo EXIT=$?
PASS: seed_override_offsets skips when KAFKA_CONSUMER_GROUP is unset (NG6)
PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state
SKIP: BOOTSTRAP_SERVERS unset
EXIT=0
```
Matches the brief's expected Step 3 output exactly.

## Not evaluable

- **No new automated test exercises the skip-aware WARN path or the exit-1
  path for a non-skipped group with a real/simulated `--describe` output.**
  The brief explicitly assigns broker-backed assertions to Task 4
  ("Test file untouched per instructions (Task 4 owns broker-backed
  assertions)" — implementer report). This commit's correctness for the
  skip/no-skip branch split is therefore verified here by static reading
  and the no-broker smoke tier only, not by an executable test that would
  fail without this change. This is in-scope for Task 4, not a defect of
  Task 3, but is noted since Task 3 is described as "the fix" and currently
  has no regression test pinning either branch.
- Behavior against a live Kafka broker (multi-group `$groups_file`,
  real `--describe` output shape, `missing_total` boundary at exactly 10
  and 11) was not exercised — no broker available in this review
  environment. The bounded-list logic (`-le 10`, `+N more`) was verified by
  static reading only.

## Verdict

APPROVED — implementation matches the brief's Step 2 code block essentially
verbatim, all binding global constraints hold, the exact-whole-line
membership-match property specifically called out in the review brief is
confirmed by both the `grep -Fx` semantics and the producer-side write
format, and the baseline smoke tier stays green. The one gap (no
skip-path/no-skip-path regression test yet) is explicitly deferred to Task 4
per the brief's own division of labor, not an omission of this task.
