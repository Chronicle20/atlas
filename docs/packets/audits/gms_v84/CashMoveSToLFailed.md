# CashMoveSToLFailed (← `CCashShop::OnCashItemResult#MOVE_S_TO_L_FAILED`)

- **IDA:** 0x47e59f
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6e MOVE_S_TO_L_FAILED; op-byte consumed by dispatcher before OnCashItemResMoveStoLFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

