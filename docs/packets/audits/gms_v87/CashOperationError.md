# CashOperationError (← `CCashShop::OnCashItemResult#OperationError`)

- **IDA:** 0x484ca3
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x51 LOAD_LOCKER_FAILED; op-byte consumed by OnCashItemResult dispatcher before OnCashItemResLoadLockerFailed). Generic failure-family arm: NoticeFailReason(Decode1).` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

