# CashLimitGoodsCountChanged (← `CCashShop::OnCashItemResult#LIMIT_GOODS_COUNT_CHANGED`)

- **IDA:** 0x472018
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x3F LIMIT_GOODS_COUNT_CHANGED; op-byte consumed by dispatcher before OnCashItemResLimitGoodsCountChanged)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |
| 2 | int32 | int32 `sn` | ✅ |  |
| 3 | int32 | int32 `remainCount` | ✅ |  |

