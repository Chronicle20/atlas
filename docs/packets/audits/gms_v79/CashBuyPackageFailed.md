# CashBuyPackageFailed (← `CCashShop::OnCashItemResult#BUY_PACKAGE_FAILED`)

- **IDA:** 0x472b2e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x82 BUY_PACKAGE_FAILED; op-byte consumed by dispatcher before OnCashItemResBuyPackageFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

