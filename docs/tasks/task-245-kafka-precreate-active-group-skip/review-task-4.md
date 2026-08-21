# Task 4 review — broker-backed assertions: active-group skip and idempotence

Commit range: `457365ea0..d941053aa` (single commit `d941053aa`).

## Scope

`git diff --stat 457365ea0..d941053aa` shows exactly one file touched:

```
deploy/k8s/base/atlas-kafka-precreate_test.sh | 138 ++++++++++++++++++++++++++
 1 file changed, 138 insertions(+)
```

`deploy/k8s/base/kafka-precreate.sh` has an empty diff for this range (`git
diff 457365ea0..d941053aa -- deploy/k8s/base/kafka-precreate.sh` → no output)
— the constraint "must NOT change in this task" holds.

The appended block (test file lines 25, 111–246) was read in full and
compared against the brief's verbatim code block
(`.superpowers/sdd/plan/task-4-brief.md:95-175` and `:196-253`). It matches
line for line, including the comments explaining the different-topic
consumer design point and the FR-3.1 regression rationale.

## Requirement-by-requirement

- **Step 1 (`CONSUMER_CMD`)** — added at `atlas-kafka-precreate_test.sh:25`,
  immediately after `CONSUMER_GROUPS_CMD`, same `${KAFKA_BIN:+$KAFKA_BIN/}`
  indirection as the other three. PASS.
- **Step 2 (FR-5.2 active-group skip)** — `atlas-kafka-precreate_test.sh:124-189`.
  Creates `TOPIC_C/D/E` + `GROUP3`, seeds `GROUP3` on `TOPIC_C` only, records
  `OFF_BEFORE`, starts a background `kafka-console-consumer.sh --group
  "$GROUP3" --topic "$TOPIC_D"` (the DIFFERENT topic than the one under
  test), traps it for cleanup, polls `group_state` to `Stable` with a bounded
  30-attempt cap, widens `TOPIC_C`'s log (2→5) so a successful reset would be
  detectable, asserts `state_is_seedable` rejects `Stable`, asserts
  `seed_group` returns exactly `2` via the `rc=0; seed_group … || rc=$?`
  pattern (does not abort under `set -eu`), and asserts `OFF_AFTER ==
  OFF_BEFORE`. PASS — all four assertions present as specified.
- **Step 3 (FR-5.1 idempotence + FR-3.1 regression)** —
  `atlas-kafka-precreate_test.sh:191-246`. Sets `$topics`/`$compact_topics`
  globals (union of `TOPIC_C`, `TOPIC_E` — the latter with no committed
  offset for `GROUP3`, the FR-3.1 regression case) and
  `KAFKA_CONSUMER_GROUP`, runs `{ seed_override_offsets; verify_group_offsets;
  }` inside `$( … 2>&1 )` command substitution twice. Pass 1 asserts exit 0
  and all four log-content checks (`already active (Stable)`, `nothing seeded
  this run (re-sync no-op)`, `(0 seeded, 1 skipped)`, the exact `WARN:` line
  naming `$TOPIC_E`); pass 2 asserts exit 0 again; a final check asserts the
  committed offset on `$TOPIC_C` across both passes is unchanged. PASS.
- **Both assertions below the `BOOTSTRAP_SERVERS` guard, SKIP cleanly** —
  confirmed live: `sh deploy/k8s/base/atlas-kafka-precreate_test.sh` (no
  `BOOTSTRAP_SERVERS` set) prints the two no-broker PASS lines, then `SKIP:
  BOOTSTRAP_SERVERS unset`, exit 0. The new code is textually below the guard
  at line 64. PASS.
- **`$KAFKA_BIN` full path on every Kafka CLI invocation** — the new code
  calls Kafka exclusively via `$TOPICS_CMD`, `$PRODUCER_CMD`,
  `$CONSUMER_GROUPS_CMD` (pre-existing) and the new `$CONSUMER_CMD`, all
  built from `${KAFKA_BIN:+$KAFKA_BIN/}`. No bare `kafka-*.sh` call
  introduced. PASS.
