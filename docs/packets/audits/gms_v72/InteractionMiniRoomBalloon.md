# InteractionMiniRoomBalloon (← `CUser::OnMiniRoomBalloon`)

- **IDA:** 0x847df1
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserCommonPacket GetUser, not OnMiniRoomBalloon)` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | string | string `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |

