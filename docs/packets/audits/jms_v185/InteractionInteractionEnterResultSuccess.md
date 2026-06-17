# InteractionInteractionEnterResultSuccess (← `CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccess`)

- **IDA:** 0x6da234
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (5; dispatch byte)` | ✅ |  |
| 1 | bytes | bytes `room (roomType + maxUsers + myPosition + per-slot avatar loop; interaction.Room substruct)` | ✅ |  |

