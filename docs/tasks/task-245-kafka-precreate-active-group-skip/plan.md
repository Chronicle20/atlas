# Kafka Precreate — Skip Offset Seeding for Active Consumer Groups — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the wave-0 `atlas-kafka-precreate` offset-seeding pass idempotent across Argo CD re-syncs — an already-active consumer group is skipped rather than reset, and `verify_group_offsets` warns instead of failing the Job for it.

**Architecture:** Five changes inside one shell script. A pure classifier (`state_is_seedable`) decides seedable-vs-active from a state token with no I/O; a probe (`group_state`) supplies that token from one `kafka-consumer-groups.sh --describe --state` call per group; `seed_group` additionally classifies Kafka's *stdout* refusal message (the refusal exits 0 — measured, design §2.3) into a distinguishable return code 2; `seed_override_offsets` skips active groups and records them into a shared temp file; `verify_group_offsets` reads that file and downgrades its hard `exit 1` gate to a `WARN:` line for skipped groups only.

**Tech Stack:** POSIX shell (executed by `bash` in `apache/kafka:3.7.2`, *sourced* by a `#!/usr/bin/env sh` test), `kafka-consumer-groups.sh`, `awk`, `grep -Fxq`. No Go.

**Spec:** [`design.md`](./design.md) (PRD: [`prd.md`](./prd.md))

## Global Constraints

Every task's requirements implicitly include this section.

- **Module root / cwd for every command in this plan:** the task worktree root
  (`.worktrees/task-245-kafka-precreate-active-group-skip`). No Go module is
  touched.
- **Dual-use portability (design §5).** `deploy/k8s/base/kafka-precreate.sh` is
  *executed* by `bash` but *sourced* by `deploy/k8s/base/atlas-kafka-precreate_test.sh`,
  whose shebang is `#!/usr/bin/env sh` (dash). Sourcing parses the whole file,
  so a bash-only **syntax** construct is a parse error even inside a function
  the test never calls. Binding rules for all new code:
  - **No arrays** (`x=(...)`), **no `local`**, **no `[[ ]]`**, **no `+=`**, **no `$'...'`**.
  - Use `case` / `[ ]`, string concatenation, `printf`, `$(( ))`.
  - Existing bash-isms (`compgen`, `${!var}`) stay exactly where they are inside
    `precreate_topics`; do not add more.
- **`$KAFKA_BIN` full path on EVERY Kafka CLI invocation.** `apache/kafka:3.7.2`'s
  `PATH` does not include the Kafka CLI tools; a bare name exits 127 and, under
  `set -euo pipefail`, crash-loops the Job. `KAFKA_BIN="/opt/kafka/bin"` is
  already defined at `deploy/k8s/base/kafka-precreate.sh:24`.
- **Preserve existing line endings.** Do not normalize.
- **The `KAFKA_CONSUMER_GROUP`-unset early return stays byte-identical and stays
  first** in both `seed_override_offsets` and `verify_group_offsets` (PRD NG1,
  FR-3.4), ahead of any Kafka call.
- **No blanket `|| true` on the reset.** The fatal set narrows by exactly one
  identified condition (FR-2.3). `set +e` regions span exactly one command
  substitution plus its `$?` read.
- **Measured Kafka facts this plan depends on** (design §2, do not re-derive):
  - `--describe --state` prints a blank first line, then a header, then one data
    row. Every row ends in exactly five whitespace-separated tokens
    (`COORDINATOR` `(ID)` `ASSIGNMENT-STRATEGY` `STATE` `#MEMBERS`), so
    `STATE = $(NF-1)`. The header's own `$(NF-1)` is the literal `STATE` and a
    single-token group name yields the header's exact `NF`, so the header MUST
    be excluded by `$(NF-1)!="STATE"`, never by line number or field count.
  - A **nonexistent group exits 0** with no data row and an `Error:` line on
    stdout. "Absent" therefore collapses naturally to "no state token".
  - **`--reset-offsets --execute` against an active group also exits 0**, with
    `Error: Assignments can only be reset if the group '<name>' is inactive, but
    the current state is <State>.` on **stdout**. Classification must key off the
    message, not the exit code, and must run *before* the exit-code check.
- **Return contract for `seed_group` after Task 2:** `0` seeded · `2` refused
  because the group is active · anything else fatal. Every caller must use
  `rc=0; seed_group … || rc=$?` so `set -e` does not abort on a deliberate `2`.
- **Baseline (already green, must stay green):**
  `shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh`
  exits 0, and `sh deploy/k8s/base/atlas-kafka-precreate_test.sh` prints the NG6
  PASS then `SKIP: BOOTSTRAP_SERVERS unset` and exits 0.

---

## File Structure

| File | Responsibility | Tasks |
|---|---|---|
| `deploy/k8s/base/kafka-precreate.sh` | The change: classifier, probe, reset classification, skip set, skip-aware verification | 1, 2, 3 |
| `deploy/k8s/base/atlas-kafka-precreate_test.sh` | No-broker classifier truth table (T1); broker-backed active-group skip + idempotence (T4) | 1, 4 |
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Header comment only — the unconditional "still empty and therefore resettable" claim | 5 |
| `docs/runbooks/sparse-environments.md` | Operator-facing re-sync semantics and how to read the new log lines | 5 |
| `deploy/k8s/overlays/pr-sparse/kustomization.yaml` | **Read-only.** Its `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` commentary (lines 270–279) asserts only that the guard "stops taking their unset-guard early return" — it does NOT assert the now-conditional empty-group invariant, so design §7 requires no edit. Task 5 re-reads it to confirm, and changes nothing. | 5 (verify only) |

---

## Task 1: `state_is_seedable` — the pure classifier and its no-broker truth table

Splitting the classification out of the Kafka call is what makes FR-1.2/FR-1.3
assertable with **no broker**, so it runs on every developer machine and in
every CI job rather than only behind the `BOOTSTRAP_SERVERS` guard (design §3.1).

### Files

- `deploy/k8s/base/kafka-precreate.sh` — add `state_is_seedable` immediately
  **above** the `seed_group` block (i.e. above the comment that currently starts
  at line 116, `# Commit end-of-log offsets for an override consumer group…`)
