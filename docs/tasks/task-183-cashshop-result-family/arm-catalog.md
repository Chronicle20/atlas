# `CCashShop::OnCashItemResult` arm catalog

Single source of wire truth for task-183. Mode bytes: DECIMAL in `cash_shop_operation.yaml`
(the tool reads ints), HEX here. `n-a` = arm absent in that version's switch, proven by full
case enumeration at the switch address (address cited in the per-arm note). Every field row is
cited to a decompile line. Shape group is a reasoning aid only — every arm is a DISCRETE struct.

| operation key | v95 handler fname | shape | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v95 | jms | fields (v95, decompile-cited) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| LIMIT_GOODS_COUNT_CHANGED | OnCashItemResLimitGoodsCountChanged | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x54 | TBD-RE | mode(disp); itemId:Decode4@0x493f47 (int32); sn:Decode4@0x493f52 (int32); remainCount:Decode4@0x493f56 (int32) |
| LOAD_INVENTORY_SUCCESS | OnCashItemResLoadLockerDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x4B | 0x4E | 0x50 | 0x58 | 0x4E | existing |
| LOAD_INVENTORY_FAILURE | OnCashItemResLoadLockerFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x4C | 0x4F | 0x51 | 0x59 | 0x4F | existing |
| LOAD_GIFT_SUCCESS | OnCashItemResLoadGiftDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x5A | TBD-RE | TBD-RE (0.3) |
| LOAD_GIFT_FAILED | OnCashItemResLoadGiftFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x5B | TBD-RE | mode(disp); reason:Decode1@0x496967 |
| LOAD_WISHLIST | OnCashItemResLoadWishDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x4F | 0x52 | 0x54 | 0x5C | 0x52 | existing |
| LOAD_WISH_FAILED | OnCashItemResLoadWishFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x5D | TBD-RE | mode(disp); reason:Decode1@0x496997 |
| UPDATE_WISHLIST | OnCashItemResSetWishDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x55 | 0x58 | 0x5A | 0x62 | 0x56 | existing |
| SET_WISH_FAILED | OnCashItemResSetWishFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x63 | TBD-RE | mode(disp); reason:Decode1@0x4969ce |
| PURCHASE_SUCCESS | OnCashItemResBuyDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x57 | 0x5A | 0x5C | 0x64 | 0x58 | existing |
| BUY_FAILED | OnCashItemResBuyFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x65 | TBD-RE | mode(disp); reason:Decode1@0x496a24; if reason∈{29,30}: goodsSN:Decode4@0x496a3d; if reason==68: extra:Decode1@0x496a97 (conditional — see report) |
| USE_COUPON_SUCCESS | OnCashItemResUseCouponDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x66 | TBD-RE | TBD-RE (0.3) |
| GIFT_COUPON_SUCCESS | OnCashItemResGiftCouponDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x68 | TBD-RE | TBD-RE (0.3) |
| USE_COUPON_FAILED | OnCashItemResUseCouponFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x69 | TBD-RE | mode(disp); reason:Decode1@0x496f9f |
| GIFT_SUCCESS | OnCashItemResGiftDone | item-blob (resolves 0x4D TODO) | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x6B | TBD-RE | TBD-RE (0.3) |
| GIFT_FAILED | OnCashItemResGiftFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x6C | TBD-RE | mode(disp); reason:Decode1@0x497224 |
| INVENTORY_CAPACITY_INCREASE_SUCCESS | OnCashItemResIncSlotCountDone | counter | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x60 | 0x63 | 0x65 | 0x6D | 0x61 | existing |
| INVENTORY_CAPACITY_INCREASE_FAILED | OnCashItemResIncSlotCountFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x61 | 0x64 | 0x66 | 0x6E | 0x62 | existing |
| INC_TRUNK_COUNT_SUCCESS | OnCashItemResIncTrunkCountDone | counter | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x6F | TBD-RE | mode(disp); trunkCount:Decode2@0x494edf (uint16, new absolute m_nTrunkCount — no separate type byte) |
| INC_TRUNK_COUNT_FAILED | OnCashItemResIncTrunkCountFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x70 | TBD-RE | mode(disp); reason:Decode1@0x4973e4 |
| INC_CHARACTER_SLOT_COUNT_SUCCESS | OnCashItemResIncCharacterSlotCountDone | counter | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x71 | TBD-RE | mode(disp); slotCount:Decode2@0x494f7f (uint16, new absolute m_nCharacterSlotCount — no separate type byte) |
| INC_CHARACTER_SLOT_COUNT_FAILED | OnCashItemResIncCharacterSlotCountFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x72 | TBD-RE | mode(disp); reason:Decode1@0x497424 |
| INC_BUY_CHARACTER_COUNT_SUCCESS | OnCashItemResIncBuyCharacterCountDone | counter | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x73 | TBD-RE | mode(disp); buyCharacterCount:Decode2@0x495007 (uint16, new absolute m_nBuyCharacterCount — no separate type byte) |
| INC_BUY_CHARACTER_COUNT_FAILED | OnCashItemResIncBuyCharacterCountFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x74 | TBD-RE | mode(disp); reason:Decode1@0x497464 |
| ENABLE_EQUIP_SLOT_EXT_SUCCESS | OnCashItemResEnableEquipSlotExtDone | counter | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x75 | TBD-RE | mode(disp); slotIndex:Decode2@0x4974c4 (uint16, indexes aEquipExtExpire[v3]/aEquipped2[v3+59+13]); days:Decode2@0x4974d2 (uint16, passed to Util::FTAddDay as day-count) — two uint16 fields, not byte+short |
| ENABLE_EQUIP_SLOT_EXT_FAILED | OnCashItemResEnableEquipSlotExtFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x76 | TBD-RE | mode(disp); reason:Decode1@0x497704 |
| CASH_ITEM_MOVED_TO_INVENTORY | OnCashItemResMoveLtoSDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x68 | 0x6B | 0x6D | 0x77 | 0x6B | existing |
| MOVE_L_TO_S_FAILED | OnCashItemResMoveLtoSFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x78 | TBD-RE | mode(disp); reason:Decode1@0x49773e |
| CASH_ITEM_MOVED_TO_CASH_INVENTORY | OnCashItemResMoveStoLDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x6A | 0x6D | 0x6F | 0x79 | 0x6D | existing |
| MOVE_S_TO_L_FAILED | OnCashItemResMoveStoLFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x7A | TBD-RE | mode(disp); reason:Decode1@0x497939 |
| DESTROY_SUCCESS | OnCashItemResDestroyDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x7B | TBD-RE | mode(disp); sn:DecodeBuffer(8)@0x495269 (int64 LARGE_INTEGER cash-item SN, matched against m_aCashItemInfo[i].liSN) |
| DESTROY_FAILED | OnCashItemResDestroyFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x7C | TBD-RE | mode(disp); reason:Decode1@0x49795e |
| EXPIRE_DONE | OnCashItemResExpireDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x7D | TBD-RE | mode(disp); sn:DecodeBuffer(8)@0x497797 (int64 LARGE_INTEGER cash-item SN, matched against m_aCashItemInfo[i].liSN) |
| REBATE_SUCCESS | OnCashItemResRebateDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x96 | TBD-RE | TBD-RE (0.3) |
| REBATE_FAILED | OnCashItemResRebateFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x97 | TBD-RE | mode(disp); reason:Decode1@0x497ade |
| COUPLE_SUCCESS | OnCashItemResCoupleDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x98 | TBD-RE | TBD-RE (0.3) |
| COUPLE_FAILED | OnCashItemResCoupleFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x99 | TBD-RE | mode(disp); reason:Decode1@0x497d32; if reason∈{29,30}: goodsSN:Decode4@0x497d4b (conditional — see report) |
| BUY_PACKAGE_SUCCESS | OnCashItemResBuyPackageDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9A | TBD-RE | TBD-RE (0.3) |
| BUY_PACKAGE_FAILED | OnCashItemResBuyPackageFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9B | TBD-RE | mode(disp); reason:Decode1@0x496d52; if reason∈{29,30}: goodsSN:Decode4@0x496d6b (conditional — see report) |
| GIFT_PACKAGE_SUCCESS | OnCashItemResGiftPackageDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9C | TBD-RE | TBD-RE (0.3) |
| GIFT_PACKAGE_FAILED | OnCashItemResGiftPackageFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9D | TBD-RE | mode(disp); reason:Decode1@0x496f32; if reason∈{29,30}: goodsSN:Decode4@0x496f4b (conditional — see report) |
| BUY_NORMAL_SUCCESS | OnCashItemResBuyNormalDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9E | TBD-RE | TBD-RE (0.3) |
| BUY_NORMAL_FAILED | OnCashItemResBuyNormalFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0x9F | TBD-RE | mode(disp); reason:Decode1@0x497b12; if reason∈{29,30}: goodsSN:Decode4@0x497b2b (conditional — see report) |
| FRIENDSHIP_SUCCESS | OnCashItemResFriendShipDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xA2 | TBD-RE | TBD-RE (0.3) |
| FRIENDSHIP_FAILED | OnCashItemResFriendShipFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xA3 | TBD-RE | mode(disp); reason:Decode1@0x497f52; if reason∈{29,30}: goodsSN:Decode4@0x497f6b (conditional — see report) |
| FREE_CASH_ITEM_DONE | OnCashItemResFreeCashItemDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xAA | TBD-RE | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x494897 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| PURCHASE_RECORD | OnCashItemResPurchaseRecord | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xAF | TBD-RE | mode(disp); goodsSN:Decode4@0x495b60 (int32, used as ZMap key or 0-check); purchased:Decode1@0x495b74 (byte, compared !=0 → bool) |
| PURCHASE_RECORD_FAILED | OnCashItemResPurchaseRecordFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB0 | TBD-RE | mode(disp); unusedByte:Decode1@0x494074 (value discarded, no further reads — see report) |
| NAME_CHANGE_BUY_DONE | OnCashItemNameChangeResBuyDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB3 | TBD-RE | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x495639 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| TRANSFER_WORLD_SUCCESS | OnCashItemResTransferWorldDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB5 | TBD-RE | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x495749 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| TRANSFER_WORLD_FAILED | OnCashItemResTransferWorldFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB6 | TBD-RE | mode(disp); reason:Decode1@0x49837e |
| GACHAPON_OPEN_SUCCESS | OnCashItemResCashGachaponOpenDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB7 | TBD-RE | TBD-RE (0.3) |
| GACHAPON_OPEN_FAILED | OnCashItemResCashGachaponOpenFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB8 | TBD-RE | mode(disp); reason:Decode1@0x4962c4 |
| GACHAPON_COPY_SUCCESS | OnCashItemResCashGachaponCopyDone | item-blob | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xB9 | TBD-RE | TBD-RE (0.3) |
| GACHAPON_COPY_FAILED | OnCashItemResCashGachaponCopyFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xBA | TBD-RE | mode(disp); reason:Decode1@0x4962fe |
| CHANGE_MAPLE_POINT_SUCCESS | OnCashItemResChangeMaplePointDone | scalar | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xBB | TBD-RE | mode(disp); sn:DecodeBuffer(8)@0x498559 (int64 LARGE_INTEGER cash-item SN); count:Decode4@0x4985cc (int32, formatted into notice string) |
| CHANGE_MAPLE_POINT_FAILED | OnCashItemResChangeMaplePointFailed | failure | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | TBD-RE | 0xBC | TBD-RE | mode(disp); NO Decode1/4 call in handler — zero packet reads beyond mode byte (see report) |

