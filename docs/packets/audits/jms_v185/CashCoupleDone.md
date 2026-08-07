# CashCoupleDone (← `CCashShop::OnCashItemResult#COUPLE_SUCCESS`)

- **IDA:** 0x48e36b
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x8a COUPLE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResCoupleDone)` | ✅ |  |
| 1 | bytes | bytes `GW_CashItemInfo blob (55B, single — appended to m_aCashItemInfo)` | ✅ |  |
| 2 | string | string `recipientName` | ✅ |  |
| 3 | int32 | int32 `itemId (second/separate reference, used only for CItemInfo::GetItemName in notice text)` | ✅ |  |
| 4 | int16 | int16 `quantity` | ✅ |  |

