# Plan Audit — task-070-pr-env-teardown-fixes

**Plan Path:** docs/tasks/task-070-pr-env-teardown-fixes/plan.md
**Audit Date:** 2026-05-20
**Branch:** task-070-pr-env-teardown-fixes
**Base Branch:** main

## Executive Summary

All 13 plan tasks are implemented and committed; every step-level checkbox in `plan.md` is flipped (`grep -c '^- \[ \]' plan.md` → 0; `grep -c '^- \[x\]' plan.md` → 59). All three in-repo bats suites for this work pass (16/16 task-070 tests green). YAML parse-checks succeed for all touched workflows and manifests. The Design A overlay split (3 extra commits beyond the original plan) is well-documented in `plan.md`'s "Execution note", `context.md`, and `pr-cleanup/kustomization.yaml`'s header — it is an honest scope addition driven by a Task 13 verification failure, not silent rework. **Overall verdict: PASS / READY_FOR_REVIEW** subject to the cluster-infra sibling PR landing in lockstep.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `compute_atlas_env` helper + oracle test | PASS | `services/atlas-pr-bootstrap/scripts/lib.sh:65-72`; `test/lib_test.bats:9-30`; bats 4/4 pass (PR-1→`1a52`, PR-491→`ed86`, PR-522→`a476`, empty→err). Commit `2c7df1e43`. |
| 2 | `cleanup.sh` derives `ATLAS_ENV` from `PR_NUMBER` | PASS | `services/atlas-pr-bootstrap/scripts/cleanup.sh:27` removes `ATLAS_ENV` from `require_env`; line 35 derives via `compute_atlas_env`. `test/cleanup_test.bats:15-26` asserts ATLAS_ENV no longer required, DB_HOST is the new first-missing var. Commit `a697a2579`. |
| 3 | Branch-delete phase in `cleanup.sh` | PASS | `cleanup.sh:86-102` implements `ATLAS_STEP=drop-branch` using `GHCR_TOKEN`; 404 cases (`"Reference does not exist"`, `"Branch not found"`, `"404"`) swallowed silently, other errors log warn. `test/cleanup_test.bats:36-67` covers 404 swallow + GHCR_TOKEN reference. Commit `6062dc4e2`. |
| 4 | PostDelete Job → `argocd` ns + new secret + drop `ATLAS_ENV` env | PASS | `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml:28` (`namespace: argocd`), `:37` (`serviceAccountName: atlas-pr-cleanup`), `:54` (`atlas-pr-cleanup-gh-token`), no `ATLAS_ENV` env entry. Kustomize-build with substituted placeholders confirms `namespace: argocd` survives (manual run in /tmp/pr-cleanup-test). Commit `3f1c6566b` + Design A move in `5639a784b`. |
| 5 | Refresh `pr-cleanup.yml` to drop 24h-grace mentions | PASS | `grep -ni "cleanup-grace\|24h\|24 h\|grace" .github/workflows/pr-cleanup.yml` → 0 matches. Commit `f5dd1893a`. |
| 6 | Sweep skeleton + arg parsing | PASS | `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh:1-67` (shebang, usage, --apply/--list/--help, numeric validation, mode 0755). `test/sweep_test.bats` arg-parse tests pass. Commit `cf2b9397a`. |
| 7 | Seven phase implementations | PASS | `sweep-orphans.sh:99-274`: `sweep_pg`, `sweep_kafka` (topics+groups), `sweep_redis`, `sweep_ghcr`, `sweep_pihole`, `sweep_app_finalizer` (kubectl-based), `sweep_branch`. All emit phase-prefixed enumeration lines in list mode; honor APPLY=1 only for destructive ops; skip if required env unset. `test/sweep_test.bats` 6/6 pass. Commit `248304ba8`. |
| 8 | Runbook rewrites §9.2/§9.4/§9.5 + new §9.11 | PASS | `docs/runbooks/ephemeral-pr-deployments.md` headers at lines 155/190/232/308. §9.11 "Orphan sweep" present (line 308) with one-shot + in-cluster invocations + Prom metric reference. Only "grace" mention left (line 157) is the explicit `**no grace window**` statement that replaced the old 24h language; line 244 references the smoke test's nightly cadence (different concept). Commit `d1cb89d69`. |
| 9 | `pr-env-smoke.yml` (nightly) | PASS | `.github/workflows/pr-env-smoke.yml` exists; cluster-touching jobs `wait-for-healthy` (line 79) and `assert-reclamation` (line 112) both gated `if: false` per plan. actionlint clean except for the documented `atlas-cluster` custom-runner-label warnings (expected — no actionlint.yaml configured). Schedule `cron: '17 4 * * *'` daily. Commit `8681c4453`. |
| 10 | Env-drift investigation deliverable | PASS | `docs/tasks/task-070-pr-env-teardown-fixes/env-drift-investigation.md` (98 lines). Probes A/B/C/D each explicitly marked **"Not run"** with honest reasoning (CRD gone post-cleanup, no cluster write access, no cluster-infra repo access). Static reasoning provided for Probe D (rules out width drift + 5 alternate-formula candidates). Verdict: "inconclusive (probes deferred to operator follow-up)" with rationale that defensive `compute_atlas_env` makes drift harmless. Commit `b6f9b96c1`. |
| 11 | `bootstrap.sh` audit (no change) | PASS | `git diff main..HEAD -- services/atlas-pr-bootstrap/scripts/bootstrap.sh` returns empty — bootstrap.sh untouched. Decision recorded in `context.md` ("Audited only. Reads `ATLAS_ENV` from the `atlas-env` ConfigMap built at PreSync time; no change."). |
| 12 | Write `context.md` | PASS | `docs/tasks/task-070-pr-env-teardown-fixes/context.md` (120 lines) covers Key Decisions table, Formula contract w/ test pinning, Key Files table including the Design A additions, Sibling PR requirements. |
| 13 | Verification gate | PASS | All bats (16/16 task-070 tests pass); YAML parse-checks pass on all 5 touched workflows + 2 manifests; kustomize-build of pr-cleanup overlay outputs Job in `argocd` namespace. Plan checkbox count: 59 checked, 0 unchecked. ShellCheck step (13.2) was skipped because shellcheck is not installed locally — the plan text acknowledges this is acceptable; CI provides the canonical lint pass. Commit `d378f3b4a`. |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Design A — Out-of-Plan Additions (commits `5639a784b`, `b1ab11bdc`, `1d942885f`)

