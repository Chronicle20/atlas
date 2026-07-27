# InteractionMiniRoomBalloonRemove (← `CUser::OnMiniRoomBalloon#Remove`)

- **IDA:** 0x938ba5
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserCommonPacket GetUser; §G3)` | ✅ |  |
| 1 | byte | byte `roomType (0 -> CChatBalloon::DestroyMiniRoomBalloon; no trailing fields)` | ✅ |  |

