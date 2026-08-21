# Plan Audit — task-252-jukebox-cash-item

**Plan Path:** docs/tasks/task-252-jukebox-cash-item/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-252-jukebox-cash-item
**Base Branch:** main (merge base d17404dbc)

This is a re-verification pass over the **full branch** (d17404dbc..a3f2519e9),
superseding the prior audit taken at commit 6360d7538. It re-confirms Tasks
1-8 (unchanged since the prior pass) and newly reviews the six commits the
prior pass never saw: `6360d7538`, `0c478156f`, `3ab503d70`, `8765e7009`,
`22db8d5fa`, `03d44e3da`, `a3f2519e9` — the last five of which constitute
Task 9 (packet coverage matrix cell for
`cash/serverbound/CashItemUseSongPlayer`), added post-plan at the user's
explicit direction, spec'd by
`docs/tasks/task-252-jukebox-cash-item/coverage-manifest.yaml` and
`cash-item-use-song-player-coverage.md`.

## Executive Summary

All 8 original plan tasks remain fully implemented (re-confirmed, unchanged
since the prior pass). Task 9 (packet coverage) is also complete: the
`CashItemUseSongPlayer` codec now has a working `candidatesFromFName` linkage,
byte-fixture tests, pinned IDA evidence across all ten in-scope GMS/JMS
versions (8 verified, 2 correctly dispositioned n-a), and a scoped one-line
fix to a matrix-builder writer-consumption grading defect that was masking
two already-correct cells. All four `packet-audit` gates (`matrix --check`,
`fname-doc --check`, `operations --check`, `dispatcher-lint`) exit 0, and
`libs/atlas-packet` and `tools/packet-audit` build and test clean. Two small
post-plan fix commits (`6360d7538`, `0c478156f`) correct real defects
(`ineffassign` dead store; a missing `api2go` relationship-stub interface
that would have broken GET decoding) without changing wire behavior or test
assertions. A third fix commit (`22db8d5fa`) repairs a genuine `-race` data
race in a test helper's concurrent-append pattern, preserving the exact
assertions. A fourth (`a3f2519e9`) corrects a self-reported false claim in
the coverage doc (a stale tool-hash that made `matrix --check` not actually
reproduce at the prior commit) — itself evidence the process is holding
itself to the "quote actual tool output" standard rather than the claim.
All affected Go modules build and test clean, including a targeted `-race`
run of every jukebox-specific test package. The three previously-documented,
deliberately-deferred items are re-confirmed as unchanged and are the only
gaps of their kind.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `ItemUseSongPlayer` serverbound codec | DONE | Unchanged since prior pass (commit 650bf48db); re-confirmed present at `libs/atlas-packet/cash/serverbound/item_use_song_player.go`. |
| 2 | `play_jukebox` saga action in `libs/atlas-saga` | DONE | Unchanged since prior pass (commit ba81811b9); re-confirmed. |
| 3 | `PLAY_JUKEBOX` command in atlas-saga-orchestrator | DONE | Unchanged since prior pass (commit 8fba7b65e); re-confirmed. |
| 4 | atlas-maps jukebox registry, processor, REST model | DONE | Unchanged since prior pass (commit 3278606dd); `Get`/`GetActive` still do not filter `ExpiresAt` (only `GetExpired`, `map/jukebox/registry.go:71`) — see Skipped/Deferred. |
| 5 | atlas-maps jukebox command consumer, expiry sweep, REST resource | DONE | Unchanged since prior pass (commit fab90abc4); re-confirmed. |
| 6 | atlas-channel jukebox REST client | DONE | Base transcription unchanged (commit 74d22004a); **improved** by commit `0c478156f`, which adds `SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs to `jukebox/rest.go` so `api2go` does not error decoding the GET response (EXT-01, copied from `worldbroadcast/rest.go`'s existing pattern). `TestGetActiveDecodesTheJukeboxResource` passes with `-race`. |
| 7 | atlas-channel type-20 item-use arm | DONE | Base arm unchanged since prior pass (commit 1886ade90); **improved** by commit `6360d7538`, which removes a dead `updateTime` store flagged by `ineffassign` and replaces it with a comment explaining `Decode` still consumes the trailing GMS≤v84 field regardless of capture. All four named tests re-run and pass with `-race` (`TestJukeboxArmSuccessCreatesTwoStepSaga`, `TestJukeboxArmRejectsSlotTemplateMismatch`, `TestJukeboxArmRejectsZeroSoundLength`, `TestJukeboxArmRejectsUnresolvableCharacter`). |
| 8 | atlas-channel jukebox broadcast and map-enter replay | DONE | Base handlers unchanged since prior pass (commit 720374ab5); **fixed for correctness** by commit `22db8d5fa`, which replaces a bare captured slice in `stubDoorAnnounceForJukebox` (`kafka/consumer/map/consumer_test.go`) with a mutex-guarded `jukeboxAnnounceRecorder`, because `handleStatusEventJukeboxStart/End` fan the announce out one goroutine per session via `ForSessionsInMap`→`ExecuteForEachSlice`, and the unsynchronized append corrupted the slice header under `-race`. All seven named tests re-run and pass with `-race`, including the byte-exact `ff ff ff ff` assertion in `TestHandleStatusEventJukeboxEnd_BroadcastsExactlyMinusOne`. |
| 9 | Packet coverage matrix cell: `cash/serverbound/CashItemUseSongPlayer` (post-plan, user-directed) | DONE | Three-commit arc (`3ab503d70`, `8765e7009`, `a3f2519e9`), spec'd by `coverage-manifest.yaml`/`cash-item-use-song-player-coverage.md`. Adds the missing `candidatesFromFName` entry in `tools/packet-audit/cmd/run.go` (without which no audit report could ever generate for this struct); byte-fixture tests + `packet-audit:verify` markers + pinned IDA evidence for 8 versions (gms_v72/79/83/84/87/92/95, jms_v185), each independently confirmed against a distinct decompiled arm address; correct n-a dispositions with positive evidence for gms_v48 (case 20 dispatched but is teleport-rock, a different feature — the cash-slot type-mapping scheme differs pre-v83) and gms_v61 (case 20 absent from the switch, verified by full jumptable enumeration); a scoped one-entry fix to `legacyConsumedSiblingWriters` (`tools/packet-audit/internal/matrix/build.go:116-118`) that unmasked two already-correct cells (gms_v72/v79) without force-promoting anything or touching the four sibling writers sharing the same fname collision. `matrix --check`, `fname-doc --check`, `operations --check`, `dispatcher-lint` all independently re-run and exit 0 in this audit (not just claimed). Commit `a3f2519e9` is itself a self-correction of a stale `Tool:`/`toolSha` mismatch in the coverage doc from the prior sub-commit — confirms the process caught and fixed its own false verification claim before this audit, rather than this audit catching it. |

**Completion Rate:** 9/9 tasks (100%) — 8 original plan tasks + the
post-plan Task 9 explicitly authorized by the user
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None skipped. Three previously-documented, deliberately-deferred items are
re-confirmed unchanged and are the only gaps of their kind — none were
introduced or masked by the six new commits:

- **`services/atlas-maps/atlas.com/maps/map/jukebox/registry.go`'s
  `Get`/`GetActive` do not filter on `ExpiresAt`** (only `GetExpired` does,
  line 71) — pre-existing, inherited verbatim from the weather registry
  template. Re-confirmed via direct read of the current file; unchanged by
  this audit's six new commits. Bounded (~1s, one sweep-tick) stale-read
  window between an entry's expiry and the sweep task's `DeleteEntry` call.

- **Two DOM-20 table-driven-test warnings** (`atlas-maps`
  `kafka/consumer/map/consumer_test.go`; `atlas-channel`
  `socket/handler/character_cash_item_use_jukebox_test.go`) — unchanged by
  this pass; neither file was touched by any of the six new commits (the
  `atlas-channel` test file touched by this pass,
  `kafka/consumer/map/consumer_test.go`, is a *different* file from the
  flagged `character_cash_item_use_jukebox_test.go`).

- **Four sibling packet writers share the `USE_CASH_ITEM` fname-collision
  defect on gms_v72/gms_v79**, left unfixed and explicitly documented as
  out of scope: `CashItemUseSuperMegaphone`, `CashItemUseMapleTV`,
  `CashItemUseMegaphone`, `CashItemUseTripleMegaphone`. Re-confirmed via
  direct read of `legacyConsumedSiblingWriters`
  (`tools/packet-audit/internal/matrix/build.go:111-119`): the allow-list
  entry added by Task 9 lists exactly one writer,
  `"CashItemUseSongPlayer": true`, under `USE_CASH_ITEM|serverbound` — the
  four siblings are not present. `coverage-manifest.yaml`'s `out_of_scope`
  section names them explicitly and states they still grade
  incomplete/n-a on gms_v72/gms_v79 after this task, which matches the
  code.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | Full suite, `-count=1`, re-run this pass. |
| `libs/atlas-saga` | PASS | PASS | Unchanged since prior pass; not touched by the six new commits. |
| `atlas-saga-orchestrator` | PASS | PASS | Unchanged since prior pass; not touched by the six new commits. |
| `atlas-maps` | PASS | PASS | Unchanged since prior pass; not touched by the six new commits. |
| `atlas-channel` | PASS | PASS | Full suite `-count=1` re-run this pass; targeted `-race -run Jukebox` re-run this pass across `kafka/consumer/map`, `socket/handler`, and `jukebox` packages — all 12 jukebox-named tests pass under `-race`, including `TestHandleStatusEventJukeboxEnd_BroadcastsExactlyMinusOne` (the test the `22db8d5fa` fix targets). |
| `tools/packet-audit` | PASS | PASS | Full suite `-count=1`, run this pass (not built/tested in the prior audit, since Task 9 postdates it). |

Additionally, all four `packet-audit` CLI gates were independently re-run in
this audit and confirmed to exit 0: `matrix --check` (rc=0),
`fname-doc --check` (rc=0, "268 structs without an audit report carry no
fname"), `operations --check` (rc=0, "0 absent-writer note(s)"), and
`dispatcher-lint` ("clean", rc=0).

No `atlas-ui` changes in this branch; no `npm run build`/`npm test` step
applies.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the controller's own flagless
  `tools/verify.sh` run, per the plan's "Final gate" section — not run here
  per instruction, since a flagless run was already in flight in this
  worktree)

## Action Items

None. All 8 original plan tasks plus the post-plan Task 9 are complete, all
global constraints hold with direct evidence, all module-local builds/tests
pass (including a `-race` re-run of every jukebox-specific test), and all
four `packet-audit` gates independently reproduce exit 0. The controller
should confirm the in-flight flagless `tools/verify.sh` run exits 0 before
proceeding to code review / PR, per the plan's Final Gate and CLAUDE.md's
"Done means verified" rule.
