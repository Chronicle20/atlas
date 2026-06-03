# InteractionOperationTradePutItem (← `CTradingRoomDlg::PutItem`)

- **IDA:** 0x7641d0
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType (nItemTI)` | ✅ |  |
| 1 | int16 | int16 `slot (nSlotPosition)` | ✅ |  |
| 2 | int16 | int16 `quantity (m_nInputNo_Result)` | ✅ |  |
| 3 | byte | byte `targetSlot (ItemIndexFromPoint)` | ✅ |  |