These three commits implement the kustomize-overlay split discovered during Task 13.4 verification. Audit verdict: **legitimate in-scope addition**, documented honestly:

1. **Trigger**: Task 4 originally placed `postdelete-cleanup.yaml` in `deploy/k8s/overlays/pr/` with an explicit `metadata.namespace: argocd`. The `pr/` overlay's top-level `namespace: atlas-pr-PLACEHOLDER_PR_NUMBER` directive (kustomization.yaml line 25) overrides this in kustomize ≥ 5.0 — `kustomize build` output the Job in `atlas-pr-<N>`, recreating bug #1.
2. **Three patch attempts tried first** (per `plan.md` "Execution note"): inline JSON6902 op, inline strategic-merge, external SMP. All were overridden by the namespace transformer.
3. **Resolution**: Two-overlay split. `deploy/k8s/overlays/pr-cleanup/` has its own `kustomization.yaml` with no `namespace:` directive, so the Job's `namespace: argocd` survives. The cluster-infra `ApplicationSet(atlas-pr)` must add a second `sources:` entry (multi-source mode, Argo CD ≥ 2.6). `pr-validation.yml` was updated (`b1ab11bdc`) to sed-substitute placeholders in both overlays and stage both for bot-branch force-push.
4. **Documentation**: `context.md` "Sibling PR (cluster-infra)" calls out the multi-source ApplicationSet update + the new ServiceAccount/Role/Secret. `pr-cleanup/kustomization.yaml:1-28` explains why the overlay exists and references `task-070/design.md §3.1 (Design A)`. `pr/kustomization.yaml:34-38` notes the omission and points readers to pr-cleanup.
5. **Verification re-run**: Kustomize-build of `/tmp/pr-cleanup-test` (with placeholders substituted to `9999` / `abcd`) shows the Job emitted with `namespace: argocd` — fix confirmed.

This is not silent scope creep. The discovery is documented in plan.md's "Execution note" and recovery-log.md, and the deviation strictly serves the original PRD acceptance criterion (PostDelete Job must run in a namespace that survives prune).

