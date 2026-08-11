# InteractionInteractionTradeConfirm (← `CTradingRoomDlg::OnTrade#TradeConfirm`)

- **IDA:** 0x6fddec
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (16 TRADE_CONFIRM; CTradingRoomDlg::OnPacket @0x6fdc9d dispatch byte). The arm reads NO body.` | ✅ |  |

