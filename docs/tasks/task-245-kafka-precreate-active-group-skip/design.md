# Kafka Precreate — Skip Offset Seeding for Active Consumer Groups — Design

Task: `task-245-kafka-precreate-active-group-skip`
PRD: [`prd.md`](./prd.md) (v1, approved)
Status: Draft
Created: 2026-08-21

---

## 1. Scope

One shell script (`deploy/k8s/base/kafka-precreate.sh`), its test harness, one
YAML header comment, one overlay comment, and one runbook section. No Go code,
no manifest spec change, no CI change (PRD NG2–NG6).

---

## 2. Measured behaviour of the Kafka CLI — and a correction to the PRD

Everything below was measured against the live broker (`kafka/kafka-broker-0`,
`/opt/kafka/bin/kafka-consumer-groups.sh`) on 2026-08-21, not recalled. The
design rests on these four facts.

### 2.1 `--describe --state` output shape

```
$ kafka-consumer-groups.sh --bootstrap-server <b> \
    --group "atlas-portals - environments - bc1c7c72-e96d-450a-b0c6-d0c16d43d6ea" \
    --describe --state | cat -A

$
GROUP                                                               COORDINATOR (ID)          ASSIGNMENT-STRATEGY  STATE                #MEMBERS$
atlas-portals - environments - bc1c7c72-e96d-450a-b0c6-d0c16d43d6ea 192.168.23.230:9092  (1)  range                Stable               2$
```

An `Empty` group renders the same shape with `-` in the strategy column — the
column is never blank, so the token count is stable:

```
Channel Service - 446b25d6-... [a435] - projection - 3e0f7e7d-... 192.168.23.230:9092  (1)  -                    Empty                0
```

**Consequences for parsing:**

- The row always ends in exactly five whitespace-separated tokens:
  `COORDINATOR` `(ID)` `ASSIGNMENT-STRATEGY` `STATE` `#MEMBERS`.
  So `STATE = $(NF-1)` and `#MEMBERS = $(NF)`, regardless of how many tokens
  the group name contributes. This is the same `NF`-anchored idiom
  `verify_group_offsets` already documents and uses (FR-1.5).
- **The first line is blank and the second is the header** — and the header's
  own `$(NF-1)` is the literal string `STATE`. A naive `NF`-anchored awk would
  therefore "find" a state of `STATE`. Token *count* cannot discriminate
  header from data either: a single-token group name yields `NF=6`, exactly
  the header's `NF`. The parser MUST exclude the header explicitly
  (`$(NF-1) != "STATE"`), not by line number or field count.

### 2.2 A nonexistent group exits **0**

```
$ kafka-consumer-groups.sh ... --group "atlas-task-245-nonexistent-probe [zzzz]" --describe --state
EXIT=0
--STDOUT--

Error: Executing consumer group command failed due to
org.apache.kafka.common.errors.GroupIdNotFoundException: Group ... not found.
--STDERR-- java.util.concurrent.ExecutionException: ...GroupIdNotFoundException...
```

Exit code 0, no header, no data row, an `Error:` line on **stdout** and a Java
stack trace on stderr. So "group absent" is naturally indistinguishable from
"probe produced nothing" — and both map to *seedable* (FR-1.2, FR-1.4). The
probe needs no special-casing for absence.

### 2.3 `--reset-offsets --execute` against an active group also exits **0**

This is the fact that changes the design.

```
$ kafka-consumer-groups.sh ... --group "atlas-portals - environments - bc1c7c72-..." \
    --topic environments --topic EVENT_TOPIC_MAP_STATUS-main --topic COMMAND_TOPIC_GUILD-main \
    --reset-offsets --to-latest --execute
EXIT=0
--STDOUT--

Error: Assignments can only be reset if the group 'atlas-portals - environments - bc1c7c72-...'
is inactive, but the current state is Stable.

GROUP                          TOPIC                          PARTITION  NEW-OFFSET
--STDERR-- (empty)
```

Confirmed for both the single-`--topic` and the repeated-`--topic` production
form, and for `--dry-run` as well as `--execute`. Exit 0. The refusal text goes
to **stdout**, which today's `seed_group` sends to `/dev/null`.

**The PRD's §1 causal claim — "`set -euo pipefail` turns that non-zero exit
into a Job failure" — is not what the CLI does.** Today `seed_group` against a
live group is a *silent no-op that reports success*. The Job cannot fail there.

