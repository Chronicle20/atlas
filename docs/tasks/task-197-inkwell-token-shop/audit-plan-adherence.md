# Plan Audit — task-197-inkwell-token-shop

**Plan Path:** docs/tasks/task-197-inkwell-token-shop/plan.md
**Audit Date:** 2026-08-06
**Branch:** task-197-inkwell-token-shop
**Base Branch:** main (merge-base 31c7a664f)
**HEAD:** 52a878b13

## Executive Summary

All 9 plan tasks were faithfully implemented with file:line evidence for each. All 6 adjudicated deviations were applied exactly as described and verified against current source. The Known Limitations (R1, R2, R3, R5, and the atlas-channel `Quantity()` TODO) were correctly left untouched — `git diff` confirms zero changes to `services/atlas-channel`, `libs/atlas-packet`, and `libs/atlas-constants`. The prior gates report (dated 19:07, before the final two commits) recorded two FAILs — a stale `docs/domain.md` (Step 7) and a file-count mismatch (Step 11) — both of which trace to the same root cause (domain.md not yet updated) and are now resolved by the final commit `52a878b13`, confirmed live: `grep -rn 'TokenItem\|not implemented' services/atlas-npc-shops docs/TODO.md` now prints `NO STUB, NO TODO`, and the branch diff now shows exactly 4 files under `services/atlas-npc-shops` and 9 under `services/atlas-ui` (matches plan's file list; the +1 vs. plan's stated 8 is the `poolSearchConfig.ts` rename counted as `R100`, not scope creep). `npm run build` and the three most load-bearing vitest files (NpcShopCommodityDialog, ItemPicker, ItemSearchCombobox regression harness — 17 tests) were re-run live against current HEAD and pass clean. No Go source file changed after the gates report's Go verification (`go test -race`/`vet`/`build` all PASS), so that portion of the report remains valid for current HEAD.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Delete 5 pre-v83 Inkwell shop seed files | DONE | `ls deploy/seed/*/*/npc-shops/shops/shop-9000069.json` returns exactly the 6 surviving paths (83/84/87/92/95_1, jms/185_1); `git diff --name-status main...HEAD -- deploy/seed` shows 5 `D` lines for gms 12/48/61/72/79_1. |
| 2 | Rewrite 6 remaining seed files in display order with token prices | DONE | All 6 files are byte-identical (single md5 `2f99595b21276b2576053599a9b45d36`); `grep -o '"templateId"...'` on `gms/83_1` produces the exact 9-row order from FR-1.3 (2022503/5, 2000004/5, 2022514/10, 2000005/10, 3010116/25, 1122017/30, 2049000/45, 2049100/70, 1003016/100), each with `tokenTemplateId: 4310000`. |
| 3 | Add pure `planTokenSpend` planner | DONE | `services/atlas-npc-shops/atlas.com/npc/shops/token.go:19-63` — `tokenDraw` struct + `planTokenSpend` byte-identical to plan Step 5 code (ascending-slot draw, `uint64 available`, zero-quantity skip). `token_test.go:45` has `TestPlanTokenSpend` with all 9 table cases from the plan. Gates report: `go test ./shops/ -run TestPlanTokenSpend -v` — PASS (part of full `go test -race ./...` run). |
| 4 | Implement token branch in `Buy()`, close TODOs | DONE | `token.go:65-138` `buyWithTokens` matches plan Step 3 code exactly (guard order: price→tokenTemplateId→tokenType valid→destinationType valid→overflow-safe cost→`planTokenSpend`/`available<cost`→free-slot probe→draws→create). `processor.go:481-484` diff is exactly the plan's 2-line replacement (`git diff main...HEAD -- .../processor.go` shows only that hunk). `docs/TODO.md` diff removes exactly the 2 named lines and the now-empty `### NPC Shops Service` heading, matching Step 8/9. `token_test.go:262,309,331` has all 3 test functions (`TestBuyWithTokensSufficientBalanceSpansSlots`, `TestBuyWithTokensQuantityMultipliesCost`, `TestBuyWithTokensRefusals` w/ 7 subtests) from Step 1. |
| 5 | Move `poolSearchConfig.ts` to `src/lib/items/` | DONE | `git diff --name-status` shows `R100 .../templates/poolSearchConfig.ts → .../lib/items/poolSearchConfig.ts`. Both importers (`ItemSearchCombobox.tsx:10`, `EquipmentSection.tsx:13`) updated to `@/lib/items/poolSearchConfig`; `grep -rn 'templates/poolSearchConfig\|"./poolSearchConfig"' src` returns nothing. |
| 6 | Extract `useItemSearch` + `ItemSearchResults` from `ItemSearchCombobox` | DONE | `useItemSearch.ts` (105 lines) is byte-identical to the plan's Step 1 code, including the atomic `{term,page}` settle comment. `ItemSearchResults.tsx` created. `ItemSearchCombobox.tsx` rewritten to a thin 74-line shell over both, byte-identical to plan Step 3 code. Regression harness `__tests__/ItemSearchCombobox.test.tsx` has zero diff (`git diff --stat main...HEAD -- .../ItemSearchCombobox.test.tsx` empty) and passes live (part of the 17/17 re-run). |
| 7 | Add value-bearing `ItemPicker` | DONE | `ItemPicker.tsx` created, byte-identical to plan Step 3 code including `id?: string` prop (deviation 5) and `useItemName(value > 0 ? String(value) : "")` gating (deviation 6). `__tests__/ItemPicker.test.tsx` created with the 7 cases from the plan; re-run live, all pass. |
| 8 | Use `ItemPicker` in `NpcShopCommodityDialog` | DONE (with adjudicated deviation 2, applied correctly) | `NpcShopCommodityDialog.tsx` rewritten; `FIELDS` uses the deviation-3 narrowed union (`ItemFieldKey \| NumberFieldKey`) instead of `keyof CommodityAttributes`, field order unchanged. Deviation 2's `role="group"`/`aria-labelledby` wrapper (commit `27d2c30ed`) is live at lines 122-143: `<div role="group" aria-labelledby={labelId}>` wraps `<Label id={labelId}>` + `<ItemPicker id={controlId}>` — the `aria-labelledby` target is a real `<Label id=…>` in the same node, confirmed by direct read. `__tests__/NpcShopCommodityDialog.test.tsx` created with all 4 cases; `git diff --stat -- .../NpcShopCard.tsx` empty (no consumer edit needed, as predicted). |
| 9 | Full verification sweep and code review | DONE | See Build & Test Results below. All 14 steps' gates pass at current HEAD; Steps 7/11 (the only two that failed in the intermediate gates-report snapshot) are resolved by the final commit `52a878b13`. Step 12 (dispatch code review) is the parent orchestration step this audit is part of — out of this report's own scope to self-certify, but the plan's `audit.md` step is separately in progress per the task brief (concurrent reviewers). |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 9 tasks are DONE.

