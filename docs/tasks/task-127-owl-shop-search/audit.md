# Plan Audit — task-127-owl-shop-search (plan-adherence-reviewer pass)

**Plan Path:** docs/tasks/task-127-owl-shop-search/plan.md
**Audit Date:** 2026-07-03
**Branch:** task-127-owl-shop-search
**Base Branch:** main
**Reviewer:** plan-adherence-reviewer (this pass only — backend-guidelines-reviewer / frontend-guidelines-reviewer passes append their own sections below, do not overwrite this one)

## Executive Summary

All 15 implementation tasks (Task 1 through Task 15) in plan.md are faithfully implemented and match the design's IDA-verified wire layouts, opcode matrix, and global constraints. Task 16 (verification gates + deployment.md) is legitimately in progress — `deployment.md` exists untracked and uncommitted, and `task-16-report.md` documents the gate run, consistent with the brief that this task is not yet finalized; this is expected and not a gap. Spot-run builds/tests/vet for `libs/atlas-constants`, `libs/atlas-packet`, `atlas-merchant`, `atlas-channel`, and `tools/packet-audit` all pass clean, and `packet-audit matrix --check` (run correctly from repo root) exits 0 with the six owl packet surfaces promoted to verified for gms_v83 and gms_v95. The task-125 registry coordination was checked (not colliding — task-125 is plan-phase only) and flagged in the report per the plan's instruction. The one accepted deviation (dropping `pkg:"merchant"` from `candidatesFromFName`) is a legitimate bugfix that corrects matrix-cell linkage, not a weakening of verification. No TODOs, stubs, or hardcoded absolute paths were found in the diff.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | FM map helper in libs/atlas-constants | DONE | `libs/atlas-constants/map/constants.go:2274-2275` (`FreeMarketEntranceId`/`FreeMarketRoomLastId`), `libs/atlas-constants/map/model.go:45-48` (`IsFreeMarketRoom`), test at `model_test.go:26`. `go test ./map/...` passes. |
| 2 | Serverbound owl codecs in libs/atlas-packet | DONE | `libs/atlas-packet/merchant/serverbound/{owl_action,owl_warp,shop_scanner_item_use}.go` + tests; `libs/atlas-packet/cash/serverbound/item_use_store_search.go` + test. Wire shapes match plan (mode byte only; `[int ownerId][int mapId]`; `[int16][int][int][byte][int]`; cash-tail `[int][byte][int]`). Commit `c238e73e2`. |
| 3 | Clientbound shop-scanner codecs + body factories | DONE | `libs/atlas-packet/merchant/clientbound/{shop_scanner_result,shop_link_result}.go` + tests, `libs/atlas-packet/merchant/shop_scanner_body.go` (mode/code constants, `WithResolvedCode` factories). Commit `816edc9cc`. |
| 4 | atlas-merchant world-scoped/ordered/capped/enriched search | DONE | `shop/processor.go:37` `MaxSearchResults = 200`; `:96` `ListingSearchCriteria`; `shop/provider.go:101` `searchListingsByItemId` with explicit tenant_id predicates (closes the `.Table()` tenant-leak bug the plan called out); `shop/rest.go` extended `ListingSearchRestModel` (OwnerId/ShopType/State/ItemSnapshot); mock updated at `shop/mock/processor.go:24,100`; README at `services/atlas-merchant/README.md:40` documents the new query params/cap/fields (note: plan cites the README path as `.../atlas.com/merchant/README.md`, but the actual/only README lives at `services/atlas-merchant/README.md` — harmless path-naming slip in the plan, content is present and correct). Commit `6f68b03bf`. |
| 5 | atlas-merchant searchcount package | DONE | `services/atlas-merchant/atlas.com/merchant/searchcount/{entity,model,administrator,provider,processor,rest}.go` all present; `entity.go:16-19` uses uuid surrogate PK + `uniqueIndex:idx_listing_search_counts_tenant_world_item` on `(tenant_id, world_id, item_id)` exactly per the Global Constraints tenant-safe-PK rule; atomic upsert via `clause.OnConflict` in `administrator.go`. `go test ./searchcount/...` passes (incl. concurrent-increment test). Commit `41580e3b3`. |
| 6 | RECORD_ITEM_SEARCH command + top-10 REST route | DONE | `kafka/message/merchant/kafka.go:27` `CommandRecordItemSearch`, `:112` body struct; consumer handler wired (per task-6-report, not re-verified line-by-line here but present and compiling); `shop/resource.go:40` route `/shop-searches/top`, `:315` `handleGetTopShopSearches`. Commit `a2cb1e495`. |
| 7 | atlas-channel merchant client extension | DONE | `services/atlas-channel/atlas.com/channel/merchant/model.go:47-48` `StateOpen=2`/`StateMaintenance=3` (mirrors atlas-merchant), `:51` `SearchListing` with all plan'd accessors, `:150` `TopSearch`; Extract functions present. `go test ./merchant/...` passes. Commit `dbd3ef47d`. |
| 8 | shopscanner registry | DONE | `services/atlas-channel/atlas.com/channel/shopscanner/{registry,registry_test}.go`; singleton `sync.Once`+`sync.RWMutex`, tenant-scoped `Key{Tenant, CharacterId}`, `SetLastSearch/GetLastSearch/SetPending/GetPending/RemovePending/ClearCharacter` all present and tested. No import of `atlas-channel/session` (dependency-free, avoids the cycle the plan warned about). Commit `1005083af`. |
| 9 | shop-scanner writer bodies + record conversion | DONE | `services/atlas-channel/atlas.com/channel/socket/writer/shop_scanner.go` + test; `ShopScannerRecords` converts channel 1-based session channel to 0-based wire byte (`byte(sl.ChannelId())-1`, confirmed at `shop_scanner.go:55`), equip rows (itemType==1) build a slotless `model.Asset`. `NewSearchListing`/`SearchListingSeed` added to `merchant/model.go` as a plain constructor (not a `*_testhelpers.go` file — compliant with the Test Helper Pattern rule). Commit `7386ef4d5`. |
| 10 | shopscanner processor (search flow, hot list, warp ladder) | DONE | `shopscanner/warp.go:14` `WarpCheck`, `:33` `EvaluateWarp` — 12-rung ladder exactly matching design §4.2 order (FM→search→own-shop→dead→shop-found→world→map-echo→shop-FM→channel→maintenance→state→listing-present); `shopscanner/processor.go:26` `NewProcessor`, `:34` `Search` (FM gate, fire-and-forget count record, search, owner-name resolution, announce, **consume only if `len(listings) > 0`** at line ~66, `SetLastSearch`), `:98` `SendHotList`. All ladder test cases from the plan present and passing. Commit `8ffe0b6b9`. |
| 11 | socket handlers + main.go registration | DONE | `socket/handler/{owl_action,owl_warp,shop_scanner_item_use}.go` created; cash 523 arm added — `character_cash_item_use.go:109` `if it == CashSlotItemTypeStoreSearch`, `:126` const, `:320` classification mapping via `item.ClassificationStoreSearch` (real constant at `libs/atlas-constants/item/constants.go:89`, not invented); `main.go:902-904` registers all three handlers, `:782-783` registers both writers. `OwlActionHandleFunc` resolves the expected mode via `atlas_packet.ResolveCode(l, readerOptions, "operations", "OPEN")` — config-driven, never hardcoded. Commit `ddd0de3fc`. |
| 12 | warp arrival auto-enter, capacity-full branch, session cleanup | DONE | `kafka/consumer/character/consumer.go:273` pending-entry check + EnterShop call inside `warpCharacter`; `kafka/consumer/merchant/consumer.go:194` `RemovePending` on `VisitorEntered`, `:300` capacity-full owl branch (announces `ShopLinkResultCodeFull`); `socket/init.go:48-50` destroyer wrapper clears `shopscanner` state before `DestroyByIdWithSpan`, importing `shopscanner` only in the bootstrap file (session package itself does not import shopscanner — no cycle). Commit `66e6851b1`. |
| 13 | Seed templates for all 6 versions | DONE | Verified programmatically (parsed JSON) for all six templates: opcodes for `OwlActionHandle`/`OwlWarpHandle`/`ShopScannerItemUseHandle` (handlers) and `ShopScannerResult`/`ShopLinkResult` (writers) match the plan's opcode matrix exactly, per version (gms_83 0x42/0x43/0x53 sb, 0x46/0x47 cb; gms_84 same as 83; gms_87 0x45/0x46 sb (no dedicated route), 0x48/0x49 cb; gms_92 0x49/0x4A sb, 0x4A/0x4B cb; gms_95 0x48/0x49/0x5A sb, 0x49/0x4A cb; jms_185 0x3A/0x3B sb, 0x40/0x41 cb). **Every** new handler entry across all six templates carries `"validator": "LoggedInValidator"` (confirmed by direct JSON parse — no validator-less entries). `ShopLinkResult` writer's `operations` table carries the full 9-code set in every template. Commit `f2620a4b0`. |
| 14 | Packet registry corrections + candidatesFromFName | DONE | `docs/packets/registry/gms_v83.yaml`/`gms_v84.yaml`: `USE_SKILL_RESET_BOOK` row fully removed (grep confirms zero remaining references in either file), `USE_SHOP_SCANNER_ITEM` row added at opcode 83 with `ida.address: 0xa0a25e` (v83) and `provenance: manual` (v84, since no v84 IDB exists — a documented, justified deviation from the brief's literal `ida-discovered` text, forced by a real `opregistry` loader invariant that rejects `ida-discovered` without an `ida.address` block; see task-14-report.md). `tools/packet-audit/cmd/run.go:619-629` adds the 6 `candidatesFromFName` cases. Coordination check performed: task-125 worktree exists but is plan-phase only (zero registry commits) — no actual collision; flagged for PR body per plan instruction (see dedicated subsection below). Commit `de16372fe`. |
| 15 | Packet verification campaign (gms_v83 + gms_v95) | DONE | All 12 evidence YAMLs present (`docs/packets/evidence/{gms_v83,gms_v95}/merchant.{serverbound,clientbound}.<Packet>.yaml`) with IDA addresses that match plan.md's anchors **exactly** (see dedicated subsection below); `STATUS.md` shows ✅ for gms_v83 and gms_v95 on all 5 tracked op rows (OWL_ACTION, OWL_WARP, USE_SHOP_SCANNER_ITEM, SHOP_SCANNER_RESULT, SHOP_LINK_RESULT); `packet-audit matrix --check` exits 0 when invoked correctly (`go run ./tools/packet-audit matrix --check` from repo root). Commits `54f68d6f3` (evidence/matrix) + `d3f245bf7` (fname-mapping fix, judged separately below). |
| 16 | Full verification gates + deployment notes | IN PROGRESS (out of scope per audit instructions) | `deployment.md` exists but is untracked/uncommitted; `task-16-report.md` documents gate runs including the 231-family WZ-data check (item 2310000 confirmed present in v83 WZ data) and this very code-review dispatch. Not evaluated further — explicitly excluded from this audit's scope. |

**Completion Rate:** 15/15 audited tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task 1-15 has direct code/config/doc evidence in the current worktree tree and in its cited commit. The only version-scoped gaps (v84/v87/v92/jms staying "seed-routed-but-unverified", `USE_SHOP_SCANNER_ITEM` unrouted on v87/v92/jms) are explicit, IDA-availability-driven design decisions documented in context.md item 14 and design.md, not silent scope cuts — they match the "only gms_v83 + gms_v95 have IDBs today" constraint stated up front in the plan's Goal line.

## Task 14/15 Packet-Audit Artifact Verification

Matrix cells (from `docs/packets/audits/STATUS.md`, current tree):

```
OWL_ACTION             | 0x042 ✅ (v83) | 0x042 ❌ (v84) | 0x045 ❌ (v87) | 0x048 ✅ (v95) | 0x03A ❌ (jms)
OWL_WARP               | 0x043 ✅ (v83) | 0x043 ❌ (v84) | 0x046 ❌ (v87) | 0x049 ✅ (v95) | 0x03B ❌ (jms)
USE_SHOP_SCANNER_ITEM  | 0x053 ✅ (v83) | 0x053 ❌ (v84) |    —   ⬜ (v87) | 0x05A ✅ (v95) |    —   ⬜ (jms)
SHOP_SCANNER_RESULT    | 0x046 ✅ (v83) | 0x048 ❌ (v84) | 0x048 ❌ (v87) | 0x049 ✅ (v95) | 0x040 ❌ (jms)
SHOP_LINK_RESULT       | 0x047 ✅ (v83) | 0x049 ❌ (v84) | 0x049 ❌ (v87) | 0x04A ✅ (v95) | 0x041 ❌ (jms)
```

All 5 tracked ops show ✅ for **gms_v83** and **gms_v95** — matches the plan's requirement that "every six packet surfaces byte-fixture verified on gms_v83 and gms_v95." v84 stays ❌ (documented as unverified-but-routed, `provenance: manual` for the registry row), v87/v92/jms show ⬜/❌ consistent with "no IDB, unrouted" for the dedicated route and "seed-routed-but-unverified" for OwlAction/OwlWarp/SHOP_SCANNER_RESULT/SHOP_LINK_RESULT — exactly the accepted state from design §5.

IDA address cross-check (evidence YAML `ida.address` field vs. plan.md's "IDA anchors" list and the opcode matrix's `USE_SHOP_SCANNER_ITEM`/cash tail):

| Packet | Plan v83 addr | Evidence v83 addr | Match | Plan v95 addr | Evidence v95 addr | Match |
|---|---|---|---|---|---|---|
| OwlAction (`CUIShopScanner::OnCreate`) | 0x8a0e9a | 0x8a0e9a | ✅ | 0x848b90 | 0x848b90 | ✅ |
| OwlWarp (`CUIShopScanResult::OnButtonClicked`) | 0x8a4423 | 0x8a4423 | ✅ | 0x848e80 | 0x848e80 | ✅ |
| ShopScannerItemUse (`CWvsContext::SendShopScannerItemUseRequest`) | 0xa0a25e | 0xa0a25e | ✅ | 0x9e10e0 | 0x9e10e0 | ✅ |
| ShopScannerResult/HotList (`CWvsContext::OnShopScannerResult`) | 0xa28c29 | 0xa28c29 (both arms) | ✅ | 0xa076c0 | 0xa076c0 (both arms) | ✅ |
| ShopLinkResult (`CWvsContext::OnShopLinkResult`) | 0x8a4e7a | 0x8a4e7a | ✅ | 0x847d60 | 0x847d60 | ✅ |

All 10 addresses (5 packets × 2 versions) match plan.md's IDA anchors exactly, byte for byte. Each evidence file also carries a `decompile_sha256` and `category: TIER1-FIXTURE`, with `verifies:` pointing at the actual fixture test names created in Tasks 2/3 (cross-checked to exist, e.g. `TestOwlActionWireShape`/`TestOwlActionRoundTrip` in `owl_action_test.go`).

The cash-arm tail codec (`ItemUseStoreSearch`) does not appear as its own STATUS.md row — it is a shared arm-tail appended to the existing `USE_CASH_ITEM` op rather than a standalone matrix packet, consistent with how other cash arm-tail codecs (e.g. pet-consumable) are treated in this codebase; task-15-report.md documents its read order (`SendScanPacket` v83 `sub_8A2407`@0x8a2407 / v95 `CUIShopScanner::SendScanPacket`@0x83f6b0) was decompiled and cross-checked against the codec as part of the ShopScannerItemUse verification, with no divergence found. This is a reasonable scoping choice, not a gap.

`packet-audit matrix --check`, run correctly from the repo root (`go run ./tools/packet-audit matrix --check`), exits 0 with no output (clean) — confirms the requester's claim independently. (Note: running the same command from inside `tools/packet-audit/` fails with "no template"/"stale" warnings because the tool resolves config paths relative to cwd — this is a documented gotcha in task-15/16's reports, not a real failure; the corrected invocation is authoritative.)

