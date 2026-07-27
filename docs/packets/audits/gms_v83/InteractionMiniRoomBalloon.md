# InteractionMiniRoomBalloon (← `CUser::OnMiniRoomBalloon`)

- **IDA:** 0x938ba5
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserCommonPacket GetUser, not OnMiniRoomBalloon; §G3)` | ✅ |  |
| 1 | byte | byte `roomType (0 = remove balloon)` | ✅ |  |
| 2 | int32 | int32 `roomId (m_dwMiniRoomSN; roomType != 0)` | ✅ |  |
| 3 | string | string `title (m_sMiniRoomTitle)` | ✅ |  |
| 4 | byte | byte `hasPassword (m_bPrivate)` | ✅ |  |
| 5 | byte | byte `pieceType (m_nGameKind)` | ✅ |  |
| 6 | byte | byte `occupancy (m_nCurUsers)` | ✅ |  |
| 7 | byte | byte `capacity (m_nMaxUsers)` | ✅ |  |
| 8 | byte | byte `inProgress (m_bGameOn)` | ✅ |  |