- **Dual-use portability (no arrays, no `local`, no `[[ ]]`, no `+=`, no
  `$'...'`)** — `grep -n "local \|\[\[\|+="` over the whole test file
  returns nothing; visual read of the new block confirms only `case`, `[ ]`,
  `$(( ))`, `printf`, string concatenation (`missing_names="$missing_names,
  $topic"` is in the pre-existing `kafka-precreate.sh`, not this file).
  `sh -n` and `bash -n` both exit 0. PASS.
- **Preserve existing line endings** — `git diff … | cat -A | grep '\^M'`
  found no CRLF in the diff. PASS.
- **`kafka-precreate.sh` untouched** — confirmed above (empty diff). PASS.
- **shellcheck** — `shellcheck -S error deploy/k8s/base/kafka-precreate.sh
  deploy/k8s/base/atlas-kafka-precreate_test.sh` → silent, exit 0 (re-run
  live, not just trusted from the report). PASS.

## Live broker-tier evidence (re-run independently)

Ran the exact command given in the review brief three times against the
live cluster reached via the session's existing `kubectl -n kafka
port-forward svc/kafka-broker 9092:9092`:

```
docker run --rm --network host -v "$PWD:/w" -w /w -e KAFKA_BIN=/opt/kafka/bin \
  -e BOOTSTRAP_SERVERS=localhost:9092 apache/kafka:3.7.2 sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

**Run 1** (clean broker state at session start): all nine PASS lines present
in order, exit 0, including all five new-assertion lines:

```
PASS: group_state reports 'Stable' for a group with a live member
PASS: seed_group returns 2 for an active group without aborting under set -eu
PASS: seed_group did not move an active group's committed offset (FR-5.2)
PASS: seed+verify skips an active group, warns on its unseeded topic, and exits 0 (FR-3.1)
PASS: a second full pass against a live environment exits 0 (FR-5.1)
PASS: two full passes changed no committed offset (FR-5.2)
```

**Run 2** (immediately re-run, no cleanup): fails at the FIRST topic-create
call — the pre-existing `atlas-precreate-test-1` topic (`TOPIC`, not any of
the new `TOPIC_C/D/E`), with `TopicExistsException`, exit 1. This
independently confirms the implementer's report: `docker run`'s PID 1 is
stable across separate invocations, so `$$`-suffixed names collide against a
persistent broker on a naive re-run.

**Run 3** (after manually deleting the six leftover
`atlas-precreate-test-{,a-,b-,c-,d-,e-}1` topics): all nine PASS lines again,
exit 0.

