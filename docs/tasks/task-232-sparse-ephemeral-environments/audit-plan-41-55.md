# Plan Audit — task-232-sparse-ephemeral-environments (Tasks 41–55)

**Plan Path:** docs/tasks/task-232-sparse-ephemeral-environments/plan.md (lines 5876–end)
**Audit Date:** 2026-08-16
**Branch:** task-232-sparse-ephemeral-environments
**Base Branch:** main
**Range audited:** `c8d44127c..418b2caf9` (116 commits)

## Executive Summary

**Confirmed: none of Tasks 41–55 were implemented on this branch.** All 15
tasks are `NOT IMPLEMENTED`. This corroborates the SDD ledger
(`.superpowers/sdd/plan/progress.md`): every session's work maps to Tasks
1–40c, and HANDOFF #12 explicitly lists "Tasks 41/42" as separate, deferred
work in its "Remaining before PR — NOT plan tasks" section — despite an
internal, unsubstantiated banner earlier in the same ledger ("ALL 55 PLAN
TASKS COMPLETE") that is contradicted by the ledger's own task-by-task
record and by every artifact check below. No commit in the 116-commit range
touches any of the files Tasks 41–55 require creating or modifying:
`tools/gen-routes.sh`, `deploy/k8s/overlays/pr-sparse/`,
`deploy/k8s/base/atlas-kafka-precreate.yaml`, `tools/mode-select.sh`,
`services/atlas-pr-bootstrap/scripts/{bootstrap,cleanup,sweep-orphans}.sh`
(within this range), `docs/tasks/.../ticker-dispositions.md`,
`docs/tasks/.../kafka-fanout-measurement.md`, `docs/tasks/.../proof.md`, or
the `env-bootstrap-guard` step in `tools/verify.sh`. Phases D, E, and F of
the plan (deployment, teardown/reclamation, and the rollout gate) have not
started.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 41 | Per-environment iteration for the 8 per-tenant ticker loops | NOT IMPLEMENTED | `services/atlas-transports/atlas.com/transports/main.go:93` still calls `tenant2.NewProcessor(l, rt.Context()).GetAll()` before `time.NewTicker` (line 111) — the exact pre-conversion shape the task's own failing test (`TestTickerDoesNotCloseOverACachedTenantList`) is written to catch. No `ForEachOwnedEnvironment` reference anywhere in `atlas-transports`, `atlas-marriages/scheduler/*`, or `atlas-asset-expiration/task/periodic.go`. The prescribed `tick_test.go` does not exist (`ls` confirms absent). |
| 42 | Remaining ticker loops + class-2/class-3 dispositions | NOT IMPLEMENTED | `docs/tasks/task-232-sparse-ephemeral-environments/ticker-dispositions.md` does not exist (`ls`: No such file or directory). `ForEachOwnedEnvironment` absent from all four target files (`atlas-mts/task/periodic.go`, `atlas-party-quests/main.go`, `atlas-saga-orchestrator/main.go`, `atlas-login/main.go`). |
| 43 | Per-service namespace variables in the ingress routing table | NOT IMPLEMENTED | `git log c8d44127c..418b2caf9 -- tools/gen-routes.sh` and `-- deploy/k8s/base/routes.conf.template.generated` both return zero commits. `deploy/k8s/base/routes.conf.template.generated` still emits `${POD_NAMESPACE}` (e.g. line 2: `atlas-merchant.${POD_NAMESPACE}.svc...`), not per-service `NS_<SERVICE>`. `deploy/k8s/base/ns-vars.generated.yaml` and `tools/gen-routes_test.sh` do not exist. No `gen-routes` reference in `tools/verify.sh`. |
| 44 | The sparse overlay | NOT IMPLEMENTED | `deploy/k8s/overlays/pr-sparse/` does not exist (`ls`: No such file or directory). Only `main`, `pr`, and `pr-cleanup` overlays are present. None of `kustomization.yaml`, `environment-record.yaml`, `ns-overrides.yaml`, or `README.md` were created. |
| 45 | Seed override consumer-group offsets at provisioning time | NOT IMPLEMENTED | `deploy/k8s/base/atlas-kafka-precreate_test.sh` does not exist. `atlas-kafka-precreate.yaml` contains no offset-seeding pass (only a pre-existing, unrelated "replay them from first-offset at every boot" comment at line 49, predating this task). `docs/runbooks/sparse-environments.md`'s only touching commit in range is `3372ff56c` (Task 29, observability) — no offset-seeding verification section was added. |
| 46 | Confirm and wire per-environment socket port and IP allocation | NOT IMPLEMENTED | `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md`'s only touching commits in range are the Phase A / Task 1–10 audit commits (`4dea1087b` … `f0a76d8b0`); none post-date or reference a ports/IP disposition row. No MetalLB pool-capacity note added to `docs/runbooks/sparse-environments.md`. `deploy/k8s/overlays/pr-sparse/kustomization.yaml` (the file this task modifies) doesn't exist because Task 44 wasn't done. |
| 47 | Bootstrap creates its own service rows instead of merging into `main`'s | NOT IMPLEMENTED | No `ATLAS_MODE` or `create_service_config` reference in `services/atlas-pr-bootstrap/scripts/bootstrap.sh` or `service-config.sh`. `bootstrap.sh:303-320`'s `upsert_service_config` is unmodified. No commits to these files inside the audited range (confirmed no matches from the grep sweep). |
| 48 | Teardown — deactivate before destroy, reclaim the control plane | NOT IMPLEMENTED | No `deactivate`, `drop-control-plane`, or `PhaseDeactivating` reference in `services/atlas-pr-bootstrap/scripts/cleanup.sh` or `libs/atlas-env/registry.go`. The `PHASES` list in `cleanup.sh` was not extended. |
| 49 | The tenant-keyed orphan sweeper | NOT IMPLEMENTED | `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` has no `sweep_tenant` function (grep returns nothing). Its last touching commit predates this branch's range entirely (`0355068ac`, task-045, an unrelated earlier PR). The script still only does namespace-prefix orphan detection, not the generic tenant-keyed pass across shared databases the task requires. |
| 50 | Affected-service determination and automatic escalation | NOT IMPLEMENTED | `tools/mode-select.sh` and `tools/mode-select_test.sh` do not exist. `.github/actions/detect-changes/action.yml` has no `mode` or `override-services` output (grep empty). |
| 51 | The per-PR label override and the mode report | NOT IMPLEMENTED | No `atlas:sparse`, `atlas:isolated`, or `ATLAS_FORCE_MODE` reference anywhere in `.github/workflows/pr-validation.yml`. Depends on Task 50's `mode-select.sh`, which doesn't exist. |
| 52 | Turn on `env-bootstrap-guard` | NOT IMPLEMENTED | `grep -n 'env-bootstrap-guard' tools/verify.sh` returns nothing — only `env-domain-guard` is wired in (line 374). `tools/env-bootstrap-guard.sh`'s own header comment (unchanged) still reads: *"Unlike env-domain-guard.sh, this guard is NOT wired into tools/verify.sh yet — Task 52 adds that once Phase C reaches zero."* This directly contradicts the SDD ledger's "ALL 55 PLAN TASKS COMPLETE ... the condition Task 52 asserts" claims (progress.md lines ~7235, 7239) — the ledger's banner is not supported by the code. |
| 53 | Measure the Kafka fan-out cost before enabling sparse by default | NOT IMPLEMENTED | `docs/tasks/task-232-sparse-ephemeral-environments/kafka-fanout-measurement.md` does not exist. This task also requires a live cluster measurement, which cannot happen before Task 44's overlay exists. |
| 54 | Make sparse the default mode | NOT IMPLEMENTED | `tools/mode-select.sh` doesn't exist (Task 50 prerequisite absent), so there is no default to flip. Depends on Tasks 43–53, none of which landed. |
| 55 | The §17 proof — environment is not deployment | NOT IMPLEMENTED | `docs/tasks/task-232-sparse-ephemeral-environments/proof.md` does not exist. This task requires a live sparse-environment deployment, which cannot exist before Task 44. |

**Completion Rate:** 0/15 tasks (0%)
**Skipped without approval:** 15 (all silently absent from the branch's diff; the SDD ledger records them as known-deferred rather than "approved skip," but no explicit user/plan sign-off to skip them is on record)
**Partial implementations:** 0

## Skipped / Deferred Tasks

All 15 tasks (41–55) are missing in full — there are no partial landings to
describe piece-by-piece; each is a clean absence of every artifact the task's
**Files** section requires.

**Impact:**

- **Tasks 41–42 (ticker per-environment iteration):** Without these, the
  eight-plus-four per-tenant ticker loops still cache a tenant list once at
  startup (design correction C4). A tenant provisioned in a sparse override
  environment after the pod started would never be picked up by these
  background loops, defeating the FR-6.1 requirement that makes autonomous
  per-tenant work environment-aware.
- **Tasks 43–46 (ingress namespacing, sparse overlay, offset seeding, LB
  allocation):** These are the deployment substrate the entire "sparse"
  feature depends on. Without `deploy/k8s/overlays/pr-sparse/` and the
  `NS_*` routing variables, there is no way to stand up a sparse
  environment at all — the feature this task (task-232) exists to build is
  entirely unbuilt at the infrastructure layer.
- **Tasks 47–49 (bootstrap service rows, teardown, orphan sweeper):** The
  single most dangerous existing-code risk the plan calls out (Task 47,
  "the single most dangerous piece of existing code in this migration") is
  unaddressed — `upsert_service_config` would still merge a sparse PR's
  tenant into `main`'s pinned service row if sparse mode were ever turned
  on, which is exactly the NG6/G7 cross-environment leak the design exists
  to prevent.
- **Tasks 50–51 (mode selection, label override):** No `tools/mode-select.sh`
  means CI has no way to decide sparse vs. isolated mode per PR, and no
  `atlas:sparse` / `atlas:isolated` labels exist to override it.
- **Task 52 (turn on `env-bootstrap-guard`):** The gate that is supposed to
  block sparse mode until every service wires the environment registry is
  still not enforced in CI — it remains in its "staged, not wired" state
  from Task 28.
- **Tasks 53–55 (fan-out measurement, default flip, §17 proof):** None of
  these can even begin without Tasks 43–44 existing; they are correctly
  blocked, not separately broken.

## Build & Test Results

Not run for this shard. No Go or deploy/ artifacts in scope for Tasks 41–55
exist to build or test — `go build`/`go test` against
`services/atlas-transports/atlas.com/transports`,
`services/atlas-mts/atlas.com/mts`, etc. would only exercise pre-existing
code untouched by this task range, and the bats suites the plan specifies
(`services/atlas-pr-bootstrap/test/*.bats`) for Tasks 47–49 were never
created. Running the existing test suites would not distinguish "these
tasks passed" from "these tasks were never attempted."

## Overall Assessment

- **Plan Adherence:** INCOMPLETE
- **Recommendation:** NEEDS_FIXES — Tasks 41–55 (Phases D, E, F, plus the
  Task 41/42 ticker carve-outs) require a full implementation pass before
  this branch can be considered to deliver the sparse-ephemeral-environments
  feature described in the plan. The branch as it stands only completes
  Phases A–C (Tasks 1–40c): the query-scope audit, Redis tenant-scoping,
  the environment registry/context plumbing, the Kafka/REST environment
  headers, and the per-tenant→per-environment wiring recipe applied across
  all services. The deployment, teardown, and rollout-gate work that turns
  that plumbing into an actual "sparse mode" has not started.

## Action Items

1. Implement Task 41: convert the eight per-tenant ticker loops
   (`atlas-transports`, `atlas-marriages` ×2, `atlas-asset-expiration`) to
   `service.ForEachOwnedEnvironment`, with the failing-test-first pattern the
   task specifies.
2. Implement Task 42: convert the remaining four ticker loops
   (`atlas-mts`, `atlas-party-quests`, `atlas-saga-orchestrator`,
   `atlas-login`) and produce `ticker-dispositions.md` covering all 18
   inventoried files.
3. Implement Task 43: `tools/gen-routes.sh` NS_* generation,
   `ns-vars.generated.yaml`, ingress Deployment patch, and the `--check`
   drift gate wired into `tools/verify.sh`.
4. Implement Task 44: create `deploy/k8s/overlays/pr-sparse/` with
   `kustomization.yaml`, `environment-record.yaml`, `ns-overrides.yaml`,
   and `README.md`.
5. Implement Task 45: offset-seeding pass in
   `atlas-kafka-precreate.yaml` plus its shell test.
6. Implement Task 46: confirm and wire per-environment LB IP/port
   allocation against live MetalLB state; fill in the query-scope-audit
   disposition row.
7. Implement Task 47: `create_service_config` sparse-mode path in
   `bootstrap.sh`/`service-config.sh`, with the negative bats test proving
   `main`'s service row is never read or written in sparse mode.
8. Implement Task 48: `deactivate` and `drop-control-plane` phases in
   `cleanup.sh`, ordered first in `PHASES`.
9. Implement Task 49: the generic `sweep_tenant` pass in
   `sweep-orphans.sh` driven from the query-scope-audit entity list.
10. Implement Task 50: `tools/mode-select.sh` + escalation rules, wired
    into `detect-changes`.
11. Implement Task 51: `ATLAS_FORCE_MODE` / `atlas:sparse` /
    `atlas:isolated` label handling and the PR mode-report comment.
12. Implement Task 52: wire `env-bootstrap-guard` into `tools/verify.sh`
    and confirm it reaches exit 0 fleet-wide.
13. Implement Task 53: measure and record the Kafka fan-out cost against a
    live cluster before proceeding to Task 54.
14. Implement Task 54: flip `mode-select.sh`'s default to `sparse` once
    Task 53's verdict is acceptable and every precondition guard passes.
15. Implement Task 55: execute the §17 proof against a real sparse
    deployment and record `proof.md`.
16. Correct the SDD ledger: its "ALL 55 PLAN TASKS COMPLETE" banner
    (progress.md, around line 7235) is contradicted by both its own later
    HANDOFF #12 text (which lists Tasks 41/42 as deferred, non-PR-blocking
    work) and by every artifact check in this audit — Tasks 43–55 are not
    mentioned in the ledger as done, attempted, or even dispatched anywhere
    after Task 40c.
