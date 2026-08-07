# CashBuyNormalFailed (← `CCashShop::OnCashItemResult#BUY_NORMAL_FAILED`)

- **IDA:** 0x47b71c
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x8e BUY_NORMAL_FAILED; op-byte consumed by dispatcher before OnCashItemResBuyNormalFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

