# Plan Audit — task-075-pr-teardown-regressions

**Plan Path:** `docs/tasks/task-075-pr-teardown-regressions/plan.md`
**Audit Date:** 2026-05-22
**Branch:** task-075-pr-teardown-regressions
**Base / HEAD:** `1528982c1` → `9062bf837`

## Executive Summary

All 12 plan tasks are implemented faithfully. 31/31 bats tests pass; `libs/atlas-kafka/consumergroup` race tests pass; `go build`/`go vet` clean on every changed module; `docker buildx bake atlas-channel atlas-login` builds clean; `docker build atlas-pr-bootstrap` produces an image that contains `/atlas/sweep-orphans.sh` (executable). The two documented justified deviations from plan — restoring `set -e` after lib.sh source in `bootstrap.sh`, and skipping `lib.sh` in the Dockerfile chmod drift check — are present and correct. The `dev/cluster-infra-coordination/` exception is wired into `.gitignore` so the example ConfigMap is tracked. Generator output is byte-identical to the committed artifact (idempotency confirmed). The pre-existing `go vet` finding in `services/atlas-login/atlas.com/login/socket/init.go:39` predates this branch and is unrelated.

**Verdict:** READY_TO_MERGE.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Phase-runner helpers + unit tests | DONE | `lib.sh:4` (`set -uo pipefail`), helpers at `:74-129`; tests in `test/lib_test.bats:32-97`. Commit `9fbe11f2`. Justified deviation: `bootstrap.sh:9-16` restores `set -e` after sourcing lib.sh (commit `520be7e6`) — preserves strict-fail. |
| 2 | rpk fixtures + jq query constants | DONE | `test/fixtures/rpk-{topic,group}-list.json` + `README.md` (commit `f70cee69`); constants at `lib.sh:144-145`. |
| 3 | cleanup.sh try-all + rpk jq fix | DONE | `scripts/cleanup.sh:17` (`set -uo pipefail`), phase functions `:46-180`, PHASES array `:185-193`, orchestration loop `:196-200`. Commits `9ad7d3bf` + `5ccd520a`. |
| 4 | sweep-orphans.sh ports to rpk + try-all | DONE | `scripts/sweep-orphans.sh:22` + `sweep_kafka:121-157` uses rpk. No kafka-*.sh references remain in `services/atlas-pr-bootstrap/`. Commit `9aa9f61b`. |
| 5 | Dockerfile + drift guard | DONE | `Dockerfile:38-44` COPY+chmod; `test/dockerfile_test.bats` covers COPY (exhaustive) + chmod (skips lib.sh — documented in-line). Commit `f26f01a2`. |
| 6 | `consumergroup.Resolve` variadic | DONE | `resolver.go:38` signature; 6 tests in `resolver_test.go`; all pass under `-race`. Commit `9617f196`. |
| 7 | Atlas-channel + atlas-login call sites | DONE | `atlas-channel/.../main.go:151` and `atlas-login/.../main.go:66` swapped to `Resolve(template, id)`. Commit `1991e3d6`. |
| 8 | gen-cleanup-env.sh + coordination artifact | DONE | `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh` (executable); `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` tracked via `.gitignore` exception. Idempotent. Commit `f5e74dd3`. |
| 9 | postdelete-cleanup envFrom switch | DONE | `postdelete-cleanup.yaml:45-71` — envFrom 3 Secrets + configMapRef atlas-pr-cleanup-env; only PR_NUMBER inline. `gen-consumer-group-patch.sh:30-36` comment rewritten; re-run produces no patch diff. Commit `e37160b4`. |
| 10 | pr-validation CI integration | DONE | `.github/workflows/pr-validation.yml:344-356` — new "Regenerate cluster-infra coordination ConfigMap artifact" step with `git diff --exit-code` + `::error::` annotation. Commit `a3f04a2c`. |
| 11 | Runbook updates | DONE | §9.4 (line 198) summary-line guidance; §9.11 (349-446) Job-form + in-cluster + envFrom; §9.12 (447) per-phase recipe table; §9.13 (476) cluster-infra coordination. Commit `9062bf83`. |
| 12 | Full local verification | DONE | bats 31/31; `go test -race`/`go vet`/`go build` clean on touched modules; `docker buildx bake atlas-channel atlas-login` succeeds; `docker build` of atlas-pr-bootstrap (support-image, not in bake) produces image containing `/atlas/sweep-orphans.sh`. |

**Completion Rate:** 12 / 12 (100%). Skipped without approval: 0. Partial implementations: 0.

## Build & Test Results

| Target | Build | Tests |
|---|---|---|
| `services/atlas-pr-bootstrap` bats | n/a | PASS (31/31) |
| `libs/atlas-kafka` | PASS | PASS (race) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS |
| `services/atlas-login/atlas.com/login` | PASS | PASS¹ |
| `atlas-channel` image (bake) | PASS | n/a |
| `atlas-login` image (bake) | PASS | n/a |
| `atlas-pr-bootstrap` image (docker build) | PASS | n/a; `/atlas/sweep-orphans.sh` present + executable |

¹ Pre-existing `go vet` finding `socket/init.go:39` (WaitGroup.Add) confirmed at base commit `1528982c1`; unrelated to this branch.

## Off-Plan Deviations (Documented & Justified)

1. **`bootstrap.sh:9-16`** — re-add `set -e` after sourcing lib.sh. Lib.sh's switch to `set -uo pipefail` (plan Task 1) would unintentionally strip `-e` from bootstrap.sh, whose flow depends on strict-fail. Caught in Task 1 code-quality review and fixed in commit `520be7e6`. Aligns with design intent (strict-fail on bootstrap, try-all on cleanup/sweep).
2. **`test/dockerfile_test.bats:22-42`** — chmod check skips `lib.sh`. lib.sh is sourced, not executed; the executable bit is irrelevant. COPY check (lines 7-20) remains exhaustive.
3. **`.gitignore:1-4`** — `dev/` switched to `dev/*` + explicit `!dev/cluster-infra-coordination/` re-include. Necessary because git cannot re-include a path under a fully-excluded parent dir; without this the Task 10 CI drift-check would fail on every PR.

## Verdict

**READY_TO_MERGE.** Every plan task has matching code, every plan-prescribed test exists and passes, and the three off-plan adjustments are documented in code with rationale that aligns with design intent.