The Job failure therefore comes from the next pass: `verify_group_offsets`
walks the full 170-topic union, finds a union topic on which the group carries
no committed offset, prints `FAIL: group '<g>' has no committed offset on topic
'<t>'` and `exit 1`. That is the only `exit 1` in the script, and it is
reachable exactly when a reset was silently refused (or when the union grew
past what the group was originally seeded on).

Corroborating measurement — the surviving sparse-era group from the PRD's
reproduction case still shows the shape that trips that gate:

```
group "Channel Service - 29047f69-1403-5c50-9cbb-deb8ce907296 [f8c5]"
  union (atlas-pr-1441 atlas-env COMMAND_/EVENT_TOPIC_*): 170 topics
  topics with a committed offset:                         138
  union topics with NO committed offset:                   31
```

`atlas-pr-1441` has since been re-provisioned onto the non-sparse
`deploy/k8s/overlays/pr` (Argo `spec.sources[0].path`), where
`KAFKA_CONSUMER_GROUP` is unset on the Job and the whole pass takes its NG6
early return — which is why the Application reads Healthy today and why the
original Job log (`ttlSecondsAfterFinished: 600`) is no longer retrievable to
confirm the `FAIL:` line directly. The above is the evidence-supported
reconstruction, stated as such.

**What this changes, and what it does not.** Every acceptance criterion in the
PRD stands unchanged. What changes is *which* change is load-bearing:

| PRD framing | Reality |
|---|---|
| FR-2 (reset-error classification) is the fix | FR-2 is a **defence-in-depth** guard. It must classify on the *message*, not the exit code, because the exit code is 0. |
| FR-3 (verify skips skipped groups) is symmetry | FR-3 is **the fix**. It is what stops the Job failing. |
| FR-1 (state probe) prevents the failure | FR-1 prevents the *silent no-op* and produces the observability FR-4 requires. Without it the reset is a lie. |

A design built only on FR-2's "non-fatal reset" would have shipped a no-op and
left the Job red.

### 2.4 Fatal reset failures still exit non-zero

Not separately re-measured here; unchanged from today's behaviour, which is
what FR-2.3 requires be preserved. The classifier below keys the *non-fatal*
branch off the refusal message and leaves everything else — any non-zero exit
without that message — fatal, so a broker-unreachable or authorization failure
keeps failing the Job exactly as it does now. The fatal set narrows by exactly
one identified condition (NFR "Failure isolation").

---

## 3. Architecture

Four functions, one new pure classifier, one shared skip set.

```
main
 ├─ precreate_topics                     (unchanged — NG3)
 ├─ seed_override_offsets
 │    ├─ [KAFKA_CONSUMER_GROUP unset] → return 0            (NG1, unchanged, first)
 │    ├─ skipped_groups=$(mktemp)                            ← shared state
 │    └─ per group:
 │         state=$(group_state "$group")                     ← FR-1.1  (1 JVM start)
 │         if state_is_seedable "$state"                     ← FR-1.2/1.3, PURE
 │            then rc=0; seed_group ... || rc=$?             ← FR-2
 │                 rc=0 → seeded
 │                 rc=2 → refused-as-active → record skip    ← FR-2.1/2.2 (race)
 │                 rc=* → fatal                              ← FR-2.3
 │            else record skip; log state                    ← FR-1.3, FR-4.1
 └─ verify_group_offsets
      ├─ [KAFKA_CONSUMER_GROUP unset] → return 0            (FR-3.4, unchanged)
      └─ per group: described=$(--describe)                  (1 JVM start, as today)
           in skip set → WARN-only report                    ← FR-3.1, FR-4.2/4.3/4.4
           else        → hard gate, exit 1 on any gap        ← FR-3.3, unchanged
```

The topology deliberately keeps the JVM-cold-start budget at **O(groups)**:
one `--describe --state` added per group, one `--reset-offsets` per group as
today, one `--describe` per group as today. The task-232 O(groups) collapse is
preserved (NFR "Performance").

### 3.1 New: `state_is_seedable <state>` — a pure classifier

Takes a state token, returns 0 (seedable) or 1 (active). No Kafka call, no I/O.

```
Empty | Dead | "" (unknown / absent / probe failed)  → seedable   (FR-1.2, FR-1.4)
anything else                                        → active     (FR-1.3)
```

An **allowlist**, not a denylist. A state Kafka adds in a future version falls
into "active" and is skipped — the safe direction, because skipping never
mutates offsets (FR-5.2), whereas a denylist would reset a group in an
unrecognised live state.

