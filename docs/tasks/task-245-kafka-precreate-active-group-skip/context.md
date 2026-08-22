# task-245 — Implementation context

Companion to [`plan.md`](./plan.md). PRD: [`prd.md`](./prd.md). Design:
[`design.md`](./design.md).

## What this task actually fixes

The PRD's §1 causal story is wrong in one place, and the design corrects it
(design §2.3) from live measurement against `kafka/kafka-broker-0`:

> `kafka-consumer-groups.sh --reset-offsets --to-latest --execute` against an
> **active** group **exits 0** and prints
> `Error: Assignments can only be reset if the group '<name>' is inactive, but
> the current state is <State>.` to **stdout** — which today's `seed_group`
> sends to `/dev/null`.

So today the reset is a *silent no-op that reports success*, and the Job cannot
fail there. The Job's failure comes from the **next** pass:
`verify_group_offsets` walks the full ~170-topic union, finds a topic the group
has no committed offset on, and `exit 1`s — the only `exit 1` in the script.

Which reorders what is load-bearing:

| PRD framing | Reality |
|---|---|
| FR-2 (reset-error classification) is the fix | **Defence-in-depth.** Must classify on the *message*, not the exit code. |
| FR-3 (verify skips skipped groups) is symmetry | **This is the fix.** Task 3. |
| FR-1 (state probe) prevents the failure | Prevents the *silent no-op* and produces the FR-4 observability. Without it the reset is a lie. |

A change built only on "make the reset non-fatal" would ship a no-op and leave
the Job red. Do not let a reviewer or a later edit collapse this back.

## Key files

| Path | Role |
|---|---|
| `deploy/k8s/base/kafka-precreate.sh` | The change. Tasks 1–3. |
| `deploy/k8s/base/atlas-kafka-precreate_test.sh` | Tasks 1 and 4. Sources the script above. |
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Task 5, header comment only. |
| `docs/runbooks/sparse-environments.md` | Task 5, new subsection at line ~159. |
| `deploy/k8s/overlays/pr-sparse/kustomization.yaml` | Read-only. Confirmed at plan time: its `PLACEHOLDER_PRECREATE_GROUPS_BLOCK` commentary (lines 270–279) asserts only that CI's patch stops the unset-guard early return, **not** the empty-group invariant. No edit needed (PRD NG4). |
| `.github/workflows/pr-validation.yml:1091-1189` | Read-only. Builds `PRECREATE_GROUPS_BLOCK` and asserts the rendered Job carries a resolved `KAFKA_CONSUMER_GROUP`. Untouched (NG4). |

## Decisions carried from design into the plan

- **`state_is_seedable` is an allowlist, not a denylist.** `Empty`/`Dead`/`""`
  seed; everything else skips. A future Kafka state falls to "active", which is
  the safe direction — skipping never mutates a committed offset (FR-5.2).
- **Parse `STATE` at `$(NF-1)`, not `#MEMBERS`** (OQ-2). FR-4.1 needs the state
  named in the log anyway; a string allowlist rejects garbage without a
  numeric guard; `#MEMBERS` collapses `Dead`/`Empty`/future zero-member-active
  into one bucket. The header must be excluded by `$(NF-1)!="STATE"`, **not**
  by line number or field count — a single-token group name yields exactly the
  header's `NF`.
- **FR-4.2's topic report lives in `verify_group_offsets`**, not in the seeding
  pass: that function already pays for the per-group `--describe`. Putting it in
  the seeding pass would add a second JVM cold start per skipped group and a
  second copy of the `NF`-anchored parse.
- **OQ-1 resolved:** bound the missing-topic list at 10 names + `+N more`.
- **OQ-4 resolved:** yes, emit the distinct all-skipped terminal line.
- **JVM cold-start budget stays O(groups)** — one added `--describe --state` per
  group; the task-232 Task 45 O(groups) collapse is preserved.

## Task sizing

Five tasks, each ≤2 files, none crossing a service. No task is deliberately
oversized. The split follows where a reviewer could reject one and approve its
neighbour:

1. Pure classifier + its no-broker truth table.
2. Seeding side of the script (probe + `seed_group` + `seed_override_offsets`).
3. Verification side of the script.
4. Broker-backed tests.
5. Docs.

**Why Task 2 bundles three functions.** `seed_group`'s new return code `2` and
its only caller's `rc=0; … || rc=$?` handling must land in the same commit —
split apart, `set -e` aborts the seeding loop on the deliberate `2`. One file,
one coherent unit.