- `deploy/k8s/base/atlas-kafka-precreate_test.sh` — add the truth-table
  assertion **after** the NG6 PASS `echo` (line 40) and **before** the
  `BOOTSTRAP_SERVERS` guard (line 42)

Patterns to copy: `deploy/k8s/base/kafka-precreate.sh:61-68` (the `case` idiom
already used in this file); `deploy/k8s/base/atlas-kafka-precreate_test.sh:31-40`
(the existing no-broker assertion's FAIL-then-`exit 1` shape).

**Interfaces**

- Produces: `state_is_seedable <state-token>` → exit 0 = seedable, exit 1 =
  active. Pure: no Kafka call, no I/O, no globals read or written. Consumed by
  Task 2 (`seed_override_offsets`) and Task 4 (broker test).

- [ ] **Step 1: Write the failing test**

Append to `deploy/k8s/base/atlas-kafka-precreate_test.sh`, between the NG6
`echo "PASS: …(NG6)"` line and the `[ -n "${BOOTSTRAP_SERVERS:-}" ] || …` guard.

Case table — every case, exact input, exact expectation:

| state token | expectation | why |
|---|---|---|
| `Empty` | seedable (rc 0) | FR-1.2 — the fresh-environment case |
| `Dead` | seedable (rc 0) | FR-1.2 |
| `""` (empty string) | seedable (rc 0) | FR-1.4 — absent group, unparseable output, or probe failure |
| `Stable` | active (rc 1) | FR-1.3 — the re-sync case |
| `PreparingRebalance` | active (rc 1) | FR-1.3 |
| `CompletingRebalance` | active (rc 1) | FR-1.3 |
| `SomeNewState` | active (rc 1) | FR-1.3 — allowlist, not denylist: a state Kafka adds later must fall to "active" |
| `STATE` | active (rc 1) | the header token must never be treated as seedable |

```sh
# state_is_seedable is an ALLOWLIST, not a denylist (design §3.1): only
# Empty, Dead and the empty string (absent group / unparseable output /
# failed probe, FR-1.4) are seedable. Anything else — including a state
# token Kafka adds in a future version — is treated as active and skipped,
# because skipping never mutates a committed offset (FR-5.2) whereas
# resetting an unrecognised live state does. Asserted without a broker, so
# this contract is enforced on every run of this script.
for state in Empty Dead ""; do
    if ! state_is_seedable "$state"; then
        echo "FAIL: state_is_seedable rejected seedable state '$state'"
        exit 1
    fi
done
for state in Stable PreparingRebalance CompletingRebalance SomeNewState STATE; do
    if state_is_seedable "$state"; then
        echo "FAIL: state_is_seedable accepted active state '$state'"
        exit 1
    fi
done
echo "PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state"
```

- [ ] **Step 2: Run the test to verify it fails**

```sh
sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected: non-zero exit, with a `state_is_seedable: not found` style error from
the shell (the function does not exist yet). It must NOT print
`PASS: state_is_seedable …`.

- [ ] **Step 3: Write the implementation**

Insert into `deploy/k8s/base/kafka-precreate.sh` above the `seed_group` comment
block:

```sh
# Classify a consumer-group state token as seedable (offsets may be reset) or
# active (a live member holds the group; Kafka will refuse the reset).
#
# Deliberately an ALLOWLIST: Empty, Dead and "" are seedable, everything else
# is active. "" covers an absent group, unparseable --describe --state output,
# and a failed probe (FR-1.4) — all of which must fall through to seed_group,
# whose own message classifier then governs. A state token Kafka adds in a
# future version falls into "active" and is skipped, which is the safe
# direction: skipping never mutates a committed offset (FR-5.2), whereas a
# denylist would reset a group in an unrecognised live state.
#
# Pure — no Kafka call, no I/O — so atlas-kafka-precreate_test.sh asserts the
# whole truth table without a broker (design §3.1).
state_is_seedable() {
    case "$1" in
        Empty|Dead|"") return 0 ;;
        *) return 1 ;;
    esac
}
```

- [ ] **Step 4: Run the test to verify it passes**

```sh
sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected, in order, exit 0:

```
PASS: seed_override_offsets skips when KAFKA_CONSUMER_GROUP is unset (NG6)
PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state
SKIP: BOOTSTRAP_SERVERS unset
```

- [ ] **Step 5: Check both shells still parse and shellcheck is clean**

```sh
bash -n deploy/k8s/base/kafka-precreate.sh
sh -n deploy/k8s/base/kafka-precreate.sh
shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected: all three silent, exit 0.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh
git commit -m "feat(kafka-precreate): add state_is_seedable classifier with no-broker truth table"
```

---

## Task 2: `group_state` probe, `seed_group` message classification, and the skip-and-record seeding pass

One coherent unit: the seeding side of `kafka-precreate.sh`. All three pieces
land together because `seed_group`'s new return code `2` requires its only
caller to handle it in the same change — landing them apart would leave a
`set -e` abort on the deliberate `2`.

**Intermediate state, by design:** after this task `seed_override_offsets`
skips active groups but `verify_group_offsets` still hard-fails them, because
nothing consumes the skip set until Task 3. This state is never shipped; Task 3
closes it. (Recorded in `context.md`.)

### Files

- `deploy/k8s/base/kafka-precreate.sh` — add `group_state`; rewrite
  `seed_group` (currently lines 116–141, comment block included) and
  `seed_override_offsets` (currently lines 143–194)

Patterns to copy: `deploy/k8s/base/kafka-precreate.sh:219` (the existing
`--describe` invocation — same `$KAFKA_BIN` full path, same `2>/dev/null`);
`deploy/k8s/base/kafka-precreate.sh:234` (the `NF`-anchored `awk` idiom this
file already documents and uses).

**Interfaces**

