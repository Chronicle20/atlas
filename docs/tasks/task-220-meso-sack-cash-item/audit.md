# Plan Audit — task-220-meso-sack-cash-item

**Plan Path:** docs/tasks/task-220-meso-sack-cash-item/plan.md
**Audit Date:** 2026-08-13
**Branch:** task-220-meso-sack-cash-item
**Base Branch:** main

## Executive Summary

All 10 plan tasks were faithfully implemented, matching the plan's prescribed code nearly verbatim across every commit. The four pre-adjudicated deviations (test-data arithmetic fix in Task 4, `retainingCache`/`SetCache` reuse in Task 5, `WithStepResult` vs. the plan's fictional `WithResult` in Task 6, and the `discardConn` test-session fix in Task 8) were each verified against source: all are test-side accommodations using pre-existing seams, with zero production-code impact. All six required build/test/vet runs pass cleanly, including the tag-gated saga-orchestrator tests that carry three of the ten tasks' only test coverage. The four scrutiny items (FR-2.6 guard untouched, FR-5.1 unlock-on-every-outcome, FR-7.2 acceptance table untouched, FR-6.1/6.2 no wire/template changes) are all confirmed by diff. `tools/lint.sh --check` reports FAIL, but this is a pre-existing environmental false-failure (stale cross-worktree golangci-lint cache referencing a sibling worktree's shifted paths, plus an unrelated node-version mismatch for `ui:node-version` — no atlas-ui files are in this diff); direct `golangci-lint run` against both flagged modules (`libs/atlas-saga`, `libs/atlas-object-id`, the latter untouched by this branch) returns "0 issues." Recommendation: READY_TO_MERGE, with a note to re-run `tools/lint.sh --check` from a clean worktree (or after `golangci-lint cache clean`) before the PR merges, purely for hygiene.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `atlas-data` parses `info/meso` | DONE | `cash/rest.go:41` adds `Meso uint32 json:"meso,omitempty"`; `cash/reader.go:76-80` adds `m.Meso = uint32(i.GetIntegerWithDefault("meso", 0))`; test `TestReaderMesoSacks`/`TestReaderMesoNotFoldedIntoSpec` added — commit `4a478449a` |
| 2 | `atlas-channel` cash view model carries `meso` | DONE | `data/cash/rest.go:9-11` adds `Meso uint32 json:"meso"`; `rest_test.go` created with two tests — commit `045726001` |
| 3 | `meso_sack_use` saga type + error code constants | DONE | `libs/atlas-saga/model.go` adds `MesoSackUse Type = "meso_sack_use"`; re-exported in both `atlas-channel/saga/model.go` and `atlas-saga-orchestrator/saga/model.go`; `atlas-channel/kafka/message/saga/kafka.go` adds `ErrorCodeMesoOverflow`/`SagaTypeMesoSackUse`; `kafka_test.go` created — commit `c76c85056` |
| 4 | `atlas-character` emits `MESO_OVERFLOW` | DONE | `kafka/message/character/kafka.go` adds `StatusEventErrorTypeMesoOverflow`; `producer.go` adds `mesoOverflowErrorStatusEventProvider`; `processor.go` sets `rejectEmit` on the overflow guard and fires it post-rollback; `meso_overflow_test.go` created — commit `05999fcac`. **Test-data deviation** (pre-adjudicated, see below) verified genuine. |
| 5 | Orchestrator threads the meso error code onto the step result | DONE | `kafka/consumer/character/consumer.go:183` replaces `StepCompleted` with `StepCompletedWithResult(e.TransactionId, false, map[string]any{"errorCode": e.Body.Error})`; `meso_error_result_test.go` created using the pre-existing `saga.SetCache` seam — commit `b0d9e8efd`. `event_acceptance.go` confirmed unchanged (see FR-7.2 below). |
| 6 | Orchestrator `meso_sack_use` compensator | DONE | `saga/compensator.go` adds interface decls, `CompensateFailedStep` dispatch arm, `compensateMesoSackUse`, `mesoSackErrorCode`, `mesoSackCharacterId`, `DispatchMesoSackRollbacks`; `saga/producer.go`'s `EmitSagaFailed` gets a `MesoSackUse` arm resolving a real characterId; `meso_sack_compensation_test.go` created (three tests) — commit `82078d127`. **API deviation** (`Saga.WithStepResult` vs. plan's fictional `Step[any].WithResult`) verified genuine and pre-existing. |
| 7 | Classify `meso_sack_use` for the timeout backstop | DONE | `saga/timer.go` adds `MesoSackUse` to `reverseWalkSagaTypes`, `allSagaTypes`, and a `dispatchTimeoutRollbacks` case calling `c.DispatchMesoSackRollbacks(s)`; two tests appended to `meso_sack_compensation_test.go` — commit `321ee221d` |
| 8 | `atlas-channel` type-19 branch | DONE | `character_cash_item_use.go` adds `CashSlotItemTypeCurrencySack = CashSlotItemType(19)`, the classification-520 return, and the dispatch arm (3 non-contiguous insertions, guard untouched — see FR-2.6 below); `character_cash_item_use_meso_sack.go` created with `cashItemDataFunc`, `buildMesoSackUseSaga`, `handleMesoSackUse`; `character_cash_item_use_meso_sack_test.go` created (4 tests) — commit `df29bec28`. **Test-session deviation** (`discardConn` vs. `nil` in `character_cash_item_use_test.go`) verified test-only, pre-existing helper reused. |
| 9 | `atlas-channel` renders the meso-sack failure | DONE | `kafka/consumer/saga/consumer.go` adds `mesoSackFailureMessage` and the `SagaTypeMesoSackUse` arm in `handleFailedEvent`, sending the pink-text message then the enable-actions `StatChanged` unlock; test appended — commit `3ec4b5df0` |
| 10 | Full verification, rollout notes, and code review | DONE | `rollout.md` created verbatim to plan spec (re-ingest prerequisite, per-tenant table, manual-acceptance checklist, known limitations) — commit `5ff4b51bf`. Step 6 (code review) was **not** run by the implementer by design — that is this audit. See note below. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

