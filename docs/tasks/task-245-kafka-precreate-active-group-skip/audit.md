# Plan Audit — task-245-kafka-precreate-active-group-skip

**Plan Path:** docs/tasks/task-245-kafka-precreate-active-group-skip/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-245-kafka-precreate-active-group-skip
**Base Branch:** main (merge base `24a33a2e6`)
**Head:** `1abf2e390`

## Executive Summary

All 5 plan tasks were implemented, each in its own commit, and each has an
implementer report, a passing `atlas-reviewer` review (APPROVED /
APPROVED_WITH_FINDINGS, zero blocking findings in every case), and a
recorded `tools/verify.sh --quick` gate. The final flagless `tools/verify.sh`
is recorded as PASS ("All checks passed", 9 checks green) at
`24a33a2e6..1abf2e390`, and I independently re-ran `shellcheck -S error`,
`bash -n`, `sh -n` on both shell files and the no-broker test tier — all
reproduced clean. The broker-backed tier (Task 4's active-group-skip and
idempotence assertions) was run live against a real `apache/kafka:3.7.2`
broker by both the implementer and, independently, the Task 4 reviewer (3x),
with all 9 PASS lines observed each time — this is recorded evidence, not a
self-report I could re-verify in this session (no port-forward established
here). No task was silently skipped or deferred. PRD NG2 (Job spec fields
unchanged) and NG6/NG1 (main inert, `KAFKA_CONSUMER_GROUP`-unset early return
untouched) both hold structurally in the diff and are asserted by the
no-broker test tier. One PRD/plan acceptance criterion — the "Live
confirmation" step in the plan's Final Gate (an actual Argo CD re-sync of a
running sparse environment) — has not yet happened, which is expected: the
branch has not been pushed and no PR/ephemeral environment exists yet.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `state_is_seedable` pure classifier + no-broker truth-table test | DONE | `deploy/k8s/base/kafka-precreate.sh:164-169` (allowlist `Empty\|Dead\|""`); `deploy/k8s/base/atlas-kafka-precreate_test.sh:43-62` (8-row truth table). Commit `6415c5ce6`. Review: `review-task-1.md` — no blocking findings, shellcheck/`bash -n`/`sh -n`/live `sh` run all reproduced clean by the reviewer. |
| 2 | `group_state` probe, `seed_group` message classification (rc 2), skip-and-record `seed_override_offsets` | DONE | `kafka-precreate.sh:142-149` (`group_state`, NF-anchored parse); `:210-236` (`seed_group`, message-keyed classification returning 2 on `"Assignments can only be reset if the group"`); `:280-330` (`seed_override_offsets` probes via `group_state`/`state_is_seedable`, records skips into `$skipped_groups`, tracks `$seeded_count`/`$skipped_count`). Commit `073117395`. Review: `review-task-2.md` — verdict rationale confirms all binding constraints met (dual-use portability, `$KAFKA_BIN` full paths, `rc=0; … \|\| rc=$?` pattern, NG6 early return byte-identical/first); one non-blocking naming-fragility note (`seed_rc` shared variable name), deferred to a follow-up, not blocking. |
| 3 | `verify_group_offsets` — skip-aware, one describe, two verdicts | DONE | `kafka-precreate.sh:354-420` — `group_skipped` computed via `grep -Fxq -- "$group" "$skipped_groups"`; hard `exit 1` gate preserved for non-skipped groups (`:392-395`, FR-3.3); skipped groups downgrade to a `WARN:`/informational line bounded at 10 names + count (`:397-417`, FR-4.2/4.3/OQ-1). Commit `457365ea0`. Review: `review-task-3.md` — APPROVED. |
| 4 | Broker-backed assertions — active-group skip and idempotence | DONE | `atlas-kafka-precreate_test.sh:111-246` — active-group skip test (FR-5.2, lines 124-189: seeds baseline offset, holds group `Stable` via a consumer on a *different* topic, asserts `seed_group` returns exactly `2` and the committed offset is unchanged); two-pass idempotence test (FR-5.1/FR-3.1, lines 191-246: asserts `WARN:` line, `(0 seeded, 1 skipped)`, re-sync no-op line, and a second full pass also exits 0 with no offset movement). Commit `d941053aa`. Review: `review-task-4.md` — APPROVED_WITH_FINDINGS (0 blocking); reviewer independently re-ran the broker tier 3x against a live `apache/kafka:3.7.2` container and observed all 9 PASS lines each clean run. One non-blocking finding: pre-existing `$$`-suffixed topic-name collision on repeated `docker run` invocations against a persistent broker — shown to be a long-standing convention issue, not introduced by this task, reproduced on the *pre-existing* first assertion's topic name, not any new Task-4 topic. |
| 5 | Documentation — Job header comment + operator runbook | DONE | `deploy/k8s/base/atlas-kafka-precreate.yaml` diff (+15/-7): the "while each group is still empty and therefore resettable" claim replaced with the conditional first-sync-vs-re-sync explanation; NG2 spec fields (`backoffLimit: 3`, `ttlSecondsAfterFinished: 600`, `Force=true,Replace=true`, `sync-wave: "0"`) confirmed byte-identical outside the comment block. `docs/runbooks/sparse-environments.md` (+50 lines): new "Re-syncs: an active group is skipped, not re-seeded (task-245)" subsection with the log-line reference table. `deploy/k8s/overlays/pr-sparse/kustomization.yaml` confirmed unchanged (NG4) — no diff for this path. Commit `1abf2e390`. Review: `review-task-5.md` — APPROVED (1 non-blocking: the WARN-table example only shows the `>10` variant, not the `<=10` no-suffix variant — an incomplete example, not an inaccuracy). |

