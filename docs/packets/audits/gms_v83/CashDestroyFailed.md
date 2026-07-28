# CashDestroyFailed (← `CCashShop::OnCashItemResult#DESTROY_FAILED`)

- **IDA:** 0x47b4c5
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6d DESTROY_FAILED; op-byte consumed by dispatcher before OnCashItemResDestroyFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

