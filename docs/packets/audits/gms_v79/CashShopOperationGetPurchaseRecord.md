# CashShopOperationGetPurchaseRecord (← `CCashShop::RequestCashPurchaseRecord`)

- **IDA:** 0x4667eb
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_get_purchase_record.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

