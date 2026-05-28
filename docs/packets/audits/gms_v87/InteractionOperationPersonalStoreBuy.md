# InteractionOperationPersonalStoreBuy (← `CPersonalShopDlg::BuyItem`)

- **IDA:** 0x74076b
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_buy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `index (a2)` | ✅ |  |
| 1 | int16 | int16 `quantity (v71)` | ✅ |  |
| 2 | int32 | int32 `itemCRC (CItemInfo::GetItemCRC) — PRESENT in v83 AND v87 AND v95. Unconditional fix holds.` | ✅ |  |

