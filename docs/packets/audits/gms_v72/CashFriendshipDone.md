# CashFriendshipDone (← `CCashShop::OnCashItemResult#FRIENDSHIP_SUCCESS`)

- **IDA:** 0x47336b
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x7b FRIENDSHIP_SUCCESS; op-byte consumed by dispatcher before OnCashItemResFriendShipDone)` | ✅ |  |
| 1 | bytes | bytes `GW_CashItemInfo blob (55B, single — appended to m_aCashItemInfo)` | ✅ |  |
| 2 | string | string `recipientName` | ✅ |  |
| 3 | int32 | int32 `itemId` | ✅ |  |
| 4 | int16 | int16 `quantity` | ✅ |  |