Splitting this out of the Kafka call is the single most valuable testability
decision in the design: it makes the FR-1.2/FR-1.3 classification assertable
**with no broker**, so it runs on every developer machine and in every CI job,
not only in the `BOOTSTRAP_SERVERS`-set path that today's seed assertions live
behind. The PRD's goal "covered by `atlas-kafka-precreate_test.sh` so it cannot
silently regress" is otherwise only half-met.

### 3.2 New: `group_state <group>` — the probe

One `kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --group
"<g>" --describe --state` via `$KAFKA_BIN` (FR-1.1 — a bare name exits 127 in
`apache/kafka:3.7.2`). Echoes the state token, or the empty string.

Parsing, per §2.1:

```awk
NF>=6 && $(NF-1)!="STATE" { print $(NF-1); exit }
```

- `NF>=6` — a data row is at minimum one group token plus the five trailing
  columns; the leading blank line and any `Error:`/stack-trace line falls out.
- `$(NF-1)!="STATE"` — excludes the header, which is otherwise
  indistinguishable for a single-token group name (§2.1).
- `exit` after the first match — one row is authoritative.

The whole call is wrapped in a narrow `set +e` region and `2>/dev/null`; any
non-zero exit, any unparseable output, and the §2.2 absent-group case all
collapse to the empty string, which `state_is_seedable` calls seedable
(FR-1.4). The probe can never itself fail the Job.

**Why parse `STATE` (`$(NF-1)`) rather than `#MEMBERS > 0` (OQ-2).** Both are
`NF`-anchored on the same measured output and both discriminate correctly
(`Empty`→0 members, `Stable`→n). `STATE` wins on three counts: FR-4.1 requires
the log line to *name the state*, so it must be read regardless; a numeric
comparison on `$(NF)` needs its own non-numeric guard for the header
(`#MEMBERS`) and for absent output, whereas the string allowlist rejects
garbage for free; and `#MEMBERS` collapses `Dead` and `Empty` and any future
zero-member-but-active state into one bucket, discarding information the
allowlist uses. Decision: parse `STATE`; do not read `#MEMBERS`.

### 3.3 Changed: `seed_group` — classify on the message, not the code

Signature and multi-`--topic` collapse unchanged, so the existing single- and
multi-topic assertions keep working verbatim (FR-5.3).

```sh
seed_group() {
    group="$1"; shift
    topic_args=""
    for topic in "$@"; do topic_args="$topic_args --topic $topic"; done

    set +e
    seed_out="$("$KAFKA_BIN/kafka-consumer-groups.sh" \
        --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$group" $topic_args --reset-offsets --to-latest --execute 2>&1)"
    seed_rc=$?
    set -e

    case "$seed_out" in
        *"Assignments can only be reset if the group"*) return 2 ;;   # FR-2.2
    esac
    if [ "$seed_rc" -ne 0 ]; then
        printf '%s\n' "$seed_out" >&2                                  # FR-2.3
        return "$seed_rc"
    fi
    return 0
}
```

Three decisions worth stating:

- **The message case runs before the exit-code check, and does not require a
  non-zero code.** Per §2.3 the refusal exits 0. Ordering it first is what
  makes the classifier correct rather than decorative.
- **The substring excludes the group name and the state.** Kafka's text is
  `Assignments can only be reset if the group '<name>' is inactive, but the
  current state is <State>.` Matching only the invariant prefix survives a
  changed quoting style, a new state name, and a group name containing glob
  metacharacters (`[f8c5]` would be a `case` pattern hazard on the *pattern*
  side; it is on the *subject* side here, which is safe, but the shorter
  pattern avoids the question).
- **`2>&1` capture replaces `>/dev/null`** (FR-2.4). The success path stays
  silent because the capture is only printed on the fatal branch.
- `set +e` spans exactly one command substitution and its `$?` read — the
  minimum region NFR "Failure isolation" allows.

Return contract: `0` seeded · `2` refused-because-active · anything else fatal.
Callers must use `rc=0; seed_group ... || rc=$?` so `set -e` does not abort the
loop on a deliberate `2`.

### 3.4 Changed: `seed_override_offsets` — probe, skip, record

The `KAFKA_CONSUMER_GROUP`-unset early return stays byte-identical and stays
first, ahead of any Kafka call (NG1, FR-3.4). Everything else is added after it.

Shared state (FR-3.2): a `skipped_groups` temp file, created here, one group
name per line — the same "globals by convention" the pass already uses for
`$topics` / `$compact_topics`. Membership is tested with
`grep -Fxq -- "$group" "$skipped_groups"`: fixed-string, whole-line, `--`
terminated, so a name containing spaces, brackets, or a leading dash is exact.

