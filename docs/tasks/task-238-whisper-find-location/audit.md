# Plan Audit — task-238-whisper-find-location

**Plan Path:** docs/tasks/task-238-whisper-find-location/plan.md
**Audit Date:** 2026-08-18
**Branch:** task-238-whisper-find-location
**Base Branch:** main (merge base 6fb58bdc5)

## Executive Summary

All 9 plan tasks were implemented and each has a corresponding, independently-written per-task review artifact (`review-task-1.md` … `review-task-9.md`) already checked into the task folder, all verdict `APPROVED`. This audit cross-checked those reviews against the actual diff and re-derived the two controller rulings from source rather than trusting either the plan or the prior reviews: the GM-widening scope (7 other services still carry `gm == 1`, confirmed by grep) and the 8-arm evidence pinning for Task 9 (confirmed 8 distinct `decompile_sha256` values in `docs/packets/evidence/gms_v92/`). All four affected Go modules (`libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-maps/atlas.com/maps`, `services/atlas-channel/atlas.com/channel`) build and test clean. No skipped, deferred, or partial tasks were found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Shared `PresenceState` enum | DONE | `libs/atlas-constants/character/presence.go` (new), `presence_test.go` (new). Exact wire values `"OFFLINE"`/`"IN_FIELD"`/`"IN_CASH_SHOP"` confirmed; `ParsePresenceState` resolves empty/unrecognised to `OFFLINE`. Commit `6839f59e5`. |
| 2 | `atlas-maps` — `state` column, model, `SetState` | DONE | `character/location/{entity,model,administrator,processor,processor_test}.go`. `upsertLocation` rewritten to `OnConflict`/`AssignmentColumns` over position columns only (`administrator.go`), confirmed by reading the code — state is excluded from `DoUpdates`. `SetState`/`SetStateIfOnline` both present on `Processor`. Commit `c99525e21`. |
| 3 | `atlas-maps` — project `state` onto REST surface | DONE | `character/location/rest.go`: `RestModel.State characterconst.PresenceState \`json:"state"\`` and `Transform` carries `State: m.State()`. `resource.go` correctly left untouched. Commit `a56c5b15d`. |
| 4 | `atlas-maps` — LOGIN/LOGOUT/CHANNEL_CHANGED transitions | DONE | `kafka/consumer/character/consumer.go`: all three handlers call `SetState` (unconditional) with correct target states; `CREATED`/`CHANGE_MAP`/`DELETED` handlers untouched. Commit `9254bd6b5`. |
| 5 | `atlas-maps` — cash-shop CHARACTER_ENTER/EXIT transitions | DONE | `kafka/consumer/cashshop/consumer.go`: `db` threaded through `InitHandlers`, both handlers use `SetStateIfOnline` (conditional, not `SetState`); `main.go:93` updated to pass `db`. Commit `296ae8b40`. |
| 6 | `atlas-channel` — state-bearing location client | DONE | `maps/location/requests.go`: `RestModel.State string`, new `Model` type with `State()` getter, `Get()` returning `ErrNotFound` on 404. `GetField`/`resolve.go`/`ResolveMapId` confirmed untouched (no diff on `resolve.go`). Commit `e09eee916`. |
| 7 | `atlas-channel` — GM level semantics | DONE | `character/model.go:69-77`: `Gm() bool { return m.gm > 0 }`, `GmLevel() int`. `session/model.go` gains `Gm()` getter. Verified repo-wide via grep that the other 7 services (atlas-npc-shops, atlas-cashshop, atlas-login, atlas-pets, atlas-consumables, atlas-messages, atlas-query-aggregator) still carry `gm == 1`, unmodified — matches the controller's explicit non-goal ruling. Commit `98ec7fd7f`. |
| 8 | `atlas-channel` — `/find` decision table | DONE | `socket/handler/character_chat_whisper.go`: `findDecision` implements the 10-row table (FR-1…FR-7 plus split arms) in the exact documented order, first-match-wins, read directly from source and confirmed to match the plan's table row-for-row (unresolved → cross-world → gm-concealed → cash-shop-local → map-local → cash-shop-remote → channel-remote → offline → never-logged-in → lookup-failed). Both `// TODO` placeholders removed (confirmed no `TODO` remains in the file). `channelId: uint32(loc.ChannelId())` — no hidden ±1 adjustment, matching the plan's explicit channel-arithmetic constraint. `location.ResolveMapId` not called in this path (only referenced in an explanatory comment). Commit `8ca62121b`. |
| 9 | Promote `gms_v92` clientbound WHISPER matrix cell | DONE | `docs/packets/ida-exports/gms_v92.json`: all 8 `CField::OnWhisper#*` arms now resolved (0 `unresolved` remaining, confirmed by direct grep-equivalent check). `docs/packets/evidence/gms_v92/` carries all 8 `field.clientbound.FieldWhisper*.yaml` files with 8 distinct `decompile_sha256` values (independently confirmed, not merely accepted from the prior review). `status.json` diff confirmed to move only the `gms_v92` `exportHashes` entry and the single target op-row cell (`incomplete`→`verified`); no unrelated v92 cell moved. `libs/atlas-packet/field/clientbound/whisper.go` has zero diff (no wire change) — confirmed via `git diff --stat` on that path. Task deliberately scoped 8-arm evidence rather than just `FieldWhisperError`, per controller ruling 2. Commit `17873fb17`. |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task has code evidence in the branch diff, and the two apparent scope questions (GM widening breadth, Task 9's evidence breadth) are both controller-ruled non-issues, independently re-verified above rather than taken on faith.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-constants` | PASS | PASS | `go build ./... && go test ./...` clean, all packages including `character`. |
| `libs/atlas-packet` | PASS | PASS | `go build ./... && go test ./...` clean, including `field/clientbound` (carries `TestWhisperByteOutputV92`). |
| `services/atlas-maps/atlas.com/maps` | PASS | PASS | `go build ./... && go test ./... -count=1` clean across all packages, including `character/location`, `kafka/consumer/character`, `kafka/consumer/cashshop`. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | Initial run hit `go build` cache contention from a concurrently-running `tools/verify.sh` on the same working tree (stale `$GOCACHE` object references, an environment artifact unrelated to this branch's code); re-run with an isolated `$GOCACHE` built and tested clean, `go test ./... -count=1` produced no `FAIL` lines across the full module including `socket/handler` (carries the `/find` decision-table tests) and `maps/location`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. No blocking or non-blocking defects found in this audit's independent re-verification of the branch diff, the two controller-ruled scope questions, and the module-local build/test gates.
