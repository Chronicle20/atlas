# InteractionInteractionTradeMesoLimit (← `CTradingRoomDlg::OnMesoLimitRefused`)

- **IDA:** 0x7358d8
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (20 TRADE_MESO_LIMIT; CTradingRoomDlg::OnPacket @0x735775 dispatch byte). The arm reads NO body.` | ✅ |  |

