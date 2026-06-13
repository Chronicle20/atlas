# InteractionOperationTradePutItem (← `CTradingRoomDlg::PutItem`)

- **IDA:** 0x847f51
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int16 | byte `` | ❌ | width mismatch |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | byte | int16 `` | ❌ | width mismatch |
| 4 | byte | byte `` | ❌ | atlas: short — missing trailing field |

