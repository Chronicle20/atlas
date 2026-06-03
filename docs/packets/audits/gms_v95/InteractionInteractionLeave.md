# InteractionInteractionLeave (← `CMiniRoomBaseDlg::OnPacketBase#Leave`)

- **IDA:** 0x637510
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (10; dispatch byte)` | ✅ |  |
| 1 | byte | byte `slot (v4)` | ✅ |  |
| 2 | byte | byte `status (read by subclass OnLeave; e.g. CTradingRoomDlg)` | ✅ |  |

