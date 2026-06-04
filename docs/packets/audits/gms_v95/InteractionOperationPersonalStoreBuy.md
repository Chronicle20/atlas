# InteractionOperationPersonalStoreBuy (← `CPersonalShopDlg::BuyItem`)

- **IDA:** 0x69a7f0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `index (nIdx)` | ✅ |  |
| 1 | int16 | int16 `quantity (v32)` | ✅ |  |
| 2 | int32 | int32 `itemCRC (CItemInfo::GetItemCRC)` | ✅ |  |