- Consumes: `state_is_seedable <state>` (Task 1).
- Produces:
  - `group_state <group>` → echoes the group's state token on stdout, or the
    empty string. Never fails, never exits non-zero.
  - `seed_group <group> <topic…>` → `0` seeded · `2` refused because active ·
    anything else fatal. Signature unchanged, so Task 4's and the existing
    single-/multi-topic assertions keep calling it verbatim.
  - `$skipped_groups` — a global holding the path to a temp file with one
    skipped group name per line. Consumed by `verify_group_offsets` in Task 3.
  - `$seeded_count` / `$skipped_count` — plain integers, FR-4.5.

- [ ] **Step 1: Add `group_state`**

Insert into `deploy/k8s/base/kafka-precreate.sh` immediately **above**
`state_is_seedable` (so the probe and its classifier read top-to-bottom):

```sh
# Probe one consumer group's state. Echoes the state token (Empty, Dead,
# Stable, PreparingRebalance, CompletingRebalance, …) or the empty string.
#
# Parsing, per the measured apache/kafka:3.7.2 output (design §2.1): the
# command prints a BLANK first line, then a header, then one data row. Real
# group names contain spaces and brackets (e.g. "Channel Service - <uuid>
# [f8c5]"), which shift every fixed forward column — but a row always ends in
# exactly five whitespace-separated tokens (COORDINATOR (ID)
# ASSIGNMENT-STRATEGY STATE #MEMBERS), so STATE is $(NF-1) regardless of how
# many tokens GROUP contributes. This is the same NF-anchored idiom
# verify_group_offsets already documents (FR-1.5).
#
# The header MUST be excluded by $(NF-1)!="STATE", not by line number and not
# by field count: a single-token group name yields NF=6, exactly the header's
# NF, so a count-based guard would "find" a state of "STATE".
#
# STATE is parsed rather than #MEMBERS (OQ-2) because FR-4.1 requires the log
# line to name the state anyway, a string allowlist rejects garbage without a
# separate numeric guard, and #MEMBERS collapses Dead, Empty and any future
# zero-member-but-active state into one bucket.
#
# A nonexistent group exits 0 with no data row and an Error: line on stdout
# (design §2.2), so absence needs no special case — it yields "". Any
# non-zero exit and any unparseable output collapse to "" as well, which
# state_is_seedable calls seedable (FR-1.4): the probe can never itself fail
# the Job.
group_state() {
    set +e
    state_out="$("$KAFKA_BIN/kafka-consumer-groups.sh" \
        --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$1" --describe --state 2>/dev/null)"
    set -e
    printf '%s\n' "$state_out" | awk 'NF>=6 && $(NF-1)!="STATE" { print $(NF-1); exit }'
}
```

- [ ] **Step 2: Rewrite `seed_group` to classify on the message**

Replace the existing `seed_group` body (`deploy/k8s/base/kafka-precreate.sh:132-141`),
and correct its header comment. The comment currently claims the group is
"empty and therefore resettable"; per design §7 that becomes conditional and
carries the §2.3 measurement so the next reader does not re-derive it.

Replace the comment block at lines 116–119:

```sh
# Commit end-of-log offsets for an override consumer group on one or more
# topics in a SINGLE kafka-consumer-groups.sh invocation. On a FIRST sync the
# group is empty and therefore resettable; on a RE-SYNC of a live environment
# the Job is deleted and recreated (Force=true,Replace=true) while the
# environment's Deployments have been joined to those very groups for hours,
# so the group is active and Kafka refuses the reset. seed_override_offsets
# probes for that ahead of time (group_state / state_is_seedable); this
# function classifies the race that remains between the probe and the reset.
#
# MEASURED (design §2.3, apache/kafka:3.7.2, both --dry-run and --execute,
# both single- and repeated---topic form): a reset refused because the group
# is active EXITS 0 and prints
#   Error: Assignments can only be reset if the group '<name>' is inactive,
#   but the current state is <State>.
# to STDOUT. Classification therefore keys off the MESSAGE, and the message
# case must run BEFORE the exit-code check — a code-only classifier is
# decorative. Before this change that output went to /dev/null and the
# refusal was a silent no-op reporting success.
#
# Returns: 0 seeded · 2 refused-because-active (non-fatal, FR-2.2) ·
# anything else fatal (FR-2.3 — broker-unreachable, authorization and
# malformed-argument failures still fail the Job). Callers MUST use
# `rc=0; seed_group … || rc=$?` so set -e does not abort on a deliberate 2.
#
# The matched substring deliberately stops before the group name and the
# state, so it survives a changed quoting style, a new state name, and a
# group name containing glob metacharacters.
```

Then the remaining comment paragraphs at lines 121–131 (the repeated-`--topic`
collapse rationale and the "topic names are plain identifiers" note) stay
verbatim, followed by:

```sh
seed_group() {
    group="$1"; shift
    topic_args=""
    for topic in "$@"; do
        topic_args="$topic_args --topic $topic"
    done

    # set +e spans exactly one command substitution and its $? read — the
    # minimum region the failure-isolation NFR allows. 2>&1 capture replaces
    # the old >/dev/null (FR-2.4); the success path stays silent because the
    # capture is only printed on the fatal branch.
    set +e
    # shellcheck disable=SC2086
    seed_out="$("$KAFKA_BIN/kafka-consumer-groups.sh" --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$group" $topic_args --reset-offsets --to-latest --execute 2>&1)"
    seed_rc=$?
    set -e

    case "$seed_out" in
        *"Assignments can only be reset if the group"*) return 2 ;;
    esac
    if [ "$seed_rc" -ne 0 ]; then
        printf '%s\n' "$seed_out" >&2
        return "$seed_rc"
    fi
    return 0
}
```

- [ ] **Step 3: Rewrite `seed_override_offsets` to probe, skip and record**

Keep every existing comment paragraph above the function
(`deploy/k8s/base/kafka-precreate.sh:143-175`) verbatim — the WHOEVER-WIRES-THIS
resolver warning, the full-topic-union rationale, and the NG6 paragraph are all
still true and still load-bearing. Append one paragraph to that block:

```sh
# A group that is ALREADY ACTIVE is already initialized — which is the end
# state this pass exists to reach — so it is skipped, not reset (FR-1.3).
# This is what makes the pass idempotent across Argo CD re-syncs: the Job
# carries Force=true,Replace=true and is recreated on every sync, while the
# environment's Deployments have been joined to those groups for hours.
# Skipping is also the safety property (FR-5.2): an environment that has been
# consuming for hours must never be silently fast-forwarded past unprocessed
# messages by a routine re-sync.
```

