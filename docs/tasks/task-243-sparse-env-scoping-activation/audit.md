# Plan Audit — task-243-sparse-env-scoping-activation

**Plan Path:** docs/tasks/task-243-sparse-env-scoping-activation/plan.md
**Audit Date:** 2026-08-20
**Branch:** task-243-sparse-env-scoping-activation
**Base Branch:** main (commit range d35509be0..HEAD)

## Executive Summary

All 13 plan tasks were implemented, tested, and reviewed; every review-flagged
defect (Tasks 9, 11, 12, 13) has a corresponding follow-up commit that lands
before the branch's tip, and each fix was independently re-verified in this
audit by reading the resulting code, not just the review artifact. All Go
modules touched (`libs/atlas-env`, `libs/atlas-kafka`,
`services/atlas-configurations`, `services/atlas-login`,
`services/atlas-channel`) build and `go test ./...` cleanly; the 142-case
`atlas-pr-bootstrap` bats suite passes in full; `tools/shell-guard.sh
--require-shellcheck` and all four deploy guards
(`pr-sparse-mirror-guard.sh`, `overlay-env-guard.sh`,
`sparse-baseline-scoping-guard.sh`, plus `kustomize build` on all three
overlays) pass. The one gap is not in the 13 tasks themselves: `plan.md`'s
"Post-plan gates" section and `prd.md` §10's Acceptance Criteria require the
live end-to-end acceptance run (three-run idempotency, Argo hard refresh,
socket binding, automatic `ACTIVE`, real login) to be executed against a
freshly created sparse PR environment and its results recorded in the task
folder — every checkbox in `prd.md` §10 is still `[ ]` and no results file
exists. That is a real, plan-mandated blocker to declaring the work "done,"
distinct from and in addition to a green `verify.sh`.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Deterministic service-id derivation | DONE | `tools/derive-service-id.sh` implements exactly the spec'd algorithm and error handling; `sh tools/derive-service-id_test.sh` — all 16 assertions PASS, including all 10 pinned UUID values from `context.md`'s table and the nil-UUID-collision regression. |
| 2 | Legacy tenant reconciliation | DONE | `libs/atlas-env/tenants.go:69-77` inserts the `tenantEnv == ""` arm exactly where specified, with the FR-3.1 comment. `go build ./... && go test ./...` in `libs/atlas-env` — `ok`. |
| 3 | Gate drop-reason distinguishability | DONE | `libs/atlas-kafka/consumer/gate.go` adds `gateReason`/six reason constants, widens `gateDroppedUnresolvable` to 3 labels only; `manager.go:631-641` carries `reason` through both drop/skip log lines exactly as specified. `go build ./... && go test ./...` in `libs/atlas-kafka` — all packages `ok`. |
| 4 | `servicesuniq` dedupe + unique index | DONE | New package `services/atlas-configurations/.../servicesuniq/migration.go` implements `Preflight`/`Migration`/keeper-rule/outbox tombstone/unique index exactly per spec, registered in `main.go:52-53` after `environmentcol.Migration` with the corrected trailing comment. `go build ./... && go test ./...` — `ok` for `servicesuniq` and every other package in the module. |
| 5 | atlas-login readiness component | DONE | `configuration/projection/state.go:78-88` adds `HasService()`; `main.go:72` adds the second `WithReadinessGate`. `go test ./configuration/projection/... -run HasService -v` — all 4 cases PASS. |
| 6 | atlas-channel readiness component | DONE | Mirrors Task 5 exactly: `state.go:78-88`, `main.go:198`. Same 4 test cases PASS with the channel-service derived id. |
| 7 | readinessProbe on login/channel Deployments | DONE | `deploy/k8s/base/atlas-login.yaml:39-45` and `atlas-channel.yaml:39-45` both carry the exact probe block (`/api/readyz`, port 8080, `initialDelaySeconds: 10`, `periodSeconds: 10`, `failureThreshold: 30`) with the required comment. |
| 8 | `service-config.sh` takes id as parameter | DONE | `new_uuid` deleted; `build_service_config` validates `$3` against the UUID regex and fails with `"requires a service id"` on empty/malformed input, matching the plan's exact error text. bats cases `sparse fails loudly when no id is supplied` / `sparse rejects a malformed id` / `sparse mode never reads or writes the pinned main service row` all pass. |
| 9 | `bootstrap.sh` upsert on derived id | DONE | `upsert_sparse_service_config` (bootstrap.sh:559-630) replaces `create_service_config`; loop reads the override set via `env_record_get \| jq ... .overrides // {} \| keys[]`; `kubectl set env` and its comment block are gone; `restart_targets` sparse branch no longer includes atlas-login/atlas-channel. RBAC comment landed identically in both `deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml` and `.../pr/sync-bootstrap.yaml` (`diff -q` — files identical). Review flagged a bare `set -e` on the override-set read (CHANGES_REQUIRED); fixed in `a4f2a6244` — confirmed the `\|\| { log error ...; exit 1; }` guard is now present at both call sites (bootstrap.sh:651, :779). |
| 10 | `bootstrap.sh` activation step | DONE | `activation_decision()` (bootstrap.sh:140-146) matches the 5-row truth table; the `ATLAS_STEP=activate` block (bootstrap.sh:775-820) rolls the override set, enforces the FR-4.2 login+channel-both-present guard, and PATCHes to `ACTIVE` via the same four-`jq`-read pattern as `record_environment_tenant`. Gated on `ATLAS_MODE=sparse`, placed immediately before `ATLAS_STEP=done`. |
| 11 | Sparse overlay — two CI tokens | DONE | Both `#PLACEHOLDER_SERVICE_ID_BLOCK` / `#PLACEHOLDER_PRECREATE_GROUPS_BLOCK` present only in `deploy/k8s/overlays/pr-sparse/kustomization.yaml` (grep across `deploy/k8s/overlays/` finds no hit in `overlays/pr/`). `kustomize build` succeeds unsubstituted on `pr-sparse` and `pr`; `pr-sparse-mirror-guard.sh`, `overlay-env-guard.sh`, `sparse-baseline-scoping-guard.sh` all exit 0. Review caught the explanatory comments repeating the bare token substring (would self-splice under the unanchored `sed -g` pass); fixed in `e1317e3ad` — confirmed the comments now read "the SERVICE_ID anchor" / "the PRECREATE_GROUPS anchor". |
| 12 | CI renders ids + resolved group names | DONE | `.github/workflows/pr-validation.yml:1037-1102` implements the derive/patch/verify loop essentially as specified, using `GROUP_LIST` (not the plan's literal `GROUPS`, which collides with bash's own reserved array — caught and documented by the reviewer, independently reproduced in `review-task-12.md`). Empty-`SERVICE_ID_BLOCK`, empty-`GROUP_LIST`, and unresolved-`%`-template guards all present with `::error::`+`exit 1`. Offline render-verification step (`:1214-1260`) checks the derived `SERVICE_ID` landed on each override Deployment (not the base literal), the bootstrap Job's `SERVICE_ID_<TYPE>` matches, and the precreate Job's `KAFKA_CONSUMER_GROUP` is present and `%`-free — the `design.md` §11 offline half. |
| 13 | Runbook | DONE | `docs/runbooks/sparse-environments.md` gained all 7 required sections: upgrade procedure (D9, `:353`), pre-flight (`:374`), `ATLAS_ENVIRONMENT` roll precondition (`:403`), binding verification (`:427`), seeded-group verification (`:454`), readiness-probe timing (`:469`), activation window (D8, `:488`). No literal absolute/home path found in the new prose (`grep` over the added section range returns nothing). Review caught the crash-loop passage naming the wrong function (`services.Migration` vs `servicesuniq.Migration`); fixed in `88d846363`. |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 13 tasks are fully implemented and independently re-verified against
the code (not just review artifacts) in this audit.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-env` | PASS | PASS | `go test ./...` — `ok` |
| `libs/atlas-kafka` | PASS | PASS | `go test ./...` — all sub-packages `ok` |
| `services/atlas-configurations/atlas.com/configurations` | PASS | PASS | `go test ./...` — every package `ok`, including new `servicesuniq` |
| `services/atlas-login/atlas.com/login` | PASS | PASS | `go test ./...` — no failures; `configuration/projection` `HasService` cases individually verified PASS |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | `go test ./...` — no failures; `configuration/projection` `HasService` cases individually verified PASS |
| `services/atlas-pr-bootstrap` (bats) | N/A (shell) | PASS | `bats services/atlas-pr-bootstrap/test` — 142/142 `ok`, 0 `not ok` |
| Shell scripts (repo-wide gate) | PASS | — | `tools/shell-guard.sh --require-shellcheck` — 71 scripts OK |
| `deploy/k8s` overlays | PASS | — | `kustomize build` clean on `main`/`pr`/`pr-sparse`; `pr-sparse-mirror-guard.sh`, `overlay-env-guard.sh`, `sparse-baseline-scoping-guard.sh` all exit 0 |

Note: a repo-wide `tools/verify.sh` was reported as already running in a
parallel context per the dispatch instructions and was not re-run here to
avoid duplication; the module-local build/test commands above were run
directly instead.

## Overall Assessment

- **Plan Adherence:** FULL — all 13 tasks completed, review findings fixed and verified, no silent skips or deferrals.
- **Recommendation:** NEEDS_REVIEW — code-complete and gate-clean, but the plan's own "Post-plan gates" section and `prd.md` §10's Acceptance Criteria (all still unchecked) require a live end-to-end run against a freshly created sparse PR environment, with results recorded in the task folder, before the branch can be called done. That run cannot be satisfied by `verify.sh` or by this audit and has not yet been performed or recorded anywhere in the task folder.

## Action Items

1. Execute the live acceptance criteria in `prd.md` §10 (freshly created sparse
   PR environment: `SERVICE_ID` binding survives an Argo hard refresh, login
   socket binds only its own tenant, three-run bootstrap idempotency, legacy
   vs. mismatched tenant message handling, automatic `ACTIVE` with no manual
   PATCH, and the full end-to-end account-creation-to-character-select login)
   per the rollout ordering in `design.md` §12, and record the results in the
   task folder — `plan.md`'s "Post-plan gates" section is explicit that these
   cannot be claimed from `verify.sh`.
2. Record the observed baseline projection catch-up time in
   `docs/runbooks/sparse-environments.md`'s "Readiness probe timing" section
   (currently states "No live measurement has been recorded here yet") once
   the live sparse environment from Action Item 1 exists, and tighten Task 7's
   `initialDelaySeconds`/`failureThreshold` only if that measurement supports it.
3. Before merging, confirm the baseline `ATLAS_ENVIRONMENT` ConfigMap-propagation
   precondition noted in `context.md` ("out of scope, and stays out") and in
   the runbook's `:403` section has actually been satisfied on the target
   cluster (affected baseline deployments rolled) — it is a live-state
   precondition of Action Item 1, not something this branch's code fixes.
