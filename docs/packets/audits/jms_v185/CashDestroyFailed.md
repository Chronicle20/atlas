# CashDestroyFailed (← `CCashShop::OnCashItemResult#DESTROY_FAILED`)

- **IDA:** 0x48e09f
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x70 DESTROY_FAILED; op-byte consumed by dispatcher before OnCashItemResDestroyFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

