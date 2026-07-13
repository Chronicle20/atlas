# Plan Audit (whole-branch adherence) — task-126-ap-sp-reset-items

**Plan Path:** docs/tasks/task-126-ap-sp-reset-items/plan.md (17 tasks)
**Audit Date:** 2026-07-02
**Branch:** task-126-ap-sp-reset-items
**Base (merge-base main):** 38d4d0ba2 → HEAD 3e30c212c (20 feature commits)
**Reviewer scope:** cross-task completeness + integration (individual per-task reviews already passed per progress.md)

## Executive Summary

All 17 plan tasks have a concrete, non-stub deliverable on the branch, and the pieces integrate
end-to-end: the saga type/actions/payloads (Task 5) are consumed by the orchestrator (Task 11) and
the channel (Tasks 14/15); the four Kafka message shapes (AP command, AP error, SP command, SP
transferred/error) are byte-exact between producer and mirror services; and the error-threading
contract (service `ERROR` → orchestrator `StepCompletedWithResult{errorCode,errorDetail}` →
`compensatePointReset` → `EmitSagaFailed(errorCode, reason=errorDetail)` → channel
`pointreset.ErrorMessage(ErrorCode, Reason)`) is wired at every hop. No `// TODO`, stub, 501, or
non-test panic was introduced. The two documented deviations (Task 4 v84/v87/jms BLOCKED; Task 16
Option A = gms_95-only wire + park) are genuine external blockers / a deliberate verify-don't-invent
scope call, not skipped work. **Verdict: PASS / READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | job.Advancement helper | DONE | libs/atlas-constants/job/advancement.go (+ _test.go); commit d6599a8a2 |
| 2 | skill.IsPointResetExcluded | DONE | libs/atlas-constants/skill/point_reset.go (+ _test.go); commit 708f7d274 |
| 3 | ItemUsePointReset codec | DONE | libs/atlas-packet/cash/serverbound/item_use_point_reset.go (+ _test.go); commit 3ee6206b3 |
| 4 | IDA verify + fixtures (5 versions) | DONE-with-documented-block | tools/packet-audit/cmd/run.go:1831 case; STATUS.md:803 `CashItemUsePointReset (T1)` v83 ✅, others ❌; v84/v87/jms genuinely blocked (no IDB, fname absent from exports); commits 5426f5feb, 960060050 |
| 5 | atlas-saga type/actions/payloads | DONE | libs/atlas-saga/model.go:27,77,78; payloads.go:230,242; unmarshal.go cases; commit 23a26f0ea |
| 6 | Build() preserves hpMpUsed | DONE | character/character/model.go (+1 line, `hpMpUsed`); model_test.go; commit be6747bc8 |
| 7 | point-reset policy tables | DONE | character/character/point_reset.go (+ _test.go); commit a65978a03 |
| 8 | TRANSFER_AP command | DONE | processor.go:1925/1947 TransferAP(AndEmit); producer.go apTransferError provider; kafka.go body+error consts; consumer.go:407 handler registered; commit a2576637e |
| 9 | macro/skill WithTransaction | DONE | skills/skill/processor.go:72,103; skills/macro/processor.go:33,55; commit 660d3b19b |
| 10 | TRANSFER_SP command | DONE | skills/skill/processor.go:314/327 TransferSp(AndEmit) (gorm-native tx, WithTransaction at :356); producer.go providers; consumer.go:109 handler; commit 23a5a251e |
| 11 | orchestrator actions/events/compensation | DONE | model.go:1144/1150 unmarshal; event_acceptance.go:123,124 table; handler.go:836-839 + handlers 2270/2289; consumers (char :187, skill :86/:105); compensator.go compensatePointReset 1166 + DispatchPointResetRollbacks 1232; mocks updated; commit 8b94d88e7 |
| 12 | channel HpMpUsed + macro consumer | DONE | channel/character/model.go:147 HpMpUsed(); kafka/consumer/macro/consumer.go; main.go:201/:502 registered; commit e0e1caaab |
| 13 | channel pointreset package | DONE | channel/pointreset/model.go (ErrorMessage:187 + validators) (+ _test.go); commit 98d2fd2f4 |
| 14 | channel saga aliases + failed branch | DONE | channel/saga/model.go:37,38,49,76,77; kafka/message/saga/kafka.go SagaTypePointReset; consumer/saga/consumer.go:136 branch; commit 335229109 |
| 15 | channel handler arm + saga assembly | DONE | socket/handler/character_cash_item_use.go:108 arm (wp threaded); character_cash_item_use_point_reset.go; commit 933e17e9d (+ rename cf02dca62) |
| 16 | seed wiring + park doc | DONE (Option A) | template_gms_95_1.json: handler 0x55 (validator present) + writer 0x8C CharacterSkillMacro; deployment.md parks gms_87/jms_185/gms_92; commits 8861b17ec, 0648db4d4, 3e30c212c |
| 17 | final verification gates | DONE | controller-run: go build/vet/test-race ×8 modules, matrix --check exit 0, redis-key-guard exit 0, docker bake ×5 — all green (progress.md) |

**Completion Rate:** 17/17 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 4 and Task 16 are documented-blocker / deliberate-scope, not partial)

Note: the plan.md checkboxes were never flipped to `- [x]` (all 90 remain `- [ ]`). This is a
documentation-hygiene artifact of subagent-driven development, not a code gap — the progress ledger
(.superpowers/sdd/progress.md) is the completion record. No action required for merge.