Then replace the function body (lines 176–194) with:

```sh
seed_override_offsets() {
    if [ -z "${KAFKA_CONSUMER_GROUP:-}" ]; then
        echo "[$(date -u +%FT%TZ)] KAFKA_CONSUMER_GROUP unset — skipping override offset seeding (main, NG6)"
        return 0
    fi

    echo "[$(date -u +%FT%TZ)] seeding end-of-log offsets for override consumer groups"
    groups_file="$(mktemp)"
    printf '%s\n' "$KAFKA_CONSUMER_GROUP" > "$groups_file"
    all_topics="$(mktemp)"
    cat "$topics" "$compact_topics" > "$all_topics"
    # Shared with verify_group_offsets by the same globals-by-convention this
    # pass already uses for $topics / $compact_topics (FR-3.2). One group name
    # per line; membership is tested with grep -Fxq -- so a name containing
    # spaces, brackets or a leading dash matches exactly.
    skipped_groups="$(mktemp)"
    seeded_count=0
    skipped_count=0

    while IFS= read -r group; do
        [ -n "$group" ] || continue
        group_current_state="$(group_state "$group")"
        if state_is_seedable "$group_current_state"; then
            seed_rc=0
            # shellcheck disable=SC2046
            seed_group "$group" $(cat "$all_topics") || seed_rc=$?
            if [ "$seed_rc" -eq 0 ]; then
                seeded_count=$(( seeded_count + 1 ))
            elif [ "$seed_rc" -eq 2 ]; then
                # The window between the probe above and the reset is real: a
                # self-heal or a rescheduled pod can join the group in between
                # (FR-2.1). Same outcome as an FR-1.3 skip.
                printf '%s\n' "$group" >> "$skipped_groups"
                skipped_count=$(( skipped_count + 1 ))
                echo "[$(date -u +%FT%TZ)] skipping group '$group': reset refused, group became active during seeding — offsets already initialized"
            else
                echo "FAIL: seeding group '$group' failed (exit $seed_rc)" >&2
                exit "$seed_rc"
            fi
        else
            printf '%s\n' "$group" >> "$skipped_groups"
            skipped_count=$(( skipped_count + 1 ))
            echo "[$(date -u +%FT%TZ)] skipping group '$group': already active ($group_current_state) — offsets already initialized"
        fi
    done < "$groups_file"

    if [ "$seeded_count" -eq 0 ] && [ "$skipped_count" -gt 0 ]; then
        echo "[$(date -u +%FT%TZ)] all $skipped_count override consumer groups were already active — nothing seeded this run (re-sync no-op)"
    fi
    echo "[$(date -u +%FT%TZ)] override consumer group offsets seeded ($seeded_count seeded, $skipped_count skipped)"
}
```

- [ ] **Step 4: Verify both shells parse, shellcheck is clean, and the existing no-broker assertions still pass**

```sh
bash -n deploy/k8s/base/kafka-precreate.sh
sh -n deploy/k8s/base/kafka-precreate.sh
shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh
sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected: first three silent and exit 0; the last prints the NG6 PASS, the
`state_is_seedable` PASS from Task 1, then `SKIP: BOOTSTRAP_SERVERS unset`, exit 0.

The NG6 assertion passing is the proof that the `KAFKA_CONSUMER_GROUP`-unset
early return survived the rewrite intact (PRD NG1).

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/base/kafka-precreate.sh
git commit -m "feat(kafka-precreate): probe group state and skip active groups when seeding offsets"
```

---

## Task 3: `verify_group_offsets` — skip-aware, one describe, two verdicts

Per design §2.3 this is **the fix**. `--reset-offsets` against a live group
already exits 0, so today's Job failure comes from this function's `exit 1` —
the only `exit 1` in the script — when a skipped group lacks a committed offset
on some union topic.

FR-4.2's reporting lives here, not in `seed_override_offsets`, on purpose: the
per-group `--describe` this function already pays for is exactly the data
needed. Computing it in the seeding pass would add a second JVM cold start per
skipped group and a second place the `NF`-anchored parse has to stay correct.

### Files

- `deploy/k8s/base/kafka-precreate.sh` — rewrite `verify_group_offsets`
  (currently lines 196–242, comment block included)

Patterns to copy: the existing `NF`-anchored `awk` at
`deploy/k8s/base/kafka-precreate.sh:234` — reuse it verbatim, including the
`NF>=9 && $(NF-7)==t {print $(NF-5)}` field arithmetic and the `head -n1`.

**Interfaces**

- Consumes: `$skipped_groups` (Task 2). May legitimately be **unset** — a test
  sources the file and calls `verify_group_offsets` without having run
  `seed_override_offsets` — so unset/missing must be treated as "nothing
  skipped" and the function must behave exactly as it does today.
- Produces: no new symbols. Exit 1 on a gap for a **non**-skipped group
  (unchanged, FR-3.3); `WARN:` and continue for a skipped one (FR-4.2).

- [ ] **Step 1: Append the skip-aware paragraph to the function's comment block**

Keep the existing comment (lines 196–205) verbatim and append:

```sh
# A group in $skipped_groups was skipped by seed_override_offsets because it
# is already ACTIVE — which means a live consumer is joined to it, which is
# the very state this gate exists to establish (FR-3.1). Re-proving it against
# the full topic union would fail the Job the first time a topic is added to a
# live environment. So for a skipped group the gate degrades to a report:
# union topics with no committed offset are named in a WARN: line and the Job
# stays green (an unseeded topic falls back to the consumer's own
# auto.offset.reset). For every other group the gate is unchanged and still
# exits 1 (FR-3.3). $skipped_groups may be unset — a test can source this file
# and call verify_group_offsets without seed_override_offsets — in which case
# nothing was skipped and every group takes the hard-gate path.
```

- [ ] **Step 2: Rewrite the function body**

Replace lines 206–242 with:

