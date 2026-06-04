# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x79e4d0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcTemplateId (this+51)` | ✅ |  |
| 1 | int16 | int16 `commodity count (loop bound; no max cap)` | ✅ |  |
| 2 | int32 | int32 `[item] itemId / TemplateId` | ✅ |  |
| 3 | int32 | int32 `[item] mesoPrice (nPrice)` | ✅ |  |
| 4 | byte | byte `[item] discountRate (nDiscountRate; GMS>=87, present in v87 @0x79e54e; absent in v83)` | ✅ |  |
| 5 | int32 | int32 `[item] token field (single int @0x79e558; IDA labels nTokenItemID; atlas TokenPrice fills this slot for v87 — v95 splits into tokenItemId+tokenPrice, v83 has none)` | ✅ |  |
| 6 | int32 | int32 `[item] period (nItemPeriod)` | ✅ |  |
| 7 | int32 | int32 `[item] levelLimit (v106)` | ✅ |  |
| 8 | int16 | int16 `[item] quantity (non-ammo branch; ammo categories 207/233 read DecodeBuffer(8) unitPrice double instead)` | ✅ |  |
| 9 | int64 | int16 `[item] slotMax (v108)` | ❌ | width mismatch |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

