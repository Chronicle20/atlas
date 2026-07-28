# CashTransferWorldFailed (← `CCashShop::OnCashItemResult#TRANSFER_WORLD_FAILED`)

- **IDA:** 0x455fe0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6F TRANSFER_WORLD_FAILED; op-byte consumed by dispatcher before OnCashItemResTransferWorldFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

