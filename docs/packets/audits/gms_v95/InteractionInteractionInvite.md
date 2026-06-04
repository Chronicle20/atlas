# InteractionInteractionInvite (← `CMiniRoomBaseDlg::OnPacketBase#Invite`)

- **IDA:** 0x637a40
- **Atlas file:** `../../libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (2; dispatch byte)` | ✅ |  |
| 1 | byte | byte `roomType (v2)` | ✅ |  |
| 2 | string | string `name (sInviter)` | ✅ |  |
| 3 | int32 | int32 `dwSN (v3)` | ✅ |  |

