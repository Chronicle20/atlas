# CashCoupleFailed (← `CCashShop::OnCashItemResult#COUPLE_FAILED`)

- **IDA:** 0x497d20
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x99 COUPLE_FAILED; op-byte consumed by dispatcher before OnCashItemResCoupleFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