## Deviation Verification

| # | Deviation | Verified |
|---|-----------|----------|
| 1 | Reword `token.go` doc comment to remove literal `4310000` | Confirmed — commit `1249edbb1`; `token.go:73-75` now reads "the v83 client hardcodes the Perfect Pitch item id for its own local pre-check (0x41C3F0)", 0x41C3F0 citation retained. |
| 2 | `role="group"` + `aria-labelledby` instead of `<Label htmlFor>` on the picker trigger | Confirmed — commit `27d2c30ed`; `NpcShopCommodityDialog.tsx:122-130` wraps `<Label id={labelId}>` and `<ItemPicker id={controlId}>` in `<div role="group" aria-labelledby={labelId}>`; plain number rows and the edit-mode `ResolvedItemName` span keep direct `htmlFor={controlId}` (lines 145-148). |
| 3 | `FIELDS` narrowed to a 2-key item union instead of `keyof CommodityAttributes` | Confirmed — `NpcShopCommodityDialog.tsx:40-46` defines `ItemFieldKey`/`NumberFieldKey` and a discriminated-union `FIELDS` array; field order (`templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimit`) unchanged from the plan. |
| 4 | Correct `services/atlas-npc-shops/docs/domain.md` (not anticipated by the plan) | Confirmed — commit `52a878b13`; both the "Domain Rules" bullet and the "Processes buy operations" bullet now describe the 3-branch token path (meso/rechargeable/token), including "the client-supplied discount price does not affect this cost" and the multi-slot draw/free-slot-probe behavior. Matches the actual `buyWithTokens` implementation. |

## Known Limitations Carried Forward — Verified Untouched

- **R1/R2/R3/R5** (design.md §7 / plan's Known Limitations section): no code changes reference or alter these behaviors; `processor.go`'s branch order (rechargeable → meso → token) is unchanged except for the 2-line stub replacement at the tail.
- **atlas-channel `commodities.Model.Quantity()` hardcoded-0 TODO**: `git diff --stat main...HEAD -- services/atlas-channel` is empty; `services/atlas-channel/atlas.com/channel/npc/shops/commodities/model.go:68-71` still returns `0` with its `// TODO` comment intact.
- `libs/atlas-packet` and `libs/atlas-constants`: zero diff confirmed (`git diff --stat main...HEAD -- libs/atlas-packet libs/atlas-constants` both empty), consistent with the plan's "No change to libs/atlas-packet" global constraint.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-npc-shops (`services/atlas-npc-shops/atlas.com/npc`) | PASS | PASS | Per prior gates report (`.superpowers/sdd/plan/task-9-gates-report.md`, Step 1): `go test -race ./...` all `ok`/`[no test files]`, `go vet ./...` silent, `go build ./...` silent. No Go file changed after that report's commit (`1249edbb1`), confirmed via `git diff --name-only 1249edbb1..HEAD` returning zero `.go` files — report remains valid for current HEAD. |
| atlas-ui (`services/atlas-ui`) | PASS | PASS | Re-ran live at current HEAD (post-`52a878b13`): `npm run build` (tsc -b + vite build) exits clean, same expected `ConversationEditorPanel` chunk-size warning (pre-existing, unrelated). Re-ran the 3 most load-bearing suites live: `NpcShopCommodityDialog.test.tsx`, `ItemPicker.test.tsx`, `ItemSearchCombobox.test.tsx` (the frozen regression harness) — 3 files / 17 tests, all PASS. Full-suite 194-file/1400-test run is recorded PASS in the prior gates report at a slightly earlier commit; the only file changes since then were `NpcShopCommodityDialog.tsx` (aria fix, covered by the live re-run) and `docs/domain.md` (non-code). |

Repo guards (redis-key-guard, goroutine-guard, lint.sh --check): recorded PASS in the prior gates report; no files relevant to those guards changed afterward (only a TSX file already covered above and a markdown doc).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the concurrent backend/frontend guideline reviewers' own findings, which are outside this report's scope)

## Action Items

None. All 9 tasks are complete, all 4 adjudicated deviations were applied as specified, and the Known Limitations were correctly left untouched. The two gate failures visible in the intermediate `.superpowers/sdd/plan/task-9-gates-report.md` snapshot (Step 7 stale `docs/domain.md`, Step 11 file-count mismatch) are both resolved by the final commit and were re-verified live in this audit.