## Task-125 (USE_SKILL_RESET_BOOK) Coordination Finding

**Finding: properly isolated, not silently overwritten.**

- The plan (context.md:65,82; design.md:100,249; plan.md Task 14 coordination note) explicitly calls out that deleting the `USE_SKILL_RESET_BOOK` rows in `docs/packets/registry/gms_v83.yaml`/`gms_v84.yaml` overlaps rows in-flight task-125 (skill-mastery-books) might also touch, and instructs the implementer to check for a task-125 worktree before landing and flag it in the PR description.
- Verified directly: `git worktree list` shows `.worktrees/task-125-skill-mastery-books` exists at commit `f48030cc4b` on branch `task-125-skill-mastery-books`. Per task-14-report.md's "Coordination check" section, this worktree was confirmed to be plan-phase only with zero registry commits at the time task-127 landed its correction — i.e., there is no actual overlapping edit to reconcile, only a documented risk that whichever branch merges second should re-check the rows.
- The correction itself is scoped narrowly: only the `USE_SKILL_RESET_BOOK` row in gms_v83.yaml/gms_v84.yaml was deleted (confirmed via grep — zero remaining references in either file); the row's replacement (`USE_SHOP_SCANNER_ITEM`) carries an explanatory `note:` field documenting the IDA evidence and superseding rationale. `USE_SKILL_RESET_BOOK` rows in gms_v87.yaml, gms_v95.yaml, and jms_v185.yaml were correctly left untouched (grep confirms they still exist) since the plan's IDA evidence only applies to v83 (and v84 by the established v84≡v83-serverbound rule).
- The coordination flag itself is carried forward into `task-16-report.md`'s "task-125 coordination flag (for PR description)" section, ready to be surfaced in the actual PR body as instructed.
- **Verdict:** this is exactly the "flag it, verify it doesn't collide, don't silently overwrite" pattern the plan asked for — not a violation.

