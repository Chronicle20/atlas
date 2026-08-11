# InteractionInteractionTradeMesoLimit (← `CTradingRoomDlg::OnMesoLimitRefused`)

- **IDA:** 0x744680
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (21 TRADE_MESO_LIMIT; CTradingRoomDlg::OnPacket @0x744ec0 dispatch byte). The arm reads NO body.` | ✅ |  |

