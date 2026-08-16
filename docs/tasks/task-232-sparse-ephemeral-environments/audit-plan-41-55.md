# Plan Audit — task-232-sparse-ephemeral-environments (Tasks 41-55)

**Plan Path:** docs/tasks/task-232-sparse-ephemeral-environments/plan.md
**Audit Date:** 2026-08-16
**Branch:** task-232-sparse-ephemeral-environments
**Base Branch:** main
**Range:** Tasks 41-55 only. Tasks 1-40 audited separately (`audit-plan-1-14.md`, `audit-plan-15-28.md`, `audit-plan-29-40.md`).

**Commits inspected (range boundary `e683d85da`..`8b5a06950`, plus fix-round commits interleaved):**
`e683d85da`, `685e9756b`, `b6bf199e5`, `554594462`, `9220e6a28`, `7343c681e`, `397d2bd6d`,
`7b1549fc5`, `81176783d`, `1bad55203`, `2a439f4dc`, `b501738be`, `41c8e85c5`, `44152eabe`,
`dddd9652a`, `5a0a7e55d`, `6cdc77092`, `b9c4a0645`, `f5b6693ce`, `87a5ae30c`, `bddc88bdb`,
`ed2a19d2d`, `9be00e383`, `816647a65`, `aa8a7b40b`, `8b5a06950`. Cross-checked against
`.superpowers/sdd/plan/progress.md` (the controller ledger, HANDOFF #15/#16 and the Task 41-54
sections) and the per-task reports at `.superpowers/sdd/plan/task-{41..52,54}-report.md`. No
`task-53-report.md` or `task-55-report.md` exists — consistent with both being not-started.

## Executive Summary