## candidatesFromFName Deviation (commit d3f245bf70) Judgment

**Verdict: reasonable, narrowing/cleanup fix — does not weaken packet verification coverage.**

- Diff reviewed directly (`git show d3f245bf70`): it removes `pkg: "merchant"` from the 6 owl `candidatesFromFName` cases added in Task 14, changing e.g. `{name: "OwlAction", pkg: "merchant", dir: csvpkg.DirServerbound}` to `{name: "OwlAction", dir: csvpkg.DirServerbound}`.
- Root cause (documented in the commit message and task-15-report.md): the `pkg` field caused `qualifiedWriterName` to prefix the struct name (`OwlAction` → `MerchantOwlAction`), which fed into the matrix's packet-id derivation (`dir(AtlasFile)+"/"+WriterName`) and produced `merchant/serverbound/MerchantOwlAction` — a string that never matched the byte-fixture markers, evidence file names, or tiers.yaml entries (all keyed as `merchant/serverbound/OwlAction`). This left the six cells stuck at partial/unlinked rather than promoting to verified.
- The fix is a **linkage bugfix in the audit tooling**, not a change to the actual packet codecs, fixtures, or verification evidence — the six struct names are unique across `atlas-packet` so `pkg` is genuinely unnecessary for `locateAtlasFile`, and the change "matches the merchant bucket convention (OpenShop et al.)" per the commit message, i.e. brings the owl cases in line with how existing merchant-bucket cases are already written elsewhere in the same file.
- Independently confirmed the fix works: `docs/packets/audits/STATUS.md` shows ✅ (not partial/🟡) for all 5 tracked owl ops on gms_v83/gms_v95 in the current tree, and `packet-audit matrix --check` exits 0 clean.
- This is a pre-commit tool-internal correction discovered and fixed by the same task before the final evidence commit landed, not a shortcut taken to make a broken verification look done. **Accepted as-is.**

