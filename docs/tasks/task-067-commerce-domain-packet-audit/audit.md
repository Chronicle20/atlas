# Plan Audit — task-067-commerce-domain-packet-audit

**Plan Path:** docs/tasks/task-067-commerce-domain-packet-audit/plan.md
**Audit Date:** 2026-05-28
**Branch:** task-067-commerce-domain-packet-audit
**Base Branch:** main (merge-base 3bab0d885)

## Plan adherence review

### Executive Summary

The plan was faithfully executed. All 12 tasks across Phase 0–4 have commit/file evidence. Phase 0 registry extension recognises both `EncodeBytes` and `EncodeEntry` with the `Encode`-wins precedence guard preserved. Every Phase 1 `❌` verdict (13 in v95) has either a fix commit or a `_pending.md` row — none silently skipped. All 8 claimed wire-bug fixes have real encoder changes, 4-variant `pt.Variants` round-trip + byte-pinned tests, and IDA-cited commit bodies; no version gate exceeds the 2-nested cap. The Phase 3 regression claim is independently re-verified: the 27 login/character/social verdict rows are byte-identical (order-independent) between base and HEAD. No `template_*.json`, `go.mod`, or `Dockerfile` was touched. Builds/tests clean.

### Task completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Phase 0 foreign-encoder survey | DONE | `f405e50f8`; `phase-0-survey.md` committed |
| 2 | Registry `EncodeBytes`+`EncodeEntry` extension | DONE | `8bc48d3a8`; registry.go:101-133 + fixture registry_test.go:40 |
| 3 | Cash constructor↔struct map | DONE | `0908da379`; survey appended |
| 4 | Phase 1a storage (9 shapes) | DONE | `c1286d80c`,`d24a3633e`,`13e666c71`,`e1d6b4192`; 9 SUMMARY rows |
| 5 | Phase 1b inventory (12 shapes) | DONE | `f06a8caab`,`74469a610`,`5783adbe0`; 12 SUMMARY rows |
| 6 | Phase 1c interaction (30 shapes) | DONE | `d6abbe5dc`,`ab8700e97`; 30 SUMMARY rows |
| 7 | Phase 1d cash (26 shapes) | DONE | `5f4ba22a5`,`0dac2f48d`; 26 SUMMARY rows |
| 8 | GMS v83 cross-version | DONE | `fb87e3848`; gms_v83.json + audits/gms_v83/ |
| 9 | GMS v87 cross-version | DONE | `589548846`,`49ad40abb`; gms_v87.json created |
| 10 | JMS v185 cross-version | DONE | `e71deb8f2`,`8b67fb93f`,`116e6af46`; gms_jms_185.json created |
| 11 | Phase 3 regression confirm | DONE | prior-task rows byte-identical base↔HEAD (re-verified) |
| 12 | Closeout / verification | DONE | `ccc5ad203`; post-phase-b.md; 4 verify cmds clean |

**Completion Rate:** 12/12 tasks (100%). **Skipped without approval:** 0. **Partial:** 0.

### Detailed findings

**1. Phase 0 registry (registry.go:97-133).** Precedence guard intact — `case "Encode"` unconditionally overwrites `entry.Calls`; `EncodeEntry`/`EncodeBytes`/`Write` all guard on `entry.Calls == nil` plus the line 98 skip (`entry.Calls != nil && name != "Encode"`). Both new method names recognised. Fixture asserts `CashInventoryItem`+4 `*Entry` types resolve. PASS.

**2. Phase 1 ❌ accountability.** 13 v95 `❌` rows enumerated; each cross-checked against `_pending.md`:
all 13 (InteractionEnter, WishList, ShopOperationIncrease{Inventory,Storage}, ChangeBatch, InteractionEnterResultSuccess, ShopOperationMove{To,From}CashInventory, InteractionUpdateMerchant, Show, OperationMemoryGameMoveStone, ShopOperationRebateLockerItem, ShopOperationSetWishlist) are documented with ack footers / per-cause rows. None silently skipped.

