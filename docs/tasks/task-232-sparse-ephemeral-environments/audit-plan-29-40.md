# Plan Audit — task-232-sparse-ephemeral-environments (Tasks 29–40)

**Plan Path:** docs/tasks/task-232-sparse-ephemeral-environments/plan.md (lines 4390–5875)
**Audit Date:** 2026-08-16
**Branch:** task-232-sparse-ephemeral-environments
**Base Branch:** main
**Range under review:** c8d44127c..418b2caf9 (116 commits)

## Executive Summary

All 13 tasks in this shard (29, 29A, 30–40, with 33–40's lettered sub-batches
treated as their parent task) were fully implemented and verified against the
git history and current tree state. Every one of the 64 fleet services now
carries `service.WithEnvironmentRegistry(serviceName)` in `main.go` and a
`wiring_test.go`; a fleet-wide grep confirms zero leftover
`SetHeaderParsers` calls missing `EnvHeaderParser` and zero leftover
`requests.RootUrl(` call sites outside the two documented, plan-approved
carve-outs (atlas-channel's `configuration/projection` and Task 41/42
ticker-bearing paths). `tools/env-bootstrap-guard.sh` exits 0 ("clean (64
service(s) checked)"). Spot-checked module builds/tests (atlas-monsters,
atlas-login, atlas-channel, atlas-world, atlas-saga-orchestrator, plus a
10-service sample across batches 33–40, plus `libs/atlas-service` and
`libs/atlas-rest`) all pass. No skipped or unresolved gaps found in this
range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 29 | Observability — log field, metric labels, alert | DONE | `libs/atlas-service/logger.go:23,51-61` (`environmentHook`, registered before the normalizer); `libs/atlas-service/logger_test.go:13,29` (`TestLoggerEmitsTheEnvironmentField` / `...OmitsTheEnvironmentFieldOnMain`, both pass); `libs/atlas-rest/requests/metrics.go:19` (`environment` label added to counter); `deploy/k8s/base/atlas-ingress.yaml:29-53` (`$http_environment` in access log + proxy header); `docs/runbooks/sparse-environments.md` created (7011 bytes, includes mode-selection, sparse floor, gate counters, alert, Loki selectors per commit 3372ff56c). |
| 29A | Provision `SERVICE_NAME` on every Deployment (BLOCKS Task 30) | DONE | Commit 71debb85a: `SERVICE_NAME` injected via downward API (`fieldRef: metadata.labels['app']`) into every base Deployment, not a hand-maintained list. Fix round 376fb33ab wired `tools/service-name-guard.sh` into `verify.sh` and extended coverage to StatefulSet/DaemonSet — addresses the "guard that cannot fire" defect class explicitly. |
| 30 | Service-wiring recipe, established on `atlas-monsters` | DONE | Commit d2bcf6e04: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md` created (290 lines); `services/atlas-monsters/atlas.com/monsters/main.go` gains `WithEnvironmentRegistry`; all 5 `SetHeaderParsers` sites and 6 `requests.RootUrl(` sites converted; `requests.ErrorRequest[T]` added to `libs/atlas-rest/requests/get.go` (+13 lines) as the plan anticipated. `wiring_test.go` present and passing. |
| 31 | `atlas-login` — wiring + socket-edge origination | DONE | Commit a9556d6e0: Bootstrap option, 5 `SetHeaderParsers` sites, 10 `requests.RootUrl(` sites converted; `newSessionContext` extracted in socket/handler with `env.Self()` paired to `tenant.WithContext`. Fix round 2fb6e4ec0 closed a second origination gap (listener-registration context in `socket/init.go`'s `NewListenerContext`) found during review — properly fixed, not deferred. Current tree: zero leftover `SetHeaderParsers`/`RootUrl(` sites in `services/atlas-login`. |
| 32 | `atlas-channel` — wiring + socket-edge origination | DONE | Commit 0f0a55bfc plus fix commits 4544cdb59/f81cc5be0/e3201c364/e1b8d4e7e/99c0e598d completed the conversion (large service, plan explicitly permits multi-commit completion for size). Current tree: zero leftover `RootUrl(` sites; the only `SetHeaderParsers` without `EnvHeaderParser` is `configuration/projection/subscriber.go:74,82` — explicitly excluded by the task text ("that projection is control-plane and is not converted here; its disposition is Task 42") and by commit message. |
| 33 | Wire environment registry — batch 1 (8 services) | DONE | Commits ce45a37b5, 29fdf3d1c→26837b858, 5edd300e9 (dead-code lint fix). All 8 services (`atlas-account`…`atlas-chalkboards`) have `WithEnvironmentRegistry(serviceName)` in main.go + `wiring_test.go` (verified by fleet sweep). Zero leftover sites. |
| 34 | batch 2 (8 services) | DONE | Commits 6a06ffae0 (batch 4 numbering note: commit subject says "batch 4" — see note below), 7a829d69b "batch 3", 8776709b8. All 8 services (`atlas-character`…`atlas-drops`) verified wired, zero leftover sites, builds/tests pass (spot-checked `atlas-character`, `atlas-inventory` proxy for the batch). |
| 35 | batch 3 (8 services) | DONE | `atlas-effective-stats`…`atlas-keys` verified wired via fleet sweep; zero leftover sites. |
| 36 | batch 4 (8 services) | DONE | Commits 54e7e0c3d, c7d7ba52c (lint fix), f93e110a2. `atlas-kites`…`atlas-mini-games` verified wired; `atlas-marriages` tickers correctly untouched (deferred to Task 41 per plan note) — `main.go:44` shows Bootstrap wired, ticker code present and unmodified by this batch. |
| 37 | batch 5 (8 services) | DONE | Commits 29e7a2ad4 "batch 5 (partial)" → 5d4139eb2 "finish batch 5…and originate on mount expiry" → 836e53768 (dead-code fix). `atlas-monster-book`…`atlas-parties` verified wired; `atlas-mts` ticker correctly untouched (`main.go:54` Bootstrap wired, ticker deferred to Task 42 per plan note). |
| 38 | batch 6 (8 services) | DONE | Commits 566a1cab2 "batch 6 (partial: conversion pass)" → 16d31323d "finish batch 6" → 9a7882e2e "originate environment in batch 6 tick paths". `atlas-party-quests`…`atlas-rates` verified wired; `atlas-party-quests` ticker correctly untouched (`main.go:91-100` PQ timer ticker present, deferred to Task 41); `atlas-quest`'s two Task-24 `producer.Produce` sites not re-edited (consistent with plan note). |
| 39 | batch 7 (8 services, split 39a/39b) | DONE | Commit 187a4b030 "batch 7 (non-saga services)" covers `atlas-reactor-actions`, `atlas-reactors`, `atlas-renders`, `atlas-reward-pools`, `atlas-rps`, `atlas-skills`, `atlas-storage`; commit bfe36ebfe (HANDOFF #11 item 2, dedicated split) covers `atlas-saga-orchestrator` separately, including its 20 `SetHeaderParsers` sites, 18+3 `requests.RootUrl(` sites, and origination in `main.go`'s recovery/reaper sweeps plus `saga/timer.go`'s per-saga timeout backstop (explicitly not a Task 41/42 carve-out — handled here). `atlas-renders` carries `service.WithoutTracer()` alongside the new `WithEnvironmentRegistry` option per plan note (`main.go:27`); `tools/envguard/bootstrap-allowlist.txt` remains empty (no allowlist entry needed since `atlas-renders` does call Bootstrap). Fleet sweep confirms zero leftover sites across all 8 services. |
| 40 | batch 8 (5 services, final) | DONE | Commits 4b7be236d (`atlas-summons`, `atlas-tenants`), a4937474e (`atlas-trades`, wiring + settlement origination), 923916e84 (`atlas-skills` cooldown-expiry origination test — technically a batch-7 follow-up), 418b2caf9 (`atlas-world`, `atlas-transports`, closing commit). `atlas-transports` ticker (C4 reference ticker) correctly untouched — Bootstrap wired at `main.go:59`, ticker deferred to Task 41. `atlas-world` confirmed genuinely 0/0 on `RootUrl(` sites (no REST client in the module) per commit message and fleet sweep. `./tools/env-bootstrap-guard.sh` currently exits 0 with "clean (64 service(s) checked)", matching the plan's Step 5 expectation. |

**Completion Rate:** 13/13 tasks (100%) — counting 29, 29A, 30, 31, 32, and 33–40 (8 tasks) as the unit; sub-batch splits (33a/33b-style renumbering seen in commit subjects, e.g. "batch 4" appearing on what the plan calls batch-2 services) are cosmetic commit-message drift, not scope drift — the service lists match the plan's batch allocation table exactly in every case checked.
**Skipped without approval:** 0
**Partial implementations:** 0 (transient `(partial)` commit messages within a batch were completed by a following commit in the same task; none left unresolved in the final tree)

## Skipped / Deferred Tasks

None skipped. The following are **plan-approved carve-outs**, not gaps, and are called out here for traceability into Tasks 41/42 (out of this shard's scope, per instructions):

- `atlas-channel/configuration/projection/subscriber.go` — control-plane projection, disposition explicitly deferred to Task 42 (Task 32 text).
- `atlas-marriages` two ticker-bearing schedulers — deferred to Task 41 (Task 36 text).
- `atlas-mts` periodic task — deferred to Task 42 (Task 37 text).
- `atlas-party-quests` PQ-timer ticker — deferred to Task 41 (Task 38 text).
- `atlas-transports` C4 reference ticker — deferred to Task 41 (Task 40 text).
- `atlas-saga-orchestrator`'s ticker was explicitly **not** deferred and was originated as part of Task 39 (commit bfe36ebfe) — this is a completion, not a gap, but noted since the plan text for Task 37/39 discusses ticker carve-outs generally.

None of these six items were audited for correctness of the Task 41/42 work itself, per shard instructions.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-service | PASS | PASS | Includes `TestLoggerEmitsTheEnvironmentField` (explicit run confirmed PASS). |
| libs/atlas-rest | PASS | PASS | |
| atlas-monsters | PASS | PASS | |
| atlas-login | PASS | PASS | |
| atlas-channel | PASS | PASS | |
| atlas-world | PASS | PASS | |
| atlas-saga-orchestrator | PASS | PASS | |
| atlas-account, atlas-buffs, atlas-character, atlas-inventory, atlas-maps, atlas-mts, atlas-portals, atlas-skills, atlas-trades, atlas-tenants | PASS | PASS | 10-service sample spanning batches 33–40; no FAIL output. |

Fleet-wide `./tools/env-bootstrap-guard.sh` → exit 0, "clean (64 service(s) checked)".

Not run: full `tools/verify.sh` (flagless) across all 64 services — out of scope for a per-shard audit; module-local build/test was used per the shard's evidence-surface constraint, supplemented by the fleet-wide guard script which is the repo's own mechanized completeness check for exactly this task family.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this shard's scope; overall branch readiness depends on other shards' findings, including Tasks 41/42 which own the deferred ticker conversions noted above)

## Action Items

None required for Tasks 29–40. For awareness of the whole-plan reviewer: confirm Tasks 41 and 42 actually pick up the six deferred sites listed above (atlas-channel projection, atlas-marriages tickers, atlas-mts periodic task, atlas-party-quests ticker, atlas-transports C4 ticker) — this shard confirms they were *correctly excluded* here, not that they were later completed.
