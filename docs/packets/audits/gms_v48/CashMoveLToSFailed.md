# CashMoveLToSFailed (← `CCashShop::OnCashItemResult#MOVE_L_TO_S_FAILED`)

- **IDA:** 0x455172
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x42 MOVE_L_TO_S_FAILED; op-byte consumed by dispatcher before OnCashItemResMoveLtoSFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

