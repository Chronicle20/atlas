# CashMoveLToSFailed (← `CCashShop::OnCashItemResult#MOVE_L_TO_S_FAILED`)

- **IDA:** 0x474087
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x61 MOVE_L_TO_S_FAILED; op-byte consumed by dispatcher before OnCashItemResMoveLtoSFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

