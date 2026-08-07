# CashTransferWorldFailed (← `CCashShop::OnCashItemResult#TRANSFER_WORLD_FAILED`)

- **IDA:** 0x47c072
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa1 TRANSFER_WORLD_FAILED; op-byte consumed by dispatcher before OnCashItemResTransferWorldFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

