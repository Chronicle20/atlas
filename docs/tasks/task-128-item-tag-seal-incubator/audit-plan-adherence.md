# Plan Audit — task-128-item-tag-seal-incubator

**Plan Path:** docs/tasks/task-128-item-tag-seal-incubator/plan.md
**Audit Date:** 2026-07-03
**Branch:** task-128-item-tag-seal-incubator
**Base Branch:** main (fork point 38d4d0ba2, head 0f555b16a8)

## Executive Summary

All 20 plan tasks were faithfully implemented; each has a matching commit and file:line
evidence. No silently skipped, deferred, or stubbed work was found in landed commits (the
only "not implemented" line is the design-mandated cube type-74 no-op; all `// TODO`
markers in changed files are pre-existing and were not introduced by this branch). Spot
builds/tests on the six heaviest modules (atlas-packet, atlas-saga-orchestrator,
atlas-inventory, atlas-channel, atlas-tenants, atlas-storage) are green. The two adjudicated
deviations (IncubatorResult extended body GMS>=95 only; IncubatorResultPayload world.Id/
channel.Id types) are present and correct. Verdict: **FAITHFUL — READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Named item ids (constants) | DONE | libs/atlas-constants/item/constants.go:251-257 |
| 2 | Asset owner in packet codec | DONE | libs/atlas-packet/model/asset.go:170-176; owner written at :219,271,297,342 |
| 3 | Serverbound sub-body codecs | DONE | cash/serverbound/item_use_{item_tag,seal,incubator}.go (+tests) |
| 4 | INCUBATOR_RESULT writer | DONE | incubator/clientbound/result.go:11,35,46 (predicate GMS>=95, adjudicated) |
| 5 | Saga types/actions/payloads | DONE | libs/atlas-saga/model.go:27-29,166-168; payloads.go:109,753-770; unmarshal.go |
| 6 | Asset owner e2e (inventory) | DONE | asset/{entity,model,builder,rest,producer,administrator}.go; kafka AssetData |
| 7 | SET_OWNER / APPLY_LOCK commands | DONE | kafka/message/compartment/kafka.go:33-34,179-184; asset/processor.go:271,287 |
| 8 | Lock-aware expiration | DONE | compartment/processor.go:942-944; asset/processor.go:305 ClearLock |
| 9 | Orch actions + asset UPDATED | DONE | event_acceptance.go:32,105-106; handler.go:865-867,1044,1058; asset consumer :334 |
| 10 | Incubator event + compensation | DONE | kafka/message/incubator/kafka.go; handler.go:1076; compensator.go:1169,1226 |
| 11 | incubator-rewards config (tenants) | DONE | configuration/{rest,provider,processor,resource,kafka,seed,mock}.go; 6 seed json |
| 12 | protectTime in atlas-data reader | DONE | data/cash/reader.go:76, rest.go:40; channel data/cash/rest.go:11 |
| 13 | Owner in channel projection | DONE | channel asset/{model,builder,rest}.go; consumer 6x SetOwner; socket/model/asset.go:15 |
| 14 | Owner in storage/merchant mirrors | DONE | storage asset chain; merchant/orchestrator/channel-merchant AssetData; +commit 0bd97c9c0e |
| 15 | Incubator client + weighted roll | DONE | channel/incubator/{rest,requests,processor,roll}.go (+roll_test.go) |
| 16 | Handler arms (tag/seal/incubator/cube) | DONE | character_cash_item_use.go:112,170,238,344; consts :361-366; writer reg main.go:798 |
| 17 | Result consumer + fail packet | DONE | kafka/consumer/incubator/consumer.go; saga consumer.go fail branch; main.go:218,551 |
| 18 | Seed template wiring | DONE | 5 templates x1 IncubatorResult; gms_92 correctly absent |
| 19 | Packet verification campaign | DONE | 5 markers + 5 evidence yaml; STATUS.md row 89 all ✅; run.go candidatesFromFName |
| 20 | Verification suite + runbook | DONE | deploy-runbook.md (35 lines incl. pre-existing updateTime follow-up) |

