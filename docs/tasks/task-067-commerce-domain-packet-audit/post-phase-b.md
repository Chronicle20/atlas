# Task-067 Post-Phase-B — Commerce-Domain Audit Closeout

## Final state

- **Source files audited:** 78 commerce-domain packet source files (15 clientbound + 63 serverbound across `cash`, `interaction`, `inventory`, `storage`).
- **Wire shapes audited (v95):** 77 SUMMARY rows — storage 9, inventory 12, interaction 30, cash 26. (The remaining shapes from the design's ~89 estimate are routers, dispatcher op-bytes, and shapes with no isolatable v95 sender — all documented in `_pending.md` rather than dropped.)
- **v95 commerce verdicts:** 62 ✅ / 13 ❌ / 2 🔍. All 13 ❌ are documented tool-limitation false positives (loop-flatten, sub-struct, int64-vs-buffer representation) with ack footers + `_pending.md` rows — **no unresolved real wire bug remains in v95**.
- **Cross-version coverage:** IDA evidence populated for **GMS v83 / v87 / v95 / JMS v185**. Per-version commerce tallies — v83: 13 ✅ / 4 ❌ / 2 🔍; v87: 9 ✅ / 2 ❌ (benign); JMS v185: 10 ✅ / 5 ❌ (JMS NX-payment protocol, deferred — see Remaining work).
- **Real wire bugs found:** 8 logical bugs (11 Phase-1 deferrals collapsed to 8 after cross-version analysis). **All 8 fixed** with 4-variant test sweeps and IDA-cited version gates. Zero template changes were needed; every fix is an atlas-packet encoder change.
- **Regression gate:** v95 re-run after all fixes — login / character / social verdicts **identical** (105/105 rows, order-independent compare). No regression from the Phase 0 registry extension.
- **Verification:** `go build`, `go vet ./libs/atlas-packet/...`, `go test -race ./libs/atlas-packet/...`, `go test -race ./tools/packet-audit/...` all clean. No `go.mod`/`Dockerfile`/service code touched → no docker build required. gitleaks scrub clean.

## Real wire bugs fixed

| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
| Show (open-storage) | `storage/clientbound/show.go` | `CTrunkDlg::SetGetItems` v95@0x76a390 / v83@0x7c5dfd / v87@0x819648 | per-tab item segmentation (count+items per set tab bit) + conditional meso (flag&2) + drop 3 spurious padding bytes | all (unconditional — identical v83→v95) |
| OperationPersonalStoreBuy / OperationMerchantBuy | `interaction/serverbound/operation_personal_store_buy.go`, `operation_merchant_buy.go` | `CPersonalShopDlg::BuyItem` v95@0x69a7f0 / v83@0x6fd261 / v87@0x74076b | add trailing `uint32 itemCRC` (`GetItemCRC`) | all (unconditional) |
| OperationPersonalStoreSetBlackList | `interaction/serverbound/operation_personal_store_set_black_list.go` | `DeliverBlackList` v83@0x6fdeda / v87@0x74146f | atlas wrote `count×byte`; client reads `count×EncodeStr(name)` → emit string entries | all (unconditional) |
| OperationChat | `interaction/serverbound/operation_chat.go` | `CheckAndSendChat` v95@0x6382a0 / v87@0x69973e / JMS@0x6db3ce | add leading `uint32 update_time` | `(GMS && ≥87) \|\| JMS` (absent in v83) |
| ShopOperationBuy | `cash/serverbound/shop_operation_buy.go` | `CCashShop::OnBuy` v95@0x48e530 / v83@0x46dadd / v87@0x477bd9 | add trailing `byte oneADay` + `int eventSN` (atlas had single `int zero`) | `GMS && ≥87` (single int in v83) |
| SPW family — BuyCouple / BuyFriendship / RebateLockerItem | `cash/serverbound/shop_operation_buy_couple.go`, `_friendship.go`, `shop_operation_rebate_locker_item.go` | `OnBuyCouple`@0x490d80, `OnBuyFriendship`@0x491b30 (v95) | leading `EncodeStr(ask_SPW)` where atlas wrote `int birthday` | `GMS && ≥95` (int in v83/v87) |
| ShopOperationGift | `cash/serverbound/shop_operation_gift.go` | `SendGiftsPacket` v95@0x487b60 / v87@0x47a168 | split gate: leading SPW string (`≥95`) + `oneADay` byte (`≥87`) | sequential guards (within 2-deep cap) |
| CashShopInventory | `cash/clientbound/shop_inventory.go` | `OnCashItemResLoadLockerDone` v95@0x494cb0 / JMS@0x48bcff | emit 4 trailing slot-counter shorts (atlas wrote 2): trunk, charSlot, **buyChar, charCount** | `(GMS && ≥95) \|\| JMS` (2 shorts in v83/v87) |

All version gates are single-depth or sequential guards (`Region()=="GMS" && MajorVersion()>=NN`, optionally `|| Region()=="JMS"`); none exceeds the 2-nested-guard cap. Each fix lands with a 4-variant byte/round-trip test (v28 / v83 / v95 / JMS v185).

## Template opcode / enum fixes

None. No `template_*.json` changes were required — every divergence was an atlas-packet encoder wire-shape issue, not a template opcode/enum drift. The cash `shop_operation_body.go` 78-constant operations/errors table showed **no >10 stale-code drift** in any single dispatcher (`CCashShop::OnCashItemResult`@0x499370 verified).

## Tooling improvements

- **Phase 0 registry extension** (`8bc48d3a8`): `TypeRegistry` pass-2 now recognises `EncodeBytes` (flat `[]byte`) and `EncodeEntry` (closure) foreign-encoder method names, with a fixture asserting `CashInventoryItem` + `AddEntry`/`QuantityUpdateEntry`/`MoveEntry`/`RemoveEntry` coverage. Confirmed working end-to-end: cash `CashInventoryItem` recursion (3 call sites) and inventory `change_batch` `ChangeEntry` family both resolved inline instead of opaque `WriteByteArray`.
- **`candidatesFromFName` wiring** (`e1d6b4192`, `5783adbe0`, `ab8700e97`, `0dac2f48d`): per-domain IDA-FName→packet mappings added for storage (CTrunkDlg), inventory (CWvsContext), interaction (CMiniRoomBaseDlg/CTradingRoomDlg/CPersonalShopDlg/CCashTradingRoomDlg), cash (CCashShop). Added an optional `pathHint` to `candidate`/`locateAtlasFile` to disambiguate struct-name collisions across packages (e.g. `Operation`/`ErrorSimple`). JMS reused the GMS FName keys (no new cases).
- **IDA exports:** `gms_v83.json` (commerce appended), `gms_v95.json` (commerce appended), `gms_v87.json` (created from scratch), `gms_jms_185.json` (created from scratch).
- **Report path normalization** (`d24a3633e`): standardised audit report paths to the canonical `../../libs/...` form (run from `tools/packet-audit/` with `../../` paths) so cross-task re-runs produce zero churn in login/character/social reports.
- **`phase-0-survey.md`:** foreign-encoder survey + cash constructor↔struct map (8 `CashShop*Body` factories → 8 target structs across `shop_inventory.go`/`shop_operation_result.go`/`shop_item_moved.go`).
- **`model.Asset.InventoryType()`:** additive public accessor (wraps existing private `inventoryType()`) enabling the storage Show per-tab segmentation fix. Zero behavior change to existing encoders.

## Remaining work

| Area | What | Why deferred |
|---|---|---|
| **JMS v185 cash-shop** | NX-point payment protocol — 5 packets (ShopOperationBuy/Gift/BuyCouple/BuyFriendship/RebateLockerItem) with JMS-specific op-bytes (0x2E/0x1E/0x24/0x1B) and field shapes | Out-of-scope JMS-only feature requiring region-dispatched encoders + template op-byte remap + NX query/charge wiring; a 3rd gate would breach the 2-guard cap. **Sibling task suggested:** "JMS v185 cash-shop NX-payment protocol support." |
| **interaction** | 7 serverbound sub-ops with no isolatable v95 sender (create, open, visit, cash_trade_open, name_change, set_visitor, invite_decline) | Senders are inlined in field/inventory-drag/cash-shop UI paths; honestly 🔍-deferred with a spike spec rather than guessed. |
| **cash** | Shapes with no isolatable v95 sender: `item_use` family (248KB `SendConsumeCashItemUseRequest` dispatcher), `shop_open`/`shop_entry` inline construction, `CashItemMovedToInventory` asset sub-struct, `CashShopGifts` | 🔍-deferred; common prefix read but per-variant layouts unverified. |
| **interaction / storage routing** | OP-FAMILY-interaction, OP-FAMILY-storage, INTERACTION-MODE-MAP, INTERACTION-CB-MODE-MAP — mode-byte → sub-op routing | Routing layer lives in atlas-channel, outside `libs/atlas-packet/` (PRD §3 non-goal). Documented for traceability. |
| **tool limitations (not bugs)** | UpdateAssets, Add, ChangeBatch, OperationMemoryGameMoveStone, InteractionEnter/EnterResultSuccess/UpdateMerchant, cash WishList/SetWishlist/Increase*/Move* | packet-audit can't model per-tab loops, sub-structs, or int64-vs-`EncodeBuffer(8)` representation. Verified wire-correct via unit tests; ack footers + `_pending.md` rows. |
| **low-confidence boundary** | `ShopOperationRebateLockerItem` `≥95` SPW gate held by family-parity inference (no v87/v95 sender symbol recovered) | Conservative — keeps v83/v95 known-good; flagged LOW CONFIDENCE in `gms_v87.json`. |
| **value plumbing** | `CashShopInventory` `buyChar`/`charCount` shorts emit 0 (non-constructor fields) | Wire-shape is now correct; populating real values is an atlas-channel domain-state concern, separate from this packet-audit task. |

See `docs/packets/ida-exports/_pending.md` → "## Sub-op enum / sub-struct deferrals — commerce domain (task-067)" for the full per-cause ledger with IDA citations.