## Count note

Design Appendix A enumerates **57 rows (9 existing + 48 NEW)**. The design/plan
summary prose ("56 arms / 47 new" — `design.md` line 473) is a miscount: the
Appendix A table itself has 57 rows (verified by direct count), and the Wave 1
task key-lists (`.superpowers/sdd/task-0.2-keymap.md`) total 48 new keys, not
47. This catalog reproduces all 57 rows. The true switch-case count at the
binary level is confirmed by Task 0.3's IDB enumeration — if that enumeration
disagrees with this 57-row table, stop and ask rather than silently
reconciling.

## Existing-9 shape assignment (from the current codec, `libs/atlas-packet/cash/clientbound/`)

Shape for the 9 existing arms is read off the current struct bodies (not
guessed): `shop_inventory.go` (`CashShopInventory`, `CashShopPurchaseSuccess`
— both carry a `CashInventoryItem`/list → `item-blob`), `shop_item_moved.go`
(`CashItemMovedToInventory`, `CashItemMovedToCashInventory` — `item-blob`),
and `shop_operation_result.go` (`LoadInventoryFailure`, `InventoryCapacityFailed`
— `failure`; `InventoryCapacitySuccess` — `counter`; `WishListLoad`,
`WishListUpdate` — fixed 10×uint32 SN array, `scalar` per design §3.2's
"small scalar bodies or lists", not `item-blob` since neither carries a
`GW_CashItemInfo`/`GW_ItemSlotBase` blob).
