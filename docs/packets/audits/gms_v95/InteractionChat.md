# InteractionChat (← `CMiniRoomBaseDlg::OnPacketBase#Chat`)

- **IDA:** 0x639ad0
- **Atlas file:** `../../libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (6; dispatch byte)` | ✅ |  |
| 1 | byte | byte `chatType (7=game-msg else font/0..3)` | ✅ |  |
| 2 | byte | byte `slot (v7 speaker position)` | ✅ |  |
| 3 | string | string `message (sText)` | ✅ |  |