Counters `seeded_count` / `skipped_count` feed FR-4.5.

### 3.5 Changed: `verify_group_offsets` — one describe, two behaviours

`verify_group_offsets` already issues exactly one `--describe` per group and
already extracts `CURRENT-OFFSET` via the `NF`-anchored awk. The change is that
the *verdict* on the extracted set branches on skip-set membership:

- **Not skipped** → unchanged. Missing or `-` offset on any union topic →
  `FAIL: ...` to stderr, `exit 1` (FR-3.3).
- **Skipped** → collect the union topics with no committed offset. Empty set →
  `skipped group '<g>': carries a committed offset on all <N> topics` (FR-4.4).
  Non-empty → a `WARN:` line naming them, and **continue** (FR-4.2).

**FR-4.2's reporting lives here, not in `seed_override_offsets`, on purpose.**
The data needed is precisely the per-group `--describe` that this function
already pays for. Computing it in the seeding pass would add a second
`--describe` per skipped group — a gratuitous JVM cold start, and a second
place where the `NF`-anchored parse has to stay correct.

Guard: `skipped_groups` may be unset when the function is reached — a test
sources the file and calls `verify_group_offsets` without having run
`seed_override_offsets`. Treat unset/missing as "nothing skipped".

### 3.6 Logging (FR-4)

| Condition | Line |
|---|---|
| FR-4.1 skip | `skipping group '<g>': already active (<State>) — offsets already initialized` |
| FR-2.2 race skip | `skipping group '<g>': reset refused, group became active during seeding — offsets already initialized` |
| FR-4.4 clean skip | `skipped group '<g>': committed offsets present on all <N> topics` |
| FR-4.2/4.3 gap | `WARN: skipped group '<g>' has no committed offset on <K> of <N> topics: <t1>, …, <t10> (+<K-10> more)` |
| FR-4.5 summary | `override consumer group offsets seeded (<S> seeded, <K> skipped)` |
| OQ-4 | `all <K> override consumer groups were already active — nothing seeded this run (re-sync no-op)` |

