# Plan Audit — task-122-attack-path-snapshots (Tasks 1-7)

**Plan Path:** docs/tasks/task-122-attack-path-snapshots/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-122-attack-path-snapshots
**Base Branch:** main (merge base `2537d8a6a`)
**Scope:** Tasks 1-7 of 14 only (sibling shard covers 8-14)

## Executive Summary

Tasks 1-7 are fully implemented and match the plan's interfaces, behavior, and test
coverage exactly, with two deliberate, well-documented deviations from the literal
code blocks in the plan (both improvements, not regressions): (1) Task 3's `Get()`
degrades to a partial model on inventory/skills REST-fallback failure instead of
erroring the whole read (commit `844a59b72`), matching the actual pre-existing
`character.ProcessorImpl` decorator behavior rather than the plan's draft code; (2)
Task 4's `ForCharacter` uses `routine.Go` instead of a raw goroutine, consistent with
project convention. Task 1's execution-time verification steps were performed
honestly, including flagging and correcting one stale escalation (`RequestReserve`,
already fixed by task-205) rather than reporting a false positive — verified directly
against `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:810-853`.
All affected atlas-channel packages build, vet, and test clean (module-local, no
`-race`/bake per this audit's scope). No skipped or partial tasks found in this range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | task-120 reconciliation gate + execution-time verifications | DONE | `docs/tasks/task-122-attack-path-snapshots/context.md:70-113` records all 6 verification steps with file:line citations; the `RequestReserve` escalation was correctly retracted (confirmed live against `services/atlas-inventory/.../compartment/processor.go:822-844`, which loops over all requests, not just the first). The one open delta (`/api/metrics` promhttp mount absent) was flagged for Task 10/12 rather than silently assumed — out of this shard's scope. |
| 2 | Snapshot registry — entry, mutators, metrics, builder enablers | DONE | `services/atlas-channel/atlas.com/channel/character/snapshot/registry.go` (682 lines) and `metrics.go` (65 lines) implement every symbol in the plan's Interfaces block verbatim: `GetRegistry`, `ComponentView`, `View`, `ComposedIfValid`, all four `Backfill*`, all update-only mutators, `Evict`/`EvictTenant`, metric constants/helpers. `character/skill/builder.go` matches the plan's code block exactly. `character.builder.go` `SetX`/`SetY` were confirmed pre-existing at merge base `2537d8a6a` (`git show 2537d8a6a:.../character/builder.go` already has them at line 138-139) — correctly a no-op, not a gap. `registry_test.go` has 17 tests covering generation-guard, no-op-when-absent, aliasing safety, tenant isolation, and concurrency. |
| 3 | Snapshot processor — Get, composition, per-component fallback + backfill, GetBuffs | DONE | `services/atlas-channel/atlas.com/channel/character/snapshot/processor.go` implements `NewProcessor`, `Get`, `GetBuffs`, `BuffsProvider`, and the four fetch seams exactly as specified. Deviation from the plan's literal code (core fallback failure still errors; inventory/skills fallback failure now degrades instead of erroring, per commit `844a59b72` "fix(atlas-channel): snapshot inventory/skills fallback failures degrade instead of erroring") is a documented correctness fix matching real `character.ProcessorImpl` decorator behavior — not a silent drop. `processor_test.go` has 10 tests including the degraded-fallback cases. |
| 4 | Lifecycle — session-destroy eviction, tenant evictor, movement position feed | DONE | `session/processor.go:416` `snapshot.GetRegistry().Evict(p.t, s.CharacterId())` inside `Destroy`; `main.go:312` `snapshot.GetRegistry().EvictTenant(tid)` in the evictor block; `movement/processor.go:79-84` folds synchronously and calls `SetPosition` before either producer/broadcast goroutine spawns. `movement/processor_test.go` has `TestForCharacter_FeedsSnapshotPositionSynchronously` and `TestForCharacter_NoEntryNoCreate`, both passing. |
| 5 | Character-status consumer snapshot handlers | DONE | `kafka/consumer/character/consumer.go:556-621` implements `handleSnapshotStatChanged`, `handleSnapshotLevelChanged`, `handleSnapshotExperienceChanged`, `handleSnapshotMapChanged`, all registered in `InitHandlers` (lines 95-113) on the same status topic as the existing handlers. `MapChanged` disposition (position+core invalidate on `UseTargetPosition=false`) matches design §10.4 exactly. `consumer_test.go` has 6 tests covering rich/nil-values, level/exp, map-changed both branches, and world-filtering. |
| 6 | Skill consumer snapshot handlers (incl. new DELETED handler) | DONE | `kafka/message/skill/kafka.go:46,76-77` adds `StatusEventTypeDeleted = "DELETED"` and empty `StatusEventDeletedBody`, confirmed matching the producer at `services/atlas-skills/atlas.com/skills/skill/producer.go:124` (`skill2.StatusEventTypeDeleted`). `kafka/consumer/skill/consumer.go:200-238` implements all three handlers, registered in `InitHandlers`. |
| 7 | Asset + compartment consumer snapshot handlers | DONE | `kafka/consumer/asset/consumer.go:618-724` implements all 8 handlers (`Created/Updated/Accepted/QuantityChanged/Moved/Deleted/Released/Expired`), all registered (lines 97-132), reusing existing `buildAssetFromCreatedBody`/`UpdatedBody`/`AcceptedBody` rather than duplicating. `kafka/consumer/compartment/consumer.go:212-277` implements all 5 handlers (`Created/Deleted/CapacityChanged/MergeComplete/SortComplete`), all invalidate-only, all registered (lines 63-83); no handlers for `RESERVED`/`RESERVATION_CANCELLED`, matching the disclosed REST-parity exception confirmed at Task 1 Step 3. `consumer_test.go` in both packages cover every handler, including a table-driven `TestHandleSnapshotCompartmentEvents_Invalidate` exercising all 5 compartment event types. |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range.

## Non-blocking observation (not a task-1-7 finding)

`movement/processor.go`'s `TeleportCharacter` (pre-existing, task-176-era inner-portal
handling, unrelated to this plan) does not call `snapshot.GetRegistry().SetPosition`,
unlike `ForCharacter`. Neither `plan.md` nor `design.md` mentions `TeleportCharacter`
in any task, so this is outside Task 4's stated acceptance criteria and not a defect
in Task 4 as planned — flagging only in case a later task (8-14) or the controller
wants to confirm this was a deliberate scope boundary and not an oversight in FR-2.5's
"position overlay always fresh" guarantee.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (module-local) | PASS | PASS | `go build ./...` clean; `go vet ./...` clean; `go test ./character/snapshot/... ./kafka/consumer/character/... ./kafka/consumer/skill/... ./kafka/consumer/asset/... ./kafka/consumer/compartment/... ./movement/... ./session/... -count=1` all `ok`. Full-module `-race` run and `docker buildx bake` are out of this shard's scope per dispatch instructions (already passed flagless at HEAD per the controller). |

## Overall Assessment

- **Plan Adherence:** FULL (for Tasks 1-7)
- **Recommendation:** READY_TO_MERGE (pending the sibling shard's verdict on Tasks 8-14)

## Action Items

None required for Tasks 1-7.
