# InventoryCapacityFailed (← `CCashShop::OnCashItemResult#InventoryCapacityFailed`)

- **IDA:** 0x497390
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6E INC_SLOT_COUNT_FAILED; op-byte consumed by dispatcher)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason arg)` | ✅ |  |

