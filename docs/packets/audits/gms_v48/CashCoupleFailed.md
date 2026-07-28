# CashCoupleFailed (← `CCashShop::OnCashItemResult#COUPLE_FAILED`)

- **IDA:** 0x45586e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5F COUPLE_FAILED; op-byte consumed by dispatcher before OnCashItemResCoupleFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