Note: every step-level checkbox (`- [ ]`) in `plan.md` remains unchecked in the committed file even though the underlying work is done and committed — this project's convention does not appear to re-check plan.md boxes post-execution (task-level headings carry no checkbox at all). This is a documentation-hygiene observation, not a completion gap; all evidence above is drawn from the actual git history and diffs, not the checkbox state.

## Skipped / Deferred Tasks

None. All 10 tasks are DONE. Task 10 Step 6 (code review) was deliberately deferred to this audit, per the adjudicated-deviations list in the audit brief — not a plan violation.

## Self-Review Cross-Check (FR-1.1 .. FR-8.2)

| Requirement | Task | Verified |
|---|---|---|
| FR-1.1/1.2/1.3 (parse + serialize `info/meso`, absent ⇒ 0) | 1 | Confirmed — `reader.go:GetIntegerWithDefault("meso", 0)`, `rest.go:Meso uint32 json:"meso,omitempty"` |
| FR-1.4 (re-ingest prerequisite) | 10 | Confirmed — `rollout.md` "Hard prerequisite: WZ re-ingest per tenant" section |
| FR-2.1 (`CashSlotItemTypeCurrencySack`) | 8 | Confirmed — `character_cash_item_use.go` const `= CashSlotItemType(19)` |
| FR-2.2 (no sub-body decode) | 8 | Confirmed — dispatch arm calls `handleMesoSackUse(l, ctx, wp)(s, itemId)` directly, no `r.Decode` call |
| FR-2.3 (resolve amount via data/cash) | 8 | Confirmed — `cashItemDataFunc(l, ctx, uint32(itemId))` → `cd.Meso` |
| FR-2.4 (fail closed on 0 / oversized) | 8 | Confirmed — `character_cash_item_use_meso_sack.go`: `cd.Meso == 0` and `cd.Meso > uint32(math.MaxInt32)` both reject before any saga is created |
| FR-2.5 (unlock on every rejection) | 8 | Confirmed — `enableActions()` called on lookup error, zero-amount, and overflow paths |
| FR-2.6 (pre-branch slot/template guard unchanged) | 8 | **Confirmed by diff.** `git diff main...HEAD -- .../character_cash_item_use.go` shows exactly 3 non-contiguous insertion hunks (the dispatch arm at ~line 486, the const at ~line 667, the classification return at ~line 733); the pre-branch guard (context.md cites lines 52-57, well outside all three hunks) has zero lines touched. |
| FR-3.1 (channel cash model) | 2 | Confirmed |
| FR-4.1/4.2 (two-step saga, ShowEffect, DestroyAsset) | 8 | Confirmed — `buildMesoSackUseSaga`: step 1 `DestroyAsset`/`consume_meso_sack`, step 2 `AwardMesos`/`award_mesos` with `ShowEffect: true` |
| FR-4.3 (award failure compensates the consume) | 6, 7 | Confirmed — `DispatchMesoSackRollbacks` reverse-walks completed `DestroyAsset` steps into `RequestCreateItem`; wired into both `CompensateFailedStep` (Task 6) and `dispatchTimeoutRollbacks` (Task 7) |
| FR-4.4 (new saga type discriminates) | 3 | Confirmed |
| FR-5.1 (unlock on every outcome) | 8/9 | **Confirmed by trace.** Success: `atlas-character`'s `statChangedProvider` hard-codes `ExclRequestSent: true` (pre-existing, `character/producer.go:238`, unedited on this branch — verified no new success-path announce was added in `handleMesoSackUse`, which returns silently after `saga.NewProcessor(...).Create(...)`). Rejections (8): `enableActions()` on every one of the three reject branches. Saga failure (9): `consumer.go`'s new arm sends the pink text then a `StatChanged` unlock — the "ONLY unlock on the failure path" per its own comment. Compensator/timeout (6/7): terminate the saga and emit `SagaFailed`, which the channel's Task-9 arm turns into the same unlock. No double-unlock path exists — the success path has exactly one `StatChanged` (the award's), each failure path has exactly one. |
| FR-5.2 (meso-ceiling message) | 9 | Confirmed — `mesoSackFailureMessage` returns `"You cannot hold any more mesos."` only for `ErrorCodeMesoOverflow`, generic text otherwise |
| FR-5.3 (no new success writer) | 8 | Confirmed — `handleMesoSackUse` returns after `saga.NewProcessor(...).Create(...)` with no `session.Announce` call on that path |
| FR-6.1/6.2 (wire verified, no codec, no template) | Global | **Confirmed.** `git diff main...HEAD --name-only \| grep -E 'libs/atlas-packet\|seed-data/templates'` returns empty. |
| FR-7.1 (`MESO_OVERFLOW` emission) | 4 | Confirmed |
| FR-7.2 (orchestrator acceptance table unchanged) | 5 | **Confirmed.** `git diff main...HEAD -- services/atlas-saga-orchestrator/.../saga/event_acceptance.go` is empty (0 lines). |
| FR-7.3 (reject, never clamp) | 4 | Confirmed — `RequestChangeMeso` still returns `ErrMesoOverflow`; the emission is additive per the code comment "Deliberate asymmetry... overflow keeps returning the error" |
| FR-8.1 (all ten versions, no raw `> N`) | 8 | Confirmed — `TestCurrencySackTypeIsNineteenOnEveryVersion` (in `character_cash_item_use_meso_sack_test.go`) iterates GMS 48/61/72/79/83/84/87/92/95 + JMS 185; no version conditional exists in `GetCashSlotItemType`'s new branch |
| FR-8.2 (per-version item availability) | 10 | Confirmed — `rollout.md`'s 10-row tenant table |

