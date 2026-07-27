# InteractionInteractionMiniGameRoom (← `CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccessMiniGame`)

- **IDA:** 0x638e30
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame_room.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (5; OnPacketBase dispatch byte)` | ✅ |  |
| 1 | byte | byte `roomType (nonzero => success; OnEnterResultStatic)` | ✅ |  |
| 2 | byte | byte `capacity (m_nMaxUsers; OnEnterResultBase @0x638e30)` | ✅ |  |
| 3 | byte | byte `yourSlot (m_nMyPosition; OnEnterResultBase)` | ✅ |  |
| 4 | byte | byte `avatar slot (<0/0xFF terminates avatar list)` | ✅ |  |
| 5 | bytes | bytes `avatar look (DecodeAvatar AvatarLook blob)` | ✅ |  |
| 6 | string | string `name (m_asUserID). IsEntrusted()=0 for games => avatar path, no int32 branch` | ✅ |  |
| 7 | int16 | int16 `jobCode (m_anJobCode[i]; v84+/JMS only — absent in v83)` | ✅ |  |
| 8 | byte | byte `avatar list terminator (0xFF)` | ✅ |  |
| 9 | byte | byte `record slot (0xFF terminates record list; COmokDlg::OnEnterResult @0x680e70)` | ✅ |  |
| 10 | int32 | int32 `record: Unknown (sub_4E42FC 20-byte = 5 x int32)` | ✅ |  |
| 11 | int32 | int32 `record: Wins` | ✅ |  |
| 12 | int32 | int32 `record: Ties` | ✅ |  |
| 13 | int32 | int32 `record: Losses` | ✅ |  |
| 14 | int32 | int32 `record: Points` | ✅ |  |
| 15 | byte | byte `record list terminator (0xFF)` | ✅ |  |
| 16 | string | string `title` | ✅ |  |
| 17 | byte | byte `gameKind (piece/sub-type)` | ✅ |  |
| 18 | byte | byte `tournament (bool)` | ✅ |  |
| 19 | byte | byte `round (tournament only)` | ✅ |  |