```sh
verify_group_offsets() {
    if [ -z "${KAFKA_CONSUMER_GROUP:-}" ]; then
        return 0
    fi

    echo "[$(date -u +%FT%TZ)] verifying override consumer group offsets are committed"
    groups_file="$(mktemp)"
    printf '%s\n' "$KAFKA_CONSUMER_GROUP" > "$groups_file"
    all_topics="$(mktemp)"
    cat "$topics" "$compact_topics" > "$all_topics"

    while IFS= read -r group; do
        [ -n "$group" ] || continue
        group_skipped=0
        if [ -n "${skipped_groups:-}" ] && [ -f "${skipped_groups:-}" ] && grep -Fxq -- "$group" "$skipped_groups"; then
            group_skipped=1
        fi
        described="$("$KAFKA_BIN/kafka-consumer-groups.sh" --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$group" --describe 2>/dev/null || true)"
        topic_total=0
        missing_total=0
        missing_names=""
        while IFS= read -r topic; do
            [ -n "$topic" ] || continue
            topic_total=$(( topic_total + 1 ))
            # GROUP is column 1 in --describe output, but real group names
            # contain spaces (e.g. "Account Service [pr-123]"), which shift
            # every fixed column number — confirmed empirically (task-232
            # Task 45 review round 1 follow-up: an awk match on $2/$4 silently
            # never matched against a real multi-word group name, and this
            # verification pass would have failed the Job on every seeded
            # group). TOPIC through CLIENT-ID are always exactly 8
            # single-token trailing columns (topic names are plain
            # identifiers, never containing spaces), so anchor from the END
            # of the line via NF instead of a fixed forward offset: TOPIC is
            # NF-7, CURRENT-OFFSET is NF-5, regardless of how many
            # whitespace-separated tokens GROUP itself contributes.
            off="$(printf '%s\n' "$described" | awk -v t="$topic" 'NF>=9 && $(NF-7)==t {print $(NF-5)}' | head -n1)"
            if [ -z "$off" ] || [ "$off" = "-" ]; then
                if [ "$group_skipped" -eq 0 ]; then
                    echo "FAIL: group '$group' has no committed offset on topic '$topic'" >&2
                    exit 1
                fi
                missing_total=$(( missing_total + 1 ))
                # Bounded at 10 names plus a count (OQ-1): the union is ~170
                # topics and deliberately a superset, so a genuine gap list is
                # either tiny or large enough that the count is the signal.
                if [ "$missing_total" -le 10 ]; then
                    if [ -z "$missing_names" ]; then
                        missing_names="$topic"
                    else
                        missing_names="$missing_names, $topic"
                    fi
                fi
            fi
        done < "$all_topics"
        if [ "$group_skipped" -eq 1 ]; then
            if [ "$missing_total" -eq 0 ]; then
                echo "[$(date -u +%FT%TZ)] skipped group '$group': committed offsets present on all $topic_total topics"
            elif [ "$missing_total" -gt 10 ]; then
                echo "WARN: skipped group '$group' has no committed offset on $missing_total of $topic_total topics: $missing_names (+$(( missing_total - 10 )) more)" >&2
            else
                echo "WARN: skipped group '$group' has no committed offset on $missing_total of $topic_total topics: $missing_names" >&2
            fi
        fi
    done < "$groups_file"
    echo "[$(date -u +%FT%TZ)] override consumer group offsets verified"
}
```

- [ ] **Step 3: Verify parse, shellcheck, and the no-broker tier**

```sh
bash -n deploy/k8s/base/kafka-precreate.sh
sh -n deploy/k8s/base/kafka-precreate.sh
shellcheck -S error deploy/k8s/base/kafka-precreate.sh
sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected: first three silent and exit 0; the last prints both PASS lines then
`SKIP: BOOTSTRAP_SERVERS unset`, exit 0.

- [ ] **Step 4: Commit**

```bash
git add deploy/k8s/base/kafka-precreate.sh
git commit -m "fix(kafka-precreate): warn instead of failing the Job for skipped active groups"
```

---

## Task 4: Broker-backed assertions — active-group skip and idempotence

Both assertions live below the `BOOTSTRAP_SERVERS` guard and `SKIP` cleanly
without a broker, matching the existing convention.

**Test-design note the implementer must not lose:** the group is kept alive by
a `kafka-console-consumer.sh` subscribed to a **different** topic than the one
under test. A console consumer auto-commits, so a consumer on the target topic
would move the very offset the test asserts is unchanged, and the assertion
would prove nothing. Committed offsets persist independently of subscription,
so the seeded offset on the target topic survives untouched while the consumer
on the other topic holds the group `Stable`.

### Files

- `deploy/k8s/base/atlas-kafka-precreate_test.sh` — append both assertions
  after the existing multi-topic PASS (currently the last line, line 87)

Patterns to copy: `deploy/k8s/base/atlas-kafka-precreate_test.sh:69-87` (the
multi-topic assertion — topic naming with `$$`, `$TOPICS_CMD`/`$PRODUCER_CMD`/
`$CONSUMER_GROUPS_CMD` indirection, FAIL-then-`exit 1` shape).

**Interfaces**

- Consumes: `group_state` and `state_is_seedable` (Tasks 1–2), `seed_group`'s
  `2` return (Task 2), `seed_override_offsets` / `verify_group_offsets`
  (Tasks 2–3), and the `$topics` / `$compact_topics` globals those two read.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Add the consumer-command variable**

`atlas-kafka-precreate_test.sh` already derives `TOPICS_CMD`, `PRODUCER_CMD`
and `CONSUMER_GROUPS_CMD` from `$KAFKA_BIN` (lines 22–24). Add one more, on the
line after `CONSUMER_GROUPS_CMD`:

```sh
CONSUMER_CMD="${KAFKA_BIN:+$KAFKA_BIN/}kafka-console-consumer.sh"
```

- [ ] **Step 2: Write the failing active-group-skip assertion (FR-5.2)**

Append to `deploy/k8s/base/atlas-kafka-precreate_test.sh`.

| step | exact action | exact expectation |
|---|---|---|
| setup | create `$TOPIC_C` (1 partition), `$TOPIC_D` (1 partition), `$TOPIC_E` (1 partition) | — |
| setup | produce `a\nb\n` to `$TOPIC_C`, `x\n` to `$TOPIC_D` | — |
| setup | `seed_group "$GROUP3" "$TOPIC_C"` | rc 0; establishes a committed offset of `2` on `$TOPIC_C` |
| record | read `$TOPIC_C`'s `CURRENT-OFFSET` via the `NF`-anchored awk into `OFF_BEFORE` | non-empty, not `-` |
| activate | start `kafka-console-consumer.sh --group "$GROUP3" --topic "$TOPIC_D" --from-beginning` in background; register a `trap` that kills it | — |
| activate | poll `group_state "$GROUP3"` up to 30 times, `sleep 2` between | reaches `Stable`; FAIL if the cap is hit |
| widen gap | produce `d\ne\nf\n` to `$TOPIC_C` | end-of-log for `$TOPIC_C` is now 5, so a successful `--to-latest` reset would visibly move the offset from 2 |
| assert 1 | `group_state "$GROUP3"` | `Stable` |
| assert 2 | `state_is_seedable "$STATE_NOW"` | returns non-zero (active) |
| assert 3 | `seed_group "$GROUP3" "$TOPIC_C"` under `set -eu` | returns exactly `2` and does NOT abort the script |
| assert 4 | re-read `$TOPIC_C`'s `CURRENT-OFFSET` into `OFF_AFTER` | `OFF_AFTER` = `OFF_BEFORE` (this is the FR-5.2 safety property) |

```sh
# FR-5.2 — the safety property. A sparse environment that has been consuming
# for hours must not be silently fast-forwarded past unprocessed messages by
# a routine Argo CD re-sync. Kafka refuses --reset-offsets on an active group,
# but MEASURED (design §2.3) it refuses with EXIT 0 and an Error: line on
# stdout, so before this change the refusal was a silent no-op reporting
# success. seed_group must now report that as return code 2, without aborting
# the caller under set -eu, and without moving the committed offset.
#
# The group is held active by a consumer on a DIFFERENT topic ($TOPIC_D) than
# the one under test ($TOPIC_C): kafka-console-consumer auto-commits, so a
# consumer on $TOPIC_C would move the very offset this test asserts is
# unchanged. Committed offsets persist independently of subscription, so the
# seeded offset on $TOPIC_C survives untouched.
TOPIC_C="atlas-precreate-test-c-$$"
TOPIC_D="atlas-precreate-test-d-$$"
TOPIC_E="atlas-precreate-test-e-$$"
GROUP3="atlas-precreate-test-group3-$$"
for t in "$TOPIC_C" "$TOPIC_D" "$TOPIC_E"; do
    "$TOPICS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$t" --partitions 1
