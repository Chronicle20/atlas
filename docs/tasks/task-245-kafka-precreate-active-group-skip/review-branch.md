# Whole-branch review — task-245-kafka-precreate-active-group-skip

Range: `24a33a2e6..1abf2e390` (5 commits: spec, design, plan, and two
implementation squash points — `6415c5ce6` state classifier, `073117395`
probe+skip, `457365ea0` WARN path, `d941053aa` broker tests, `1abf2e390` docs).

Per-task reviews (`review-task-{1..5}.md`) already exist and are all
APPROVED / APPROVED_WITH_FINDINGS with 0 blocking. This review covers only
the cross-task seam: does the classifier → probe/skip → WARN-path → test
chain work as ONE mechanism, and the three deferred minors.

## Scope

`git diff --stat 24a33a2e6..1abf2e390`:

```
deploy/k8s/base/atlas-kafka-precreate.yaml         |   22 +-  (comment only)
deploy/k8s/base/atlas-kafka-precreate_test.sh      |  159 +++ (new assertions)
deploy/k8s/base/kafka-precreate.sh                 |  198 +++ (production logic)
docs/runbooks/sparse-environments.md               |   50 +  (docs)
docs/tasks/.../{context,design,plan,prd}.md        | 2053 +  (task artifacts, not reviewed as code)
```

`scope_confirmed`: the diff matches the brief exactly — a `state_is_seedable`
classifier, a `group_state` probe wired into `seed_override_offsets`, a
skip-aware WARN path in `verify_group_offsets`, broker-backed tests
asserting the new contract, and matching docs. No surprise files, no drift.

## Seam 1 — `$skipped_groups` handoff (`seed_override_offsets` → `verify_group_offsets`)

`kafka-precreate.sh:295` creates `skipped_groups="$(mktemp)"` unconditionally
on every `seed_override_offsets` call that reaches past the
`KAFKA_CONSUMER_GROUP` unset guard — so the file exists (possibly empty) by
the time `verify_group_offsets` runs it reads at `kafka-precreate.sh:368`:

```
if [ -n "${skipped_groups:-}" ] && [ -f "${skipped_groups:-}" ] && grep -Fxq -- "$group" "$skipped_groups"; then
```

- **Unset case**: if `KAFKA_CONSUMER_GROUP` is unset, both functions return
  at their own identical early guard (`kafka-precreate.sh:281`,
  `:355`) before `skipped_groups` is ever touched — `verify_group_offsets`
  never reads the unset variable. Consistent, no bug.
- **Empty-file case**: a run where nothing is skipped leaves `skipped_groups`
  pointing at a zero-byte file; `grep -Fxq` correctly finds no match for any
  group, `group_skipped` stays `0`, every group takes the pre-existing
  hard-fail path (`:392-395`). Verified by inspection, matches design intent.
- **Exact-match safety**: `grep -Fxq --` is fixed-string (`-F`) and
  whole-line (`-x`), so a group name containing brackets/spaces/dashes (the
  documented real shape, e.g. `"Account Service [pr-123]"`) cannot false-match
  a substring of another group's name and the leading-dash case is protected
  by `--`. Correct.
- **"Skipped for one topic, seeded for another" — cannot happen.**
  `state_is_seedable` is evaluated once per **group** (`:301-302`), and on
  the seedable branch `seed_group` is called exactly once with the **full**
  `$all_topics` union in a single invocation (`:305`, `cat "$all_topics"` —
  `topics` + `compact_topics`, same union `verify_group_offsets` iterates at
  `:363`). A group is therefore classified atomically for every topic in the
  run — there is no code path that seeds a subset and skips a subset of the
  same group in the same pass. The reviewer prompt's named risk does not
  exist in this design; confirmed by reading both loops, not assumed.
- **Cleanup**: none of `groups_file` / `all_topics` / `skipped_groups` /
  `topics` / `compact_topics` are ever `rm`'d. Pre-existing for
  `topics`/`compact_topics` (task-232); `skipped_groups` is new but the same
  convention. Non-blocking — the Job's pod filesystem is torn down with the
  pod, and mktemp files are individually small. Noted, not filed.

## Seam 2 — `seed_group` return-code contract (0 / 2 / other) against every call site