## Global Constraints Spot-Check

| Constraint | Status | Evidence |
|---|---|---|
| FM map range 910000000-910000022 | HOLDS | `libs/atlas-constants/map/constants.go:2274-2275`; used consistently in `shopscanner/processor.go` (search-entry gate), `shopscanner/warp.go` (both current-map and shop-map FM checks), `socket/handler/owl_action.go` (owl-action gate). |
| Search cap 200 | HOLDS | `shop/processor.go:37` `const MaxSearchResults = 200`; enforced via `.Limit(MaxSearchResults)` in `shop/provider.go:115`. Plan's cap-at-200 test (`TestSearchListings_CapAt200`) exercises ascending truncation of the expensive tail. |
| Owl consumed only on ≥1-result search | HOLDS | `shopscanner/processor.go` `Search`: consumption call (`consumable.NewProcessor(...).RequestItemConsume(...)`) is gated by `if len(listings) > 0`, confirmed directly in the current file (search count increment happens unconditionally beforehand, matching the plan's "increment on every executed search, consume only on results" split). |
| Same-channel-only warp | HOLDS | `shopscanner/warp.go` `EvaluateWarp`: `if c.ShopChannelId != c.SessionChannelId { return ...CodeClosed, false }` — no channel-change code path exists anywhere in the diff. |
| Config-resolved mode bytes/codes, never hardcoded | HOLDS | `socket/handler/owl_action.go:26` `atlas_packet.ResolveCode(l, readerOptions, "operations", "OPEN")`; `libs/atlas-packet/merchant/shop_scanner_body.go` uses `atlas_packet.WithResolvedCode("operations", ..., ...)` for all three writer bodies (RESULT/HOT_LIST mode bytes and every ShopLinkResult code). |
| Every seed-template handler entry has `validator: LoggedInValidator` | HOLDS | Confirmed by parsing all six templates programmatically — every `OwlActionHandle`/`OwlWarpHandle`/`ShopScannerItemUseHandle` entry in all six templates carries `"validator": "LoggedInValidator"`. No validator-less entries found. |
| Counts table: uuid PK + unique(tenant_id, world_id, item_id) | HOLDS | `searchcount/entity.go:16-19`: `Id uuid.UUID primaryKey`, `uniqueIndex:idx_listing_search_counts_tenant_world_item` applied to all three of `TenantId`/`WorldId`/`ItemId`. |
| Builder/constructor test patterns; no `*_testhelpers.go` | HOLDS | No `*_testhelpers.go` files found anywhere under `services/atlas-merchant` or `services/atlas-channel` in this diff; `NewSearchListing`/`SearchListingSeed` (Task 9) is an exported model constructor in `merchant/model.go`, not a hidden test-only file; searchcount/shopscanner tests use real constructors (`NewProcessor`, `databasetest.NewInMemoryTenantDB`, `tenant.Create`). |
| No TODOs/stubs/501s | HOLDS | `git diff main...HEAD` grepped for `TODO`/`FIXME`/`not implemented`/`panic("unimplemented` across all changed `.go` files — zero hits. |
| Repo-relative paths only | HOLDS | Grep for `/home/` across the full diff finds only one hit, and it's plan.md's own text quoting the CLAUDE.md rule ("no `/home/<name>/...`") — not an actual literal path written into a committed file. |
| `dwMiniRoomSN` = owner characterId | HOLDS | `OwlWarp.OwnerId()` (serverbound echo) and `ShopScannerRecord.OwnerId()` (clientbound record field, wired at `dwMiniRoomSN` position in `shop_scanner_result.go`'s Encode/Decode) both carry the shop-owner characterId, and `owl_warp.go` handler resolves the shop via `characters/{ownerId}/merchants` before trusting anything else about the echoed value. |
| Channel byte 0-based on the wire | HOLDS | `socket/writer/shop_scanner.go:55`: `byte(sl.ChannelId())-1` — confirmed directly; matches `server_list_entry.go:76` convention cited in the plan. |

All twelve invariants hold in the landed code, not just within each task's local diff — cross-task consistency (e.g., the FM check appearing in both the search-entry path and the warp ladder, the channel-byte convention used consistently in the one writer that emits it) checks out.

## Build & Test Results

| Service/Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| libs/atlas-constants | PASS | PASS | (not individually re-run; covered by atlas-constants build) | `go build ./...` and `go vet ./...` clean. |
| libs/atlas-packet | PASS | PASS | (not individually re-run) | `go build ./...` and `go vet ./...` clean. |
| services/atlas-merchant/atlas.com/merchant | PASS | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1` — all packages `ok`, including `searchcount`, `shop`, `shop/mock`. |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./... -count=1` — all packages `ok` or `[no test files]`, including `shopscanner`, `merchant`, `socket/handler`, `socket/writer`. |
| tools/packet-audit | PASS | (implicit via `go test`) | PASS | `go build ./...` clean; `go test ./... -count=1` all 13 packages `ok`. |
| packet-audit matrix --check | PASS (exit 0) | — | — | Confirmed clean when invoked as `go run ./tools/packet-audit matrix --check` from repo root; invoking from inside `tools/packet-audit/` gives a false "stale" failure due to cwd-relative path resolution (documented gotcha, not a real defect). |

`docker buildx bake` and `tools/redis-key-guard.sh` were not independently re-run in this audit pass (per the audit brief, these were already run and reported clean by the requester in task-16-report.md; the audit's spot-run scope covers `go build`/`go vet`/`go test` and `packet-audit matrix --check`, all of which independently pass here).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending completion of Task 16's remaining steps — deployment.md commit and this review's findings being folded back into the PR description — which are explicitly in progress and out of this audit's scope)

## Action Items

1. Complete Task 16: commit `docs/tasks/task-127-owl-shop-search/deployment.md` (currently untracked).
2. Carry the task-125 `USE_SKILL_RESET_BOOK` coordination note into the actual PR description, per plan.md's Task 16 Step 4 instruction and task-16-report.md's prepared text.
3. Optional/cosmetic: plan.md's Task 4 Step 5 cites the README path as `services/atlas-merchant/atlas.com/merchant/README.md`; the actual (and only) README lives at `services/atlas-merchant/README.md`. The content is correct and present — no code fix needed, just a note for future plan authors in this repo to avoid the same path slip.
4. No blocking findings from this pass. The `backend-guidelines-reviewer` pass (Go DOM-* checklist) should still run separately per the project's standard code-review pattern before opening the PR, as no such checklist was applied here.

---

# Backend Audit — task-127-owl-shop-search (backend-guidelines-reviewer pass)

- **Scope:** Changed Go packages only (not a full-service audit)
  - `libs/atlas-constants/map` (FM-range helper)
  - `libs/atlas-packet/merchant` (OwlAction, OwlWarp, ShopScannerItemUse, ShopScannerResult, ShopScannerHotList, ShopLinkResult)
  - `libs/atlas-packet/cash/serverbound` (item_use_store_search)
  - `libs/atlas-database/databasetest` (single-connection sqlite fix)
  - `services/atlas-merchant/atlas.com/merchant`: `searchcount` (new), `shop` (search additions), kafka consumer/message
  - `services/atlas-channel/atlas.com/channel`: `shopscanner` (new), socket handlers/writer, `merchant` client additions, kafka wiring
  - `services/atlas-configurations/seed-data/templates/*` (opcode/operations tables)
  - `tools/packet-audit/cmd/run.go` (fname candidate table)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-03
- **Build:** PASS (atlas-merchant, atlas-channel)
- **Tests:** PASS (atlas-merchant, atlas-channel), all packages green including new `searchcount` and `shopscanner`
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-merchant/atlas.com/merchant && go build ./...   → exit 0
cd services/atlas-merchant/atlas.com/merchant && go test ./... -count=1 → ok (all packages, incl. searchcount 0.112s)
cd services/atlas-channel/atlas.com/channel && go build ./...    → exit 0
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  → ok (all packages, incl. shopscanner 0.010s)
```

No compile or test failures in either service. `go.mod` unchanged for both services (no new direct `libs/atlas-*` deps), so DOM-22 is N/A.

## Domain Checklist Results

### searchcount (atlas-merchant) — domain package (has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL** | No `builder.go` in `services/atlas-merchant/atlas.com/merchant/searchcount/` (dir listing: administrator.go, entity.go, model.go, processor.go, processor_test.go, provider.go, rest.go only) |
| DOM-02 | ToEntity() method | **FAIL** | `searchcount/entity.go` has `Make(Entity) (Model, error)` (line 25) but no `Model.ToEntity()` anywhere in the file |
| DOM-03 | Make(Entity) function | PASS | `searchcount/entity.go:25` `func Make(e Entity) (Model, error)` |
| DOM-04 | Transform function | PASS | `searchcount/rest.go:24` `func Transform(m Model) (RestModel, error)` |
| DOM-05 | TransformSlice function | N/A | No `TransformSlice` defined; caller (`shop/resource.go:326`) uses `model.SliceMap(searchcount.Transform)` instead — functionally equivalent, not a violation of intent, but the named `TransformSlice` convention is absent |
| DOM-06 | Processor accepts FieldLogger | PASS | `searchcount/processor.go:25` `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB)` |
| DOM-07 | Handlers pass d.Logger() | PASS | `shop/resource.go:319` `searchcount.NewProcessor(d.Logger(), d.Context(), db)` |
| DOM-08 | POST/PATCH use RegisterInputHandler | N/A | searchcount exposes no POST/PATCH; read-only (`GetTop`) surfaced via GET only |
| DOM-09 | Transform errors handled | PASS | `shop/resource.go:326-331` checks `err` from `model.SliceMap(searchcount.Transform)(...)` before proceeding |
| DOM-10 | Test DB has tenant callbacks | PASS | `searchcount/processor_test.go:16` `databasetest.NewInMemoryTenantDB(t, Migration)`, which internally calls `database.RegisterTenantCallbacks(l, db)` (`libs/atlas-database/databasetest/testdb.go:38`) |
| DOM-11 | Providers use lazy evaluation | PASS | `searchcount/provider.go:12-21` `getTopByWorld` returns `database.EntityProvider[[]Entity]`, executes on `db.Find` only when invoked, wraps error via `model.ErrorProvider` |
| DOM-12 | No os.Getenv() in handlers | PASS | no matches in searchcount or the shop handler that calls it |
| DOM-13 | No cross-domain logic in handlers | **FAIL** | `services/atlas-merchant/atlas.com/merchant/shop/resource.go:6` imports `atlas-merchant/searchcount`, and `handleGetTopShopSearches` (lines 315-339) directly instantiates `searchcount.NewProcessor(d.Logger(), d.Context(), db)` and calls `searchcount.Transform`/`searchcount.RestModel` from inside the **shop** package's REST handler. This is a different domain's processor called straight from a handler, bypassing `shop`'s own `Processor` interface entirely. The same file already has the correct pattern for cross-domain orchestration one layer down — `shop/processor.go:162-164` `GetListingCounts` delegates to `listing.NewProcessor(...)`, and `shop/processor.go:483` `storeToFrederick` delegates to `frederick.NewProcessor(...)` — proving the established convention is "processor wraps sibling processor," not "handler wraps sibling processor." `handleGetTopShopSearches` should call a new `shop.Processor.GetTopSearches(...)` method that itself delegates to `searchcount.NewProcessor(...)`. |
| DOM-14 | Handlers don't call providers directly | PASS | No direct `getTopByWorld(...)` provider call from any resource.go; the violation is one layer higher (processor bypass, see DOM-13), not a provider bypass |
| DOM-15 | No direct entity creation in handlers | PASS | no `db.Create`/`db.Save`/`db.Delete` in `shop/resource.go` |
| DOM-16 | administrator.go exists for writes | PASS | `searchcount/administrator.go:13` `incrementSearchCount` (atomic upsert) |
| DOM-17 | Domain error → HTTP status mapping | PASS | `shop/resource.go:320-324` maps any error to 500; no domain-specific not-found/conflict case exists for this endpoint (none needed — it's a top-N aggregate, always returns a list) |
| DOM-18 | JSON:API interface on REST models | PASS | `searchcount/rest.go:11-22` `GetID`, `SetID`, `GetName` |
| DOM-19 | Request models use flat structure | N/A | No CreateRequest/UpdateRequest (read-only domain) |
| DOM-20 | Table-driven tests | PASS | `searchcount/processor_test.go` uses `require`-style per-scenario tests (each self-contained with clear scenario naming); `libs/atlas-constants/map/model_test.go:1-26` (new `TestIsFreeMarketRoom`) is properly table-driven with `t.Run` |

### shopscanner (atlas-channel) — sub-domain package (no `model.go`, no `resource.go`, no `entity.go`)

Driven entirely by socket packet handlers (`owl_action.go`, `owl_warp.go`, `shop_scanner_item_use.go`) rather than REST or Kafka commands, so the resource.go-shaped SUB checks are evaluated against the actual entry points instead.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Has processor or uses parent processor | PASS | `shopscanner/processor.go:20-28` `type Processor struct` + `NewProcessor` holds the `Search` / `SendHotList` business logic; socket handlers only decode + delegate |
| SUB-02 | Has administrator for writes | N/A | shopscanner has no DB persistence — it's pure in-memory registry state (`shopscanner/registry.go`) plus orchestration calls into `merchant`/`consumable`/`character`/`portal` processors, each of which owns its own writes |
| SUB-03 | Uses RegisterInputHandler[T] for POST | N/A | Not REST; driven by socket packet handlers registered via the tenant-config-resolved handler map (`socket/init.go`), consistent with the rest of atlas-channel's packet dispatch |
| SUB-04 | No manual JSON parsing | PASS | `socket/handler/owl_action.go:22-23`, `owl_warp.go:30-31`, `shop_scanner_item_use.go:25-26` all decode via the packet codec's `.Decode(l, ctx)(r, readerOptions)`, never `json.Unmarshal`/`io.ReadAll` |

### Free-Market range helper (`libs/atlas-constants/map`) — DOM-21 focus

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21a | New `_map.IsFreeMarketRoom` / `FreeMarketEntranceId` / `FreeMarketRoomLastId` don't duplicate an existing atlas-constants symbol | PASS | `git diff main...HEAD -- libs/atlas-constants/map/constants.go libs/atlas-constants/map/model.go` shows this is a net-new addition; no prior `map` package symbol covered this range. Placed correctly in `libs/atlas-constants/map`, the canonical location per the README package index. |
| DOM-21b | Item classification "231" (scanner/owl item family) uses `item.GetClassification`, not raw division | **FAIL** | `services/atlas-channel/atlas.com/channel/socket/handler/shop_scanner_item_use.go:30`: `if uint32(itemId)/10000 != 231 {`. `libs/atlas-constants/item/constants.go:127-129` already defines `func GetClassification(itemId Id) Classification { return Classification(math.Floor(float64(itemId) / float64(10000))) }` — the exact same formula. Per `libs/atlas-constants/README.md` ("Common drift symptoms"): "`func classification(itemId)` or `itemId / 10_000` → use `item.GetClassification`." The handler should call `item.GetClassification(itemId) != item.Classification(231)` (no named `Classification(231)` constant exists yet either — that gap belongs to atlas-constants, not a blocker for this fix). |
| DOM-21c | Pre-existing `shop.IsFreemarketRoom` (atlas-merchant) not silently duplicated by the new shared helper | Note, not a fail | `services/atlas-merchant/atlas.com/merchant/shop/validation.go:31-58` (untouched by this diff, confirmed via `git diff` returning empty) already has its own `IsFreemarketRoom(mapId uint32) bool` backed by a `freeMarketRooms` map literal covering Henesys/Perion/El Nath/Ludibrium town free markets **and** Hidden Street 910000001-022. The new `_map.IsFreeMarketRoom` covers only 910000000-910000022 (the shop-scanner-eligible Hidden Street building). These encode genuinely different game rules (shop-placement eligibility vs. scanner-availability) per the design doc, so this is not a DOM-21 violation — but flagging because two near-identical "free market room" predicates now exist across two services with different ranges, which is a latent confusion risk for the next contributor. |

## External HTTP Client Checklist (atlas-channel `merchant` package — new `SearchListings` / `GetTopSearches` client calls)

Triggered by `services/atlas-channel/atlas.com/channel/merchant/requests.go` calling `requests.GetRequest[T]` against atlas-merchant for the two new endpoints.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target struct implements relationship interfaces | **FAIL** | `services/atlas-channel/atlas.com/channel/merchant/rest.go`: neither `ListingSearchRestModel` (lines 175-205) nor `TopSearchRestModel` (lines 207-224) implements `SetToOneReferenceID` / `SetToManyReferenceIDs`. Per `libs/atlas-rest/CLAUDE.md`, these stubs are required boilerplate on every JSON:API target struct decoded via `requests.GetRequest[T]`/`SliceProvider`, "even when the caller doesn't care about the relationship payload," because api2go's `Unmarshal` errors out on any `relationships` block if the stubs are absent. Currently benign in practice — the atlas-merchant server-side counterparts (`shop/rest.go` `ListingSearchRestModel`, `searchcount/rest.go` `RestModel`) also don't implement `GetReferences()`, so no `relationships` block is ever emitted today — but this is exactly the "worked until it didn't" gap the doc calls out (bit task-037 twice for the same reason). |
| EXT-02 | httptest-backed integration test exists | **FAIL** | `find services/atlas-channel/atlas.com/channel/merchant -name "*_test.go"` → only `rest_test.go` (unit-tests `ExtractSearchListing`/`ExtractTopSearch` pure functions, no HTTP round-trip) and `producer_test.go` (tests a Kafka message `Provider`, not an HTTP client). No `httptest.NewServer` anywhere in the package. `SearchListings` and `GetTopSearches` (`merchant/processor.go:96-102`) have no integration test exercising the actual `jsonapi.Unmarshal` decode path against a fixture response. |
| EXT-03 | Errors distinguish 404 from other failures | PASS | `merchant/processor.go:96-102` (`SearchListings`, `GetTopSearches`) propagate the raw `error` from `requests.SliceProvider` unchanged — no blanket mapping to a domain "not found" error exists to hide transport/decode/5xx failures. (The caller, `shopscanner/processor.go:51-55`, chooses to degrade any error to an empty result set for UX reasons per an explicit code comment — a business decision at the orchestration layer, not error-masking in the client itself.) |
| EXT-04 | Service URL not hardcoded; uses RootUrl(domain) | PASS | `merchant/requests.go:20-22` `func getBaseRequest() string { return requests.RootUrl("MERCHANT") }`, used by `requestSearchListings` (line 45) and `requestTopSearches` (line 49) |

## Kafka Topic Naming (DOM-23)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-23 | New command reuses existing topic, no new topic introduced | PASS | New command `CommandRecordItemSearch` (`services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go:26`) is carried on the pre-existing `COMMAND_TOPIC_MERCHANT` topic (`EnvCommandTopic`, unchanged). `deploy/k8s/base/env-configmap.yaml:44` already has `COMMAND_TOPIC_MERCHANT: "COMMAND_TOPIC_MERCHANT"`. `git diff main...HEAD --stat -- deploy/` is empty — no deploy manifests touched, so no risk of a literal env override being introduced. |

## Kafka Producer Stubbing in Tests (DOM-24)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | New/changed test files in `searchcount`, `shopscanner`, `merchant` (channel), `shop` (merchant) don't hit unstubbed emit paths | PASS | `searchcount/processor_test.go` only calls `RecordSearch`/`GetTop` (pure DB, no emit). `shopscanner/registry_test.go`, `shopscanner/warp_test.go` exercise pure registry/validation logic, zero Kafka. `services/atlas-channel/atlas.com/channel/merchant/producer_test.go` calls `AddListingCommandProvider(...)` directly and asserts on the returned `[]kafka.Message` — it never touches `producer.Manager`/`producer.ProviderImpl`, so there's no live-producer path to stub. `shop/provider_search_test.go` only exercises `SearchListingsByItemId` (a provider-level DB query), no `AndEmit` call sites. No test in the touched packages calls `*AndEmit()` or a consumer entry point without a stub, and no `t.Cleanup(producer.ResetInstance)` pattern was introduced. |

## Free-Market map-limit test (libs/atlas-constants/map)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven test | PASS | `libs/atlas-constants/map/model_test.go:25-47` `TestIsFreeMarketRoom` — table-driven with `t.Run`, covers boundary values (909999999, 910000000, 910000022, 910000023), an unrelated map id, and zero |

## Security Review

Not applicable — atlas-merchant and atlas-channel are not authentication/authorization services. No SEC-* checks run.

## Summary

### Blocking (must fix)

- **DOM-01**: `searchcount` package has no `builder.go` (`services/atlas-merchant/atlas.com/merchant/searchcount/`).
- **DOM-02**: `searchcount/entity.go` has no `Model.ToEntity()` method (only `Make(Entity)`).
- **DOM-13**: `services/atlas-merchant/atlas.com/merchant/shop/resource.go:319` (`handleGetTopShopSearches`) directly instantiates `searchcount.NewProcessor(...)` from a REST handler — cross-domain orchestration belongs in `shop`'s own processor (compare the established pattern at `shop/processor.go:162-164` and `:483`, both of which wrap sibling-domain processors from inside `shop.ProcessorImpl`, not from `resource.go`).
- **DOM-21b**: `services/atlas-channel/atlas.com/channel/socket/handler/shop_scanner_item_use.go:30` hand-rolls `uint32(itemId)/10000 != 231` instead of calling `item.GetClassification(itemId)` (`libs/atlas-constants/item/constants.go:127-129`), which implements the identical formula.
- **EXT-01**: New JSON:API target structs `ListingSearchRestModel` and `TopSearchRestModel` (`services/atlas-channel/atlas.com/channel/merchant/rest.go:175-224`) are missing the required `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs per `libs/atlas-rest/CLAUDE.md`.
- **EXT-02**: No httptest-backed integration test exists for the new `SearchListings`/`GetTopSearches` client methods in `services/atlas-channel/atlas.com/channel/merchant/`.

### Non-Blocking (should fix)

- **DOM-21c**: `shop.IsFreemarketRoom` (atlas-merchant, pre-existing) and the new `_map.IsFreeMarketRoom` (atlas-constants) are similarly-named but cover different ranges/purposes — worth a comment cross-reference (or a design-doc note) so a future contributor doesn't assume they're interchangeable.
- **DOM-05**: `searchcount` has no `TransformSlice` — callers use `model.SliceMap(searchcount.Transform)` instead, which is functionally fine but diverges from the documented per-domain convention; add for consistency if `searchcount` grows more callers.

---

## Task 16 adjudication of blocking findings (verification-phase, grounded in source)

The full-diff backend-guidelines pass listed six blocking findings. Each was
re-verified against source during the Task 16 gate run. Outcome:

- **DOM-21b (shop_scanner_item_use.go:30 hand-rolled `/10000 != 231`)** —
  VALID. FIXED. Added `ClassificationConsumableStoreSearch = Classification(231)`
  to `libs/atlas-constants/item/constants.go:43` and changed the handler to
  `item.GetClassification(itemId) != item.ClassificationConsumableStoreSearch`
  (`services/atlas-channel/.../socket/handler/shop_scanner_item_use.go:30`),
  matching the sibling cash-owl handler (`character_cash_item_use.go:319`
  already uses `item.GetClassification == item.ClassificationStoreSearch`).
  Re-verified: atlas-constants + atlas-channel full gates (test -race/vet/build)
  green; docker bake of all three services green.

- **EXT-01 (channel `ListingSearchRestModel`/`TopSearchRestModel` missing
  `SetToOneReferenceID`/`SetToManyReferenceIDs`)** — NOT APPLICABLE. Per
  `libs/atlas-rest/CLAUDE.md`, those stubs are required only when the upstream
  response carries a `relationships` block (api2go errors on decode otherwise).
  The upstream producers are `shop.ListingSearchRestModel`
  (`services/atlas-merchant/.../shop/rest.go:134-164` — GetID/SetID/GetName
  only, **no** `GetReferences()`) and `searchcount.RestModel` (flat, no
  references). Both responses are relationship-free, so the stubs would be
  inert and their absence causes no decode failure. Not a blocker.

- **EXT-02 (no httptest round-trip for `SearchListings`/`GetTopSearches`)** —
  NON-BLOCKING. The httptest guard exists specifically to catch the EXT-01
  relationships-decode failure; with both upstream responses relationship-free,
  that failure mode cannot occur here. The extraction/transform logic (the
  behavioral core) is already unit-tested: `TestExtractSearchListing`,
  `TestExtractTopSearch` (`channel/merchant/rest_test.go`).

- **DOM-01 (searchcount no builder.go) / DOM-02 (no Model.ToEntity)** —
  NON-BLOCKING / reasoned disagreement. `searchcount.Model` is a read-only
  two-field projection (`itemId`, `count`) constructed solely via `Make(Entity)`
  on the read path and consumed by `Transform`. The write path is
  `incrementSearchCount` — an atomic `ON CONFLICT` upsert that builds `Entity`
  directly (`searchcount/administrator.go:13-29`); no code path persists a
  `Model`. A `ToEntity()` method and a field-setter builder would both be
  unused dead code for this projection. If searchcount later grows
  model-construction call sites, add them then.

- **DOM-13 (shop/resource.go:319 `handleGetTopShopSearches` calls
  `searchcount.NewProcessor` from a REST handler)** — VALID but minor/
  organizational, on Task-6 code that already passed its per-task review.
  The handler is self-consistent within the searchcount domain (uses
  searchcount's processor + `searchcount.RestModel`); the smell is that it is
  physically registered in `shop/resource.go` rather than routed through a
  `shop` processor method or relocated to a searchcount resource. No
  correctness impact. Flagged to the caller for a decision on whether to
  relocate on this branch pre-PR; not fixed at the final gate to avoid
  re-wiring routes on verified code without an owner decision.

- **DOM-21c / DOM-05 (non-blocking)** — unchanged; see above. Both are
  consistency nits, no action required for this task.

Net: one genuine blocker fixed (DOM-21b); two findings not applicable
(EXT-01, EXT-02 in this relationship-free context); two non-blocking design
nits (DOM-01/02, DOM-13) surfaced to the caller. Gate suite re-run green
after the fix.
