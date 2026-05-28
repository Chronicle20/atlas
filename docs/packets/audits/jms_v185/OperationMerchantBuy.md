# OperationMerchantBuy (← `CPersonalShopDlg::BuyItem#Merchant`)

- **IDA:** 0x762365
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_merchant_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nIdx (index). entrusted op-byte 0x1F consumed by dispatcher` | ✅ |  |
| 1 | int16 | int16 `Src (quantity)` | ✅ |  |
| 2 | int32 | int32 `ItemCRC (CItemInfo::GetItemCRC). PRESENT in JMS` | ✅ |  |

