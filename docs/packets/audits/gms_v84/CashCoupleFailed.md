# CashCoupleFailed (← `CCashShop::OnCashItemResult#COUPLE_FAILED`)

- **IDA:** 0x47ea60
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x8b COUPLE_FAILED; op-byte consumed by dispatcher before OnCashItemResCoupleFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