done
printf 'a\nb\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_C"
printf 'x\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_D"

seed_group "$GROUP3" "$TOPIC_C"

committed_offset() {
    "$CONSUMER_GROUPS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$1" --describe 2>/dev/null \
        | awk -v t="$2" 'NF>=9 && $(NF-7)==t {print $(NF-5)}' | head -n1
}

OFF_BEFORE="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ -z "$OFF_BEFORE" ] || [ "$OFF_BEFORE" = "-" ]; then
    echo "FAIL: setup — seed_group left '$TOPIC_C' without a committed offset"
    exit 1
fi

"$CONSUMER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" \
    --group "$GROUP3" --topic "$TOPIC_D" --from-beginning >/dev/null 2>&1 &
CONSUMER_PID=$!
# Never leave a consumer attached to the shared broker on a mid-test failure.
trap 'kill "$CONSUMER_PID" 2>/dev/null || true' EXIT INT TERM

# Bounded poll — a hard attempt cap, never an unbounded while.
STATE_NOW=""
attempt=0
while [ "$attempt" -lt 30 ]; do
    STATE_NOW="$(group_state "$GROUP3")"
    [ "$STATE_NOW" = "Stable" ] && break
    attempt=$(( attempt + 1 ))
    sleep 2
done
if [ "$STATE_NOW" != "Stable" ]; then
    echo "FAIL: group '$GROUP3' did not reach Stable within 30 attempts (last state: '$STATE_NOW')"
    exit 1
fi
echo "PASS: group_state reports 'Stable' for a group with a live member"

# A successful --to-latest reset would now move $TOPIC_C's offset from 2 to 5.
printf 'd\ne\nf\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_C"

if state_is_seedable "$STATE_NOW"; then
    echo "FAIL: state_is_seedable called '$STATE_NOW' seedable"
    exit 1
fi

seed_rc3=0
seed_group "$GROUP3" "$TOPIC_C" || seed_rc3=$?
if [ "$seed_rc3" -ne 2 ]; then
    echo "FAIL: seed_group against an active group returned $seed_rc3, expected 2"
    exit 1
fi
echo "PASS: seed_group returns 2 for an active group without aborting under set -eu"

OFF_AFTER="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ "$OFF_AFTER" != "$OFF_BEFORE" ]; then
    echo "FAIL: active group's committed offset moved $OFF_BEFORE -> $OFF_AFTER (FR-5.2)"
    exit 1
fi
echo "PASS: seed_group did not move an active group's committed offset (FR-5.2)"
```

- [ ] **Step 3: Write the failing idempotence assertion (FR-5.1, and the FR-3.1 regression test)**

Driving `main` twice is impractical here: `precreate_topics` uses `compgen`,
i.e. it requires bash, while this harness is `#!/usr/bin/env sh`. Take the
PRD-sanctioned function-level equivalent (design §6.2 item 5) — the same
property at a lower seam, not a scope reduction: set the `$topics` /
`$compact_topics` globals `seed_override_offsets` and `verify_group_offsets`
read, then run the pair twice.

The union deliberately includes `$TOPIC_E`, which `$GROUP3` has **no** committed
offset on. That is the FR-3.1 regression case: before this change,
`verify_group_offsets` would `exit 1` there.

| run | expectation |
|---|---|
| 1 | the pair exits 0; output contains `already active (Stable)`, `nothing seeded this run (re-sync no-op)`, `(0 seeded, 1 skipped)`, and `WARN: skipped group '<GROUP3>' has no committed offset on 1 of 2 topics: <TOPIC_E>` |
| 2 | the pair exits 0 again (FR-5.1) |
| after both | `$TOPIC_C`'s committed offset still equals `OFF_BEFORE` (FR-5.2 across a full re-run) |