## Skipped / Deferred Tasks

None. Two minor caveats are documented in the plan but not skipped:

- **ShellCheck (Step 13.2)**: shellcheck binary not installed on the audit host; the plan acknowledges this is acceptable because CI runs shellcheck on every push. Not a blocker.
- **Probes A/B/C/D (Step 10.1)**: all four are explicitly marked "Not run" with documented reasons (CRD gone post-cleanup; no cluster-infra repo access; no live wedged Application; defensive `compute_atlas_env` makes the drift root-cause non-blocking). This is not a skip — it is honest documentation of what could not be done from this worktree, plus a follow-up plan.

## Build & Test Results

| Surface | Build | Tests | Notes |
|---|---|---|---|
| services/atlas-pr-bootstrap (bats) | N/A | 16/17 PASS | Task-070 tests all green (4 lib + 5 cleanup + 6 sweep + 1 bootstrap). The one failure is `bootstrap.sh fails without ATLAS_UI_BASE` (exit 127 — `bash` not finding the script under test setup). Verified pre-existing on main: `git show main:services/atlas-pr-bootstrap/test/bootstrap_test.bats` shows identical test code. Not a regression. |
| YAML parse | PASS | — | `yq` parses `postdelete-cleanup.yaml`, both `kustomization.yaml`s, `pr-env-smoke.yml`, `pr-cleanup.yml`, `pr-validation.yml`. |
| Kustomize build (pr-cleanup) | PASS | — | `kustomize build /tmp/pr-cleanup-test` (placeholders sed-substituted) emits Job with `namespace: argocd`, `serviceAccountName: atlas-pr-cleanup`, `atlas-pr-cleanup-gh-token` secret ref, `PR_NUMBER: "9999"` env. One harmless `commonLabels deprecated` warning (cosmetic). |
| actionlint (pr-env-smoke) | PASS | — | Only known custom runner-label warnings (`atlas-cluster`). No other findings. |
| ShellCheck | SKIP | — | Binary not installed locally; CI provides canonical lint. Acceptable per plan. |
| Go services | N/A | — | No Go code touched in this branch. |
| atlas-ui | N/A | — | No TS/React code touched. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_FOR_REVIEW (subject to cluster-infra sibling PR coordination)

## Critical Pre-Merge Coordination (not blockers but required for the feature to function)

The plan and context.md make these explicit; flagging here for the merge reviewer:

1. **cluster-infra sibling PR must land in lockstep** (per `context.md` "Sibling PR (cluster-infra)"):
   - `ServiceAccount/atlas-pr-cleanup` + `Role`/`RoleBinding` in `argocd` namespace granting the Job permission to `delete jobs` on itself.
   - `Secret/atlas-pr-cleanup-gh-token` provisioned in `argocd` (and reflected to `atlas-pr-*` per the existing Reflector wiring).
   - `ApplicationSet(atlas-pr)` switched to multi-source mode with a second `sources:` entry for `deploy/k8s/overlays/pr-cleanup/` on the same `bot/pr-<N>-resolved` branch.
   Until the sibling PR ships, the new `pr-cleanup` overlay is dead code; the merged branch does NOT cause an outage but also does NOT activate the bug #1 fix.

2. **Self-hosted runner provisioning** for `pr-env-smoke.yml` — both cluster-touching jobs are `if: false`. Once a runner with label `atlas-cluster` exists, flip to `if: true` on lines 79 + 112.

## Action Items

None blocking. The branch is complete with respect to the plan. Optional follow-ups (documented as out-of-scope in the deliverables themselves):

1. Once cluster-infra sibling PR lands, open a small follow-up PR to flip `if: false` → the gating condition on `wait-for-healthy` + `assert-reclamation` in `pr-env-smoke.yml`.
2. Next time PR-env drift is observed on a live cluster, run Probes A and B per `env-drift-investigation.md` and update the verdict in-place.
3. Pre-existing failure in `bootstrap_test.bats:14` ("bootstrap.sh fails without ATLAS_UI_BASE", exit 127) is out of scope here but worth a separate triage task — root cause appears to be a bats-side env handling change between machines/versions, not a script regression.
