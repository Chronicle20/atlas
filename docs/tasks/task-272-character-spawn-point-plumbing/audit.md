# Plan Audit — task-272-character-spawn-point-plumbing

**Plan Path:** docs/tasks/task-272-character-spawn-point-plumbing/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-272-character-spawn-point-plumbing
**Base Branch:** main (commit range b284bcebf..61e5e4b94)

## Executive Summary

All 9 tasks in the plan (8 implementer tasks + 1 acceptance sweep) were completed exactly as specified, with faithful file-for-file, line-for-line correspondence between the plan's prescribed diffs and the actual commits. All eight `Model.SpawnPoint()` accessors now return `uint32` from `m.spawnPoint`; the two wire narrowing casts (`byte(c.SpawnPoint())`) are present at exactly the two specified call sites; the four already-correct `Extract`s were left untouched; `atlas-character` and `libs/atlas-packet` show zero diff. Every new/amended test fixture uses a non-zero `spawnPoint` value and none rely on `Extract∘Transform` idempotence alone to prove the fix. Independent re-verification (build + targeted tests + full test suite) passed for all 8 affected Go modules with no failures.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-channel — accessor, Extract, CHARACTER_DATA wire cast | DONE | `character/model.go:240` un-stubbed to `uint32`/`m.spawnPoint`; `rest.go:149` adds `spawnPoint: m.SpawnPoint,` to `Extract`; `socket/writer/character_data.go:47` casts `byte(c.SpawnPoint())`; `rest_test.go` and new `character_data_test.go:TestBuildCharacterData_SpawnPoint` (in-range + truncation subtests) added exactly as specified. Commit c4b1fd3be. |
| 2 | atlas-login — accessor, Extract, CHARACTER_LIST wire cast | DONE | `character/model.go:222` un-stubbed; `rest.go:153` adds `SetSpawnPoint(m.SpawnPoint).` to the builder chain; `socket/writer/character_list.go:56` casts `byte(c.SpawnPoint())`; stale doc comment in `rest_test.go` corrected as specified; new `character_list_test.go:TestToCharacterListEntry_SpawnPoint` added (in-range + truncation). Commit a03a83ea4. |
| 3 | atlas-query-aggregator — accessor and un-cast REST re-serve | DONE | `character/model.go:224` un-stubbed; `rest.go:128` drops the `uint32(...)` cast on `Transform`'s `SpawnPoint:` field, preserving uint32 fidelity; `Extract` (line 139) confirmed untouched (Task 9 diff check below); new `rest_test.go` adds `TestExtract_SpawnPoint` and `TestTransform_SpawnPointPreservesUint32` (300, above byte ceiling). Commit 299779b35. |
| 4 | atlas-cashshop — accessor and Extract | DONE | `character/model.go:211` un-stubbed; `rest.go:152` adds `spawnPoint: m.SpawnPoint,` to `Extract`; `rest_test.go` fixture changed from `SpawnPoint: 0` to `SpawnPoint: 11` and assertion added. Commit c027e20d3. |
| 5 | atlas-pets — accessor, Extract, and Transform | DONE | `character/model.go:207` un-stubbed; `rest.go` adds `spawnPoint: m.SpawnPoint` to `Extract` AND `SpawnPoint: m.spawnPoint` to `Transform` (the one service needing both legs per design §5.1); `rest_test.go` fixture gains `spawnPoint: 55`, plus new `TestExtract_SpawnPoint`. The other ~26 dropped pets fields were left untouched — no scope creep. Commit 2dc3d66a5. |
| 6 | atlas-npc-shops — accessor, positional fields, builder setters | DONE | `character/model.go:208` un-stubbed; `rest.go:161-163` adds `x: rm.X`, `y: rm.Y`, `stance: rm.Stance` to `Extract` (the pre-existing `spawnPoint: rm.SpawnPoint,` line left alone, confirmed by diff); `builder.go` adds `SetX`/`SetY`/`SetStance` one-line setters in the gofmt-aligned block form specified; `rest_test.go` adds direct-literal assertions for `SpawnPoint()`/`X()`/`Y()`/`Stance()` plus a new builder-anchored `TestTransform_PositionalFieldsFromBuilder`. Commit 64344f5d8. |
| 7 | atlas-consumables — accessor only | DONE | `character/model.go:213` un-stubbed; `rest.go` untouched (confirmed by Task 9's diff check — only the query-aggregator `Transform` cast removal appears in that diff); `rest_test.go` adds the `275`-valued (above byte ceiling) assertion against the existing fixture. Commit f56e6fc70. |
| 8 | atlas-messages — accessor only | DONE | `character/model.go:205` un-stubbed; `rest.go` untouched (confirmed by Task 9's diff check); the separate `Stance()` stub at `model.go:225` was left alone as instructed (out of scope); `rest_test.go` adds the `SpawnPoint() != 11` assertion to the existing round-trip fixture. Commit e4f376d2e. |
| 9 | Acceptance sweep and repo-wide gate | DONE | All six acceptance steps re-run independently and reproduced: (1) exactly nine `func (m Model) SpawnPoint()` hits (8 fixed + atlas-character), all `uint32`; (2) zero `return 0` survivors; (3) `git diff --stat b284bcebf -- services/atlas-character libs/atlas-packet` empty; (4) the four "unchanged Extract" diff for consumables/messages/query-aggregator shows only the query-aggregator `Transform` cast-removal hunk, no `Extract` body touched; (5) both wire casts present (`character_data.go:47`, `character_list.go:56`); (6) prior report states flagless `tools/verify.sh` exited 0 on 61e5e4b94 — not re-run here per the task brief's "already established" instruction, but independently corroborated by re-running `go build ./...` and `go test ./... -count=1` for all 8 modules below, all green. Docs commit 61e5e4b94 adds per-task review artifacts and an agent ledger; no production code in that commit. |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped, partially implemented, or silently deferred. All plan-specified test fixtures use non-zero `spawnPoint` values; no test relies on `Extract∘Transform` round-trip equality as the sole evidence of the fix (each service's diff includes at least one direct-literal assertion, per the plan's Global Constraints). The npc-shops builder setters (`SetX`/`SetY`/`SetStance`) were added with no production caller, exactly as the plan calls out as expected (design §5.2 overriding PRD FR-8) — not a gap.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; targeted `TestTransformRoundTrip` and `TestBuildCharacterData_SpawnPoint` (both subtests) pass. |
| atlas-login | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; targeted `TestTransformRoundTrip` and `TestToCharacterListEntry_SpawnPoint` (both subtests) pass. |
| atlas-query-aggregator | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestExtract_SpawnPoint` and `TestTransform_SpawnPointPreservesUint32` pass. |
| atlas-cashshop | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestTransformRoundTrip` passes. |
| atlas-pets | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestExtract_SpawnPoint` passes. |
| atlas-npc-shops | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestTransformRoundTrip` and `TestTransform_PositionalFieldsFromBuilder` pass. |
| atlas-consumables | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestTransformRoundTrip` passes. |
| atlas-messages | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` no failures; `TestTransformRoundTrip` passes. |
| repo-wide (`tools/verify.sh`) | PASS | PASS | Per task brief, already established: flagless `tools/verify.sh` exited 0 on 61e5e4b94 (not re-run in this audit; corroborated independently by the per-module build/test runs above). |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. No fixes required before this plan can be considered complete.