All 13 code/doc tasks in this range (41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 54) are
faithfully implemented, each verified directly against the working tree rather than trusted from
the ledger's account. Tasks 53 and 55 are correctly NOT_APPLICABLE-for-now: both require a live
deployed sparse environment and the `atlas_kafka_gate_*` counters this branch introduces, neither
of which exists yet. No fabricated measurement, no invented `proof.md`, no estimated numbers
anywhere in the diff — verified by absence. Task 54 Step 2 (flip the default) is correctly a
no-op by construction of Tasks 50-51; the Task 54 commit (`8b5a06950`) touches only two doc files,
confirmed by `git diff --name-only`. Module-local `go build`/`go test` is clean across all eight
touched Go modules (no FAIL), and all three shell-test suites relevant to this range
(`mode-select_test.sh`, `gen-routes.sh --check`, `gen-tenant-tables.sh --check`) pass, as does the
`atlas-pr-bootstrap` bats suite at 87/87 — matching the ledger's own count exactly.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 41 | Per-environment iteration, 8 per-tenant ticker loops (transports/marriages×2/asset-expiration) | DONE | `services/atlas-transports/atlas.com/transports/main.go:102,119,149` (startup, tick, shutdown all use `service.ForEachOwnedEnvironment`); `services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go:99`, `proposal_expiry.go:93`; `services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go:119`. Commit `e683d85da` + fix `b6bf199e5` (tenant-filter bug found and fixed same session). |
| 42 | Remaining ticker loops + class-2/class-3 dispositions | DONE | `services/atlas-mts/.../periodic.go:171`, `services/atlas-party-quests/.../main.go:104,141`, `services/atlas-saga-orchestrator/.../main.go:255,338`, `services/atlas-login/.../main.go:207` all call `service.ForEachOwnedEnvironment`. `docs/tasks/task-232-sparse-ephemeral-environments/ticker-dispositions.md` exists, documents an inventory correction 18→19 files, and accounts for all 19 (4 Task-41 + 4 Task-42 class-1, 6 class-2, 4 class-3, 1 newly-found 19th file) — a genuine re-derivation, not the plan's assumed 18. Commit `9220e6a28`. |
| 43 | Per-service namespace vars in ingress routing table | DONE | `tools/gen-routes.sh`, `tools/gen-routes_test.sh`, `deploy/k8s/base/ns-vars.generated.yaml` all present; `tools/verify.sh:393` runs `step "routes drift" ./tools/gen-routes.sh --check`. Ran `./tools/gen-routes.sh --check` directly → `gen-routes: up to date`, exit 0. Commit `397d2bd6d`. |
| 44 | The sparse overlay | DONE | `deploy/k8s/overlays/pr-sparse/` contains `kustomization.yaml`, `environment-record.yaml`, `ns-overrides.yaml`, `README.md`, `ingress-route.yaml`, `patches/`, plus sync-bootstrap/predelete-purge/pihole jobs per the plan's "what it keeps" list. Commits `7b1549fc5`..`1bad55203` (fix round added `pr-sparse-mirror-guard.sh`, per ledger). |
| 45 | Seed override consumer-group offsets at provisioning | DONE | `seed_group` extracted into sourceable `deploy/k8s/base/kafka-precreate.sh` (testable, per plan's Step 1 mandate), referenced from `atlas-kafka-precreate.yaml`'s kustomization and exercised by `atlas-kafka-precreate_test.sh` (single- and multi-topic cases). Commits `81176783d`..`41c8e85c5` (two review fix rounds, per ledger, resolving Kafka CLI path issues). |
| 46 | Confirm/wire per-environment socket port and IP allocation | DONE | `grep -n "lb-allocate" deploy/k8s/overlays/pr-sparse/kustomization.yaml` → `92:  - path: patches/lb-allocate.yaml`, present as required. Commit `b501738be`, ledger records live-cluster pool confirmation and runbook update. |
| 47 | Bootstrap creates its own service rows instead of merging into `main`'s | DONE | `build_service_config()` in `services/atlas-pr-bootstrap/scripts/service-config.sh:72` branches on `${ATLAS_MODE:-isolated} = sparse` (line 81) to build a fresh-UUID row rather than merging. bats: "sparse mode never reads or writes the pinned main service row" and "isolated mode still merges into the pinned row" both `ok` in the local run. Commit `44152eabe`, fix `5a0a7e55d` (restored unconditional-replace in isolated POST branch per re-review). |
| 48 | Teardown — deactivate before destroy, reclaim control plane | DONE | `services/atlas-pr-bootstrap/scripts/cleanup.sh:577-588` `PHASES=(deactivate do_deactivate  drop-control-plane do_drop_control_plane  sweep-tenant do_sweep_tenant  drop-dbs ... )` — `deactivate` is first, before every `do_drop_*`. `do_deactivate` (line 168) PATCHes DEACTIVATING then DELETED. Two review fix rounds landed real defects (dead-code reclaim filter fixed by adding `services.Environment`, partial-PATCH zeroing fixed to read-modify-write) — confirmed present: `services/atlas-configurations/atlas.com/configurations/services/service/rest.go:20,66,94` all carry `Environment string`, and `processor.go:122,132,142` set `rm.Environment = e.Environment`. Commits `dddd9652a`, `6cdc77092`, `b9c4a0645`. |
| 49 | Tenant-keyed orphan sweeper | DONE | `sweep_tenant()` at `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh:208`, refuses empty tenant id (line 210), driven from a generated `tenant-tables.txt`. `tools/gen-tenant-tables.sh --check` → up to date, exit 0, matching the generator's expected output. bats: `sweep-orphans.sh --sweep-tenant deletes only the named tenant's rows via psql` and `refuses an empty tenant id` both `ok`. CI wiring confirmed three-point (job def `pr-validation.yml:260`, `needs:` at `1113`, result check at `1133`). Commit `f5b6693ce`, fix `87a5ae30c` (CI-twin-job finding). |
| 50 | Affected-service determination and automatic escalation | DONE | `tools/mode-select.sh` and `tools/mode-select_test.sh` present; ran `./tools/mode-select_test.sh` → PASS, exit 0. `.github/workflows/pr-validation.yml:289` `mode-select` job present, `needs:` at `1113`, result check via `MODE_SELECT_GUARD_RESULT` at `1134`. Commits `bddc88bdb`, fix `ed2a19d2d` (escalation-default inversion + `BASELINE_OVERLAY` decl, both confirmed correct per ledger's independent re-check). |
| 51 | Per-PR label override and mode report | DONE | `tools/mode-select_test.sh:46-82` implements `check_forced`/`check_forced_overrides`/conflict-case; `check_forced_overrides` specifically covers the override-set (not just the mode-string), which is the fix for the empty-override-set defect the controller found mid-task. `.github/workflows/pr-validation.yml:350,355` build the `atlas:isolated`/`atlas:sparse` change-label lines. Commit `9be00e383`, fix `816647a65`. |
| 52 | Turn on `env-bootstrap-guard` | DONE | `tools/verify.sh:380` `step "env bootstrap guard" ./tools/env-bootstrap-guard.sh`, gated on the `touched` predicate per the plan's Step 2 snippet. Commit `aa8a7b40b`. |
| 53 | Measure Kafka fan-out cost before enabling sparse by default | NOT_APPLICABLE (deliberately deferred) | No `kafka-fanout-measurement.md` exists anywhere in the tree (`ls` → No such file). Ledger records the controller queried live Prometheus directly (`atlas_kafka_gate_processed_total` → empty result vector) and found the counters this task needs are introduced by this branch and not deployed; escalated to the user, who chose to proceed with 54-55 and defer 53 post-deploy. No fabricated table, no placeholder numbers, no "expected acceptable" language anywhere in `docs/runbooks/sparse-environments.md` — confirmed by direct grep (see below). This is the correct, honest disposition per the task brief's explicit instruction. |
| 54 | Make sparse the default mode | DONE | `tools/mode-select.sh:215` unconditional fallthrough prints `sparse` — the default IS sparse, confirmed live: `printf 'services/atlas-monsters/.../processor.go\n' \| ./tools/mode-select.sh` → `sparse` (reproduced). Task 54's own commit `8b5a06950` diff is `docs/runbooks/ephemeral-pr-deployments.md` and `docs/runbooks/sparse-environments.md` ONLY (`git diff 8b5a06950~1 8b5a06950 --name-only`) — `tools/mode-select.sh` is untouched by this commit, confirming Step 2 was correctly treated as a no-op already satisfied by Tasks 50-51, not faked with an invented diff. Preconditions (7 guard scripts) all ran clean per the ledger; NetworkPolicy check correctly reported as "9 exist cluster-wide, zero in any atlas-* namespace — not a blocker" rather than a bare "none found." |
| 55 | The §17 proof — environment is not deployment | NOT_APPLICABLE (deliberately deferred) | `proof.md` does NOT exist (`ls` → No such file). All ten plan steps are live-cluster measurements (`kubectl`, `logcli`, a real game client for Step 2, `atlas_kafka_gate_*` metrics for Step 6) against a deployed sparse environment, none of which exists pre-merge. Correct disposition — an invented or empty-row `proof.md` would have been the Critical finding; its absence is not one. |

**Completion Rate:** 13/15 tasks DONE (86.7%), 2/15 correctly deferred (not gaps)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

**Task 53** and **Task 55** are the two intentionally-deferred tasks, both blocked on the same
external prerequisite: a live sparse environment deployed from this branch, which cannot exist
until the branch merges. The ledger documents the controller verified this by querying live
Prometheus directly rather than assuming — `atlas_kafka_gate_processed_total` returned an empty
result set, proving the counters Task 26 (this branch) introduces are not yet in the cluster.
Both were escalated to the user, who explicitly chose to proceed with Tasks 54-55's non-blocked
work and defer the two live-measurement tasks post-deploy. This audit found no artifact
(`kafka-fanout-measurement.md`, `proof.md`) fabricated to paper over the gap, and confirmed the
governing runbook uses honest "unmeasured" / "deferred" language rather than invented numbers:

```
$ grep -n "unmeasured\|Task 53\|fabricat\|expected acceptable\|estimate" docs/runbooks/sparse-environments.md
72:### Fan-out cost is unmeasured at the time sparse became the default
76:the gate above is what filters it back down per-environment. Task 53 was
80:live data to measure against yet. **Task 53 is deferred to a post-deploy
249:concurrency. See "Fan-out cost is unmeasured" above: the Kafka fan-out cost
251:not been measured (Task 53, deferred). Treat 10 as "the pool physically
```

Impact: neither task blocks the merge decision on this branch's own terms — Task 53 gates Task
54's *default flip*, and the user explicitly accepted that risk; Task 55 is a sign-off artifact
for the PRD, not a code dependency of any other task. Both are legitimate post-deploy follow-up
work, not silently dropped scope.

## Build & Test Results

| Service / Check | Build | Tests | Notes |
|---|---|---|---|
| atlas-transports | PASS | PASS | `go build ./...` clean; `go test ./...` no FAIL |
| atlas-marriages | PASS | PASS | same |
| atlas-asset-expiration | PASS | PASS | same |
| atlas-mts | PASS | PASS | same |
| atlas-party-quests | PASS | PASS | same |
| atlas-saga-orchestrator | PASS | PASS | same |
| atlas-login | PASS | PASS | same |
| atlas-configurations | PASS | PASS | same (covers Task 48's `services.Environment` field) |
| `tools/mode-select_test.sh` | n/a | PASS | exit 0, "PASS" printed |
| `tools/gen-routes.sh --check` | n/a | PASS | "gen-routes: up to date", exit 0 |
| `tools/gen-tenant-tables.sh --check` | n/a | PASS | "tenant-tables.txt is up to date", exit 0 (two services correctly skipped as undeployed, per Task 49's ledger ruling) |
| atlas-pr-bootstrap (bats) | n/a | PASS | 87/87 `ok`, 0 `not ok` — matches ledger's reported count exactly |

`tools/verify.sh`, `tools/lint.sh`, and docker bake were NOT run per instruction (controller
owns those; a run is reported in flight).

## Overall Assessment

- **Plan Adherence:** FULL (for the 13 tasks in scope for this branch to complete; 2 tasks
  correctly deferred with documented, user-approved reasoning and no fabricated evidence)
- **Recommendation:** READY_TO_MERGE for Tasks 41-52 and 54's content, contingent on the
  controller's own flagless `tools/verify.sh` run (in flight) passing, and on the user's
  standing decision to treat Tasks 53/55 as a post-deploy follow-up rather than a merge blocker.

## Action Items

1. None required within this task range — all in-scope work (41-52, 54) is implemented,
   tested, and matches its own commit history and the controller ledger's independently-verified
   claims.
2. Post-merge follow-up (already tracked, not a defect of this range): execute Task 53's
   Kafka fan-out measurement and Task 55's §17 proof against a real deployed sparse environment,
   per the user's decision recorded in `.superpowers/sdd/plan/progress.md`.