**Completion Rate:** 5/5 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None of the 5 plan tasks were skipped or partially implemented.

One item outside the 5 numbered tasks — the plan's **Final Gate → Live
confirmation** step (an actual Argo CD re-sync of a running `pr-sparse`
environment showing `atlas-kafka-precreate` `Complete` and the Application
`Healthy`) — has not yet been performed. This is expected at this point in
the workflow: `git status` shows the branch has no upstream (`no remote
branch`) and `gh pr list` returns no open PR, so no ephemeral environment
exists yet to observe. This is not a silently-skipped task; it is a
step that structurally requires the branch to be pushed and a sparse
ephemeral environment to run for a while first (per the plan's own note that
`atlas-pr-1441` is no longer a valid subject since it moved off
`pr-sparse`). It should be re-confirmed against this branch's own ephemeral
environment once one exists, and is called out here as an outstanding item
rather than folded into the 5-task completion rate.

Three minor, explicitly non-blocking findings were carried by reviewers to
the final whole-branch review (per `.superpowers/sdd/plan/progress.md`):
1. `seed_rc` is a shared, non-namespaced global between `seed_group` and its
   caller (`kafka-precreate.sh:225,303-305`) — fragile to a future edit, not
   a current defect.
2. `atlas-kafka-precreate_test.sh`'s `$$`-suffixed topic names collide on a
   second `docker run` invocation against the same persistent broker (PID 1
   is stable inside a container) — a pre-existing convention issue in the
   file, not introduced or worsened by this branch.
