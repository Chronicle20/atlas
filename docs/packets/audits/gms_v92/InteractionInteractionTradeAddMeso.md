# InteractionInteractionTradeAddMeso (← `CTradingRoomDlg::OnPutMoney`)

- **IDA:** 0x743e70
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (16 TRADE_ADD_MESO; CTradingRoomDlg::OnPacket @0x744ec0 dispatch byte)` | ✅ |  |
| 1 | byte | byte `side (recipient-relative room side)` | ✅ |  |
| 2 | int32 | int32 `amount (ASSIGNED: this[side + N] = Decode4(), not accumulated)` | ✅ |  |

