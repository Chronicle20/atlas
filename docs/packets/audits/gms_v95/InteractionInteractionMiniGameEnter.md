# InteractionInteractionMiniGameEnter (← `CMiniRoomBaseDlg::OnPacketBase#EnterMiniGame`)

- **IDA:** 0x638f80
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame_enter.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (4; OnPacketBase dispatch byte)` | ✅ |  |
| 1 | byte | byte `slot (m_apAvatar index; OnEnterBase @0x638f80)` | ✅ |  |
| 2 | bytes | bytes `avatar look (DecodeAvatar AvatarLook blob)` | ✅ |  |
| 3 | string | string `name (m_asUserID)` | ✅ |  |
| 4 | int16 | int16 `jobCode (m_anJobCode[i]; v84+/JMS only - absent in v83; OnEnterBase @0x638f80)` | ✅ |  |
| 5 | int32 | int32 `record: Unknown (COmokDlg::OnEnter @0x6812e0 -> GW_MiniGameRecord::Decode @0x4f2ad0 DecodeBuffer(20) = 5 x int32)` | ✅ |  |
| 6 | int32 | int32 `record: Wins` | ✅ |  |
| 7 | int32 | int32 `record: Ties` | ✅ |  |
| 8 | int32 | int32 `record: Losses` | ✅ |  |
| 9 | int32 | int32 `record: Points` | ✅ |  |