**3. 8 wire-bug fixes.** Each has (a) a real encoder change, (b) a 4-variant `pt.Variants` round-trip test + byte-pinned hex assertions, (c) IDA address citation in the commit body. Verified: shop_operation_gift_test.go pins v83/v87/v95 hex explicitly; gift split-gate is two *sequential* single-depth guards (gift.go:49-57), not nested. The `(GMS&&>=87)||JMS` predicates (operation_chat.go:31-33, shop_inventory.go:131-133) are extracted to boolean helpers → zero nesting in encoder bodies. Automated nesting scan across all changed commerce encoders: **0 OVER CAP**.

**4. Phase 3 regression.** Re-derived: 27 prior-domain verdict rows identical (order-independent diff empty). Only non-commerce SUMMARY change is row reordering from the path-normalization commit `d24a3633e`; the lone `❌` (CharacterList) is a pre-existing task-028 sub-struct limitation present at base. NO login/character/social encoder modified on this branch.

**5. Deferrals legitimacy.** Honest scope calls, well-documented:
- 7 interaction sub-ops "no isolatable v95 sender" — 🔍 with spike-spec rationale (senders inlined in field/drag/UI paths).
- Cash shapes with no v95 sender (item_use family, shop_open/entry, asset sub-structs) — 🔍, common prefix read, per-variant unverified.
- 5 JMS NX-payment packets — deferred to a named sibling task with full IDA op-byte evidence (0x2E/0x1E/0x24/0x1B) and a hard-cap rationale (3rd gate would breach the 2-guard cap). Two in-scope JMS bugs (chat, locker counters) WERE fixed this pass rather than deferred — correct triage.

**6. Scope confirmations.** `git diff` confirms: zero `template_*.json` changes, zero `go.mod`/`Dockerfile` changes, no service code touched (only `libs/atlas-packet` + `tools/packet-audit` + docs). gitleaks scrub clean (no `/home/` paths in any audit report or survey). `model.Asset.InventoryType()` is a clean additive accessor (asset.go:104-105) wrapping the private method.

### Build & test results

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | `go test -race ./...` clean |
| tools/packet-audit | PASS | PASS | `go test -race ./...` clean (registry fixture green) |
| services/atlas-channel (consumer) | PASS | n/a | `go build ./...` exit 0 |

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

### Action Items

None blocking. Optional follow-up: spin up the named sibling task "JMS v185 cash-shop NX-payment protocol support" tracked in `_pending.md`.

---

## Backend guidelines audit

**Auditor:** backend-guidelines-reviewer (adversarial)
**Date:** 2026-05-28
**Scope:** Go encoder + tooling changes on `task-067-commerce-domain-packet-audit` (`3bab0d885..HEAD`).
**Modules audited:** `libs/atlas-packet` (encoder library), `tools/packet-audit` (tooling). Neither is a DDD service — no `model.go`/`processor.go`/`resource.go` domain packages exist, so the service-oriented DOM-01..DOM-20, SUB-*, EXT-*, SCAFFOLD-*, and SEC-* checklists are N/A. The applicable checks are general Go hygiene + DOM-21 (constant duplication) + the task's immutability/version-gate/test-fidelity focus areas.

### Objective gate (Phase 1)
- `cd libs/atlas-packet && go vet ./...` — clean (exit 0).
- `cd libs/atlas-packet && go test -race ./...` — all packages PASS (no failures).
- `cd tools/packet-audit && go vet ./...` — clean (exit 0).
- `cd tools/packet-audit && go test -race ./...` — all packages PASS.
- **Build/Test: PASS.**

