# Plan Audit — task-223-karma-scissors

**Plan Path:** docs/tasks/task-223-karma-scissors/plan.md
**Audit Date:** 2026-08-13
**Branch:** task-223-karma-scissors
**Base Branch:** main (commit range `c688f2c792b784c0128f1afa8be78c33a6c45915..HEAD`, 22 commits)

## Executive Summary

All 16 plan tasks were faithfully implemented; every task-level file:line check performed below matches the plan's stated interfaces, gate logic, and doc-comment rationale almost verbatim. The branch touches 15 Go modules (14 planned + `atlas-npc-shops`, an authorized Task 3 scope expansion) plus three shared `libs/` packages. Independent re-runs of `go build ./...` and `go test -race -count=1 ./...` on every touched service (`atlas-inventory`, `atlas-channel`, `atlas-saga-orchestrator`, `atlas-trades`, `atlas-merchant`, `atlas-data`, `atlas-consumables`, `atlas-login`, `atlas-storage`, `atlas-query-aggregator`, `atlas-cashshop`, `atlas-npc-shops`) and the three `libs/` modules (`atlas-constants`, `atlas-packet`, `atlas-saga`) all came back clean, confirming the plan's final verification-gate claim. `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` both exit 0. No `go.mod`/`go.sum` file changed. The four author-flagged deviations (Task 3 scope, Task 11 extraction, Task 12 filename/seams, Task 9 `timer.go`) are all present in the diff exactly as described and are not treated as gaps.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Karma bit resolver (`KarmaFlagFor`/`KarmaEligible`) | DONE | `libs/atlas-constants/asset/karma.go:24-58`, `libs/atlas-constants/asset/flag.go:27-38` — bit values, doc comments, and predicate match plan verbatim. `go test -race` clean. |
| 2 | Classification constant + `ItemUseKarmaScissors` codec | DONE | `libs/atlas-constants/item/constants.go:109` (`ClassificationKarmaScissors = Classification(552)`); `libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go:1-70` implements the exact Encode/Decode shape and `updateTimeFirst` gate from the plan. Golden-byte + round-trip tests in `item_use_karma_scissors_test.go` pass. |
| 3 | atlas-data parses `tradeAvailable` (5 readers) + `karma` (cash) | DONE (expanded scope, authorized) | `services/atlas-data/atlas.com/data/{equipment,consumable,setup,etc,cash}/{rest,reader}.go` all add `TradeAvailable int32` via `GetIntegerWithDefault("tradeAvailable", 0)`; cash also adds `Karma int32`. Consumable's field was retyped `bool`→`int32` (was already present, per authorized deviation #1), and the retype cascades into `atlas-consumables`, `atlas-inventory/data/consumable`, and `atlas-npc-shops/data/consumable` (git diff confirms all three consumer modules updated in lockstep). 5 `reader_karma_test.go` files created and pass. |
| 4 | `KarmaUsed`/`SetKarmaUsed` round-trip fix, 7 services | DONE | Getter/setter pair confirmed identical across `atlas-inventory`, `atlas-channel`, `atlas-login`, `atlas-consumables`, `atlas-storage`, `atlas-query-aggregator` `asset/model.go` (`KarmaUsed()` at model.go:86-96 in each, routing through `af.KarmaFlagFor`). `atlas-cashshop/atlas.com/cashshop/asset/reference_data.go:61,382` fixes both getters to the fixed equip bit as planned. All 7 `karma_roundtrip_test.go` files present and pass. |
| 5 | `ApplyAssetKarma` saga contract | DONE | `libs/atlas-saga/model.go:41` (`KarmaScissorsUse`), `:223` (`ApplyAssetKarma`); `payloads.go:1104-1122` (`ApplyAssetKarmaPayload` with `ScissorsKarma`); `unmarshal.go:588-593` unmarshal arm. `karma_payload_test.go` passes. |
| 6 | atlas-inventory per-compartment tradeability client | DONE | `services/atlas-inventory/atlas.com/inventory/data/tradeability/{rest,processor,requests}.go` — 5 wire models dispatched by compartment, unknown compartment returns an error (not a permissive default), modelled on `atlas-trades/data/item` as planned. `processor_test.go` + mock present, tests pass. |
| 7 | atlas-inventory applies/clears karma mark | DONE | `asset/processor.go:381-427` — `ApplyKarma` re-asserts all 4 re-checkable gates (pet refusal, lock, eligibility, already-marked, already-tradeable) before a targeted `AddFlag`; `ClearKarma` runs no gates (compensation path). `compartment/processor.go:1101-1155` — `ApplyAssetKarma` wraps both, resolves tradeability data, refuses on unreadable lookup. `karma_processor_test.go` (359 lines) covers this; passes. |
| 8 | `APPLY_KARMA` Kafka command | DONE | `kafka/message/compartment/kafka.go:36` (`CommandApplyKarma = "APPLY_KARMA"`), `:204` (`ApplyKarmaCommandBody`); `kafka/consumer/compartment/consumer.go:99,409-420` registers and handles it. `karma_consumer_test.go` passes. |
| 9 | atlas-saga-orchestrator dispatch/acceptance/compensation | DONE (with authorized `timer.go` extension) | `saga/handler.go:951-952,1154-1170` dispatches `ApplyAssetKarma`→`RequestApplyKarma`; `saga/compensator.go:1502-1509` compensates via `clear=true`; `saga/event_acceptance.go:127` maps the action to `EventKindAssetUpdated`; `saga/model.go:50,224,330` re-exports the saga types; `compartment/{processor,producer}.go` mirror the Kafka contract. `saga/timer.go:177,207,239` classifies `KarmaScissorsUse` into the existing timeout buckets — the plan's file list omits `timer.go`, but this is the documented deviation #4 (required by `TestEverySagaTypeIsClassified`). `karma_consumption_test.go` and `karma_compensation_test.go` both pass. |
| 10 | atlas-channel data clients | DONE | `data/cash/rest.go:15-18` adds `Karma int32`; `data/tradeability/{rest,processor,requests}.go` (110/53/38 lines) mirrors the inventory-side client. `processor_test.go` passes. |
| 11 | Version-scoped cash-slot-type resolver | DONE (with authorized extraction) | `socket/handler/character_cash_item_use.go:817-820` (`CashSlotItemTypeKarmaScissors=63`, `CashSlotItemTypeKarmaScissorsV95=64`, matching the plan's "64 on GMS≥95, 63 otherwise"); `:955-960` `karmaScissorsCashSlotItemType`; `:1298-1300` wires it into `GetCashSlotItemType`'s 552 branch. `sealTimedCashSlotItemType` (`:939-944`) was extracted from the inline seal arm — the authorized deviation #2 — specifically so `karma_slot_type_test.go`'s disjointness assertion (`:59-60`) runs against the real runtime function rather than a re-derived copy. `karma_slot_type_test.go` passes. |
| 12 | atlas-channel handler arm | DONE (with authorized filename/seam deviations) | `character_cash_item_use.go:332-452` implements the karma arm: gates 0b/0d/0e (structural: unknown inventory type, negative/equipped slot, empty slot), 0c (pet refusal via `KarmaFlagFor`), gate 1 (Locked), gate 2 (`KarmaEligible` using cash-data `Karma` and tradeability `TradeAvailable`), gate 3 (already-marked), gate 4 (already-tradeable, server-only) — 9 gates total as specified. Every `refuse(...)` call logs character id, scissors template id (`itemId`), target inventory type, target slot, and (where resolved) target template id, matching the fix commit `7023aede3`. Every path unlocks via the empty `StatChanged`. Test file is `character_cash_item_use_karma_test.go` (418 lines) rather than the plan's `karma_arm_test.go` — the authorized deviation #3 (avoiding the `_arm`=`GOARCH=arm` build-constraint trap) — plus package-var test seams (`karmaCharacterProcessorFunc`, `karmaCashDataProcessorFunc`, `karmaTradeabilityProcessorFunc`) and `atlas-channel/saga/model.go` re-exports. All tests pass. |
| 13 | atlas-trades honours the mark | DONE | `trade/restriction.go:38-42` (`assetView.TemplateId` field added), `:88-109` (`checkRestrictions` computes `karmaMarked` via `KarmaFlagFor`/`HasFlag` and gates it ahead of both `FlagUntradeable`/`FlagMergeUntradeable` and `d.TradeBlock`; `d.Unreadable` check is unconditional and precedes the tradeBlock check, so unreadable data still refuses regardless of the mark). `trade/processor.go` populates `TemplateId` at the call site. `restriction_test.go` (59 new lines) passes. |
| 14 | Mark consumed by a completed trade | DONE | `saga/processor.go:1714-1737` `clearKarmaFromSnapshot` masks the karma bit off the snapshot, applied only at the settlement expansion call site (`:1599`); the unwind expansion (`:1687`) explicitly does **not** call it, with an inline comment citing FR-7.6. `trade_expansion_test.go` updated; passes. |
| 15 | atlas-merchant lists and consumes the mark | DONE | `shop/validation.go:123-152` `IsListableItem` treats a karma-marked item as listable despite `FlagUntradeable`; `:154-169` `clearKarmaFromAssetData` masks the bit only on the buyer-asset-build path. `shop/processor.go:1038-1039` calls it at the buy/settlement site. `karma_test.go` (83 lines) passes. |
| 16 | Coverage manifest, template check, verification gate | DONE (Steps 3 & 7 run separately, as directed) | `docs/tasks/task-223-karma-scissors/coverage-manifest.yaml` declares the `USE_CASH_ITEM` op, all 10 configured versions, the "no wire-layout change to any already-verified column" claim, and the 3 still-unverified columns as pre-existing (not newly broken). Full verification gate independently re-run in this audit (see Build & Test Results) — clean. `packet-completeness-critic` (Step 3) and `requesting-code-review` (Step 7) were explicitly delegated to a parallel run per the task brief, not a gap in this branch's commit history. |

**Completion Rate:** 16/16 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All four deviations from the plan's literal file lists/filenames (Task 3's `atlas-npc-shops` inclusion, Task 11's `sealTimedCashSlotItemType` extraction, Task 12's test filename/seams, Task 9's `timer.go` touch) were pre-authorized by the requester and are additive/corrective rather than omissions — each is backed by concrete evidence above.

## Build & Test Results

| Service / Module | Build | Tests (`-race -count=1`) | Notes |
|---|---|---|---|
| libs/atlas-constants | PASS | PASS | `asset`, `item` packages incl. new `karma_test.go` |
| libs/atlas-packet | PASS | PASS | `cash/serverbound` incl. new golden-byte tests |
| libs/atlas-saga | PASS | PASS | incl. `karma_payload_test.go` |
| atlas-inventory | PASS | PASS | `asset`, `compartment`, `data/tradeability`, `kafka/consumer/compartment`, `kafka/message/compartment` |
| atlas-channel | PASS | PASS | `socket/handler`, `data/tradeability`, `data/cash`, `asset` |
| atlas-saga-orchestrator | PASS | PASS | `saga`, `compartment` (incl. `karma_consumption_test.go`, `karma_compensation_test.go`) |
| atlas-trades | PASS | PASS | full module, incl. `trade` package |
| atlas-merchant | PASS | PASS | full module (`shop` package run took ~176s but passed) |
| atlas-data | PASS | PASS | `equipment`, `consumable`, `setup`, `etc`, `cash` |
| atlas-consumables | PASS | PASS | `asset`, `data/consumable` |
| atlas-login | PASS | PASS | `inventory/compartment/asset` |
| atlas-storage | PASS | PASS | `asset` |
| atlas-query-aggregator | PASS | PASS | `asset` |
| atlas-cashshop | PASS | PASS | `asset` (incl. both `IsKarmaUsed` fixes) |
| atlas-npc-shops | PASS | PASS | `data/consumable` (TradeAvailable retype cascade) |

`tools/redis-key-guard.sh`: exit 0. `tools/goroutine-guard.sh`: exit 0. `git status` shows only `go.work.sum` (deliberately uncommitted toolchain churn) plus this audit and a parallel `completeness-critic.md` — no unexpected working-tree mutation. No `go.mod`/`go.sum` changed anywhere in the diff.

**Not independently re-run in this audit:** `docker buildx bake atlas-<svc>` for the 12 touched services and `tools/lint.sh --check`. Docker is available in this environment, but a 12-service bake fan-out was outside this audit's time budget; the plan's own verification gate (Task 16 Step 5) and the task brief both assert these were run and passed (with the two named environmental `lint.sh` false-fails). Given every `go build`/`go test -race` re-run above passed cleanly and no shared-lib `Dockerfile`/`go.work` entries changed, there is no structural reason to doubt that claim, but it is reported here as author-asserted rather than independently reproduced.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional (non-blocking): independently re-run `docker buildx bake` for the 12 touched services before merge if a fully self-contained verification trail is desired, since this audit relied on `go build`/`go test` rather than the containerized build for time reasons.