`grep -rn 'seed_group '` in the diff shows exactly two call sites: the
sourced production call in `seed_override_offsets` (`:305`) and the four
broker-backed test invocations in `atlas-kafka-precreate_test.sh`
(`:74, :99, :134, :177`). All five sites use the mandated
`rc=0; seed_group … || rc=$?` pattern; none rely on `$?` after any
intervening command. Traced:

- **rc 0 (seeded)**: `seed_rc -eq 0` → `seeded_count++` (`:306-307`). Test at
  `atlas-kafka-precreate_test.sh:74-83` and `:99-109` confirm a committed
  offset actually lands.
- **rc 2 (active refusal)**: `seed_rc -eq 2` → appended to `$skipped_groups`,
  `skipped_count++`, non-fatal (`:308-314`). Test at `:176-189` confirms rc 2
  specifically (not "any nonzero") and confirms the committed offset is
  unchanged (FR-5.2).
- **rc other (fatal)**: falls to the `else` branch, `exit "$seed_rc"`
  (`:315-317`) — this is a **script-level `exit`**, not a function `return`,
  so it terminates the whole Job process immediately; `verify_group_offsets`
  is never reached for this run at all. Confirmed by reading the call site,
  not inferred.
- **Global `seed_rc` reuse without `local`** (deferred triage item 1, see
  below): traced all three return paths in `seed_group`
  (`kafka-precreate.sh:210-236`) against the caller's read of `$?` — the
  caller's `seed_rc=$?` reads the function's actual `return` status (bash
  guarantees this regardless of what a same-named global variable holds
  internally), not the internal `$seed_rc` variable by name. Concretely: on
  the message-match branch (`:229`, `return 2`), the internal `seed_rc`
  variable set at `:225` from the command-substitution's `$?` may still hold
  `0` (a refused reset exits 0 per the design's measured behaviour) — but the
  caller's `$?` after the function call is `2` regardless, because that is
  what `return 2` sets. No observable bug. Confirms the Task 2 reviewer's
  finding independently.

## Seam 3 — genuinely broken group still fails the Job (highest-risk regression)

This is the one the WARN path could plausibly swallow. Traced every path
that reaches a broken/unseedable-for-a-real-reason group:

1. **`seed_group` itself fails for a non-active reason** (broker unreachable,
   auth failure, malformed topic/group argument): `seed_out` does not match
   `"Assignments can only be reset if the group"`, `seed_rc -ne 0` is true,
   function `return`s the real nonzero code (`kafka-precreate.sh:231-234`).
   Caller's `elif` for `2` is false, falls to `else: exit "$seed_rc"`
   (`:315-317`) — **kills the whole script**, `verify_group_offsets` never
   runs. A broken group cannot reach the WARN path through this route.
2. **`group_state` probe itself fails** (broker unreachable during the probe):
   collapses to `""` by design (`:137-141`), which `state_is_seedable`
   allowlists as seedable (`:166`) — the probe deliberately never fails the
   Job on its own, deferring to `seed_group`'s own attempt, which is route 1
   above (a broker that's down for the probe is down for the reset too, and
   that failure is fatal per route 1).
3. **A group genuinely has a missing committed offset AFTER a successful, real
   seed attempt (not a skip)**: `group_skipped` is `0` for any group not
   written to `$skipped_groups`, i.e. every group that took the `seeded_count`
   branch. `verify_group_offsets:392-395` preserves the exact pre-existing
   `FAIL: ... ; exit 1` statement for `group_skipped -eq 0`, now merely gated
   by the new conditional (diff confirms this is the *unmodified* line,
   moved under an `if`, not rewritten). Behaviourally identical to
   pre-task-245 for every group not in `$skipped_groups`.

Conclusion: the WARN path is reachable **only** for a group this same script
run itself put into `$skipped_groups`, which happens **only** on the
allowlisted-seedable-state skip or the rc-2 "became active mid-run" skip —
never on a fatal `seed_group` failure or a broker-probe failure. A genuinely
broken group still fails the Job via `exit 1` (unseeded, non-skipped) or
`exit "$seed_rc"` (seed attempt itself failed). No swallow found.

**Coverage gap (non-blocking, pre-existing shape)**: the broker-backed test
suite does not exercise route 3 (a real `FAIL: ... exit 1` for a
non-skipped, non-seeded group) — confirmed in
`.superpowers/sdd/plan/task-4-report.md:140` ("the still-unhandled-gap case
... not triggered here"). This was equally untested before task-245 (the
`exit 1` statement is unchanged, not new), so it is not a regression this
branch introduces, but it does mean the highest-risk path is verified here by
static trace only, not by a broker assertion. Worth a future test (stand up a
group missing an offset that was never in `$skipped_groups`), not blocking
this PR since the code path itself is provably unmodified.

## Seam 4 — awk field-parsing robustness

`group_state`'s `awk 'NF>=6 && $(NF-1)!="STATE" { print $(NF-1); exit }'`
(`kafka-precreate.sh:148`) is checked against `design.md:24-55`'s measured
`--describe --state` output (blank line, header, one data row, group name
containing spaces) — the NF-anchored formula matches the measured shape
exactly, and the broker test confirms it end-to-end: `group_state` correctly
reports `Stable` for a live single-member group
(`atlas-kafka-precreate_test.sh:157-166`, passed per
`task-4-report.md:99`). `verify_group_offsets`'s pre-existing
`NF>=9 && $(NF-7)==t {print $(NF-5)}` (unchanged by this branch) is reused
consistently. No parsing defect found; this is measured against real
`apache/kafka:3.7.2` output, not assumed.

## Deferred-minor triage

1. **`seed_rc` shared global, no `local` (`kafka-precreate.sh:225,303-305`)**
   — **No fix.** Independently retraced all three `seed_group` return paths
   against the caller's `$?` read (Seam 2 above); the caller never reads the
   internal `$seed_rc` variable by name, only the function's actual return
   status via `$?`, which bash sets correctly regardless of the internal
   global collision. The file's own dual-shell (`sh`-sourceable /
   `bash`-executed) constraint is a real reason to avoid `local` here (POSIX
   `sh` doesn't guarantee it; the test harness is `#!/usr/bin/env sh` and
   sources this file). Confirmed no bug — same conclusion as the Task 2
   reviewer, reached independently. File as accepted risk, not a follow-up.

