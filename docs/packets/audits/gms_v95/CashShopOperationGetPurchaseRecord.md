# CashShopOperationGetPurchaseRecord (← `CCashShop::RequestCashPurchaseRecord`)

- **IDA:** 0x4823c0
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_get_purchase_record.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nCommoditySN (serialNumber)` | ✅ |  |