```sh
# FR-5.1 — running the pass twice against the same live environment exits 0
# both times, and (FR-5.2) changes no committed offset. Also the FR-3.1
# regression test: $TOPIC_E is in the union and $GROUP3 has no committed
# offset on it, which before this change made verify_group_offsets exit 1 and
# fail the whole Job.
#
# main is not driven directly because precreate_topics uses compgen (bash) and
# this harness is #!/usr/bin/env sh; $topics / $compact_topics are the globals
# precreate_topics would otherwise set, so setting them here exercises the
# same seam.
topics="$(mktemp)"
compact_topics="$(mktemp)"
printf '%s\n%s\n' "$TOPIC_C" "$TOPIC_E" > "$topics"
: > "$compact_topics"
KAFKA_CONSUMER_GROUP="$GROUP3"
export KAFKA_CONSUMER_GROUP

# Command substitution is a subshell, so verify_group_offsets' exit 1 fails
# the substitution instead of killing this script — which is what lets the
# assertion report rather than merely die.
if ! pass1="$( { seed_override_offsets; verify_group_offsets; } 2>&1 )"; then
    echo "FAIL: first seed+verify pass against an active group did not exit 0"
    printf '%s\n' "$pass1"
    exit 1
fi
case "$pass1" in
    *"already active (Stable)"*) ;;
    *) echo "FAIL: first pass did not log the active-group skip"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"nothing seeded this run (re-sync no-op)"*) ;;
    *) echo "FAIL: first pass did not log the all-skipped re-sync no-op line"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"(0 seeded, 1 skipped)"*) ;;
    *) echo "FAIL: first pass did not report the seeded/skipped counts"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"WARN: skipped group '$GROUP3' has no committed offset on 1 of 2 topics: $TOPIC_E"*) ;;
    *) echo "FAIL: first pass did not WARN about the unseeded union topic"; printf '%s\n' "$pass1"; exit 1 ;;
esac
echo "PASS: seed+verify skips an active group, warns on its unseeded topic, and exits 0 (FR-3.1)"

if ! pass2="$( { seed_override_offsets; verify_group_offsets; } 2>&1 )"; then
    echo "FAIL: second seed+verify pass did not exit 0 (FR-5.1)"
    printf '%s\n' "$pass2"
    exit 1
fi
echo "PASS: a second full pass against a live environment exits 0 (FR-5.1)"

OFF_FINAL="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ "$OFF_FINAL" != "$OFF_BEFORE" ]; then
    echo "FAIL: committed offset moved $OFF_BEFORE -> $OFF_FINAL across two passes (FR-5.2)"
    exit 1
fi
echo "PASS: two full passes changed no committed offset (FR-5.2)"
```

- [ ] **Step 4: Run the no-broker path and the shellcheck gate**

```sh
sh deploy/k8s/base/atlas-kafka-precreate_test.sh
shellcheck -S error deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected: the test prints both no-broker PASS lines then `SKIP: BOOTSTRAP_SERVERS unset`
and exits 0 (the new assertions are below the guard); shellcheck silent, exit 0.

- [ ] **Step 5: Run the broker tier against a real Kafka**

Reach the cluster broker with a port-forward in one shell:

```sh
kubectl -n kafka port-forward svc/kafka-broker 9092:9092
```

then, in the worktree:

```sh
BOOTSTRAP_SERVERS=localhost:9092 sh deploy/k8s/base/atlas-kafka-precreate_test.sh
```

Expected, all PASS lines present and exit 0:

```
PASS: seed_override_offsets skips when KAFKA_CONSUMER_GROUP is unset (NG6)
PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state
PASS
PASS: seed_group seeds every topic in a single multi-topic call
PASS: group_state reports 'Stable' for a group with a live member
PASS: seed_group returns 2 for an active group without aborting under set -eu
PASS: seed_group did not move an active group's committed offset (FR-5.2)
PASS: seed+verify skips an active group, warns on its unseeded topic, and exits 0 (FR-3.1)
PASS: a second full pass against a live environment exits 0 (FR-5.1)
PASS: two full passes changed no committed offset (FR-5.2)
```

If no broker is reachable, this step is BLOCKED, not skipped — report it and do
not mark the task done on the SKIP path alone. `KAFKA_BIN` must point at a Kafka
CLI install; the harness defaults to `/opt/kafka/bin` and falls back to bare
names on `PATH` when that directory is absent.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/base/atlas-kafka-precreate_test.sh
git commit -m "test(kafka-precreate): assert active-group skip and re-sync idempotence"
```

---

## Task 5: Documentation — the Job header comment and the operator runbook

### Files

- `deploy/k8s/base/atlas-kafka-precreate.yaml` — header comment only, lines
  17–24. **No spec change**: `backoffLimit: 3`, `ttlSecondsAfterFinished: 600`,
  `Force=true,Replace=true`, sync-wave 0 all stay exactly as they are (PRD NG2).
- `docs/runbooks/sparse-environments.md` — the section "Verifying a consumer
  group is seeded (FR-4.9, FR-5.3)" begins at line 125; insert a new
  subsection after the paragraph ending `…no separate inference is needed.`
  (line 158) and before `To check a specific group by hand:` (line 160).
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — **read-only**; re-read
  lines 270–279 to confirm no edit is needed (Step 3).

- [ ] **Step 1: Correct the Job's header comment**

In `deploy/k8s/base/atlas-kafka-precreate.yaml`, replace the paragraph at lines
17–24 with:

```yaml
# Also seeds (task-232 Task 45, design §6.3) committed end-of-log offsets for
# every override consumer group named in KAFKA_CONSUMER_GROUP — see
# kafka-precreate.sh for the mechanism and NG6 (main is inert: the pass is
# skipped whenever KAFKA_CONSUMER_GROUP is unset). Whether a group is
# resettable depends on which sync this is. On a FIRST sync the group is still
# empty, and every Deployment sits at sync-wave 10 behind this Job, so the
# reset lands before any consumer joins. On a RE-SYNC of a live environment
# the Force=true,Replace=true above deletes and recreates this Job while the
# environment's Deployments have been joined to those groups for hours: the
# group is active, Kafka refuses the reset, and the pass SKIPS it (task-245).
# An active group is already initialized, which is the end state the pass
# exists to reach, so skipping is both correct and the safety property — a
# routine re-sync must never fast-forward a live environment past unprocessed
# messages. The script is mounted from the atlas-kafka-precreate-script
# ConfigMap (kustomization.yaml) instead of living inline here so it is
# independently sourceable and testable (atlas-kafka-precreate_test.sh).
```

- [ ] **Step 2: Add the runbook subsection**

Insert into `docs/runbooks/sparse-environments.md` between line 158 and line 160:

````markdown
### Re-syncs: an active group is skipped, not re-seeded (task-245)

The Job carries `argocd.argoproj.io/sync-options: Force=true,Replace=true`, so
Argo CD deletes and recreates it on **every** sync — including a re-sync of an
environment whose Deployments have been running, and joined to those very
groups, for hours. Kafka refuses `--reset-offsets` on a group that is not
inactive, so on a re-sync the pass **probes each group's state first and skips
any group that is already active**. An active group is by definition already
initialized, which is the end state the seeding pass exists to reach.

This is also the safety property: an environment that has been consuming for
hours must never be silently fast-forwarded past unprocessed messages by a
routine re-sync.

Reading the Job log, per group, there are three outcomes:

| Log line | Meaning |
|---|---|
| `skipping group '<g>': already active (Stable) — offsets already initialized` | Re-sync of a live environment. Normal. |
| `skipping group '<g>': reset refused, group became active during seeding — offsets already initialized` | A pod joined the group between the state probe and the reset. Same outcome, same normality. |
| `skipped group '<g>': committed offsets present on all <N> topics` | The common re-sync case — skipped, and nothing is missing. |
| `WARN: skipped group '<g>' has no committed offset on <K> of <N> topics: <names> (+<M> more)` | Skipped, and these union topics were never seeded for it. Not a failure. |
| `override consumer group offsets seeded (<S> seeded, <K> skipped)` | Terminal summary. |
| `all <K> override consumer groups were already active — nothing seeded this run (re-sync no-op)` | Every group was skipped: this run did nothing, by design. |

A `WARN:` line usually means a **topic was added to the environment after the
group went live**. A live group cannot be re-seeded, so that topic keeps no
committed offset for it and the consumer's own `auto.offset.reset` governs
where it starts. The Job stays green: failing here would wedge an environment
on a benign mid-life topic addition. If the WARN names topics the group
actually consumes and the starting position matters, delete the environment's
Deployments (emptying the groups) and re-sync, or accept the
`auto.offset.reset` position.

The skip covers every shape that reaches the pass with a pre-existing active
group, not only a re-sync: a re-created override reusing a previous
environment's `ATLAS_ENV` suffix, or an environment whose baseline was
switched, are handled identically.

A fresh environment is unaffected. Its groups are `Empty`, every one is reset
to end-of-log, and `verify_group_offsets` still hard-fails the Job (`FAIL:
group '<g>' has no committed offset on topic '<t>'`, exit 1) if any is missing
— the activation gate is narrowed to exactly the groups where it is
unprovable, and is preserved everywhere it can still be enforced.

The behaviour is covered by `deploy/k8s/base/atlas-kafka-precreate_test.sh`:
the `state_is_seedable` truth table runs without a broker, and the
active-group skip and two-pass idempotence assertions run when
`BOOTSTRAP_SERVERS` is set.
````

- [ ] **Step 3: Confirm the pr-sparse overlay comment needs no change**

```sh
sed -n '265,282p' deploy/k8s/overlays/pr-sparse/kustomization.yaml
```

Expected: the `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` commentary asserts only that
CI's patch makes `seed_override_offsets` / `verify_group_offsets` "stop taking
their unset-guard early return". It does **not** assert the now-conditional
empty-group invariant, so no edit is required (design §7). Change nothing here;
PRD NG4 forbids any functional change to this file.

- [ ] **Step 4: Confirm the new subsection landed in the right section**

```sh
grep -n "^#\{2,3\} " docs/runbooks/sparse-environments.md
```

Expected: `### Re-syncs: an active group is skipped, not re-seeded (task-245)`
appears **between** `## Verifying a consumer group is seeded (FR-4.9, FR-5.3)`
and `### KAFKA_CONSUMER_GROUP must be resolved, post-substitution group names`.

The new text must contain no literal home or absolute path (`/home/…`,
`/Users/…`) — committed docs under `docs/` use repo-relative paths or
placeholders, and this is an enforced repo convention.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/base/atlas-kafka-precreate.yaml docs/runbooks/sparse-environments.md
git commit -m "docs(kafka-precreate): document re-sync skip semantics and the new Job log lines"
```

---

## Final gate (controller, after Task 5)

- [ ] **Flagless verification**

```sh
tools/verify.sh
```

Expected: exit 0. Only the flagless invocation counts. No Go module is touched,
so the relevant guards are the shell/manifest ones and the rendered-manifest
assertions PRD NG4 requires stay untouched.

- [ ] **Live confirmation (design §6.4)**

An Argo CD re-sync of a running **sparse** environment must leave
`atlas-kafka-precreate` `Complete` and the Application `Healthy`.

```sh
kubectl get application -A | grep atlas-pr-<N>
kubectl -n atlas-pr-<N> get jobs
kubectl -n atlas-pr-<N> logs job/atlas-kafka-precreate
```

**`atlas-pr-1441` — the PRD's reproduction case — is no longer a valid
subject.** It has been re-provisioned onto the non-sparse
`deploy/k8s/overlays/pr` and no longer sets `KAFKA_CONSUMER_GROUP` on the Job,
so it would exercise the NG6 early return and prove nothing. The subject must
be an environment on `deploy/k8s/overlays/pr-sparse` whose groups have reached
`Stable` — this branch's own ephemeral environment, if it is provisioned
sparse, once it has been up long enough.
