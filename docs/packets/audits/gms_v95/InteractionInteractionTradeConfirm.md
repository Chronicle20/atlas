# InteractionInteractionTradeConfirm (← `CTradingRoomDlg::OnTrade#TradeConfirm`)

- **IDA:** 0x763f20
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (17 TRADE_CONFIRM; CTradingRoomDlg::OnPacket @0x7649a0 dispatch byte). The arm reads NO body.` | ✅ |  |

