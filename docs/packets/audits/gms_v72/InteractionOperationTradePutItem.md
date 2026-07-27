# InteractionOperationTradePutItem (← `CTradingRoomDlg::PutItem`)

- **IDA:** 0x6ff1be
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType` | ✅ |  |
| 1 | int16 | int16 `slot` | ✅ |  |
| 2 | int16 | int16 `quantity` | ✅ |  |
| 3 | byte | byte `targetSlot` | ✅ |  |