3. The runbook's WARN-line reference table shows only the `missing_total >
   10` variant with the `(+M more)` suffix, not the `<=10` no-suffix variant
   — an incomplete example, not an inaccuracy.

None of these are blocking; all were explicitly triaged as non-blocking by
their respective task reviewers.

## PRD Non-Goals Verification

- **NG1 / NG6 (main inert, `KAFKA_CONSUMER_GROUP`-unset early return
  unchanged).** `seed_override_offsets` (`kafka-precreate.sh:280-284`) and
  `verify_group_offsets` (`:355-357`) both still return immediately, before
  any Kafka call, when `KAFKA_CONSUMER_GROUP` is unset — unchanged in
  substance from the pre-branch code. Asserted live and reproduced by this
  audit: `sh deploy/k8s/base/atlas-kafka-precreate_test.sh` (no
  `BOOTSTRAP_SERVERS`) prints `PASS: seed_override_offsets skips when
  KAFKA_CONSUMER_GROUP is unset (NG6)` before anything else, exit 0.
- **NG2 (Job spec unchanged).** `git diff 24a33a2e6..1abf2e390 --
  deploy/k8s/base/atlas-kafka-precreate.yaml` touches only the header
  comment block (lines 15-24 pre-change → the new 15-lines-longer comment).
  `backoffLimit: 3`, `ttlSecondsAfterFinished: 600`,
  `argocd.argoproj.io/sync-options: Force=true,Replace=true`, and
  `argocd.argoproj.io/sync-wave: "0"` are all present, unmodified, outside
  the diff hunk (confirmed via `grep -nE
  "backoffLimit|ttlSecondsAfterFinished|Force=true|sync-wave"`).
- **NG3 (topic pre-creation untouched).** `precreate_topics` (lines 40-114)
  falls entirely before the first `+` hunk in the diff; `xargs -P 16` at
  line 83 is unchanged.
- **NG4 (CI group-list resolution untouched).** `git diff … --
  deploy/k8s/overlays/pr-sparse/kustomization.yaml` produces no output — the
  file is byte-identical on this branch. Task 5's brief explicitly directed
  a read-only re-confirmation rather than an edit, and the implementer's
  report / reviewer both confirm no edit was needed or made.
- **NG5 (full-topic-union seeding unchanged).** `seed_override_offsets`
  still builds `$all_topics` from `cat "$topics" "$compact_topics"`
  (`kafka-precreate.sh:290`), the same full-union construction as before.
- **NG6 (no Go/TypeScript changes).** `git diff --stat 24a33a2e6..1abf2e390`
  touches only `deploy/k8s/base/*.sh`, `deploy/k8s/base/*.yaml`,
  `docs/runbooks/sparse-environments.md`, and the task's own
  `docs/tasks/task-245-…/*.md` artifacts. No `.go` or `.ts`/`.tsx` file
  appears in the diff.

## Build & Test Results

No Go or TypeScript changes on this branch, so there is no `go build`/`go
test`/`npm run build` step to run — consistent with the task instructions
and confirmed by `git diff --stat`.

| Check | Result | Notes |
|---|---|---|
| `shellcheck -S error` (both shell files) | PASS | Re-run independently in this audit session: silent, exit 0. |
| `bash -n` (both shell files) | PASS | Re-run independently: both OK. |
| `sh -n` (both shell files) | PASS | Re-run independently: both OK. |
| No-broker test tier (`sh atlas-kafka-precreate_test.sh`) | PASS | Re-run independently: both no-broker PASS lines + `SKIP: BOOTSTRAP_SERVERS unset`, exit 0. |
| Broker-backed test tier (9 PASS lines) | PASS (recorded, not independently re-run this session) | No `kubectl -n kafka port-forward` was established in this audit session per the task instructions. Evidence relied upon: implementer's `task-4-report.md` (one clean run after removing a leftover-topic collision) and the Task 4 reviewer's independent 3x re-run (`review-task-4.md`), both showing all 9 PASS lines and exit 0 against a live `apache/kafka:3.7.2` container. |
| `tools/verify.sh` (flagless) | PASS (recorded) | `.superpowers/sdd/plan/progress.md`: "FLAGLESS GATE: PASS. `tools/verify.sh` (no flags, full merge-base diff 24a33a2e6..1abf2e390) exited 0 — 'All checks passed.' 9 checks green … Go/UI/tools/hook suites skipped as not-applicable." Not independently re-run in this audit session (out of scope per this audit's instructions; the shell/manifest-level checks above were re-run directly instead). |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the plan's own "Live
  confirmation" step, which requires pushing the branch and letting a
  `pr-sparse` ephemeral environment run long enough for its groups to reach
  `Stable` — a post-push step, not a defect in the work done so far)

## Action Items

1. After pushing this branch and opening the PR, perform the plan's Final
   Gate "Live confirmation" step against this branch's own `pr-sparse`
   ephemeral environment (once its groups have reached `Stable`): confirm
   `atlas-kafka-precreate` reaches `Complete` and the Argo CD Application
   stays `Healthy` across a re-sync. `atlas-pr-1441` is no longer a valid
   subject (re-provisioned off `pr-sparse`).
2. Optional follow-up (non-blocking, already triaged by reviewers): rename
   the shared `seed_rc` global to reduce future-edit fragility
   (`kafka-precreate.sh:225,303-305`).
3. Optional follow-up (non-blocking, already triaged by reviewers): make
   `atlas-kafka-precreate_test.sh`'s broker-tier topic names collision-proof
   across repeated `docker run` invocations against a persistent broker
   (e.g. `--delete --if-exists` teardown at the top of the tier, or a
   less PID-dependent uniqueness token).
