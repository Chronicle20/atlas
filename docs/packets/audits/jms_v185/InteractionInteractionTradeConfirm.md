# InteractionInteractionTradeConfirm (← `CTradingRoomDlg::OnTrade#TradeConfirm`)

- **IDA:** 0x845ed5
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (15 TRADE_CONFIRM; CTradingRoomDlg::OnPacket @0x845d95 dispatch byte). The arm reads NO body.` | ✅ |  |