2. **`$$`-suffixed test topic names collide on a second `docker run` against
   a persistent broker** — **No fix on this branch.** Confirmed the
   collision is on `atlas-precreate-test-1` / `-a-1` / `-b-1`, names
   generated by the pre-existing (task-232) first three assertions, not by
   any topic name this branch introduces (`TOPIC_C/D/E`, `GROUP3` are new but
   collide under the identical mechanism only because the whole file shares
   one `$$`). This is a long-standing property of running the same
   containerized test script twice against one persistent broker with a
   stable PID 1, not something task-245 changed or worsened. A genuine fix
   (e.g., broker-timestamp or random suffix) is a task-232-scoped
   maintenance item, not part of this brief. File as a follow-up if desired;
   does not block this PR.

3. **Runbook WARN-line table missing the `<=10` no-suffix row
   (`docs/runbooks/sparse-environments.md`, table after line ~170;
   code at `kafka-precreate.sh:415`)** — **Fix on this branch.** This is a
   one-line, zero-risk documentation completeness gap: the table already
   shows the `>10` variant with `(+<M> more)` but omits the sibling `<=10`
   variant with no suffix, which is the MORE common real-world shape (a
   handful of newly-added topics, not >10). It's producible immediately (one
   markdown table row, no design decision, no ambiguity) and CLAUDE.md's
   "finish producible work" standard applies — this is not a case requiring a
   new task or an out-of-scope call, it's a same-file completion of the row
   already being added in this branch's own docs commit
   (`1abf2e390`). Recommend adding before merge, not as a follow-up.

## Verdict rationale

No blocking findings on the cross-task seam. The classifier → probe/skip →
WARN mechanism is coherent: the highest-risk regression (a genuinely broken
group swallowed by the new WARN path) does not occur, traced through every
return code and every guard. The one substantive gap — broker-test coverage
of the still-unmodified hard-fail path — is pre-existing and not a
regression. The one merge-worthy but non-blocking item is the runbook table
row (item 3), which is trivial and safe to fix on this branch, but its
absence does not misrepresent behaviour badly enough to block — the WARN
message itself is self-describing at runtime either way.

## Not evaluable

- The broker-backed test tier (`atlas-kafka-precreate_test.sh` broker
  assertions) was not re-run in this session — no `kubectl -n kafka
  port-forward` was established, per the controller's explicit instruction.
  Relied on the recorded evidence in
  `.superpowers/sdd/plan/task-4-report.md` (full PASS transcript, cleanup
  confirmed) rather than re-executing.
