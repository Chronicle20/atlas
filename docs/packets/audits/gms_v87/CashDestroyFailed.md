# CashDestroyFailed (← `CCashShop::OnCashItemResult#DESTROY_FAILED`)

- **IDA:** 0x486ca3
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x72 DESTROY_FAILED; op-byte consumed by dispatcher before OnCashItemResDestroyFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

