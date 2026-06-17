# CashLoadInventoryFailure (← `CCashShop::OnCashItemResult#LOAD_INVENTORY_FAILURE`)

- **IDA:** 0x47957c
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4c LOAD_LOCKER_FAILED; op-byte consumed by OnCashItemResult dispatcher before OnCashItemResLoadLockerFailed). Generic failure-family arm: NoticeFailReason(Decode1).` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

