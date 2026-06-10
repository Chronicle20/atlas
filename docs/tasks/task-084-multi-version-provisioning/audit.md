# Plan Audit — task-084-multi-version-provisioning

**Plan Path:** docs/tasks/task-084-multi-version-provisioning/plan.md
**Audit Date:** 2026-06-10
**Branch:** task-084-multi-version-provisioning
**Base Branch:** main

## Executive Summary

All 12 plan tasks were faithfully implemented with no silent skips, stubs, or TODOs. Every deliverable file exists and matches the plan's intent; the three documented intentional deviations (helper location, `grep -- ` guard, three hardening follow-up commits) are present and benign. All verification gates pass: 59/59 bats tests, `tools/gen-lb-ports_test.sh` → ALL PASS, `gen-lb-ports.sh --check` exit 0, and zero Go/go.mod/go.sum changes (so the Go build/test/bake and redis-key-guard gates are correctly N/A). Recommendation: READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Port-derivation helper + bats | DONE | `services/atlas-pr-bootstrap/scripts/version-ports.sh:10-24` (`derive_login_port`=major×100 with int-reject regex `^[0-9]+$`, `derive_channel_port`=+1); test `version-ports_test.bats:9-31` (5 cases incl. non-int reject). Commit `74e8b66c1`. |
| 2 | Ship version-ports.sh in image + chmod-skip | DONE | `Dockerfile:57` `COPY scripts/version-ports.sh`; chmod line `Dockerfile:66` correctly omits it; `dockerfile_test.bats:34` skips it from chmod check. Commit `715c9c672`. |
| 3 | versions.json (7) + schema | DONE | `versions.json:4-12` has 7 versions (gms 12/83/84/87/92/95, jms 185); `jq '.versions\|length==7'` → true. `versions.schema.json:1-25` draft-07, `additionalProperties:false`, requires region/major/minor. Commit `4bebc8cea`. |
| 4 | gen-lb-ports.sh + tests | DONE | `gen-lb-ports.sh`: two labels `container-ports`/`service-ports` (43-63), `versions_sorted` sort_by major (38-40), dup-major reject (31-35), missing-marker reject incl. END-marker guard (92-96), `--check` diff mode (98-102). Test `gen-lb-ports_test.sh` → ALL PASS (18 assertions incl. dup, missing-marker, missing-END, idempotent). Commits `e4abc4005`, `f643f91fd`. |
| 5 | Markers wrap port blocks in both YAMLs | DONE | `atlas-login.yaml:20-28,54-83` and `atlas-channel.yaml:20-28,54-83`: BEGIN/END markers wrap container + service blocks; `8080` (login:29, chan:29) and `loadBalancerIP` (login:84, chan:84) sit outside markers. Commit `5db9d02d6`. |
| 6 | Regenerate to complete set + idempotent | DONE | login containerPorts = 1200/8300/8400/8700/9200/9500/18500 (+8080); channel = 1201/8301/8401/8701/9201/9501/18501 (+8080) — exact match to plan. `gen-lb-ports.sh --check` exit 0 (idempotent). Commit `bc58b14ed`. |
| 7 | CI drift-check job wired into gate | DONE | `pr-validation.yml:107-116` job; `needs` includes `gen-lb-ports` (480); `LBPORTS_RESULT` capture (497); summary row (507); failure condition (512). Commit `7e7f86256`. |
| 8 | service-config.sh + tests + COPY/skip + trimmed templates | DONE | `service-config.sh:18-50` (`build_login_entry`/`build_channel_entry`/`merge_tenant_entry`); `service_config_test.bats` (7 cases, all pass); `Dockerfile:58` COPY + `dockerfile_test.bats:35` skip; `login-service.json:14` `"tenants": []`; `channel-service.json:24` channel port `0`. Commits `9a5c138b9`, `7742eb130`. |
| 9 | Rewire bootstrap.sh to additive merge | DONE | `bootstrap.sh:25-28` sources both helpers; `upsert_service_config` (285-349) reads live config, builds id-keyed entry per shape, merges onto LIVE attributes (308-314), skip-PATCH-when-equal guard (321-322); three shape call sites login/channel/none (352-358). Commit `33ac33236`. |
| 10 | Image bake validation (verification-only) | DONE | COPY/source chain verified statically: `Dockerfile:57-58` COPY both sourced helpers; `service-config.sh:9-14` resolves version-ports.sh from `/atlas` at runtime; `bootstrap.sh:26,28` sources `/atlas/version-ports.sh` + `/atlas/service-config.sh`. No commit expected (plan: commit only if bake surfaced a fix); none made — correct. |
| 11 | Runbook §9.14 + onboarding note | DONE | `ephemeral-pr-deployments.md:585-628` §9.14 add-a-version + additive-bootstrap guarantee + coexistence repro; `onboarding.md:61-69` version-derived ports note. Commits `2b3ef11c7`, `9507638a5`. |
| 12 | Final verification gate | DONE | bats 59/59 pass; `gen-lb-ports_test.sh` ALL PASS; `--check` exit 0; no Go changes. |

**Completion Rate:** 12/12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No SKIPPED, PARTIAL, DEFERRED, or NOT_APPLICABLE tasks.

## Intentional Deviations (verified, not gaps)

1. Shared helper lives at `services/atlas-pr-bootstrap/scripts/version-ports.sh` (not `tools/lib/...`) — documented in plan.md §13-19; bake context forces it. Both consumers source the one physical file (`gen-lb-ports.sh:18,23` repo-relative; `bootstrap.sh:26` via `/atlas`). Confirmed.
2. `gen-lb-ports_test.sh:16` `assert_contains` uses `grep -qF -- "$2"` (the `--` guard) — a real bug fix for needles starting with `-`. Confirmed.
3. Three hardening follow-up commits beyond the 12 tasks: `f643f91fd` (END-marker guard + `trap ... RETURN` temp cleanup, `gen-lb-ports.sh:83,94-96`), `7742eb130` (`unset _sc_dir` + id-contract comment, `service-config.sh:15,40`), `9507638a5` (runbook §9.14 numbering). All polish, no scope change. Confirmed.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-pr-bootstrap (shell) | N/A | PASS | `bats services/atlas-pr-bootstrap/test/` → 59/59 ok |
| tools/gen-lb-ports (shell) | N/A | PASS | `tools/gen-lb-ports_test.sh` → ALL PASS (18 assertions) |
| LB manifest drift | N/A | PASS | `tools/gen-lb-ports.sh --check` → exit 0 |
| Go modules | N/A | N/A | `git diff --name-only main...HEAD \| grep .go/go.mod/go.sum` → empty; Go/bake/redis-key-guard gates correctly N/A |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None.
