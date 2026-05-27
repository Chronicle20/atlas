# Plan Audit — task-074-docker-build-shared-dockerfile

**Plan Path:** `docs/tasks/task-074-docker-build-shared-dockerfile/plan.md`
**Audit Date:** 2026-05-21
**Branch:** `task-074-docker-build-shared-dockerfile`
**Base Branch:** `main`

## Executive Summary

All 20 plan tasks are implemented faithfully. Every deliverable specified in the plan landed on the branch (shared `Dockerfile`, `docker-bake.hcl`, rewritten `docker-build` composite action, collapsed `pr-validation.yml`/`main-publish.yml` build jobs, rewritten compose blocks, deleted legacy Dockerfiles + helper scripts, rewritten tools, updated `CLAUDE.md`). The four deviations called out by the dispatcher (Tasks 1/5 inline fixes, Task 3 justified skip, `docker-bake.hcl` HCL-evaluator workaround) are each documented in the file headers and/or `design.md` §3.2/§3.3 with explicit rationale and match what was empirically necessary. Working tree is clean, full-fleet `docker buildx bake all-go-services` succeeded out-of-band, and all 55 bake targets exactly mirror `services.json`'s 55 `type=="go-service"` entries.

**Verdict: READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| Pre-A | Verify worktree state | DONE | branch + cwd correct (`git branch --show-current` = `task-074-…`; `pwd` ends `.worktrees/task-074-…`) |
| 1 | Create shared `Dockerfile` at repo root | DONE | `Dockerfile` present, syntax pragma on line 1. Commit `4c216f400` (initial) plus two fix-up commits folded into Task 1 surface: `5fe16a5f9` (drop `go.sum` for `atlas-retry`/`atlas-service` — those libs carry no external deps) and `01fd97d33` (guard `cp config.yaml` because only 3 of 55 services ship one). Both deviations are documented in the `Dockerfile` body comments (lines 29–31, 99–104). |
| 1 (deviation) | go.work synthesis vs plan's `COPY go.work` | ACKNOWLEDGED | Commit `e9ac294c2`. Plan Step 1 wrote `COPY go.work go.work.sum ./`; in practice that failed workspace-load because repo-root `go.work` lists ~50 sibling-service module dirs not present in any single-service build context. Mitigation: inline `RUN { printf ...; for L in <17 libs>; ... } > go.work` (lines 73–89 of `Dockerfile`) — exactly the §3.3 fallback documented in `design.md` and recorded as the empirical outcome in §3.2. |
| 2 | Empirical test: does `go.work` alone resolve atlas-* without `-replace`? | DONE | `design.md` §3.2 "Recorded outcome" updated with the [2026-05-21] entry confirming workspace-only resolution works once the inline `go.work` synthesis lands. Commit thread visible in `design.md` line 210. |
| 3 | (Conditional) Add parameterized `-replace` block | NOT_APPLICABLE | Plan explicitly says skip if Task 2 Step 1 succeeded. Task 2 succeeded post-§3.3 mitigation; skip is justified. `Dockerfile` correctly contains no `go mod edit -replace` block. |
| 4 | Smoke-test built `atlas-account` image | DONE | Per dispatcher note, full-fleet `docker buildx bake all-go-services` succeeded (which transitively builds atlas-account through the shared `Dockerfile`); per-image `Cmd: [/server]` is encoded in `Dockerfile` line 123 and `ExposedPorts 8080/tcp` on line 114. No fix-up commit needed in this task surface. |
| 5 | Create `docker-bake.hcl` at repo root | DONE | `docker-bake.hcl` present (commit `16bb9258f`). Targets: 55 (matches `services.json` go-services exactly — `diff` between bake `--print` keys and `jq` over services.json is empty). |
| 5 (deviation) | HCL `jsondecode(file(...))` workaround | ACKNOWLEDGED | Plan Step 1 specified `locals { config = jsondecode(file(...)) }`. docker buildx bake's HCL evaluator does NOT support `locals` or `file()`. Workaround: inline `go_services = [...]` top-level identifier with header comments (lines 1–18) calling out that `services.json` remains canonical and recommending a CI parity check as follow-up. Empirically verified by running `docker buildx bake --print`. Plan Step 2's "expected 54" was stale text — `services.json` has 55 go-services and the bake file correctly enumerates 55. |
| 6 | Full-fleet local bake of all go services | DONE | Dispatcher confirms `docker buildx bake all-go-services` green; no fixes required after the deviations above. |
| 7 | Replace `.github/actions/docker-build/action.yml` | DONE | File matches plan Step 1 verbatim (input schema, per-target `--set` flag construction, global `*.platform` / `*.cache-from` / `*.cache-to` / `*.output` / `*.attest` flags, `Summary` step). Commit `c0b824549`. |
| 8 | Rewrite `pr-validation.yml` `build-docker` job | DONE | Single bake job `build-docker` (line 134) consumes `detect-changes.outputs.docker-services-matrix`, derives `targets` + `image-name-map` via `jq`, applies per-PR tag (`pr-${PR_NUMBER}-${SHORT_SHA}` when `deploy-env` labeled, else `pr-${PR_NUMBER}` / `pr-dispatch`). `update-pr-overlay` (line 235) and `pr-validation-complete` (line 366) preserved unchanged with correct `needs` graph. Commit `a0b72d997`. |
| 9 | Collapse `main-publish.yml` AMD64/ARM64 matrix jobs | DONE | `build-amd64` (line 118) and `build-arm64` (line 157) collapsed to single bake jobs with `latest-<arch>` + `main-<sha>-<arch>` tags and `atlas-bake-<arch>` cache scopes. `create-manifest` (line 196) and `update-image-tags` (line 245) preserved as per-service matrices. Commit `0e0990a4d`. |
| 10 | Rewrite `deploy/compose/docker-compose.core.yml` build blocks | DONE | 0 remaining `dockerfile: services/atlas-` lines; 52 `SERVICE: atlas-` lines; 53 `dockerfile: Dockerfile$` lines (52 go-services + 1 atlas-assets static service which correctly uses its own service-local `Dockerfile` via `context: ../../services/atlas-assets`). Commit `819f266bc`. |
| 11 | Rewrite `deploy/compose/docker-compose.socket.yml` build blocks | DONE | 0 remaining `dockerfile: services/atlas-` lines; 2 `SERVICE: atlas-` lines (atlas-login, atlas-channel); 2 `dockerfile: Dockerfile$` lines. Commit `418e25210`. |
| 12 | Confirm `docker-compose.yml` requires no changes | DONE | Verification-only; not modified on this branch. |
| 13 | Delete in-scope `services/atlas-*/Dockerfile` files | DONE | `git diff --diff-filter=D` shows 55 per-service Dockerfile deletions (plan said 54; the actual go-service count is 55). The 3 untouched survivors remain: `services/atlas-assets/Dockerfile`, `services/atlas-pr-bootstrap/Dockerfile`, `services/atlas-ui/Dockerfile`. Commit `4f6bab23d`. |
| 14 | Delete every `Dockerfile.dev` and `Dockerfile.debug` | DONE | 43 `Dockerfile.dev` + 43 `Dockerfile.debug` = 86 deletions (matches plan's "~86" estimate). Commit `cfb6202b2`. |
| 15 | Delete `tools/inject-dockerfile-replace.sh` | DONE | File gone from `tools/`. Commit `8b8630c8c`. |
| 16 | Rewrite `tools/build-services.sh` as bake wrapper | DONE | File matches plan Step 1 verbatim: `exec docker buildx bake "$@"`. Commit `d8a1e4ae4`. |
| 17 | Update `tools/import-lib.sh` docstring | DONE | Header block (lines 4–17) matches plan Step 2 text covering go.work + Dockerfile COPY append steps + bake verification line. Commit `3a1835275`. |
| 18 | Update `tools/import-service.sh` docstring | DONE | Header block (lines 4–17) matches plan Step 1 text covering services.json + go.work + bake verification + no-per-service-Dockerfile callout. Commit `f0b7bb121`. |
| 19 | Rewrite `CLAUDE.md` Build & Verification section | DONE | Section updated with bake-centric verification, `all-go-services` callout, and one-place rule for new libs. Commit `c7fd79bfd`. |
| 20 | End-to-end local sanity sweep | DONE | Tree clean; only 3 surviving service Dockerfiles (assets, pr-bootstrap, ui); `docker buildx bake all-go-services` green per dispatcher; `go test -race ./...` clean per dispatcher; `go vet ./...` only has pre-existing `libs/atlas-rest` WaitGroup warnings shared with `main` (not introduced); `go.work.sum` refresh committed as `950256bbd`. |

**Completion Rate:** 20/20 tasks (100%), plus correctly-skipped Task 3 (conditional gate).
**Skipped without approval:** 0
**Partial implementations:** 0

## Acknowledged Deviations

These were called out by the dispatcher as expected, justified deviations from the plan's verbatim text. None are gaps.

1. **`Dockerfile` synthesizes `go.work` inline** (commit `e9ac294c2`) instead of `COPY go.work go.work.sum ./`. Repo-root `go.work` references ~50 sibling-service modules and 2 `tools/*` modules whose `go.mod` files don't exist in any single-service build context. This is the §3.3 fallback predicted in `design.md`. Recorded in `design.md` §3.2 line 210 and in the `Dockerfile` lines 73–76 comment block.
2. **`Dockerfile` drops `go.sum` from `atlas-retry`/`atlas-service` COPY lines** (commit `5fe16a5f9`). Those two libs have zero external deps so `go mod download` never generates a `go.sum`. Plan's verbatim text would fail with "file not found". Documented at `Dockerfile` lines 29–31.
3. **`Dockerfile` guards `cp config.yaml`** (commit `01fd97d33`). Only 3 of 55 services ship a `config.yaml`; the guard emits an empty placeholder when absent so the runtime-stage `COPY --from=build-env /app/config.yaml /` has a stable source path. Documented at `Dockerfile` lines 99–104.
4. **`docker-bake.hcl` inlines `go_services = [...]`** (commit `16bb9258f`) instead of `locals { config = jsondecode(file(...)) }`. Empirically verified that bake's HCL evaluator has neither `locals` nor `file()`. Mitigation noted in file header lines 1–18, including the recommendation to add a CI parity check between bake's `go_services` and `services.json`'s `select(.type=="go-service") | .name`. The plan's "expected 54" count in Task 5 Step 2 is stale — actual count is 55, and the bake file correctly enumerates 55.

## Build & Test Results

This task touches no Go source. Per dispatcher's pre-audit confirmation:

| Surface | Result | Notes |
|---------|--------|-------|
| `docker buildx bake all-go-services` | PASS | 55/55 targets succeed end-to-end |
| `go test -race ./...` (all changed modules) | PASS | dispatcher confirmed clean |
| `go vet ./...` | PASS-EQUIVALENT | only pre-existing `libs/atlas-rest` WaitGroup warnings, also present on `main`, not introduced by this branch |
| `go build ./...` | PASS | dispatcher confirmed clean |
| `docker buildx bake --print` (auditor re-ran) | PASS | parses cleanly; 55 targets enumerated; matches `services.json` exactly |
| `git status` | CLEAN | empty working tree |
| Parity check: bake targets vs `services.json` go-services | PASS | `diff` produces no output (both are the same 55 names) |
| Surviving per-service Dockerfiles | PASS | exactly the 3 documented exceptions (`atlas-assets`, `atlas-pr-bootstrap`, `atlas-ui`) |

## Overall Assessment

- **Plan Adherence:** FULL — every task completed with documented, justified deviations only.
- **Recommendation:** READY_TO_MERGE.

## Action Items

None blocking. Optional follow-up suggested by `docker-bake.hcl` header comment: add a CI check that `go_services` in `docker-bake.hcl` exactly mirrors `jq -r '.services[]|select(.type=="go-service")|.name' .github/config/services.json | sort` so future drift surfaces in PR validation instead of at runtime. This is a separate, optional task and does not need to land in this PR.