## Adjudicated Deviations — Verification

1. **Task 4 test-data overflow arithmetic.** Verified genuine: starting balance is `1,000,000,000` (`meso_overflow_test.go`); first credit of `2,147,483,647` lands at `3,147,483,647` (< `math.MaxUint32` = 4,294,967,295); the second `2,147,483,647` credit would land at `5,294,967,294`, which exceeds `MaxUint32` — the overflow branch is genuinely exercised. Test asserts the post-first-call balance (`3,147,483,647`) is unchanged after the rejected second call.
2. **Task 5 `retainingCache` via `saga.SetCache`.** Verified `SetCache` is a pre-existing exported seam: `git show main:.../saga/cache.go` contains `func SetCache(c Cache)` at line 72, predating this branch. No new production seam was added.
3. **Task 6 `Saga.WithStepResult` vs. plan's `Step[any].WithResult`.** Verified `WithStepResult` is a pre-existing method: `git show main:.../saga/model.go` contains `func (s Saga) WithStepResult(index int, result map[string]any) (Saga, error)` at line 634. The test file carries an explicit "NOTE on the confirm-before-writing helpers" comment documenting the substitution, matching the plan's own escape hatch instruction.
4. **Task 8 `discardConn` swap in `character_cash_item_use_test.go`.** Verified `discardConn` is a pre-existing test type in `character_damage_test.go:524`, present since at least commit `6496b9c87` (task-213, predates this branch). The change (`session.NewSession(..., nil)` → `session.NewSession(..., discardConn{})`) is test-only, documented as behaviorally neutral for every other caller, and required because the new reject path's `session.Announce` chain reaches `s.con.Write` for real (unlike prior callers of this helper).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data | PASS | PASS | `go build ./... && go test -race ./... && go vet ./...` clean |
| atlas-channel | PASS | PASS | `go build ./... && go test -race ./... && go vet ./...` clean |
| atlas-character | PASS | PASS | `go build ./... && go test -race ./... && go vet ./...` clean |
| atlas-saga-orchestrator (no tags) | PASS | PASS | `go build ./... && go test -race ./... && go vet ./...` clean, but **misses the `//go:build test` files** — Tasks 5-7's dedicated tests do not compile under this invocation |
| atlas-saga-orchestrator (`-tags=test`) | PASS | PASS | `go test -race -tags=test ./...` clean; explicitly re-verified with `-v -run 'TestMesoSack\|TestHandleCharacterMesoErrorEventThreadsErrorCode\|TestEverySagaTypeIsClassified'` — all 6 named tests individually PASS |
| libs/atlas-saga | PASS | PASS | `go build ./... && go test -race ./... && go vet ./...` clean |

