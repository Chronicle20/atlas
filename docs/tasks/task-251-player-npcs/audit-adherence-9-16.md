# Plan Audit — task-251-player-npcs (Tasks 9-16)

**Plan Path:** docs/tasks/task-251-player-npcs/plan.md
**Audit Date:** 2026-08-22
**Branch:** task-251-player-npcs
**Base Branch:** main
**Scope:** Tasks 9-16 only (plan.md lines 595-1149)

## Executive Summary

All 8 tasks in this range (9-16) are implemented, match the plan's tables/signatures, and
correctly incorporate the two controller rulings named in progress.md: `Placement.ScriptId`
is `uint32` (`position/types.go:37`), and equipment is sourced from `atlas-inventory` via a
dedicated `inventory/` client package rather than `atlas-character`. `go build ./...` and
`go test ./... -count=1` from the module root both pass cleanly (17 packages, all `ok`). Two
review-round findings recorded in progress.md (Task 13's `_map.Id` typing, Task 16's
`conversationPath` hardcoded `false`) were independently verified fixed in the current source.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 9 | `routing/` — job category → Hall of Fame map | DONE | `routing/routing.go` implements `HallOfFameMapFor`, `IsPodiumMap`, `IsHallOfFameMap`, `JobCategory` exactly per plan table; Pirate-wire-500 resolved via `constants.SkillJobSet`/`job.IsAIdentity` (routing.go:78-80); Noblesse exclusion at routing.go:94-98. `routing` package tests pass. |
| 10 | `allocation/` — script-id pool | DONE | `allocation/allocation.go` implements `PoolMin/PoolMax/BranchBase/BranchSize`, `BranchFor(set, jobId, mapId)` (the corrected 3-arg signature per progress.md's Task 10 ruling), `BranchRange`, `Allocate`; GM formula `26 + 4*(mapId/100000000)` at allocation.go:64. `pool.go` present for the impure usable-set builder. `allocation` tests pass. |
| 11 | `position/` — grid and podium positioners | DONE | `position/types.go`, `grid.go`, `podium.go` implement `Tuning`, `Rect`, `Point`, `Placement`, `SnapFunc`, `GridPitch`, `NextGridPosition`, `Reorganize`, `PodiumPosition`, `Encode/DecodePodiumState`, `RaisePodiumStep`. **`Placement.ScriptId` is `uint32`** (types.go:37) — confirms the recorded Task 11 follow-up ruling (commit `5dd60245a` per progress.md) actually landed in source, not just claimed. `position` tests pass. |
| 12 | Persistence — entity/model/builder/administrator/provider | DONE | `playernpc/entity.go` defines both `Entity` (player_npcs) and `EquipmentEntity` (player_npc_equipment) with the PRD §6 unique indexes (`idx_pn_tw_script`, `idx_pn_tw_map_name`, `idx_pn_tw_map_object`) and cascade-delete association; includes `OverallJobRank` column (entity.go:50) and the design-required `Step` column (entity.go:20-30, added by Task 15's later resolution, documented in the entity's own doc comment). `playernpc` package tests pass (includes `model_test.go`, `administrator_test.go`). |
| 13 | Read clients | DONE (post-fix) | Five client packages present: `character/`, `data/npc/`, `data/map/`, `ranking/`, `configuration/`, plus `inventory/` (added in the fix round, superseding the plan's original character-based equipment source per the controller ruling). `data/map/model.go:5,24` uses `_map.Id` (not raw `uint32`), confirming the review-blocking finding (`_map.Id` retype, commit `cc6851b6e`) is applied in current source. All five+one client packages' tests pass. |
| 14 | `snapshot/` and `eligibility/` | DONE | `snapshot/snapshot.go` implements `Capture(characterId, worldId, cp, ip, rp)` — signature matches the "New signature Task 15 consumes" block in progress.md, sourcing equipment from `inventory.Processor` (snapshot.go:79,84) not `character`. Cash/masked slot convention implemented at snapshot.go:118-159 (visible = `Position*-1`, masked = `+100`, FR-5.2 1-11/101-111 boundary via `equipmentPositions`). `eligibility/eligibility.go` implements `Evaluate(cfg, char, existingCount, conversationPath)` with the design §9.1 shared predicate (conversationPath gate at eligibility.go:34-36 ahead of level/gm/duplicate checks). `snapshot` and `eligibility` tests pass. |
| 15 | The deploy transaction | DONE | `playernpc/processor.go` implements `Deploy`/`Redeploy`/`RemoveById`/`Remove`/`GetById`/`GetByMap` per the recorded contract. `advisoryLock` (administrator.go:92-113) issues `pg_advisory_xact_lock` inside `database.ExecuteTransaction` (processor.go:193-194) — transaction-scoped per design. `resolvePodiumPosition`/`podiumRank` present (processor.go:430-472) with the `world_job_rank - 1` semantics the fix round (commit `92be6a7cc`) verified and pinned with a podium subtest. Event-after-commit discipline present (`DEPLOYED`/`REPOSITIONED` constants, emitted post-commit per processor.go:33-37 doc and code structure). `playernpc` tests pass (5.4s, includes `processor_test.go`). |
| 16 | REST resource | DONE (post-fix) | `playernpc/rest.go` registers all plan-required routes under `/player-npcs`, with `/eligibility` registered **before** `/{id}` (rest.go:181-186, comment explicitly cites the ordering hazard). `handleGetEligibility` calls `eligibility.Evaluate(cfg, c, existingCount, true)` (rest.go, conversationPath literal `true`) — confirms the Task 16 blocking-finding fix (commit `d908c3c95`, `false`→`true`) is present in current source, not just claimed in progress.md. `main.go:122` wires `playernpc.InitializeRoutes(GetServer())(db)`. Seven `.bruno/*.bru` request files present under `services/atlas-player-npcs/.bruno/` (Deploy, Redeploy, Get, Get list, Get Eligibility, Delete, Delete-by-character), matching the fix round's stated addition. `resource_test.go` (25.0K) present; `playernpc` package tests pass. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. No task was found unchecked, silently skipped, or deferred beyond what
progress.md already records and resolves (the Task 11 `Placement.ScriptId` follow-up and the
Task 13 equipment-source rewire were both controller-ruled corrections, and both are confirmed
landed in source above, not merely claimed).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-player-npcs | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — 17 packages, all `ok` (routing, allocation, position, playernpc, character, data/map, data/npc, ranking, configuration, inventory, eligibility, snapshot, kafka/consumer/*). No failures, no skips. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch verdict depends on
  the other shards' ranges and any outstanding cross-cutting findings such as the repo-wide
  SQLite FK-pragma issue noted in progress.md, which is explicitly out of scope for task-251).

## Action Items

None for Tasks 9-16. Two items are recorded in progress.md as constraints for *later* tasks
(21, 22) in this plan and are out of this shard's range — they are not action items against
Tasks 9-16's own completeness:

1. Task 21 (bulk `Remove` is N transactions/emits, not atomic) — informational, correctly
   documented as an accepted, non-blocking design tradeoff by the Task 15 review.
2. Task 22 (eligibility endpoint's `worldId` defaults to 0; caller must pass it explicitly) —
   correctly identified as the calling task's responsibility, not a Task 16 defect.