**OQ-1 (bounding) — resolved: bound at 10 names plus a `+N more` count.** The
union is 170 topics and the seeding superset is deliberate (NG5), so a genuine
gap list is either tiny (a topic added mid-life) or large enough that the count
is the signal and a full dump would bury the rest of the Job log. Ten names is
enough to recognise a pattern (one family, one service's topics) without
scrolling. FR-4.3's "name the topics rather than only a count" is satisfied:
names come first, the count qualifies them.

**OQ-4 — resolved: yes**, emit the distinct all-skipped line. It is the single
line that makes "this run did nothing, by design" legible in Argo CD's Job log
view, which is the exact triage question this task exists to answer.

---

## 4. Alternatives considered

### A. Blanket `|| true` on the reset (rejected)

One character, fixes the symptom.

Rejected on three counts. It violates FR-2.3 explicitly — broker-unreachable
and authorization failures would stop failing the Job, and a sparse
environment would silently come up with an unseeded group replaying main's
entire retention window, which is precisely the failure task-232 Task 45
existed to prevent. It produces no FR-4 observability: the log cannot
distinguish "seeded" from "silently did nothing". And per §2.3 it **would not
have fixed the bug at all** — the reset already exits 0; the failure is in
`verify_group_offsets`.

### B. Make `verify_group_offsets` non-fatal for all groups (rejected)

Also fixes the symptom, and is smaller than the chosen design.

Rejected because `verify_group_offsets` *is* the activation gate (task-232
FR-5.3): a completed, healthy wave-0 Job is the observable readiness signal
that holds every Deployment at wave 10 until seeding is proven. Making it
advisory for every group deletes that signal for the fresh-environment case
too, where it is load-bearing and correct. The chosen design narrows the gate
to exactly the groups where it is unprovable (active ones), preserving it
everywhere it can still be enforced.

### C. Detect activity from `--describe`'s `CONSUMER-ID` column (rejected)

`verify_group_offsets` already runs `--describe`; a live member shows a
non-`-` `CONSUMER-ID`, so activity could be inferred from a call already being
made — zero added JVM starts.

Rejected because the inference is needed in `seed_override_offsets`, which runs
*before* `verify_group_offsets`, so the "free" call is not yet available and
would have to be hoisted — restructuring both functions to share a describe
cache in order to save ~1.5s per group on a Job that already runs for minutes.
It also infers rather than observes: `--describe --state` reports the state
Kafka will actually enforce in `--reset-offsets`, whereas `CONSUMER-ID`
presence is a proxy that diverges during rebalance. NFR "Correctness / safety"
asks for the skip to be structurally true, not probable. Rejected for
directness, not cost.

### D. `--reset-offsets --dry-run` as the probe (rejected)

Run the dry-run first; if it prints the refusal, skip.

It is the most faithful possible probe — the identical code path Kafka uses to
refuse. Rejected because it costs the same JVM start as `--describe --state`
while yielding strictly less information (no state token for FR-4.1, and per
§2.3 the same exit code either way), and because passing 170 `--topic` flags to
a call whose only purpose is to be refused is a confusing thing to leave in a
script. §3.3's message classifier already covers the case D would catch, in the
one place it can also catch the post-probe race.

### E. Turn the Job into an Argo CD `PreSync` hook (rejected — NG2)

Out of scope by the PRD, and it would not help: the Deployments are already
running during a re-sync regardless of when in the sync the Job fires.

---

## 5. Portability constraints

`kafka-precreate.sh` is **executed** by `bash` in `apache/kafka:3.7.2` and
**sourced** by `atlas-kafka-precreate_test.sh`, whose shebang is
`#!/usr/bin/env sh` (dash on most Linux hosts). Sourcing parses the entire
file, so a bash-only *syntax* construct is a parse error in the test even
though the surrounding function is never called there.

Binding rules for the new code:

- **No arrays.** `skipped=(...)` is a dash parse error. The skip set is a temp
  file; counters are plain integers.
- **No `local`.** The file already uses globals throughout; keep it.
- **No `[[ ]]`, no `+=`, no `$'...'`.** Use `case`/`[ ]`, string concatenation,
  `printf`.
- **`$KAFKA_BIN` full paths on every Kafka CLI invocation** (FR-1.1 / the
  exit-127 crash-loop the file's header comment documents).
- Existing bash-isms (`compgen`, `${!var}`) stay where they are — they are
  runtime calls inside `precreate_topics`, which the test never invokes, and
  they parse today. Do not add more.
- Preserve existing line endings (repo convention).

The `if ! (return 0 2>/dev/null); then main "$@"; fi` executed-vs-sourced probe
is untouched.

---

## 6. Testing

`atlas-kafka-precreate_test.sh` has two tiers: assertions above the
`BOOTSTRAP_SERVERS` guard that always run, and broker-backed assertions below
it that `SKIP` cleanly. The design puts as much as possible in the first tier.

### 6.1 No-broker tier (always runs)

1. **Existing NG6 assertion** — unchanged, still first, still passing.
2. **`state_is_seedable` truth table** (new). `Empty`, `Dead`, `""` → seedable.
   `Stable`, `PreparingRebalance`, `CompletingRebalance`, and a synthetic
   future token (e.g. `SomeNewState`) → active. This is the FR-1.2/FR-1.3
   contract, and it is the assertion that makes the allowlist-not-denylist
   decision (§3.1) permanent.

### 6.2 Broker tier (`BOOTSTRAP_SERVERS` set)

3. **Existing single-topic and multi-topic `seed_group` assertions** —
   unchanged, must still pass (FR-5.3).
4. **Active-group skip** (FR-5.2, PRD acceptance criterion). Create a topic,
   produce, `seed_group` to establish a known committed offset, record it.
   Start `kafka-console-consumer.sh --group <g> --topic <t>` in the background;
   poll `group_state` in a bounded loop (fixed attempt cap, not an unbounded
   `while`) until it reports `Stable`, failing the test if the cap is reached.
   Produce more messages so a successful reset would visibly move the offset.
   Then assert: `group_state` reports an active state; `state_is_seedable`
   says no; `seed_group` returns 2 and does **not** abort the script under
   `set -eu`; and the committed offset is byte-identical to the recorded one.
   Kill the consumer in a `trap`-registered cleanup so a mid-test failure does
   not leave a consumer attached to the shared broker.

   The offset-unchanged assertion is the one that proves FR-5.2 — the safety
   property. `seed_group` returning 2 without an offset change is what "a
   routine re-sync does not fast-forward a live environment past unprocessed
   messages" means operationally.
5. **Idempotence** (FR-5.1). Prefer driving `main` twice: export two
   `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` vars pointing at the test topics, set
   `KAFKA_CONSUMER_GROUP` to the still-active test group, and assert both runs
   exit 0 and the group's committed offset is unchanged across them. `main` is
   already sourced and callable, so this needs no new seam. If the harness
   proves impractical (`precreate_topics` requires `compgen`, i.e. it must run
   under bash — so this assertion may need `bash "$0"` re-exec or a bash-only
   sub-case), fall back to the PRD-sanctioned function-level equivalent:
   `seed_override_offsets` then `verify_group_offsets` twice in sequence,
   asserting exit 0 and no offset movement. Decide at plan time; the fallback
   is not a scope reduction, it is the same property at a lower seam.

   Note this assertion is also the FR-3.1 regression test: with the active
   group in the skip set and a union topic it has no offset on,
   `verify_group_offsets` must warn and exit 0 rather than `exit 1`.

### 6.3 Repo gate

Flagless `tools/verify.sh` must exit 0. No Go module is touched, so the
relevant guards are shellcheck/formatting and the rendered-manifest assertions
that NG4 requires stay untouched.

### 6.4 Live confirmation

An Argo CD re-sync of a running **sparse** environment leaves
`atlas-kafka-precreate` `Complete` and the Application `Healthy`. Note that
`atlas-pr-1441` — the PRD's reproduction case — has been re-provisioned onto
the non-sparse `deploy/k8s/overlays/pr` and no longer sets
`KAFKA_CONSUMER_GROUP` on the Job (§2.3), so it is **no longer a valid
subject**: it would exercise the NG6 early return and prove nothing. The
subject must be an environment on `deploy/k8s/overlays/pr-sparse` whose groups
have reached `Stable` — this branch's own ephemeral environment, if it is
provisioned sparse, once it has been up long enough.

---

## 7. Documentation changes

| File | Change |
|---|---|
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Header comment: "while each group is still empty and therefore resettable" becomes conditional — empty on a first sync, already-active-and-therefore-already-initialized on a re-sync, skipped in that case. No spec change. |
| `deploy/k8s/base/kafka-precreate.sh` | `seed_group`'s own header comment carries the same correction, plus the §2.3 measurement (the refusal exits 0) so the next reader does not re-derive it. |
| `deploy/k8s/overlays/pr-sparse/kustomization.yaml` | The `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` commentary asserts only that the guard stops taking its early return; it does not assert the now-conditional empty-group invariant. Comment change only if a re-read at implementation time finds otherwise (NG4 — no functional change either way). |
| `docs/runbooks/sparse-environments.md` | §"Verifying a consumer group is seeded": document re-sync semantics, the three per-group log outcomes and how to read them, that a live group cannot be re-seeded and what that means for a topic added mid-life (the consumer's own `auto.offset.reset` governs), and — per OQ-3 — that any other shape reaching the pass with a pre-existing active group (a re-created override reusing a previous env's `ATLAS_ENV` suffix, a switched baseline) is covered by the same skip. |

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| A first sync races: a Deployment joins a group before wave 0 seeds it, so it is skipped and never seeded at all. | Argo CD holds wave 10 behind the wave-0 Job, so this requires a pre-existing environment — which is the re-sync case, where skipping is correct. The FR-4.2 warning surfaces it if it ever happens otherwise. |
| The refusal message text changes in a future Kafka version, so the FR-2.2 classifier stops matching. | The classifier is defence-in-depth behind the FR-1 state probe (§2.3), and the fatal branch is the safe direction — a Job failure, not a silent bad reset. `state_is_seedable`'s allowlist is version-independent. |
| The `NF`-anchored parse breaks on a Kafka version that adds a trailing column. | Same allowlist safety: an unrecognised token → seedable → the reset is attempted → the message classifier catches an active group. Two independent mechanisms must both fail to produce a wrong outcome. |
| Test 4 leaves a console consumer attached to a shared broker. | `trap`-registered cleanup; bounded poll loop with a hard attempt cap. |

---

## 9. Resolved open questions

- **OQ-1** — Bound the FR-4.2 topic list at 10 names plus `+N more`. §3.6.
- **OQ-2** — Parse `STATE` at `$(NF-1)`, not `#MEMBERS`; measured output and
  reasoning in §2.1 and §3.2. The header must be excluded by
  `$(NF-1)!="STATE"`, not by field count.
- **OQ-3** — Yes: a re-created override reusing a previous environment's
  `ATLAS_ENV` suffix, or a switched baseline, reaches the pass with a
  pre-existing active group. The same skip covers it; the runbook says so (§7).
- **OQ-4** — Yes: emit a distinct all-skipped terminal line. §3.6.