**Completion Rate:** 20/20 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. gms_92 template rows are the only deliberately-omitted artifact and are correct per
context.md §Open items (no verified opcode, no v92 IDB) — an absent row is a safe no-op.

## Adjudicated Deviations (verified handled, not gaps)

- **IncubatorResult extended body GMS>=95 only** — result.go:46 gates the 12-byte tail to
  `GMS && MajorVersion>=95` (not the plan's ">=87"). Corrected via live IDA in Task 19
  (commit c815473ae1); all 5 evidence cells + STATUS.md re-pinned. Correct.
- **IncubatorResultPayload uses world.Id / channel.Id** (payloads.go:772-773), wire-identical
  to byte; handler passes `f.WorldId()`/`f.ChannelId()` directly (handler :332-333). Correct.
- **Task 14 extended to orchestrator deposit/withdraw** — commit 0bd97c9c0e also fixed
  assetDataFrom{Compartment,StorageProjection}Asset conversions (compartment/rest.go,
  saga/processor.go, storage/rest.go). In-scope gap closed. Correct.

## Two Flagged Items

**1. Outer ItemUse updateTime gate (GMS>=95) — PRE-EXISTING, follow-up is correct.**
`libs/atlas-packet/cash/serverbound/item_use.go:38,50` gates `updateTime` read/write to
`GMS && MajorVersion>=95`. This file was NOT touched by the branch (confirmed via
`git diff --name-only`). The handler derives `updateTimeFirst` from the same predicate
(character_cash_item_use.go:37), so the new sub-body codecs are internally consistent with
the existing outer codec — the branch introduces no new inconsistency. A fix would touch
every cash-item arm (pet/chalkboard/field-effect included), so scoping it to its own
follow-up (documented in deploy-runbook.md item 2) is the correct call, not an in-scope gap.

**2. GetCashSlotItemType returns 65 for both SealTimedV95 and a non-GMS95
ClassificationCharacterCreation sub-case — PRE-EXISTING table ambiguity, contained.**
`character_cash_item_use.go:650` (CharacterCreation, non-GMS95, 543xxx) returns
CashSlotItemType(65), which collides with SealTimedV95=65 (:365). The GetCashSlotItemType
body is pre-existing code — the branch only added the const names and the seal arm. Before
task-128, type 65 fell through to the warn log; now it routes to the seal arm. Reaching harm
requires a non-GMS95 client to (a) possess a 543xxx CharacterCreation item and (b) send a
seal-shaped sub-body — which its own client-side type resolution would not produce for that
classification. Even if reached, the seal arm's server-side guards contain it: Equip-only
inventory check (:175), slot-occupancy check (:179), launder-prevention (:184), and the
inventory-side ApplyLock rejecting non-lock expirations. Not a task-128 regression in new
logic; a latent version-table ambiguity worth a follow-up cleanup but non-blocking.

## Build & Test Results (spot-check)

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | incubator/clientbound, cash/serverbound, model green |
| libs/atlas-saga-orchestrator | PASS | PASS | saga + saga/mock green |
| atlas-inventory | PASS | PASS | compartment + asset green |
| atlas-channel | PASS | PASS | incubator + asset green |
| atlas-tenants | PASS | PASS | configuration green; mock compiles (11 new methods) |
| atlas-storage | PASS | PASS | asset green |

Executor's reported 10/10 modules + 7/7 bakes + matrix/guard clean is consistent with the
spot-check; bakes and full -race suite not re-run in this audit.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional follow-ups (both out of task-128 scope):
1. Fix the outer `cash/serverbound/ItemUse` updateTime gate to match v87/v95/jms unconditional
   write (affects all cash-item arms) — already tracked in deploy-runbook.md item 2.
2. Disambiguate the CashSlotItemType(65) collision between SealTimedV95 and non-GMS95
   ClassificationCharacterCreation in GetCashSlotItemType.
