# InteractionOperationTradeConfirm (← `CTradingRoomDlg::Trade`)

- **IDA:** 0x7646b0
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_trade_confirm.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count (number of crc entries)` | ✅ |  |
| 1 | int32 | int32 `data/itemId (per-entry first)` | ✅ |  |
| 2 | int32 | int32 `crc (per-entry second)` | ✅ |  |

