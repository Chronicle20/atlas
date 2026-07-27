# CashFriendshipFailed (← `CCashShop::OnCashItemResult#FRIENDSHIP_FAILED`)

- **IDA:** 0x47ec0e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x95 FRIENDSHIP_FAILED; op-byte consumed by dispatcher before OnCashItemResFriendShipFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

