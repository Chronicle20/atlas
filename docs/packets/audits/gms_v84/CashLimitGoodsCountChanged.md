# CashLimitGoodsCountChanged (← `CCashShop::OnCashItemResult#LIMIT_GOODS_COUNT_CHANGED`)

- **IDA:** 0x47c1bc
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4a LIMIT_GOODS_COUNT_CHANGED; op-byte consumed by dispatcher before CCashShop::OnCashItemResLimitGoodsCountChanged)` | ✅ |  |
| 1 | int32 | int32 `itemId (int32)` | ✅ |  |
| 2 | int32 | int32 `sn (int32)` | ✅ |  |
| 3 | int32 | int32 `remainCount (int32, updates client-local CS_LIMITGOODS array remaining-count)` | ✅ |  |

