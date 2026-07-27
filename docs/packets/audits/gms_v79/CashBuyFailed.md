# CashBuyFailed (← `CCashShop::OnCashItemResult#BUY_FAILED`)

- **IDA:** 0x472913
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x50 BUY_FAILED; op-byte consumed by dispatcher before OnCashItemResBuyFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

