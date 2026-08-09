# `CCashShop::OnCashItemResult` arm catalog

Single source of wire truth for task-183. Mode bytes: DECIMAL in `cash_shop_operation.yaml`
(the tool reads ints), HEX here. `n-a` = arm absent in that version's switch, proven by full
case enumeration at the switch address (address cited in the per-arm note). Every field row is
cited to a decompile line. Shape group is a reasoning aid only — every arm is a DISCRETE struct.

| operation key | v95 handler fname | shape | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v95 | jms | fields (v95, decompile-cited) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| LIMIT_GOODS_COUNT_CHANGED | OnCashItemResLimitGoodsCountChanged | scalar | 0x29 | 0x2B | 0x33 | 0x3F | 0x47 | 0x4A | 0x4C | 0x54 | 0x4A | mode(disp); itemId:Decode4@0x493f47 (int32); sn:Decode4@0x493f52 (int32); remainCount:Decode4@0x493f56 (int32) |
| LOAD_INVENTORY_SUCCESS | OnCashItemResLoadLockerDone | item-blob | 0x2A | 0x2F | 0x37 | 0x43 | 0x4B | 0x4E | 0x50 | 0x58 | 0x4E | existing |
| LOAD_INVENTORY_FAILURE | OnCashItemResLoadLockerFailed | failure | 0x2B | 0x30 | 0x38 | 0x44 | 0x4C | 0x4F | 0x51 | 0x59 | 0x4F | existing |
| LOAD_GIFT_SUCCESS | OnCashItemResLoadGiftDone | item-blob | n-a | 0x31 | 0x39 | 0x45 | 0x4D | 0x50 | 0x52 | 0x5A | 0x50 | mode(disp); count:Decode2@0x496554 (uint16); giftList:DecodeBuffer(98×count)@0x49657d — COUNT-prefixed LIST of GW_GiftList, 98-byte fixed record (struct confirmed via `type_inspect(GW_GiftList)`, cross-checked against the read-loop @0x49658c-0x496922): liSN offset 0 (8B, `_LARGE_INTEGER` SN, re-echoed verbatim into the ack COutPacket 0x9A via `mov edx,[ebx-0Ch]`/`mov ecx,[ebx-8]`@0x49686f/0x496876); nItemID offset 8 (4B int32, `mov eax,[ebp-11h]`@0x496683); sBuyCharacterName (a.k.a. `sSendCharacterName` per `CUIReceiveGift::SetValues`@0x781620 param name) offset 12 (13B fixed null-terminated char buffer, strlen/memcpy@0x496652-0x49667a); sText offset 25 (73B fixed null-terminated char buffer, strlen/memcpy@0x496609-0x496638). 8+4+13+73=98, byte-exact match to the named struct |
| LOAD_GIFT_FAILED | OnCashItemResLoadGiftFailed | failure | n-a | 0x32 | 0x3A | 0x46 | 0x4E | 0x51 | 0x53 | 0x5B | 0x51 | mode(disp); reason:Decode1@0x496967 |
| LOAD_WISHLIST | OnCashItemResLoadWishDone | scalar | 0x2C | 0x33 | 0x3B | 0x47 | 0x4F | 0x52 | 0x54 | 0x5C | 0x52 | existing |
| LOAD_WISH_FAILED | OnCashItemResLoadWishFailed | failure | 0x2D | 0x34 | 0x3C | 0x48 | 0x50 | 0x53 | 0x55 | 0x5D | 0x53 | mode(disp); reason:Decode1@0x496997 |
| UPDATE_WISHLIST | OnCashItemResSetWishDone | scalar | 0x32 | 0x39 | 0x41 | 0x4D | 0x55 | 0x58 | 0x5A | 0x62 | 0x56 | existing |
| SET_WISH_FAILED | OnCashItemResSetWishFailed | failure | 0x33 | 0x3A | 0x42 | 0x4E | 0x56 | 0x59 | 0x5B | 0x63 | 0x57 | mode(disp); reason:Decode1@0x4969ce |
| PURCHASE_SUCCESS | OnCashItemResBuyDone | item-blob | 0x34 | 0x3B | 0x43 | 0x4F | 0x57 | 0x5A | 0x5C | 0x64 | 0x58 | existing |
| BUY_FAILED | OnCashItemResBuyFailed | failure | 0x35 | 0x3C | 0x44 | 0x50 | 0x58 | 0x5B | 0x5D | 0x65 | 0x59 | mode(disp); reason:Decode1@0x496a24; if reason∈{29,30}: goodsSN:Decode4@0x496a3d; if reason==68: extra:Decode1@0x496a97 (conditional — see report) |
| USE_COUPON_SUCCESS | OnCashItemResUseCouponDone | item-blob | 0x36 | 0x3D | 0x45 | 0x51 | 0x59 | 0x5C | 0x5E | 0x66 | 0x5A | mode(disp); itemCount:Decode1@0x4986d3 (byte); items:DecodeBuffer(55×itemCount)@0x498707 (LIST of GW_CashItemInfo, via CashInventoryItem-equiv 55-byte blob); maplePoint:Decode4@0x498829 (int32); uliCount:Decode4@0x498832 (int32); uliList:DecodeBuffer(8×uliCount)@0x49886d — COUNT-prefixed LIST of a packed `_ULARGE_INTEGER` record, 8 bytes/entry, bit layout confirmed by disasm of the read-loop @0x4988a0-0x4989ae (base pointer = aniNew.a+2): quantity offset 0 (u16, `movzx ecx, word ptr [ebx-2]`@0x4988a3, formatted as the `%d` count in the notice text); slotPos offset 2 (u16, `movzx ebp, word ptr [ebx]`@0x4988a7, passed as `nPos` to `CCSWnd_Inventory::SetSelectedNo`@0x498908); itemId offset 4 (i32, `mov edi,[ebx+2]`@0x4988a0, full 32-bit — itemId/1000000 derives the inventory-tab category passed to `CCtrlTab::SetTab`@0x4988e8, and itemId itself is passed to `CItemInfo::GetItemName`@0x498919). 2+2+4=8, stride confirmed by `add ebx,8`@0x4989ae; meso:Decode4@0x4989cb (int32) |
| GIFT_COUPON_SUCCESS | OnCashItemResGiftCouponDone | item-blob | 0x38 | 0x3F | 0x47 | 0x53 | 0x5B | 0x5E | 0x60 | 0x68 | 0x5C | mode(disp); recipientName:DecodeStr@0x498e42; itemCount:Decode1@0x498e54 (byte); items:DecodeBuffer(55×itemCount)@0x498eee (LIST of GW_CashItemInfo, 55-byte blob, same helper as USE_COUPON_SUCCESS); maplePoint:Decode4@0x498fc0 (int32) |
| USE_COUPON_FAILED | OnCashItemResUseCouponFailed | failure | 0x39 | 0x40 | 0x48 | 0x54 | 0x5C | 0x5F | 0x61 | 0x69 | 0x5D | mode(disp); reason:Decode1@0x496f9f |
| GIFT_SUCCESS | OnCashItemResGiftDone | item-blob (resolves 0x4D TODO) | 0x3B | 0x42 | 0x4A | 0x56 | 0x5E | 0x61 | 0x63 | 0x6B | 0x5F | mode(disp); recipientName:DecodeStr@0x497084; itemId:Decode4@0x49709a (int32, name-lookup key); quantity:Decode2@0x4970a3 (uint16); nxCashSpent:Decode4@0x4970b7 (int32) — TRUE SHAPE: NO item-blob at all (contra the shape column's "item-blob" label); pure scalar body — this resolves the legacy 0x4D gift TODO |
| GIFT_FAILED | OnCashItemResGiftFailed | failure | 0x3C | 0x43 | 0x4B | 0x57 | 0x5F | 0x62 | 0x64 | 0x6C | 0xA3 | mode(disp); reason:Decode1@0x497224 |
| INVENTORY_CAPACITY_INCREASE_SUCCESS | OnCashItemResIncSlotCountDone | counter | 0x3D | 0x44 | 0x4C | 0x58 | 0x60 | 0x63 | 0x65 | 0x6D | 0x61 | existing |
| INVENTORY_CAPACITY_INCREASE_FAILED | OnCashItemResIncSlotCountFailed | failure | 0x3E | 0x45 | 0x4D | 0x59 | 0x61 | 0x64 | 0x66 | 0x6E | 0x62 | existing |
| INC_TRUNK_COUNT_SUCCESS | OnCashItemResIncTrunkCountDone | counter | n-a | 0x46 | 0x4E | 0x5A | 0x62 | 0x65 | 0x67 | 0x6F | 0x63 | mode(disp); trunkCount:Decode2@0x494edf (uint16, new absolute m_nTrunkCount — no separate type byte) |
| INC_TRUNK_COUNT_FAILED | OnCashItemResIncTrunkCountFailed | failure | n-a | 0x47 | 0x4F | 0x5B | 0x63 | 0x66 | 0x68 | 0x70 | 0x64 | mode(disp); reason:Decode1@0x4973e4 |
| INC_CHARACTER_SLOT_COUNT_SUCCESS | OnCashItemResIncCharacterSlotCountDone | counter | 0x3F | 0x48 | 0x50 | 0x5C | 0x64 | 0x67 | 0x69 | 0x71 | 0x65 | mode(disp); slotCount:Decode2@0x494f7f (uint16, new absolute m_nCharacterSlotCount — no separate type byte) |
| INC_CHARACTER_SLOT_COUNT_FAILED | OnCashItemResIncCharacterSlotCountFailed | failure | 0x40 | 0x49 | 0x51 | 0x5D | 0x65 | 0x68 | 0x6A | 0x72 | 0x66 | mode(disp); reason:Decode1@0x497424 |
| INC_BUY_CHARACTER_COUNT_SUCCESS | OnCashItemResIncBuyCharacterCountDone | counter | n-a | n-a | 0x52 | n-a | n-a | n-a | n-a | 0x73 | 0x67 | mode(disp); buyCharacterCount:Decode2@0x495007 (uint16, new absolute m_nBuyCharacterCount — no separate type byte) |
| INC_BUY_CHARACTER_COUNT_FAILED | OnCashItemResIncBuyCharacterCountFailed | failure | n-a | n-a | 0x53 | n-a | n-a | n-a | n-a | 0x74 | 0x68 | mode(disp); reason:Decode1@0x497464 |
| ENABLE_EQUIP_SLOT_EXT_SUCCESS | OnCashItemResEnableEquipSlotExtDone | counter | n-a | n-a | n-a | 0x5E | 0x66 | 0x69 | 0x6B | 0x75 | 0x69 | mode(disp); slotIndex:Decode2@0x4974c4 (uint16, indexes aEquipExtExpire[v3]/aEquipped2[v3+59+13]); days:Decode2@0x4974d2 (uint16, passed to Util::FTAddDay as day-count) — two uint16 fields, not byte+short |
| ENABLE_EQUIP_SLOT_EXT_FAILED | OnCashItemResEnableEquipSlotExtFailed | failure | n-a | n-a | n-a | 0x5F | 0x67 | 0x6A | 0x6C | 0x76 | 0x6A | mode(disp); reason:Decode1@0x497704 |
| CASH_ITEM_MOVED_TO_INVENTORY | OnCashItemResMoveLtoSDone | item-blob | 0x41 | 0x4A | n-a | 0x60 | 0x68 | 0x6B | 0x6D | 0x77 | 0x6B | existing |
| MOVE_L_TO_S_FAILED | OnCashItemResMoveLtoSFailed | failure | 0x42 | 0x4B | n-a | 0x61 | 0x69 | 0x6C | 0x6E | 0x78 | 0x6C | mode(disp); reason:Decode1@0x49773e |
| CASH_ITEM_MOVED_TO_CASH_INVENTORY | OnCashItemResMoveStoLDone | item-blob | 0x43 | 0x4C | 0x54 | 0x62 | 0x6A | 0x6D | 0x6F | 0x79 | 0x6D | existing |
| MOVE_S_TO_L_FAILED | OnCashItemResMoveStoLFailed | failure | 0x44 | 0x4D | 0x55 | 0x63 | 0x6B | 0x6E | 0x70 | 0x7A | 0x6E | mode(disp); reason:Decode1@0x497939 |
| DESTROY_SUCCESS | OnCashItemResDestroyDone | scalar | 0x45 | 0x4E | 0x56 | 0x64 | 0x6C | 0x6F | 0x71 | 0x7B | 0x6F | mode(disp); sn:DecodeBuffer(8)@0x495269 (int64 LARGE_INTEGER cash-item SN, matched against m_aCashItemInfo[i].liSN) |
| DESTROY_FAILED | OnCashItemResDestroyFailed | failure | 0x46 | 0x4F | 0x57 | 0x65 | 0x6D | 0x70 | 0x72 | 0x7C | 0x70 | mode(disp); reason:Decode1@0x49795e |
| EXPIRE_DONE | OnCashItemResExpireDone | scalar | 0x47 | 0x50 | 0x58 | 0x66 | 0x6E | 0x71 | 0x73 | 0x7D | 0x71 | mode(disp); sn:DecodeBuffer(8)@0x497797 (int64 LARGE_INTEGER cash-item SN, matched against m_aCashItemInfo[i].liSN) |
| REBATE_SUCCESS | OnCashItemResRebateDone | item-blob | 0x5C | 0x65 | 0x6F | 0x7D | 0x85 | 0x88 | 0x8A | 0x96 | 0x88 | mode(disp); sn:DecodeBuffer(8)@0x4979b6 (int64 LARGE_INTEGER, matched against existing m_aCashItemInfo[i].liSN — same pattern as DESTROY_SUCCESS/EXPIRE_DONE); amount:Decode4@0x4979bd (int32) — NO item-blob (contra the shape column's "item-blob" label) |
| REBATE_FAILED | OnCashItemResRebateFailed | failure | 0x5D | 0x66 | 0x70 | 0x7E | 0x86 | 0x89 | 0x8B | 0x97 | 0x89 | mode(disp); reason:Decode1@0x497ade |
| COUPLE_SUCCESS | OnCashItemResCoupleDone | item-blob | 0x5E | 0x67 | 0x71 | 0x7F | 0x87 | 0x8A | 0x8C | 0x98 | 0x8A | mode(disp); item:DecodeBuffer(0x37=55)@0x497bad (single GW_CashItemInfo blob, appended to m_aCashItemInfo); recipientName:DecodeStr@0x497bd1; itemId:Decode4@0x497be7 (int32); quantity:Decode2@0x497bfa (uint16) |
| COUPLE_FAILED | OnCashItemResCoupleFailed | failure | 0x5F | 0x68 | 0x72 | 0x80 | 0x88 | 0x8B | 0x8D | 0x99 | 0x8B | mode(disp); reason:Decode1@0x497d32; if reason∈{29,30}: goodsSN:Decode4@0x497d4b (conditional — see report) |
| BUY_PACKAGE_SUCCESS | OnCashItemResBuyPackageDone | item-blob | 0x60 | 0x69 | 0x73 | 0x81 | 0x89 | 0x8C | 0x8E | 0x9A | 0x8C | mode(disp); itemCount:Decode1@0x496b94 (byte); items:DecodeBuffer(55×itemCount)@0x496bc1 (LIST of GW_CashItemInfo, 55-byte blob); trailingCount:Decode2@0x496c19 (uint16, branches notice-text format) |
| BUY_PACKAGE_FAILED | OnCashItemResBuyPackageFailed | failure | 0x61 | 0x6A | 0x74 | 0x82 | 0x8A | 0x8D | 0x8F | 0x9B | 0x8D | mode(disp); reason:Decode1@0x496d52; if reason∈{29,30}: goodsSN:Decode4@0x496d6b (conditional — see report) |
| GIFT_PACKAGE_SUCCESS | OnCashItemResGiftPackageDone | item-blob | 0x62 | 0x6B | 0x75 | 0x83 | 0x8B | 0x8E | 0x90 | 0x9C | 0x8E | mode(disp); recipientName:DecodeStr@0x496df0; packageId:Decode4@0x496e06 (int32, CItemInfo::GetSpecialName key); unused1:Decode2@0x496e08 (uint16, read+discarded); unused2:Decode2@0x496e0f (uint16, read+discarded); nxCashSpent:Decode4@0x496e27 (int32) — NO item-blob (contra the shape column's "item-blob" label) |
| GIFT_PACKAGE_FAILED | OnCashItemResGiftPackageFailed | failure | 0x63 | 0x6C | 0x76 | 0x84 | 0x8C | 0x8F | 0x91 | 0x9D | 0x8F | mode(disp); reason:Decode1@0x496f32; if reason∈{29,30}: goodsSN:Decode4@0x496f4b (conditional — see report) |
| BUY_NORMAL_SUCCESS | OnCashItemResBuyNormalDone | item-blob | 0x64 | 0x6D | 0x77 | 0x85 | 0x8D | 0x90 | 0x92 | 0x9E | 0x90 | mode(disp); count:Decode4@0x495349 (int32); list:DecodeBuffer(8×count)@0x495375 — COUNT-prefixed LIST of the SAME packed `_ULARGE_INTEGER` record as USE_COUPON_SUCCESS's second list (base pointer = aniNew.a+2, disasm @0x495391-0x4953d7): quantity offset 0 (u16 — present on the wire per the shared record shape, but this handler never dereferences it); slotPos offset 2 (u16, `movzx ebp, word ptr [esi]`@0x495394, passed as `nPos` to `CCSWnd_Inventory::SetSelectedNo`@0x4953d2); itemId offset 4 (i32, `mov ecx,[esi+2]`@0x495391, full 32-bit — itemId/1000000 derives the inventory-tab category passed to `CCtrlTab::SetTab`@0x4953b6). Stride confirmed by `add esi,8`@0x4953d7 — NO GW_CashItemInfo item-blob (contra the shape column's "item-blob" label) |
| BUY_NORMAL_FAILED | OnCashItemResBuyNormalFailed | failure | 0x65 | 0x6E | 0x78 | 0x86 | 0x8E | 0x91 | 0x93 | 0x9F | 0x91 | mode(disp); reason:Decode1@0x497b12; if reason∈{29,30}: goodsSN:Decode4@0x497b2b (conditional — see report) |
| FRIENDSHIP_SUCCESS | OnCashItemResFriendShipDone | item-blob | 0x68 | 0x71 | 0x7B | 0x89 | 0x91 | 0x94 | 0x96 | 0xA2 | 0x94 | mode(disp); item:DecodeBuffer(0x37=55)@0x497dcd (single GW_CashItemInfo blob, appended to m_aCashItemInfo); recipientName:DecodeStr@0x497df1; itemId:Decode4@0x497e07 (int32); quantity:Decode2@0x497e1a (uint16) — identical shape to COUPLE_SUCCESS |
| FRIENDSHIP_FAILED | OnCashItemResFriendShipFailed | failure | 0x69 | 0x72 | 0x7C | 0x8A | 0x92 | 0x95 | 0x97 | 0xA3 | 0x95 | mode(disp); reason:Decode1@0x497f52; if reason∈{29,30}: goodsSN:Decode4@0x497f6b (conditional — see report) |
| FREE_CASH_ITEM_DONE | OnCashItemResFreeCashItemDone | scalar | n-a | n-a | n-a | n-a | n-a | n-a | 0x9E | 0xAA | 0xA1 | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x494897 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| PURCHASE_RECORD | OnCashItemResPurchaseRecord | scalar | n-a | n-a | 0x84 | 0x92 | 0x9A | 0x9D | 0xA3 | 0xAF | 0x9D | mode(disp); goodsSN:Decode4@0x495b60 (int32, used as ZMap key or 0-check); purchased:Decode1@0x495b74 (byte, compared !=0 → bool) |
| PURCHASE_RECORD_FAILED | OnCashItemResPurchaseRecordFailed | failure | n-a | n-a | 0x85 | 0x93 | 0x9B | 0x9E | 0xA4 | 0xB0 | 0x9E | mode(disp); unusedByte:Decode1@0x494074 (value discarded, no further reads — see report) |
| NAME_CHANGE_BUY_DONE | OnCashItemNameChangeResBuyDone | scalar | 0x6C | 0x7D | 0x88 | 0x96 | 0x9E | 0xA1 | 0xA7 | 0xB3 | 0xA5 | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x495639 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| TRANSFER_WORLD_SUCCESS | OnCashItemResTransferWorldDone | scalar | 0x6E | 0x7B | 0x8A | 0x98 | 0xA0 | 0xA3 | 0xA9 | 0xB5 | 0xAE | mode(disp); cashItemInfo:DecodeBuffer(0x37=55)@0x495749 (GW_CashItemInfo blob) — decompile shows an item-blob shape, NOT a small scalar; shape column left as-is per task scope, flagged here |
| TRANSFER_WORLD_FAILED | OnCashItemResTransferWorldFailed | failure | 0x6F | 0x7E | 0x8B | 0x99 | 0xA1 | 0xA4 | 0xAA | 0xB6 | 0xAF | mode(disp); reason:Decode1@0x49837e |
| GACHAPON_OPEN_SUCCESS | OnCashItemResCashGachaponOpenDone | item-blob | n-a | n-a | n-a | n-a | n-a | 0xA5 | 0xAB | 0xB7 | n-a | mode(disp); sn:DecodeBuffer(8)@0x494adc (int64 LARGE_INTEGER, matched against existing m_aCashItemInfo[i].liSN); remain:Decode4@0x494aea (int32, new qty — 0 removes locker entry); isCashItem:Decode1@0x494af6 (byte flag); CONDITIONAL if isCashItem≠0: newItem:DecodeBuffer(0x37=55)@0x494b5d (single GW_CashItemInfo blob, appended to m_aCashItemInfo); resultCode:Decode4@0x494b6b (int32); resultParam2:Decode1@0x494b6d (byte) |
| GACHAPON_OPEN_FAILED | OnCashItemResCashGachaponOpenFailed | failure | n-a | n-a | n-a | n-a | n-a | 0xA6 | 0xAC | 0xB8 | n-a | mode(disp); reason:Decode1@0x4962c4 |
| GACHAPON_COPY_SUCCESS | OnCashItemResCashGachaponCopyDone | item-blob | n-a | n-a | n-a | n-a | n-a | 0xA7 | 0xAD | 0xB9 | n-a | mode(disp); flag1:Decode1@0x494bab (byte); flag2:Decode1@0x494bb5 (byte); unused1:Decode4@0x494bb8 (int32, discarded); unused2:Decode4@0x494bbf (int32, discarded); lostItemId:Decode4@0x494bcd (int32); lostNumber:Decode4@0x494bd6 (int32); CONDITIONAL if flag1≠0 AND flag2≠0: item:DecodeBuffer(0x37=55)@0x494bf4 (single GW_CashItemInfo blob) |
| GACHAPON_COPY_FAILED | OnCashItemResCashGachaponCopyFailed | failure | n-a | n-a | n-a | n-a | n-a | 0xA8 | 0xAE | 0xBA | n-a | mode(disp); reason:Decode1@0x4962fe |
| CHANGE_MAPLE_POINT_SUCCESS | OnCashItemResChangeMaplePointDone | scalar | n-a | n-a | n-a | n-a | n-a | 0xA9 | 0xAF | 0xBB | n-a | mode(disp); sn:DecodeBuffer(8)@0x498559 (int64 LARGE_INTEGER cash-item SN); count:Decode4@0x4985cc (int32, formatted into notice string) |
| CHANGE_MAPLE_POINT_FAILED | OnCashItemResChangeMaplePointFailed | failure | n-a | n-a | n-a | n-a | n-a | 0xAA | 0xB0 | 0xBC | n-a | mode(disp); NO Decode1/4 call in handler — zero packet reads beyond mode byte (see report) |

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

## Per-version wire divergences (for Wave 1/3 gating)

Grouped by theme. All figures pulled directly from the Task 0.4/0.5 harvest
reports in `.superpowers/sdd/` — cited inline, not invented. Reason-code
values are DECIMAL throughout (matching the harvest reports' convention).

### 1. Conditional-goodsSN failure arms — per-version reason-code trigger sets

Six arms (`BUY_FAILED`, `COUPLE_FAILED`, `BUY_PACKAGE_FAILED`,
`GIFT_PACKAGE_FAILED`, `BUY_NORMAL_FAILED`, `FRIENDSHIP_FAILED`) share an
identical wire *shape*: `reason:Decode1`, then a conditional `goodsSN:Decode4`
that fires only when `reason` is one of two specific values. The wire shape
(does a 4-byte goodsSN follow the reason byte) is unchanged across every
version audited; only the *trigger values* of the already-present `reason`
byte differ per version — a DOM-25 config-resolved, version-specific value,
never to be hardcoded from any single version's codec:

| version | trigger set | source |
|---|---|---|
| v95 | {29, 30} | `arm-catalog.md` v95 fields column (pre-existing) |
| v83 | {191, 192} | `task-0.4-v83-modes.md` divergence notes (BUY_FAILED, COUPLE_FAILED, BUY_PACKAGE_FAILED, GIFT_PACKAGE_FAILED, BUY_NORMAL_FAILED, FRIENDSHIP_FAILED rows) |
| v87 | {206, 207} | `task-0.4-v87-modes.md` "Cross-arm divergence note" — gate is `reason==207 (0xCF)` or `reason==206 (0xCE)` |
| v84 | {200, 201} | `task-0.4-v84-modes.md` — directly RE'd for BUY_FAILED/COUPLE_FAILED; the other 4 (BUY_PACKAGE_FAILED, GIFT_PACKAGE_FAILED, BUY_NORMAL_FAILED, FRIENDSHIP_FAILED) are presumed-by-pattern, not independently re-decompiled (flagged as such in that report) |
| v72 | {166, 167} | `task-0.5-v72-modes.md` per-arm table (all 6 arms individually cited) |
| v79 | {180, 181} | `task-0.5-v79-modes.md` "Cross-arm pattern" section (all 6 arms) |
| jms | {205, 206} | `task-0.4-jms-modes.md` per-arm table (BUY_FAILED, COUPLE_FAILED, BUY_PACKAGE_FAILED, GIFT_PACKAGE_FAILED, BUY_NORMAL_FAILED, FRIENDSHIP_FAILED all cite `reason∈{205,206}`) |
| v48 | {133, 134} | `task-0.5-v48-modes.md` per-arm table (all 6 arms cite `reason∈{133,134}`) |
| v61 | {149, 150} (+ separate `{127,128,129}` gate on a *different* conditional — the `SendTransferFieldPacket` redirect, not the goodsSN read; `BUY_NORMAL_FAILED` uses `{127,128}` only, missing 129 — a v61-internal inconsistency) | `task-0.5-v61-modes.md` per-arm divergence list |

`BUY_FAILED`'s v95 entry additionally documents a THIRD conditional —
`if reason==68: extra:Decode1` — that is **v95-only**. Every other version
audited (v83, v84, v87, v72, v79, v61, v48, jms) has at most the two-branch
shape (plain failure notice, or goodsSN-bearing branch); none has this
third `reason==68` read. Confirmed absent explicitly in: v83 ("No separate
'extra:Decode1 if reason==68' branch exists in v83"), v87 ("v87 has no
third conditional... that branch is absent in v87's body"), v72 ("v72 also
has NO third conditional"), v79 ("v79 has NO third 'extra:Decode1 if
reason==68' branch"), v61 (no third-conditional call documented), v48 ("No
second 'extra:Decode1 if reason==68' field exists in v48"), jms ("no second
`reason==68` conditional at all").

### 2. jms omits v95's trailing `nxCashSpent:Decode4` on two SUCCESS arms

- **GIFT_SUCCESS**: jms's resolved handler (`sub_48D3CE`, renamed
  `OnCashItemResGiftDone`, mode `0x5F`) reads `recipientName:DecodeStr`,
  `itemId:Decode4`, `quantity:Decode2` — and stops. v95 reads a fourth field,
  `nxCashSpent:Decode4`, that jms does not. (`task-0.4-jms-modes.md`,
  "Unnamed-handler resolution" + per-arm table.)
- **GIFT_PACKAGE_SUCCESS**: jms (mode `0x8E`) reads `recipientName:DecodeStr`,
  `packageId:Decode4`, `unused1:Decode2`, `unused2:Decode2` — and stops,
  missing v95's trailing `nxCashSpent:Decode4`. (`task-0.4-jms-modes.md`
  per-arm table, GIFT_PACKAGE_SUCCESS row.)

### 3. Legacy existing-9-arm shape diffs (v48/v61/v72/v79)

- **LOAD_INVENTORY_SUCCESS trailing Decode2 counters (v61/v72/v79)**: all
  three versions carry undocumented trailing uint16 field(s) after the
  item-blob list that the terse "existing" catalog note doesn't itemize:
  - v61: TWO trailing `Decode2` fields (`this[291]`, `this[292]`) after the
    item list (`task-0.5-v61-modes.md`, LOAD_INVENTORY_SUCCESS divergence note).
  - v72: TWO trailing `Decode2` fields (`trailingA@0x4711ce`,
    `trailingB@0x4711e2`) after the item-blob (`task-0.5-v72-modes.md` per-arm
    table).
  - v79: TWO extra trailing fields at `this[291]`/`this[292]`, same shape as
    v61/v72 (`task-0.5-v79-modes.md` per-arm table, "likely trunk-count/
    character-slot-count echoes").
- **v48 LOAD_INVENTORY_SUCCESS folds the LOAD_GIFT_SUCCESS payload in**: v48's
  case 42 body reads `cashCount:Decode2 → cashItems:DecodeBuffer(55×cashCount)
  → giftCount:Decode2 → giftList:DecodeBuffer(98×giftCount) →
  trailingField:Decode2` — the gift-list block (98-byte records, byte-identical
  width to v95's `GW_GiftList`) is delivered as a trailing block of the
  locker-load response itself, not as a separately-dispatched
  `LOAD_GIFT_SUCCESS` case. Confirmed by two independent sessions:
  `task-0.5-v48-modes.md` (initial hypothesis, full case enumeration proving
  no standalone gift-load case exists) and `task-0.5-v48-giftfold.md` (full
  decompile + re-enumeration + `func_query` name-space sweep, all confirming
  the fold). `LOAD_GIFT_FAILED` remains a genuine absence in v48 (no
  corresponding failure branch in case 42's body, no handler function anywhere
  in the binary) — this is NOT folded, it simply doesn't exist.
- **CASH_ITEM_MOVED_TO_INVENTORY decodes a full `GW_ItemSlotBase` in v48/v61
  (and v79)**: rather than the fixed 55-byte `GW_CashItemInfo` blob every
  other item-blob arm in this function uses, this arm reads
  `slotField:Decode2` (v48) / `slotIndex:Decode2` (v61/v79) followed by a
  `GW_ItemSlotBase::Decode()` call — a polymorphic item-slot decoder — because
  moving a cash-locker entry into the character inventory produces a real
  inventory item, not a cash-item record. Cited: `task-0.5-v48-modes.md`
  (finding #4), `task-0.5-v61-modes.md` (per-arm divergence list), and
  `task-0.5-v79-modes.md` (per-arm table, "item:GW_ItemSlotBase::Decode(...)").
  **v72 does not have this arm at all** — `CASH_ITEM_MOVED_TO_INVENTORY` and
  `MOVE_L_TO_S_FAILED` are both `n-a` in v72 (proven by full switch
  enumeration + `func_query name_regex="MoveLtoS"` returning zero hits
  anywhere in the binary — only the opposite `MoveStoL` direction exists in
  this build). This is an existing-9 arm going `n-a` in a legacy version, per
  the task brief's explicit call-out.
- **INC_BUY_CHARACTER_COUNT_SUCCESS reads `slotIndex:Decode2` +
  `GW_ItemSlotBase::Decode` in v72**: v72's arm at this catalog key is a
  MATERIALLY different wire contract than v95's — v95 is a bare
  `buyCharacterCount:Decode2` absolute-counter update; v72 instead reads a
  slot index plus a full item-slot struct, matches it against the existing
  locker list by SN, and removes it ("buy character slot by consuming a
  specific locker item"). `task-0.5-v72-modes.md` flags this explicitly as a
  "MAJOR divergence... NOT the same shape family" and recommends design
  review before gating this arm `MajorAtLeast`. (Note: this key is `n-a` in
  v48/v61/v79/v83/v84/v87 — v72 is the only legacy version where it's
  present, and jms also has it present at `0x67`/103 with the v95-style
  bare-counter shape per `task-0.4-jms-modes.md`, an unresolved shape
  question across v72 vs jms/v95 that the modeling phase should reconcile.)
- **INVENTORY_CAPACITY_INCREASE_SUCCESS reads a `(tabType, newCount)` pair in
  v48**: `tabType:Decode1` (validated 1–5) then `newCount:Decode2` (validated
  `>previous` and `≤80`) — not a single absolute-count field. Also seen with
  the same two-field shape in v61 (`type:Decode1` + `count:Decode2`,
  validated ≤80) and v72 (`type:Decode1` + `newCount:Decode2`, validated ≤96,
  with the note that its *sibling* new counters INC_TRUNK/INC_CHARACTER_SLOT
  do NOT carry the leading type byte — bare `Decode2` only). Cited:
  `task-0.5-v48-modes.md` finding #5, `task-0.5-v61-modes.md` divergence list,
  `task-0.5-v72-modes.md` per-arm table.
- **PURCHASE_SUCCESS / CASH_ITEM_MOVED_TO_CASH_INVENTORY read a single fixed
  blob with no count prefix in v72**: both existing arms are a bare
  `DecodeBuffer(55)` (single `GW_CashItemInfo`, no leading count field) in
  v72 — flagged for scrutiny against the current Go codec's list-oriented
  assumption when the legacy codec is wired (`task-0.5-v72-modes.md`).

### 4. v84 — 6 arms present that v83 lacks; otherwise uniform +3

v84's `CCashShop::OnCashItemResult` (`0x47c291`, RE'd this pass — not
derived) has **54 present-case arms**, 6 more than v83's 48. The 48 arms
shared with v83 all obey a uniform `mode = v83_mode + 3` offset exactly,
including matching gap positions between consecutive present arms
(`task-0.4-v84-modes.md`, "cross-checked value-by-value across the full
ordered case list, not just spot-checked"). The 6 arms present in v84 but
`n-a` in v83 are genuinely new to this build, with no `+3`-derived
counterpart, each independently field-shape-verified against the v95
handler body (not just case-position inference):

- `GACHAPON_OPEN_SUCCESS` = 0xA5 (165)
- `GACHAPON_OPEN_FAILED` = 0xA6 (166)
- `GACHAPON_COPY_SUCCESS` = 0xA7 (167)
- `GACHAPON_COPY_FAILED` = 0xA8 (168)
- `CHANGE_MAPLE_POINT_SUCCESS` = 0xA9 (169)
- `CHANGE_MAPLE_POINT_FAILED` = 0xAA (170)

`INC_BUY_CHARACTER_COUNT_SUCCESS`/`FAILED` and `FREE_CASH_ITEM_DONE` remain
`n-a` in v84, same as v83 — not backported. A yaml/template derivation
purely as "v83 value + 3" would have silently produced `n-a` for these 6
arms; this pass corrects that by RE'ing the real v84 dispatcher directly
(previously it was template-derived/assumed-uniform-+3 per the yaml's old
header comment — see the yaml's updated header for the corrected status).

### 5. jms-specific notes

- `GIFT_FAILED` sits at `0xA3` (163) in jms — not adjacent to `GIFT_SUCCESS`
  (`0x5F`/95), unlike every other version where SUCCESS/FAILED pairs are
  positionally adjacent. Confirmed HIGH confidence (not merely
  best-available) via direct structural/call-graph comparison against v95's
  real handler body in `task-0.4-jms-resolve.md` §2 — both read a single
  `reason:Decode1` and share the identical distinguishing
  `SendGiftsPacket`-vs-`NoticeFailReason` branch-on-gifts-flag control flow.
- `CHANGE_MAPLE_POINT_FAILED` is `n-a` in jms — but with an important
  caveat documented in `task-0.4-jms-resolve.md` §1: the request-side
  feature (`SendChangeMaplePoint`) IS reachable and wired in this jms v185
  build (double-click item 5222002 in the Cash Locker → `COutPacket(0xA7)`).
  **Updated by task-207:** an exhaustive `push 0A7h` scan of the jms_v185
  binary found `0xA7` has two real senders, not one:
  `CCashShop::SendChangeMaplePoint` (`0x4851be`, `COutPacket` construction at
  `0x4851d3`) and `CUICashItemGachapon::OnButtonClicked` (`0xa6e309`,
  `COutPacket` construction at `0xa6e3a4`) — both encode an 8-byte serial and
  are indistinguishable on the wire. GMS separates these into two distinct
  opcodes on every version that has both (e.g. v87 169 vs 171); jms_v185
  collapses them onto the single opcode 167. The registry now carries both
  senders as separate rows at `direction: serverbound, opcode: 167` in
  `docs/packets/registry/jms_v185.yaml`.
  Four zero-decode candidate handlers in the switch are structurally
  identical to v95's "zero packet reads, canned notice" shape for this arm
  and cannot be disambiguated further without decoding jms's Japanese
  StringPool resource text (out of IDA-MCP's tool surface) — but none
  carries any maple-point-specific side effect, so the arm itself has no
  dispatcher case wired to it. This is a different *kind* of n-a than the
  gachapon/CHANGE_MAPLE_POINT_SUCCESS arms (verifiably absent from the whole
  binary) — here the client-send path exists but the result dispatcher has
  no corresponding case.

### 6. Free-cash-item / purchase-record are era-gated, not monotonic

`FREE_CASH_ITEM_DONE` is `n-a` in v48/v61/v72/v79/v83/v84, present in
v87 (`0x9E`), v95 (`0xAA`), and jms (`0xA1`) — i.e. it first appears at v87,
not at some earlier legacy version, and the v72 harvest separately confirms
a *same-named* free-cash-item notice (`OnNoticeFreeCashItem`) exists in v72
but as a wholly separate top-level packet op (289, via `CCashShop::OnPacket`)
— not a mode-byte sub-arm of `OnCashItemResult` at all, so it does not count
as a `present` mode for this catalog. `PURCHASE_RECORD`/`PURCHASE_RECORD_FAILED`
are `n-a` in v48/v61, present starting at v72 (`task-0.5-v48-modes.md`,
`task-0.5-v61-modes.md`, `task-0.5-v72-modes.md`) — the task brief's
"gachapon/world-transfer/maple-point/name-change/purchase-record LIKELY n-a
in legacy" prediction was only half right for v79: gachapon and maple-point
ARE n-a there, but `TRANSFER_WORLD_SUCCESS/FAILED`, `NAME_CHANGE_BUY_DONE`,
and `PURCHASE_RECORD/PURCHASE_RECORD_FAILED` are all PRESENT in v79
(`task-0.5-v79-modes.md`, "Design-prediction correction").

### 7. Cross-version reason-code renumbering is systemic, not per-arm noise

Beyond the six goodsSN-gated arms in §1, the same "reason-code enum differs
per version, wire shape does not" pattern recurs in the `SendTransferFieldPacket`
redirect gate used by many `*_FAILED` handlers: v83 gates on
`reason∈{162,163,164}`, v84 on `{171,172,173}` (also seen in v84's
`GACHAPON_OPEN_FAILED`), v48 on `{112,113,114}` (with the `{112,113}`-only
inconsistency in `BUY_NORMAL_FAILED`), v61 on `{127,128,129}` (with the `{127,128}`-only
inconsistency in `BUY_NORMAL_FAILED`), v79 on
`{154,155,156}` (with `BUY_NORMAL_FAILED` again the outlier at `{154,155}`
only). None of this changes the wire byte layout — it is exclusively a
version-specific reason-code lookup table concern for the codec
implementation (DOM-25), never a struct-shape concern.

## JMS-only arms (task-183 follow-up)

A code-review audit found 10 arms of the JMS v185 dispatcher
(`CCashShop::OnCashItemResult` @ `0x48b5a5`, raw `CInPacket::Decode1` switch —
re-verified 2026-07, IDA session `3c4bb8b1`, `MapleStory_dump_SCY.exe`) that the
original task-183 pass left unmodeled. They target `sub_XXXX` functions with NO
client symbol and NO GMS/legacy-canonical equivalent — absent from every GMS +
legacy switch (all fully enumerated and named in task-183 Waves 0/1). The four
bodyless canned-notice / no-op arms below are the "four zero-decode candidate
handlers … cannot be disambiguated further without decoding jms's Japanese
StringPool" flagged in §5 above: they are now modeled explicitly by their
verifiable behavior (each shows StringPool notice #N, or is a genuine no-op) —
the Japanese notice *text* is still not resolvable through IDA-MCP, but the
dispatcher case, the target address, and the wire shape (bodyless) all are.

**Every IDB name below is BEHAVIOR-DERIVED** (JMS DEVM build is stripped; no
PDB). Each name describes what the handler *verifiably does* at the cited
address; none is presented as the client's real symbol. The 9 non-shared subs
were renamed + `idb_save`d in session `3c4bb8b1`; mode 164 targets the shared
generic `nullsub_2` (left as-is — a genuine empty function reused elsewhere).

| mode | addr | behavior-derived IDB name | atlas KEY | wire (after mode byte) | reason/result treatment |
|---|---|---|---|---|---|
| 76  | 0x48ba24 | `CCashShop__OnCashItemResShowGiftResultNotice` | `GIFT_RESULT_NOTICE` | `reason:Decode1` | errors-resolved (notice arm) |
| 77  | 0x48ba3f | `CCashShop__OnCashItemResLoadReceivedGiftDone` | `LOAD_RECEIVED_GIFT_SUCCESS` | `flag:Decode1` + `count:Decode4` + N × `ReceivedGiftEntry` (176B) | plain flag |
| 96  | 0x48d4d4 | `CCashShop__OnCashItemResLimitGoodsStockChanged` | `LIMIT_GOODS_STOCK_CHANGED` | `result:Decode1` + (result∈{205,206}) `itemId:Decode4` | plain result (gates wire shape) |
| 146 | 0x48e6c9 | `CCashShop__OnCashItemResShowNotice1089` | `SHOW_NOTICE_1089` | *(bodyless)* | — |
| 147 | 0x48e6f7 | `CCashShop__OnCashItemResTransferWorldNoticeReason` | `TRANSFER_WORLD_NOTICE_REASON` | `reason:Decode1` | errors-resolved (notice arm) |
| 162 | 0x48c321 | `CCashShop__OnCashItemResRefreshLocker` | `REFRESH_LOCKER` | `item:DecodeBuffer(0x37=55)` (CashInventoryItem) | — |
| 164 | 0x48c370 | `nullsub_2` (shared; not renamed) | `CLIENT_NO_OP` | *(bodyless)* — client no-op | — |
| 166 | 0x48c26e | `CCashShop__OnCashItemResShowNotice1465` | `SHOW_NOTICE_1465` | *(bodyless)* | — |
| 167 | 0x48c373 | `CCashShop__OnCashItemResRefreshLockerOrNotice` | `REFRESH_LOCKER_OR_NOTICE` | `flag:Decode1` + `item:DecodeBuffer(0x37=55)` (CashInventoryItem) | plain flag |
| 168 | 0x48c413 | `CCashShop__OnCashItemResShowNotice1464` | `SHOW_NOTICE_1464` | *(bodyless)* | — |

Per-arm decompile evidence (all cite the JMS `3c4bb8b1` decompile):

- **76 `GIFT_RESULT_NOTICE`** (`0x48ba24`): reads one `Decode1` reason byte and
  passes it to the gift-result notice `sub_48F0F2` (`0x48f0f2`), which maps
  214/215/216 → StringPool 625/626/627 (else 624). Reason is a NoticeFailReason-
  class UI code → body resolves it from the writer `errors` table (sibling
  failure-arm pattern); struct field is a plain `errorCode byte`.
- **77 `LOAD_RECEIVED_GIFT_SUCCESS`** (`0x48ba3f`): `Decode1` flag (v26; ==0 →
  show StringPool notice 626 after the loop), `Decode4` count, then count ×
  `DecodeBuffer(0xB0=176)` records, then ACKs `COutPacket(0xF5)`. See the
  176-byte record layout below. Behavior-derived name: it loads the received-gift
  list into the locker gift UI.
- **96 `LIMIT_GOODS_STOCK_CHANGED`** (`0x48d4d4`): `Decode1` result (v4);
  **only when result==205 or 206** it reads `Decode4` itemId and calls
  `UpdateStock`/`ChangeLimitGoodsState`/`ChangePage`; then `NoticeFailReason(result)`;
  then if result∈{177,178,179} `SendTransferFieldPacket`. The result byte
  **controls the wire shape** (conditional itemId), so it is a **plain field, not
  config-resolved** — matching the family's treatment of protocol status codes
  (`GachaponOpenDone.resultCode`). Config-resolving a byte that gates a
  conditional read is unsafe (the resolved value must equal 205/206 to include
  the itemId).
- **146 `SHOW_NOTICE_1089`** (`0x48e6c9`): no `Decode*`; shows StringPool notice
  `0x1089` (4233) via `CUtilDlg::Notice`. Bodyless (mode byte only).
- **147 `TRANSFER_WORLD_NOTICE_REASON`** (`0x48e6f7`): `Decode1` reason (v3) →
  `NoticeFailReason(reason)`; then if reason∈{177,178} `SendTransferFieldPacket`.
  Wire is unconditional (`mode + reason`); the transfer is a client side-effect,
  not a wire read, so the reason is safely errors-resolved (notice-arm pattern);
  plain `errorCode byte` struct field.
- **162 `REFRESH_LOCKER`** (`0x48c321`): `DecodeBuffer(0x37=55)` a single
  `GW_CashItemInfo` into a fresh locker slot, then refreshes the locker window.
  No leading byte. Reuses the existing `CashInventoryItem` + `decodeCashInventoryItemSkipPadding`.
- **164 `CLIENT_NO_OP`** (`0x48c370`): dispatcher routes mode 164 to the shared
  `nullsub_2` — an empty function that reads nothing and does nothing. Genuine
  client no-op; modeled bodyless per INV-1.
- **166 `SHOW_NOTICE_1465`** (`0x48c26e`): no `Decode*`; StringPool notice
  `0x1465` (5221). Bodyless.
- **167 `REFRESH_LOCKER_OR_NOTICE`** (`0x48c373`): `Decode1` flag (v12),
  `DecodeBuffer(0x37=55)` a `GW_CashItemInfo`; then either refreshes the locker
  window (when the client-local `*(this+1236)` flag is set) or shows StringPool
  5219/5216 (selected by the flag byte). Flag is a binary selector → plain byte
  (like `PurchaseRecordDone.purchased`).
- **168 `SHOW_NOTICE_1464`** (`0x48c413`): no `Decode*`; StringPool notice
  `0x1464` (5220). Bodyless.

### JMS 176-byte received-gift record (`ReceivedGiftEntry`)

The mode-77 list element is one `DecodeBuffer(0xB0 = 176)` blob per entry — a
`GW_GiftList`-shaped record with **no named type** in the stripped DEVM IDB (it
is NOT the 98-byte `GiftListEntry`/`GW_GiftList` used by `LOAD_GIFT_SUCCESS`).
Byte offsets derived from the `sub_48BA3F` disassembly of the field reads
(`0x48ba9b`-`0x48bbca`):

| offset | field | size | disasm evidence |
|---|---|---|---|
| 0x00 | reserved head | 12 | not accessed by the handler |
| 0x0C | `itemId` (int32) | 4 | `mov edi,[eax+0Ch]`; arg1 to gift-display helpers `sub_869D96`/`sub_86A57C`, paired with `itemName` |
| 0x10 | `data1` (int32) | 4 | `mov ecx,[eax+10h]` (helper arg2; semantics unresolved) |
| 0x14 | `data2` (int32) | 4 | `mov ecx,[eax+14h]` (helper arg3) |
| 0x18 | `giftType` (int32) | 4 | `cmp [eax+18h],0`; ==0 → points/coupon MapleTip path, !=0 → gift-notice path |
| 0x1C | `text` (string) | 101 | `Assign(p+0x1C)` null-terminated within the 0x1C-0x80 region |
| 0x81 | `sender` (string) | 33 | `Assign(p+0x81)` null-terminated within the 0x81-0xA1 region |
| 0xA2 | `itemName` (string) | 14 | `Assign(p+0xA2)` null-terminated within the 0xA2-0xAF region |

12 + 4 + 4 + 4 + 4 + 101 + 33 + 14 = **176**. The head + string regions are
fixed-size (the strings are null-terminated within their regions), so `Encode`
always emits exactly 176 bytes. Fields at offsets the handler does not read
(the 12-byte head) are modeled as a reserved region rather than invented.

### Verification

All 10 arms are jms_v185-only cells. Each carries: a `#`-entry in
`run.go candidatesFromFName`; a discrete struct in
`libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`; a body func +
operation-key const in `shop_operation_body.go`; a `jms_v185` mode in
`cash_shop_operation.yaml` (n-a on all other versions → no key); the key in the
`template_jms_185_1.json` CashShopOperation writer `operations` map (via
`packet-audit operations`); a `#`-suffix export entry in `gms_jms_185.json`; a
generated audit report in `docs/packets/audits/jms_v185/Cash*.{json,md}`
(all ✅); a byte-fixture with a `packet-audit:verify` marker in
`shop_operation_result_jms_test.go`; and a pinned TIER1-FIXTURE evidence record.
The `CASHSHOP_OPERATION` op-row's `jms_v185` cell remains `verified` (worst-of
aggregation — honest only because all 10 new arms verify).
