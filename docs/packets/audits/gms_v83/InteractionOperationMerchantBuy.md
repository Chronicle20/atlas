# InteractionOperationMerchantBuy (← `CPersonalShopDlg::BuyItem#Merchant`)

- **IDA:** 0x6fd261
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_merchant_buy.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `index (arg0)` | ✅ |  |
| 1 | int16 | int16 `quantity (a2[0])` | ✅ |  |
| 2 | int32 | int32 `itemCRC (CItemInfo::GetItemCRC) — present in v83 AND v95` | ✅ |  |

