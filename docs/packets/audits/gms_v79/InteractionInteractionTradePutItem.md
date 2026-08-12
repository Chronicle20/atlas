# InteractionInteractionTradePutItem (← `CTradingRoomDlg::OnPutItem`)

- **IDA:** 0x7357bf
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (15 TRADE_PUT_ITEM; CTradingRoomDlg::OnPacket @0x735775 dispatch byte)` | ✅ |  |
| 1 | byte | byte `side (recipient-relative room side; indexes this[side + N] item grid)` | ✅ |  |
| 2 | byte | byte `tradeSlot (1..9 within that side's grid)` | ✅ |  |
| 3 | byte | bytes `asset (GW_ItemSlotBase::Decode; model.Asset substruct)` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