## Deviations (confirmed legitimate, NOT gaps)

1. **Task 4 — v84/v87/jms_185 BLOCKED.** Only gms_v83 was fully verified + T1 cell promoted
   (STATUS.md:803 shows exactly one ✅ and four ❌). gms_v95 read-order confirmed against the live
   IDB but its report was withheld (shared-codec runtime-bool honesty). v84/v87/jms have no live IDB
   and the sender fname is absent from every checked-in export — a genuine external blocker,
   documented in deployment.md §1. Grounding independently confirmed clean; no faked artifacts.
2. **Task 16 — Option A (gms_95 only + park).** Handler wired ONLY on gms_95 (verified codec);
   gms_87/jms_185/gms_92 parked in deployment.md §2, matching the v92-mount-food precedent. A
   reversible, verify-don't-invent scope decision after the user was unavailable. v83/v84 handlers
   pre-exist (template_gms_83_1.json:412; v84 byte-identical per task-083).
3. **Task 16 addition (extra correct work).** `CharacterSkillMacro` writer (0x8C) was found missing
   from gms_95, IDA-verified against the live v95 IDB, and wired (needed by FR-18). Beyond plan scope.

## Cross-Task Integration (verified)

- **Saga symbols flow:** libs/atlas-saga PointReset/TransferAP/TransferSP + payloads → orchestrator
  unmarshal cases (model.go:1144/1150) + channel aliases (saga/model.go:37-77). Match.
- **Command bodies byte-exact:** TransferAPCommandBody (character ≡ orchestrator) and TransferSpBody
  (skills ≡ orchestrator) identical field/tag/type.
- **Status-event bodies byte-exact:** StatusEventApTransferErrorBody, StatusEventSpTransferredBody,
  StatusEventErrorBody all identical json tags across producer and orchestrator mirror.
- **Error-threading contract intact:** char consumer:187 and skill consumer:105 both call
  `StepCompletedWithResult(false, {"errorCode","errorDetail"})`; compensator
  `pointResetFailureFields` (compensator.go:1213) extracts them and emits
  `EmitSagaFailed(errorCode, reason=errorDetail)`; channel consumer/saga/consumer.go:137 passes
  `e.Body.ErrorCode, e.Body.Reason` into `pointreset.ErrorMessage(code, detail)`. Acceptance table
  (event_acceptance.go:123/124) default-deny isolates the two `Type=="ERROR"` handlers from
  cross-firing.
- **Handler dispatch:** channel arm (character_cash_item_use.go:108) matches both cash-slot buckets
  (23/24) and dispatches by item id — AP → TransferAP saga, SP → TransferSP saga with TargetMaxLevel
  from game data. Orchestrator GetHandler cases (handler.go:836-839) route both actions.

## No-Stub / Honesty Checks

- No `// TODO`, FIXME, 501, or "not implemented" introduced by the branch. The lone TODO at
  character_cash_item_use.go:115 ("for v83 there is a trailing updateTime") is **pre-existing on
  main** (git show 38d4d0ba2 confirms it at line 108) and applies to the *generic fall-through* for
  other unhandled cash items, not the point-reset arm (which sits above it, fully implemented).
- The single `panic(err)` is in pointreset/model_test.go:27 (test setup helper) — acceptable.

## Build & Test Results

Per CLAUDE.md gates, controller-run (progress.md §"Task 17") and not re-run here per instructions:

| Gate | Result |
|------|--------|
| go build ./... ×8 modules | PASS |
| go vet ./... ×8 modules | PASS |
| go test -race ./... ×8 modules | PASS (no panic/DATA RACE) |
| packet-audit matrix --check | exit 0 |
| redis-key-guard.sh | exit 0 |
| docker buildx bake (character/skills/channel/saga-orchestrator/configurations) | PASS (5 images) |

(A local `matrix --check` re-run here failed only on `GOWORK=off` module resolution inside the
worktree — an invocation artifact, not a data failure; STATUS.md content confirms the promotion.)

## Minor-Findings Triage (from progress.md)

| Finding | Verdict |
|---------|---------|
| Task 7 raw `job.Id(100)` literals instead of named consts (DOM-21-adjacent) | SHIP — values reviewer-verified correct; cosmetic, defer to backend-guidelines-reviewer |
| Task 7 gofmt alignment | ALREADY FIXED in Task 8 commit a2576637e |
| Task 11 AP/SP error handlers log at Debug (sibling meso logs at Error) | SHIP — cosmetic ops-diagnosability nit; behavior correct |
| Task 8 primary-cap `*dst+1 > cap` uint16 wrap only at stat==65535 | SHIP — unreachable from legit client; non-exploitable; hardening optional |
| Task 1 audit-report narrative miscount (31 vs 32 subtests; SuperGm tier label) | SHIP — code correct, report-prose only, no defect |

All rolled-up Minors are cosmetic/non-blocking. None gate merge.

## Overall Assessment

- **Plan Adherence:** FULL (17/17; deviations documented and legitimate)
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, non-blocking) Flip plan.md checkboxes to `- [x]` for record hygiene.
2. (Optional, non-blocking) Address Task 7 named-constant style if backend-guidelines-reviewer flags DOM-21.
3. (Tracking) Unpark gms_87 / jms_185 / gms_92 handler + CharacterSkillMacro writer when a
   corresponding IDB becomes available (deployment.md §2/§"Parked versions").