**Deliberate intermediate state after Task 2.** `seed_override_offsets` skips
active groups while `verify_group_offsets` still hard-fails them, because
nothing consumes the skip set until Task 3. Never shipped; Task 3 closes it.
A reviewer seeing Task 2 in isolation should not flag this as a defect.

## Traps

- **Dual-shell parse.** `kafka-precreate.sh` is *executed* by `bash` but
  *sourced* by a `#!/usr/bin/env sh` (dash) test. Sourcing parses the whole
  file, so a bash-only **syntax** construct is a parse error even inside a
  function the test never calls. No arrays, no `local`, no `[[ ]]`, no `+=`,
  no `$'…'`. Existing `compgen`/`${!var}` inside `precreate_topics` are runtime
  calls that parse fine — leave them, add no more. Always run **both**
  `bash -n` and `sh -n`.
- **`$KAFKA_BIN` full path on every Kafka CLI call.** A bare name exits 127 in
  `apache/kafka:3.7.2` and crash-loops the Job under `set -euo pipefail`.
- **The test's active-group consumer must subscribe to a DIFFERENT topic than
  the one under test.** `kafka-console-consumer.sh --group` auto-commits, so a
  consumer on the target topic would move the very offset the test asserts is
  unchanged, and the FR-5.2 assertion would prove nothing. Committed offsets
  persist independently of subscription, so a consumer on `$TOPIC_D` holds the
  group `Stable` while `$TOPIC_C`'s seeded offset stays put.
- **Driving `main` twice in the test is impractical**: `precreate_topics` uses
  `compgen` (bash) and the harness is `sh`. The plan takes the design's
  sanctioned function-level equivalent — set `$topics`/`$compact_topics`, run
  `seed_override_offsets; verify_group_offsets` twice. Same property, lower
  seam; not a scope reduction.
- **`verify_group_offsets` calls `exit 1`.** In a sourced test that kills the
  test process, so the idempotence assertions run the pair inside a command
  substitution (a subshell) and check its exit status.
- **`$skipped_groups` may be unset** when `verify_group_offsets` is reached —
  a test can source the file and call it without `seed_override_offsets`.
  Unset/missing means "nothing skipped" and the hard gate applies.
- **No blanket `|| true`.** The fatal set narrows by exactly one identified
  condition. Broker-unreachable, authorization and malformed-argument failures
  must still fail the Job (FR-2.3).
- **The `KAFKA_CONSUMER_GROUP`-unset early return stays first and byte-identical**
  in both functions (NG1). The existing NG6 assertion is the regression guard.

## Verification

- Baseline at plan time, both already green and both must stay green:
  `shellcheck -S error deploy/k8s/base/kafka-precreate.sh deploy/k8s/base/atlas-kafka-precreate_test.sh`
  → exit 0; `sh deploy/k8s/base/atlas-kafka-precreate_test.sh` → NG6 PASS then
  `SKIP: BOOTSTRAP_SERVERS unset`, exit 0.
- `tools/shell-guard.sh` (invoked by `tools/verify.sh` as the "shell tooling
  guard") globs **`tools/*.sh` only** — it does not cover `deploy/k8s/base/`.
  So the flagless `tools/verify.sh` will NOT catch a shellcheck regression in
  these two files. Run shellcheck on them by hand at every task, as the plan's
  steps do.
- **The broker tier is a real gate, not an optional extra.** `sh
  atlas-kafka-precreate_test.sh` with no `BOOTSTRAP_SERVERS` exits 0 by
  SKIPping — that is not evidence. Task 4 must be run against a live broker
  (`kubectl -n kafka port-forward svc/kafka-broker 9092:9092`, then
  `BOOTSTRAP_SERVERS=localhost:9092 …`). If no broker is reachable, report
  BLOCKED rather than marking Task 4 done.
- Flagless `tools/verify.sh` must exit 0 before the branch is called done.
- Live confirmation: a re-sync of a **sparse** environment leaves the Job
  `Complete` and the Application `Healthy`. `atlas-pr-1441` is no longer a
  valid subject — it was re-provisioned onto the non-sparse
  `deploy/k8s/overlays/pr`, where `KAFKA_CONSUMER_GROUP` is unset on the Job
  and the whole pass takes its NG6 early return.