No leaked `kafka-console-consumer.sh` process on the host after any run
(`ps aux | grep kafka-console-consumer` empty; the container's `--rm` plus
the script's own `trap … EXIT INT TERM` account for this).

## The two points to weigh

**1. Repeatability / the `$$` collision.** Confirmed live: Run 2 above
reproduces the collision, and critically it fails on the *pre-existing*
`atlas-precreate-test-1` topic — the very first topic the *original*,
untouched-by-this-task assertion creates — not on any of the new
`TOPIC_C/D/E` names. This is decisive: the fragility is a property of the
whole file's long-standing `$$`-suffix convention against a persistent
broker under a fixed-PID container runtime, not something Task 4 introduced
or made worse. Task 4 follows the existing convention exactly as directed
("Patterns to copy … topic naming with `$$`"). A fix (e.g. `--delete
--if-exists` teardown at the top of the broker tier, or a truly unique
per-run token) would be a repo-wide convention change touching the
pre-existing assertions too — out of Task 4's stated scope
(`atlas-kafka-precreate_test.sh` append-only, `kafka-precreate.sh`
untouched) and not a defect introduced by this diff. Non-blocking finding,
correctly assignable to a follow-up rather than this task.

**2. Can either new assertion pass for the wrong reason?**

- FR-5.2 (offset-unchanged): the test design deliberately widens `TOPIC_C`'s
  log from 2 to 5 messages *after* establishing the baseline and *before*
  the refused reset attempt (`atlas-kafka-precreate_test.sh:168-169`). A
  `--to-latest` reset that silently succeeded (the pre-fix bug) would move
  the offset to 5; the assertion only passes if `OFF_AFTER == OFF_BEFORE ==
  2`. This closes the "passes trivially because the offset never had
  anywhere to move" hole — the gap-widening step is exactly what the design
  brief flagged as load-bearing for this reason, and it is present and
  correctly ordered. Verified live: the run shows `PASS: seed_group did not
  move an active group's committed offset` with the gap having genuinely
  widened first.
- Consumer-topic separation: the background consumer subscribes to
  `$TOPIC_D`, never `$TOPIC_C` (`--group "$GROUP3" --topic "$TOPIC_D"`,
  `atlas-kafka-precreate_test.sh:147-148`), so it cannot itself move
  `TOPIC_C`'s committed offset via auto-commit and self-validate the
  assertion. Confirmed by reading the exact invocation, not by trusting the
  comment.
- Background consumer cleanup: `trap 'kill "$CONSUMER_PID" 2>/dev/null ||
  true' EXIT INT TERM` is registered immediately after backgrounding
  (`:150-151`), before the poll loop, so an early FAIL/exit still kills it.
  Live evidence: `ps aux | grep kafka-console-consumer` was empty on the host
  after every run, and inside the `docker run --rm` container run this is
  additionally backstopped by container teardown.
- The idempotence pair genuinely exercises `verify_group_offsets`'s `exit 1`
  path being caught: it runs inside `$( { seed_override_offsets;
  verify_group_offsets; } 2>&1 )`, a subshell via command substitution, so a
  hard `exit 1` inside `verify_group_offsets` (which would otherwise kill the
  whole sourcing test script under `set -eu`) only fails the substitution.
  This was exercised — `$TOPIC_E` has no committed offset for `GROUP3` and is
  in the union, so before this whole feature `verify_group_offsets` would
  have hit exactly that `exit 1` case; the assertion for the `WARN:` line
  proves the code takes the soft-skip branch instead. Live run confirms the
  `WARN: skipped group '<GROUP3>' has no committed offset on 1 of 2 topics:
  <TOPIC_E>` line printed (not directly visible in the terse PASS-only
  console capture above, but the pass1 assertion at
  `atlas-kafka-precreate_test.sh:228-231` explicitly `case`-matches on it,
  and the run exited with `PASS: seed+verify skips an active group, warns on
  its unseeded topic, and exits 0 (FR-3.1)` rather than failing that case
  clause).

No wrong-reason pass scenario found for either assertion.

## Judgment on the stated goal — does this close the "no live evidence" gap?

Yes. Tasks 1–3's reviewers recorded "not evaluable: no broker-backed test
exercises the WARN path / the exit-1 path / the active-group skip." This
task's two new assertions, independently re-run against a live broker three
times in this review, directly exercise:

- the WARN path (pass 1's `WARN: skipped group … has no committed offset on
  1 of 2 topics` assertion, driven by `$TOPIC_E`);
- the `verify_group_offsets` `exit 1` path being avoided for a skipped group
  (same assertion — before this feature, this exact union would have hit
  the hard-gate `exit 1`);
- the active-group skip itself (`seed_group` returning `2` against a Kafka
  group genuinely held `Stable` by a live consumer, not a mocked state).

All three were previously not evaluable and are now evaluable and observed
passing against a real broker.

## Not evaluable

- None within this unit's scope. (The pre-existing `$$`-collision fragility
  is evaluable — and was evaluated above — but a fix for it is out of this
  task's scope, not something this review could not check.)

## Verdict

APPROVED_WITH_FINDINGS — one non-blocking finding (pre-existing `$$`
collision on repeated `docker run` invocations against a persistent broker,
not introduced or worsened by this diff, not in scope to fix here).
