# InteractionOperationTradePutItem (← `CTradingRoomDlg::PutItem`)

- **IDA:** 0x7c359f
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `invType (a4)` | ✅ |  |
| 1 | int16 | int16 `srcSlot (a2)` | ✅ |  |
| 2 | int16 | int16 `destSlot/qty (m_pHead)` | ✅ |  |
| 3 | byte | byte `trade-window slot (v25[0])` | ✅ |  |

