# Plan Audit — task-241-duey-parcel-delivery (Tasks 11-20)

**Plan Path:** docs/tasks/task-241-duey-parcel-delivery/plan.md
**Audit Date:** 2026-08-19
**Branch:** task-241-duey-parcel-delivery
**Base Branch:** main
**Scope:** Plan Tasks 11-20 only (sibling shards cover 1-10 and 21-28)

## Executive Summary

All 10 plan tasks in this range (11-20) are implemented, tested, code-reviewed to
a clean verdict in the execution ledger, and verified independently against the
branch diff for this audit. Every claimed interface (saga actions/payloads,
orchestrator composite expansion, custody dispatch package, handler/compensation
wiring, atlas-parcel custody consumer, channel fee/validation, DUEY_ACTION
handler + ParcelSend saga, receive/discard/close arms, ShowParcel action/command,
and the NPC `open_duey` entry point with all 8 seed files) has direct file:line
evidence in the working tree. `go build ./...` and `go test ./... -count=1` pass
clean with no failures across all five affected modules (`libs/atlas-saga`,
`saga-orchestrator`, `atlas-parcel`, `atlas-channel`, `atlas-npc-conversations`).
No task in this range was skipped, deferred, or left partially implemented.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 11 | Saga library — parcel custody actions, types and payloads | DONE | `libs/atlas-saga/model.go:74-79,234-237` (Types/Actions); `payloads.go:987,1017,1070,1077` (four payload structs); `unmarshal.go:474-497` (four case arms); `unmarshal_test.go:736,798,855,888` (four unmarshal tests). Reviewed APPROVED (0 blocking) per ledger; commit `bc69a8a76`. |
| 12 | Orchestrator — composite expansion for parcel custody | DONE | `saga/processor.go:1232-1234,2162,2301` (`expandTransferToParcel`/`expandWithdrawFromParcel`); `saga/model.go:206-209,347-350,1560-1579` (aliases + payload switch); `saga/parcel_expansion_test.go:47,214` (both tests). Went through 2 fix rounds (RISK-2 meso-only guard, `Sscanf` error check); final re-review APPROVED clean, commit range `bc69a8a76..1a7cc20d6`. |
| 13 | Orchestrator — parcel custody dispatch package and status consumer | DONE | `parcel/processor.go:78-146` (`AcceptToParcelAndEmit`/`ReleaseFromParcelAndEmit`/`RestoreParcelAndEmit`/`RemoveParcelAndEmit`); `kafka/message/parcel/custody/kafka.go`, `kafka/consumer/parcel/custody/consumer.go` present; `main.go:34,128,160` registers `parcelCustody`. 1 fix round (idempotency comment); final re-review APPROVED clean, commit range `1a7cc20d6..6e557896d`. |
| 14 | Orchestrator — handler dispatch, event acceptance, compensation | DONE | `saga/handler.go:152-153,226,264,944-946,2539,2601` (interface + dispatch + impls + `parcelP` wiring); `saga/event_acceptance.go:93-95,259-260,458-460` (event kinds + acceptance table + outcome table); `saga/compensator.go:2536-2580,2674-2681,2909-2938` (`assetDataFromParcelSnapshot`, reverse-walk arms, late-compensation table); `saga/parcel_compensation_test.go:114,312` (`TestParcelSendCompensation`, `TestParcelReceiveCompensation`). Reviewed APPROVED, 0 fix rounds, commit `964196ee0`. |
| 15 | atlas-parcel custody consumer | DONE | `parcel/processor_custody.go:77,160,205,223` (`AcceptCustody`/`ReleaseCustody`/`RestoreCustody`/`RemoveCustody`); `kafka/consumer/custody/consumer.go`, `kafka/message/custody/kafka.go`, `kafka/producer/custody/producer.go` present; `main.go:4,61-63` registers custody consumer; `kafka/consumer/custody/consumer_test.go:111` `TestCustodyCommands` (10 subtests). 1 fix round closed a real item-property data-loss defect (`ItemLevel`/`RingId`/`ViciousCount` mapping into `AssetData`, `asset_data.go:38,55-56`); re-review APPROVED clean, commit range `964196ee0..281ce4242`. |
| 16 | atlas-channel — fee, meso limit and send validation | DONE | `parcel/fee.go:9-22,36,59` (all six constants + `Fee`/`TotalCost`); `parcel/validation.go:31-50` (`ValidateSend` with cap/limit/message checks); `parcel/fee_test.go:5,32` and `parcel/validation_test.go:24`. Independently re-verified by reviewer (Python recomputation) — APPROVED_WITH_FINDINGS, 0 blocking, 2 deferred non-blocking notes (unreachable overflow branch, unrelated trade-tax table). Commit `66125ac19`. |
| 17 | atlas-channel — DUEY_ACTION handler and the ParcelSend saga | DONE | `socket/handler/duey_action.go:20-23,31-63` (mode dispatcher); `duey_action_send.go:208` (`buildParcelSendSaga`); `main.go:132,1011` registers `parcelsb.DueyActionHandle`; `duey_action_send_test.go:205` `TestDueyActionSend` (13 subtests). One deviation (`CountPending(recipientId, worldId)` vs brief's one-arg signature) ruled correct against `atlas-parcel/parcel/resource.go:81-95`'s hard-required `filter[worldId]`. One test-only fix round (saga step assertions strengthened past length-only checks); final re-review APPROVED, commit range `281ce4242..680a44ff0` + fix `3b8de6cc0`. |
| 18 | atlas-channel — receive, discard and close arms | DONE | `socket/handler/duey_action_receive.go:101,190,257` (`resolveParcelByWireId`, `buildParcelReceiveSaga`, discard announce); `duey_action.go:43-63` routes RECEIVE/DISCARD/CLOSE; `parcel/requests.go:54` discard PATCH body. Two fix rounds: (1) moved the wire-id projection into the `parcel` package as exported `WireId` to avoid an import cycle with Task 21's emit side, and closed missing RECEIVE test coverage; final re-review APPROVED_WITH_FINDINGS, 0 blocking, commit range `3b8de6cc0..4d75fe7ed` + fix `0124fa9d8`. |
| 19 | Saga `ShowParcel` action and the SHOW_PARCEL command | DONE | `libs/atlas-saga/model.go:243` (`ShowParcel Action`); `payloads.go:1091` (`ShowParcelPayload`); `kafka/message/parcel/kafka.go:17-25` (`EnvCommandTopic`/`CommandTypeShowParcel`/`ShowParcelCommand`); `parcel/processor.go:158-164` (`ShowParcelAndEmit`); `saga/handler.go:2623-2639` (`handleShowParcel`, self-completing); `saga/event_acceptance.go:264` (`ShowParcel: {}`). Reviewed APPROVED_WITH_FINDINGS, 0 blocking, commit `a476d638f`. |
| 20 | NPC conversation entry point for Duey | DONE | `operation_executor.go:2133-2134` (`case "open_duey"`); `operation_executor_test.go:492` `TestExecuteOpenDuey`; all 8 seed files present at `deploy/seed/{gms/72_1,79_1,83_1,84_1,87_1,92_1,95_1}/npc-conversations/npc/npc-9010009.json` and `deploy/seed/jms/185_1/npc-conversations/npc/npc-9010009.json`. Reviewer independently confirmed byte-identical seeds via md5sum and no hardcoded NPC-id constant (design §9.6). APPROVED, 0 blocking, commit `a6acc9f38`. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task in the 11-20 range has direct code evidence, a passing test
suite, and a clean (or cleanly closed) review verdict recorded in
`.superpowers/sdd/plan/progress.md` and `docs/tasks/task-241-duey-parcel-delivery/reviews/`.

Two items were explicitly deferred *within* completed tasks (both non-blocking,
both documented as intentional, neither a silently dropped requirement):
- Task 16 NB1: the `TotalCost`-overflow branch in `ValidateSend` is provably
  unreachable given `MaxParcelMeso` — defensive code matching design §6.2, not
  a missing test.
- Task 18 NB1 (carried as an open item): the same-mailbox wire-id collision
  branch in `resolveParcelByWireId` is correctly wired but has no dedicated
  test constructing an actual collision. Not requested by any fix brief;
  flagged in the ledger as an item to fold into a later atlas-channel task or
  state in the PR description.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-saga | PASS | PASS | `go test ./... -count=1` — ok |
| atlas-saga-orchestrator | PASS | PASS | All packages ok, including `parcel` and `saga` |
| atlas-parcel | PASS | PASS | `kafka/consumer/custody` and `parcel` packages ok |
| atlas-channel | PASS | PASS | All packages ok, including `parcel` and `socket/handler` |
| atlas-npc-conversations | PASS | PASS | All packages ok, including `conversation` |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch
  readiness depends on the other shards' audits and the still-outstanding
  repo-wide `tools/verify.sh` flagless run, which is tracked separately and
  was not re-run by this audit per instructions)

## Action Items

None required for Tasks 11-20. Two carry-forward notes for the PR description
(already tracked in the execution ledger, not new findings from this audit):
1. Task 16 NB1 — document that the `TotalCost` overflow branch in
   `parcel/validation.go` is intentionally unreachable defensive code.
2. Task 18 open item — no test exists for the same-mailbox wire-id collision
   branch in `resolveParcelByWireId`; note as an accepted gap or add coverage
   in a follow-up if this branch is touched again.
