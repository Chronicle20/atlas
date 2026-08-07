# CashBuyPackageFailed (← `CCashShop::OnCashItemResult#BUY_PACKAGE_FAILED`)

- **IDA:** 0x4540da
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x61 BUY_PACKAGE_FAILED; op-byte consumed by dispatcher before OnCashItemResBuyPackageFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

