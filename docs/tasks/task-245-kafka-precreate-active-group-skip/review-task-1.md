# Review — Task 1: `state_is_seedable` classifier and no-broker truth table

Commit range: `f50f1893f..6415c5ce6` (single commit `6415c5ce6`)
Brief: `.superpowers/sdd/plan/task-1-brief.md`
Report: `.superpowers/sdd/plan/task-1-report.md`

## Scope

`git diff --stat f50f1893f..6415c5ce6`:

```
deploy/k8s/base/atlas-kafka-precreate_test.sh | 21 +++++++++++++++++++++
deploy/k8s/base/kafka-precreate.sh            | 20 ++++++++++++++++++++
2 files changed, 41 insertions(+)
```

Both files are exactly the two named in the brief. No unrelated files touched.
Diff matches the brief's Step 1 / Step 3 code blocks byte-for-byte (verified
by direct comparison of `git diff` hunks against the brief's fenced code
blocks).

## Findings

### 1. Requirement coverage — PASS

`deploy/k8s/base/kafka-precreate.sh:113-137` (new `state_is_seedable`, inserted
directly above the `# Commit end-of-log offsets…`/`seed_group` comment block,
matching the brief's placement instruction) implements the allowlist exactly:
`Empty|Dead|"") return 0 ;; *) return 1 ;;`.

Truth table in `deploy/k8s/base/atlas-kafka-precreate_test.sh:42-64` covers
all 8 rows enumerated in the brief:

| brief row | covered | file:line |
|---|---|---|
| `Empty` seedable | yes | atlas-kafka-precreate_test.sh:47 (loop `for state in Empty Dead ""`) |
| `Dead` seedable | yes | same loop |
| `""` seedable | yes | same loop |
| `Stable` active | yes | atlas-kafka-precreate_test.sh:53 (loop `for state in Stable PreparingRebalance CompletingRebalance SomeNewState STATE`) |
| `PreparingRebalance` active | yes | same loop |
| `CompletingRebalance` active | yes | same loop |
| `SomeNewState` active (future-state-falls-to-active) | yes | same loop |
| `STATE` (header token never seedable) | yes | same loop |

All 8 cases present, none omitted, none duplicated.

### 2. Assertions actually assert — PASS

Each loop iteration calls `state_is_seedable "$state"` and branches on its
real exit code (`if ! state_is_seedable "$state"; then … exit 1; fi` for the
seedable loop, `if state_is_seedable "$state"; then … exit 1; fi` for the
active loop) — not a mock, not a string compare on an unrelated value. A
regression that flipped the `case` arms, or that made `state_is_seedable`
always `return 0`, would fail the active-loop branch (`Stable` etc.) and the
script would print `FAIL: state_is_seedable accepted active state 'Stable'`
and exit 1. Confirmed by manually flipping the implementation to `return 0`
unconditionally is unnecessary here — the branch structure is inspectable and
unambiguous; the PASS echo is reached only after both loops complete without
hitting an `exit 1`, so it cannot print PASS while silently skipping a check.

### 3. Purity of `state_is_seedable` — PASS

`kafka-precreate.sh:132-137`: single `case "$1"` on the positional parameter,
two `return` statements. No Kafka CLI invocation, no `$KAFKA_BIN` reference,
no file I/O, no read or write of a global variable. Confirmed by direct
inspection of the 6-line function body — nothing else is possible to invoke
in that span.

### 4. Dual-use portability (global constraint) — PASS

No arrays, no `local`, no `[[ ]]`, no `+=`, no `$'...'` anywhere in the new
code. `case`/`return` only in `kafka-precreate.sh`; `for`/`if`/`case`/`exit`
only in the test file — all POSIX `sh` constructs. Confirmed:

```
$ bash -n deploy/k8s/base/kafka-precreate.sh   # OK
$ sh -n deploy/k8s/base/kafka-precreate.sh     # OK
$ bash -n deploy/k8s/base/atlas-kafka-precreate_test.sh  # OK
$ sh -n deploy/k8s/base/atlas-kafka-precreate_test.sh    # OK
```

### 5. `$KAFKA_BIN` full-path constraint — N/A, correctly so

`state_is_seedable` makes no Kafka CLI invocation, so the "every Kafka CLI
call uses `$KAFKA_BIN`" constraint does not apply to this task's new code.
No violation introduced.

### 6. NG6 early-return byte-identical / stays first — PASS (unchanged)

This task did not touch `seed_override_offsets` or `verify_group_offsets`.
`git diff` confirms zero hunks against those functions in this commit; the
NG6 guard block in the test file (`atlas-kafka-precreate_test.sh:27-40`) is
untouched apart from the new block being appended *after* it, as the brief
required ("between the NG6 PASS echo … and the BOOTSTRAP_SERVERS guard").

### 7. Baseline stays green — PASS, reproduced directly

```
$ shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh
(silent, exit 0)

$ sh deploy/k8s/base/atlas-kafka-precreate_test.sh
PASS: seed_override_offsets skips when KAFKA_CONSUMER_GROUP is unset (NG6)
PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state
SKIP: BOOTSTRAP_SERVERS unset
EXIT:0
```

Matches the brief's Step 4 expected output exactly, in order, exit 0.

### 8. Placement / ordering — PASS

New function sits immediately above the `seed_group` comment block in
`kafka-precreate.sh` (confirmed via diff hunk context `@@ -113,6 +113,26 @@
precreate_topics() {`, i.e. inserted right after `precreate_topics` ends and
before the `seed_group` comment). New test block sits between the NG6 PASS
echo and the `BOOTSTRAP_SERVERS` guard (diff hunk context `@@ -39,6 +39,27 @@
case "$skip_out" in`, i.e. right after the NG6 `case` block's closing and
before the guard line), exactly as the brief specified.

## Not evaluable

None. The full surface of this task (a pure function plus a no-broker test)
was reviewable within the diff itself; no Kafka broker interaction, no
cross-service seam, and no dependency on code this unit didn't touch.

## Verdict

All 8 truth-table rows present and each genuinely exercises the classifier's
real return code; the function is pure and portable per the dual-use
constraint; baseline stays green (shellcheck, `bash -n`/`sh -n` on both
shells, and a live `sh` run of the test script all reproduced clean). No
scope creep — `seed_group`/`seed_override_offsets`/`verify_group_offsets`
untouched, correctly deferred to later plan tasks.
