# Plan Audit — task-207-cash-shop-surprise

**Plan Path:** docs/tasks/task-207-cash-shop-surprise/plan.md
**Audit Date:** 2026-08-10
**Branch:** task-207-cash-shop-surprise (worktree HEAD `92fddbb61`)
**Base Branch:** main (branch point `1e0a321b8`)

## Executive Summary

All 20 plan tasks are implemented with file:line evidence matching the plan's specified interfaces, file structure, and per-version opcode/mode-byte tables. `go build ./...` and `go test ./... -count=1` are clean with zero `FAIL` lines across `libs/atlas-packet`, `services/atlas-reward-pools`, `services/atlas-cashshop`, and `services/atlas-channel`. All seven explicit rulings that intentionally deviate from the plan text (marker format, jms `0xA7` dual-sender rows, Task 8 duplicate-key containment, Task 19 fname tie-break, `economy-and-trade.md` non-existence, v84/v87/v92/v95 alias-row `❌`, and live-tenant "not run") were followed correctly and are traceable to committed evidence. No `// TODO`, stub, or 501 was found in the branch's Go diff (the one `TODO` grep hit is a pre-existing, unrelated line immediately adjacent to inserted constants, confirmed via `git diff` context). Working tree is clean.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Serverbound `CashItemGachaponButton` codec | DONE | `libs/atlas-packet/cash/serverbound/item_gachapon_button.go` + `_test.go` exist; `CashItemGachaponHandle` const, `NewCashItemGachaponButton`, `CashId()`, `Operation()`, `Encode`/`Decode` all present matching plan signatures exactly. |
| 2 | Clientbound `CashItemGachaponResult` codecs | DONE | `libs/atlas-packet/cash/clientbound/item_gachapon_result.go` + `_test.go` exist; `CashItemGachaponSuccess`/`Failed`, `CashItemGachaponSuccessBody`/`FailedBody` via `atlas_packet.WithResolvedCode` (DOM-25 compliant, no hard-coded mode byte). |
| 3 | Registry entries + jms `0xA7` attribution | DONE (per ruling 2) | `CASH_ITEM_GACHAPON_BUTTON` present in `gms_v83.yaml:2867`, `gms_v79.yaml:2503`, `jms_v185.yaml:2956-2964`. jms opcode 167 carries BOTH rows (`CASHSHOP_SURPRISE`/`SendChangeMaplePoint` and `CASH_ITEM_GACHAPON_BUTTON`/`OnButtonClicked`) with an explanatory `note:` field, per the ruling that overrides the plan's original "correct the mis-attribution" instruction. `arm-catalog.md:251-262` amended in place recording "two senders," not a single corrected attribution — matches ruling 2 exactly. |
| 4 | `atlas-reward-pools` cash-surprise kind | DONE | `gachapon/builder.go`: `KindCashSurprise = "cash-surprise"` (line 18), `isValidKind` closed union (line 97-98), `DefaultKind` unchanged. |
| 5 | `commodity_id` on pool items | DONE | `item/model.go:47-50` `CommodityId()`, `item/builder.go:57-89` `SetCommodityId`/`ErrCommodityIdRequired` validation gate. |
| 6 | Flat-weight selection, `ErrEmptyPool`, commodity on reward | DONE | `reward/processor.go:34,40,70,101,133` — `ErrEmptyPool`, `usesFlatWeights` predicate used in both `SelectReward` and `GetPrizePool` branches as the plan specifies. |
| 7 | Tenant-configured Surprise box template ids | DONE | `configuration/tenant/cashshop/surprise/rest.go` created; `configuration/registry.go:55-68` `DefaultSurpriseBoxTemplateId` + `GetSurpriseBoxTemplateIds` with the documented default-on-unconfigured fallback. |
| 8 | Idempotency ledger | DONE (per ruling 3) | `surprise/opening/entity.go`, `administrator.go`, `administrator_test.go` all present. Duplicate-key detection lives in `surprise/opening/duplicate.go` (`isDuplicateKeyError`, Postgres SQLSTATE 23505 + sqlite extended codes), NOT in `libs/atlas-database`'s shared `TranslateError` — confirmed via `git diff 1e0a321b8...HEAD -- libs/atlas-database/` returning empty. Matches the ruling exactly. |
| 9 | Reward-pools REST client | DONE | `services/atlas-cashshop/atlas.com/cashshop/rewardpool/{rest,requests,processor,processor_test}.go` all present. |
| 10 | Capacity rule | DONE | `surprise/capacity.go` + `capacity_test.go` present. |
| 11 | Kafka command/status-event contracts | DONE | `CommandTypeOpenSurprise`, `StatusEventTypeSurpriseOpened`, `StatusEventTypeSurpriseFailed` present and mirrored byte-identically in both `atlas-cashshop/kafka/message/cashshop/kafka.go` and `atlas-channel/kafka/message/cashshop/kafka.go`. |
| 12 | Open orchestration | DONE | `surprise/processor.go:58` `NewProcessor(l, ctx, db) Processor`, `:72` `OpenAndEmit(...)` — matches plan's exact interface. |
| 13 | Consume `OPEN_SURPRISE` command | DONE | `kafka/consumer/cashshop/consumer.go:141-146` `handleOpenSurprise` with the `c.Type != cashshop.CommandTypeOpenSurprise` shared-topic guard the plan required. |
| 14 | Channel handler, command producer, writer registration | DONE | `socket/handler/cash_item_gachapon.go` + `_test.go` present; `cashshop/processor.go:30,207-214` `OpenSurprise(accountId, characterId, cashId) error` mints a transaction id and produces via `OpenSurpriseCommandProvider`. |
| 15 | Channel announces the result | DONE | `kafka/consumer/cashshop/consumer.go:164-233` `handleStatusEventSurpriseOpened`/`Failed`, each guarded on event `Type`, writing via `session.Announce(...)(cashpkt.CashItemGachaponResultWriter)(cashpkt.CashItemGachaponSuccessBody/FailedBody(...))`. |
| 16 | Tenant socket-config template routing | DONE | All 7 templates carry `"handler": "CashItemGachaponHandle"` (v79 through jms_v185); all 6 result-capable versions (v83/v84/v87/v92/v95/jms_v185) additionally carry `"writer": "CashItemGachaponResult"` with `"fname": "CCashShop::OnCashItemGachaponResult"`. v79 correctly has NO writer entry, matching the v79-serverbound-only ruling. |
| 17 | atlas-ui — widen pool kind | DONE | `types/models/reward-pool.ts:1` `RewardPoolKind = "gachapon" \| "incubator" \| "cash-surprise"`; `lib/schemas/reward-pools.schema.ts` `cashSurprisePoolSchema`/`CashSurprisePoolFormData`, `cashSurpriseItemSchema`/`CashSurpriseItemFormData` present. |
| 18 | atlas-ui — cash-surprise pool/item forms | DONE | `PoolFormDialog.tsx:23,103-104` and `PoolItemDialog.tsx:29,43,56-57,133-143,214-225` wire the `cash-surprise` kind, `needsCommodity` gate, and the commodity id field end-to-end. |
| 19 | Coverage matrix — verify and prove every cell | DONE | `docs/packets/audits/status.json`/`STATUS.md` regenerated (diff confirms machine-generated changes only); `STATUS.md:718` shows `CASH_ITEM_GACHAPON_BUTTON` ✅ across v79/v83/v84/v87/v92/v95/jms_v185 at the exact plan-specified opcodes; `STATUS.md:485` shows `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` ✅ at 0x14D/0x154/0x15E/0x180/0x188/0x16D. Row `:486` (the alias `GACHAPON_OPEN_RESULT` row on v84/v87/v92/v95) is left `❌`, explicitly documented as a coverage decision in `context.md` and `plan.md:2950`, matching ruling 6. `tools/packet-audit/cmd/seed_fname.go` gained the `indexRegistryByOpcode` tie-break (`pickCandidate` prefers the template's bound implementation) — matches ruling 4. |
| 20 | Full verification gauntlet and documentation | DONE | `services/atlas-cashshop/docs/domain.md` (+51 lines) and `docs/kafka.md` (+31 lines) updated. `context.md` §7 (lines 174-206) records the verification gauntlet with explicit "not run" rows for live-tenant acceptance and the service-registration guard (both correctly justified, not silently skipped) — matches ruling 7. `docs/research/missing-features/economy-and-trade.md` does not exist in the tree (`git ls-files` confirms untracked/absent) — matches ruling 5, left off the branch. |

**Completion Rate:** 20/20 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 20 tasks have direct file:line evidence of implementation. The items that deviate from the plan's literal text (registry marker format, jms `0xA7` dual rows, duplicate-key detection locality, seed-fname tie-break, `economy-and-trade.md` absence, four alias-row `❌` cells, live-tenant acceptance rows) are all covered by the seven explicit rulings supplied in the audit brief and are traceable to committed, documented evidence rather than silent omissions.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — no FAIL lines. |
| services/atlas-reward-pools/atlas.com/reward-pools | PASS | PASS | `go build ./...` clean; `gachapon`, `item`, `reward`, `seed`, `global` packages all `ok`. |
| services/atlas-cashshop/atlas.com/cashshop | PASS | PASS | `go build ./...` clean; `surprise`, `surprise/opening`, `rewardpool`, `configuration`, `kafka/message/cashshop`, `kafka/consumer/cashshop` all `ok`. |
| services/atlas-channel/atlas.com/channel | PASS | PASS | `go build ./...` clean; `socket/handler`, `session`, `world`, etc. all `ok`. |

(atlas-ui build/vitest and the full guard/bake gauntlet were reported as already verified by the invoking session at commit `92fddbb61` and were not re-run per instructions.)

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. No gaps found.
