# Plan Audit — task-098-baseline-only-bootstrap

**Plan Path:** docs/tasks/task-098-baseline-only-bootstrap/plan.md
**Audit Date:** 2026-06-14
**Branch:** task-098-baseline-only-bootstrap
**Base Branch:** main

## Executive Summary

All 8 plan tasks were faithfully implemented with file:line evidence for every step. The
diff is confined to exactly the 6 implementation files in the plan's File Structure table
plus the task-098 planning artifacts — no stray edits. Every acceptance gate passes: bats
61/0, shellcheck clean of new findings, kustomize exit 0, and the authoritative grep gate
empty. The two folded-in code-quality cleanups (the `/api/data/process` docstring note and the
`retry 240 10`→`retry 60 5` window comment) are present and correct. No deferred, skipped, or
stubbed work. Verdict: faithful.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Failing tests for baseline preflight | DONE | bootstrap_test.bats:19-94 — `prq_env`/`make_shims`/`write_fixture_tenant` helpers + the 404 fail-fast test (68-81) and 000 MinIO-unreachable test (83-94); existing `require_env` tests kept (7-17). Commit 01e1e5d1c. |
| 2 | Implement preflight in bootstrap.sh | DONE | `CANONICAL_TENANT_JSON` (33), `MINIO_PROBE_RETRIES` (36), `MINIO_PROBE_SLEEP` (37); `baseline_object_status` (129-133), `baseline_reachable`/`BASELINE_PROBE_CODE` (139-143), `probe_baseline_object` (150-156), `preflight_baseline` (164-182); wired call at 187 (`ATLAS_STEP=preflight-baseline preflight_baseline`) immediately after the TENANT_ID shape guard (46-49). Commit a50567cf6. |
| 3 | Remove full-mode; restore-only data step | DONE | Header comment rewritten to task-098 baseline-only (2-12); `WZ_CANONICAL`/`BOOTSTRAP_MODE` defaults deleted (only `MINIO_ENDPOINT` remains, 29); `canonical_baseline_exists`/`resolve_mode` gone (grep empty); `/api/data/wz` wait-ready probe dropped (197 is the lone `/api/data/status` line); tenant-create uses `$CANONICAL_TENANT_JSON` (206-208); data step collapsed to restore-only (404-429, no `case`/`mode`/`full` branch). Commit 26e135d9b. |
| 4 | Clean stale `resolve_mode` in lib.sh | DONE | `log()` comment generalized — "a since-removed mode-resolution helper" replaces the named `resolve_mode` reference (lib.sh:7-12). Commit a1190ddaf. |
| 5 | Remove WZ init container + volume | DONE | sync-bootstrap.yaml diff removes the main container `volumeMounts` (`/opt/wz`), the entire `initContainers:` block (`fetch-wz-canonical`), and the `volumes:` `wz-canonical` emptyDir (29 deletions). Commit d6c985add. |
| 6 | Rewrite ephemeral-PR runbook §9.1 | DONE | Top cross-ref fixed to "baseline-only bootstrap" (8); §9.1 rewritten to "Data provisioning: baseline-only" with fail-fast section + MinIO stand-up; `mc cp atlas.zip` / refresh / BOOTSTRAP_MODE content removed. Commit 05244a240. |
| 7 | Update migration-runbook note | DONE | Impact note rewritten to "baseline-only … fails fast … prerequisite" (canonical-version-migration.md ~29-34); step-4 wording "the ephemeral baseline-only bootstrap" replaces `auto`-mode (~110). `grep auto.?mode` empty. Commit c459ef121. |
| 8 | Full verification sweep | DONE | Gates re-run below; all pass. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No `// TODO`, no stubs, no deferred follow-ups.

## Gate Results

| Gate | Result | Notes |
|------|--------|-------|
| `bats test/` | PASS | 61 pass, 0 fail (includes the 2 new preflight tests + 2 `require_env` + pre-existing lib/sweep/port tests). |
| `shellcheck scripts/*.sh` | PASS | Only PRE-EXISTING findings: SC1091 on sourced files, SC2034 on ATLAS_STEP (seed), SC1010 on `done`, SC2317 in untouched cleanup.sh. No new findings attributable to this branch. `BASELINE_PROBE_CODE` cross-function global is NOT flagged. |
| `kustomize build overlays/pr` | PASS | exit 0; lone warning is the pre-existing `commonLabels` deprecation. |
| Authoritative acceptance grep | PASS | `grep -rnE 'BOOTSTRAP_MODE\|WZ_CANONICAL\|fetch-wz-canonical\|/opt/wz\|resolve_mode\|atlas-canonical/atlas\.zip'` over `services/atlas-pr-bootstrap/` + `deploy/k8s/overlays/pr/` → empty (exit 1). |
| `git diff main...HEAD --stat` scope | PASS | Exactly the 6 impl files + task-098 planning docs. No stray edits. |

### Build & Test Results (per CLAUDE.md service table)

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-pr-bootstrap | N/A | PASS (bats 61/0) | Bash shell-container service, no `go.mod`; Go build/test/bake rules do not apply. shellcheck + kustomize substitute for static checks. |

## Folded-in Cleanups (verified, not regressions)

- `/api/data/process` docstring note removed: the `data_processing_done` doc comment now
  reads "This stability check is used after a baseline restore to wait for atlas-data to finish
  writing the restored documents." (bootstrap.sh:100-101), replacing the old WZ-ingest sentence.
- Window comment corrected `retry 240 10`→`retry 60 5` (bootstrap.sh ~91: "With the `retry 60 5 …`
  call shape, STABLE_REQUIRED=3 gives a ≥ 10 s no-write window").
- No `api/data/process` or `api/data/wz` residual anywhere in bootstrap.sh.

## Plan-vs-reality nuance (expected, not a finding)

The runbook retains one deliberate negative sentence — "There is no `atlas.zip` to upload."
(ephemeral-pr-deployments.md:55) — which is plan-specified verbatim (Task 6 Step 2). The
authoritative gate excludes bare `atlas.zip`, so this is correct by design, not a leak.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None.