### Findings

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GO-01 | Model immutability (private fields, value-receiver getters, no shared-state leak) | PASS | New fields all private with value-receiver getters: `cash/clientbound/shop_inventory.go:67-71` (`buyCharacterCount`/`characterCount` + getters lines 81-82); `cash/serverbound/shop_operation_buy.go:14-17,28-29`; `shop_operation_gift.go:17-21,33-35`; `interaction/serverbound/operation_chat.go:13-16,19`. |
| GO-02 | `model/asset.go` `InventoryType()` is a clean additive accessor (no internal-state leak) | PASS | `model/asset.go:104-107` — returns `inventory.Type` by value, delegating to private `inventoryType()` (line 99-102); no pointer/slice exposed, immutability intact. |
| GO-03 | Version gates use `Region()` as string, not world.Id | PASS | All gates compare `t.Region() == "GMS"` / `== "JMS"`: `shop_operation_buy.go:30,49`; `shop_operation_buy_couple.go:46,61`; `shop_operation_gift.go:45,49,62,66`; `operation_chat.go:31`; `shop_inventory.go:131`. No `world.Id` coercion. |
| GO-04 | Version guards ≤2 nested, predicates readable | PASS | Deepest nesting is a single `if/else` inside the returned closure. Multi-clause predicates extracted to named helpers `chatHasUpdateTime` (`operation_chat.go:25-27`) and `cashInventoryHasExtraCounts` (`shop_inventory.go:128-131`). No magic-number sprawl — versions inline as `>= 87` / `>= 95` with IDA-cited comments. |
| GO-05 | No new `interface{}` params / no `reflect` | PASS | `git diff` of changed src shows no added `interface{}` params (only pre-existing `options map[string]interface{}` signature) and zero `reflect` usage. |
| GO-06 | No `*_testhelpers.go` / no test-only constructors on prod types | PASS | `find libs/atlas-packet -name '*_testhelpers.go'` → empty. Test fixtures are local in-file funcs (`storage/clientbound/show_test.go:12-21` `testAsset`/`etcAsset`) built via the immutable `model.NewAsset(...).SetStackableInfo(...)` chain. |
| GO-07 | Tests assert real wire bytes (hex + round-trip `Available()==0`), use `pt.Variants` + existing `New*` constructors | PASS | `pt.RoundTrip` enforces `reader.Available()==0` (`test/roundtrip.go:21-22`). Byte-pinning: `shop_operation_gift_test.go:53-74` (per-version hex), `show_test.go:39-51` (offset/count assertions). Variant sweep: `shop_operation_gift_test.go:12`, `show_test.go:31,86,129`. |
| GO-08 | Tooling (packet-audit) Go hygiene: error handling, no dead code, no stubs | PASS | `registry.go:108-127` adds `EncodeEntry`/`EncodeBytes` recognition with correct `entry.Calls == nil` "Encode-wins" precedence guard; no swallowed errors, no `TODO`/`panic` introduced in `cmd/run.go` diff. Tests added (`registry_test.go`, `analyzer_test.go`, `idasrc/export_test.go`) and pass. |
| DOM-21 | No duplication of shared constants | MINOR | `storage/clientbound/show.go:14-21` redeclares storage tab-flag bit values (`showFlagCurrency=2`…`showFlagCash=64`) that already exist as `storage.StorageFlag*` in `libs/atlas-packet/storage/operation_body.go:30-35`. **Justified exception:** `storage` (parent) imports `storage/clientbound` (`operation_body.go:8`), so importing back would be an import cycle. Inventory types correctly reuse `inventory.TypeValue*` from `libs/atlas-constants`. Risk = silent drift if `StorageFlag` values change; the `// mirrors storage.StorageFlag` comment documents intent but no compile-time link enforces parity. |

### Observations (not attributable to this task)
- **gofmt alignment drift** in changed getter blocks (`shop_inventory.go`, `shop_operation_buy.go`, `shop_operation_gift.go`, `asset.go`, `show.go`, et al.). Verified the same files were already gofmt-unclean at merge-base `3bab0d885`; this task neither introduced nor regressed it. `go vet` (the gate) is clean. INFO only.
- **`itemCRC` added unconditionally** (no version gate) in `operation_merchant_buy.go:31,42` and `operation_personal_store_buy.go:35,44`, unlike the gated chat/cash fields. This is a wire-correctness decision outside the guideline checklist — flagged for the wire reviewer, not a guidelines fail.

### Verdict

**PASS / READY_TO_MERGE** — no BLOCKER or MAJOR backend-guideline violations. One MINOR (DOM-21 constant mirror in `show.go`) is a documented, import-cycle-forced exception. Immutability, version-gate discipline (`Region()`-as-string, ≤2 nesting), and test fidelity (real wire bytes, variant sweep, no test-only constructors) all hold. Build and `-race` tests clean in both modules.
