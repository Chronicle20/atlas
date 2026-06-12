# InteractionInteractionEnter (← `CMiniRoomBaseDlg::OnPacketBase#Enter`)

- **IDA:** 0x638f80
- **Atlas file:** `../../libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (4; dispatch byte)` | ✅ |  |
| 1 | bytes | bytes `visitor (slot + DecodeAvatar + userID str + jobCode; interaction.Visitor substruct)` | ✅ |  |