Repo guards (Task 10 Step 3), run from worktree root: `tools/redis-key-guard.sh` (exit 0), `tools/goroutine-guard.sh` (exit 0), `tools/skill-job-id-guard.sh` (exit 0), `tools/buff-duration-guard.sh` (exit 0) — all clean.

`tools/lint.sh --check`: reports **FAIL** — 3 failing targets: `lint:libs/atlas-object-id`, `lint:libs/atlas-saga`, `ui:node-version`. Investigated and determined **spurious / environmental, not a task-220 defect**:
- The golangci-lint run is polluted by stale cache entries referencing a sibling worktree (`task-216-energy-charge`) whose files have since moved/been removed — every `libs/atlas-object-id` and `libs/atlas-saga` "failure" line in the raw log is a `generated_file_filter processor` warning pointing at a `task-216-energy-charge/...` path with `no such file or directory`, followed immediately by `0 issues.` for the actual lint pass.
- Ran `golangci-lint run ./...` directly (same pinned binary, `.cache/tools/bin/golangci-lint-v2.12.2`) against both flagged modules with no cross-worktree noise: `libs/atlas-saga` → `0 issues.`; `libs/atlas-object-id` → `0 issues.` (the latter is not even touched by this branch's diff).
- `ui:node-version` fails because the shell's active node is v24 vs. the required v22 (`nvm use 22`) — irrelevant here since this task touches zero atlas-ui files. This matches the known project issue "lint.sh --check false-fails without nvm / cross-worktree golangci-lint lock contention."

No real formatting or lint issues were found in any file this task touched.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Hygiene, non-blocking) Re-run `tools/lint.sh --check` from a shell with `nvm use 22` active and/or after `golangci-lint cache clean`, to get a clean `--check` pass on record before merge — the underlying code is already lint-clean per direct `golangci-lint run` on both flagged modules.
2. (Hygiene, non-blocking) Consider checking off `plan.md`'s step-level checkboxes to reflect the completed state, or note in the plan's header that this project does not re-check boxes post-execution — current state could mislead a future skim-reader into thinking nothing was done.

## Verification confirmation

`git status` in the worktree is clean at the end of this audit (no files modified, staged, or committed by this audit process).
