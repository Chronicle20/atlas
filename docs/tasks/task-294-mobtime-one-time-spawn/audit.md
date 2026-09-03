# Plan Audit — task-294-mobtime-one-time-spawn

**Plan Path:** docs/tasks/task-294-mobtime-one-time-spawn/plan.md
**Audit Date:** 2026-09-03
**Branch:** task-294-mobtime-one-time-spawn
**Base Branch:** main

## Executive Summary

All 7 plan tasks are fully implemented and match the plan's specified interfaces, code, and comments essentially verbatim, with three legitimate self-correcting follow-up commits (a rediskeyguard-compliance fix, an atomicity restoration via Lua, and a mop-up of leftover dead mock methods outside Task 7's authorized file list). `go build ./...` and `go test ./... -count=1` both pass cleanly across all packages in `services/atlas-maps/atlas.com/maps`, and `go test -race` on the touched packages (`map/monster`, `map`) is also clean. No `Spawnable` references remain anywhere in the module. No task was skipped, stubbed, or silently deferred.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Carry `Hide` and classify spawn points in the data package | DONE | `data/map/monster/model.go:17` (`Hide bool`), `rest.go:19,52`, `classify.go` (new, matches plan verbatim), `classify_test.go` (`TestExtractCarriesHide`, `TestClassify`) — commit `b7cdb2716` |
| 2 | Split the registry into recurring / one-time / meta hashes | DONE | `map/monster/registry.go` — `SpawnPointRegistry{hashes, oneTime, meta}`, `newRegistry`, `recurringKey/oneTimeKey/metaKey`, `initializeScript` (3-key Lua), `Count`/`CountOneTime`; `registry_test.go:171,273,324,368` (`TestInitializeForMap_PartitionsByMobTimeAndHide`, `TestInitializeForMap_IsIdempotent`, `TestRegistryKeys_AreV2AndDistinct`, `TestFlushTenant_ClearsAllThreeHashes`) — commit `84c6c5695`. Note: `InitializeForMap` carries one extra back-compat branch (recurring-hash-len fallback when unmarked-seeded) not in the plan's Step 5 listing — additive safety net, does not contradict any plan requirement. |
| 3 | Atomic one-time claim and re-arm | DONE | `registry.go` — `claimOneTimeScript` (HSETNX), `ClaimOneTimeSpawnPoints`, `RearmOneTime` — commit `6dbf44b86`. Plan's Step 4 specified `RearmOneTime` via raw `r.client.HDel`; actual code uses `rearmOneTimeScript` (Lua HDEL) after two follow-up commits (`2c3cb80cd` routed through `TenantKeyedHash` for the redis key-guard, then `16fc28cde` restored atomicity via Lua because the guard fix broke the exactly-once HDEL semantics). Functionally equivalent/stronger than plan text; concurrency test `TestRearmOneTime_ConcurrentTrueExactlyOnce` added and passing. |
| 4 | Fire the one-time batch in `SpawnMonsters` | DONE | `map/monster/processor.go:97-131` — claim-and-fire block, recurring-only denominator (`registry.Count`), `CountOneTime` zero-branch log, updated `SpawnMonsters` doc comment (lines 72-77) — matches plan text verbatim. Commit `54ea73400`. |
| 5 | `DESTROY_FIELD` command envelope and producer | DONE | `kafka/message/monster/kafka.go` — `EnvCommandTopic`, `CommandTypeDestroyField`, `FieldCommand[E]`, `DestroyFieldBody{}` (all field-for-field with atlas-monsters' `fieldCommand`/`destroyFieldCommandBody` per `services/atlas-monsters/.../kafka/consumer/monster/kafka.go:13,26,103,252`); `map/producer.go` — `destroyFieldCommandProvider`; `map/producer_test.go:23` — `TestDestroyFieldCommandProvider_MatchesConsumerEnvelope` (cross-service seam test). Commit `b850f4073`. |
| 6 | Re-arm on field empty in `map.ProcessorImpl.Exit` | DONE | `map/processor.go:107-134` — field-empties block calling `RearmOneTime` and conditionally emitting `DESTROY_FIELD`, matches plan text verbatim including comments. Commit `4619cd5af`. Supporting `TestMain` changes in `kafka/consumer/cashshop/testmain_test.go` and `kafka/consumer/character/testmain_test.go` (registry init against miniredis) were required because `Exit` now reaches the registry singleton — not explicitly named in Task 6's file list but a necessary, correctly-scoped consequence. |
| 7 | Remove the dead `Spawnable` surface | DONE | `data/map/monster/processor.go` — interface shrunk to exactly `SpawnPointProvider`/`GetSpawnPoints`, no `Spawnable`/`SpawnableSpawnPointProvider`/`GetSpawnableSpawnPoints` remain; `services/atlas-maps/docs/domain.md:350-351` updated as specified. Commit `125bb8459`, plus follow-up `4457c3547` which removed four additional dead mock methods on `mockMonsterDataProcessor` (map/processor_test.go) and `mockSpawnDataProcessor` (map/monster/registry_test.go) that Task 7's authorized file list missed. `grep -rn "GetSpawnableSpawnPoints\|SpawnableSpawnPointProvider\|Spawnable("  services/` returns zero hits after `4457c3547`. |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No plan task was skipped, stubbed, or left incomplete.

Note on plan checkbox state: all 52 step-level checkboxes in `plan.md` remain `- [ ]` (unchecked) despite the work being fully implemented and committed. This is a documentation-hygiene gap, not an implementation gap — every step's code, test, and commit evidence is present in the branch history. Flagged for completeness; not a blocking finding since the code and tests independently confirm completion.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-maps (services/atlas-maps/atlas.com/maps) | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` — all packages `ok`, no failures. `go test -race ./map/monster/... ./map/...` — PASS, no race reports. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Non-blocking, cosmetic) Check off the plan's step-level checkboxes in `plan.md` to reflect actual completion state, or note in the PR why they were left unchecked.
