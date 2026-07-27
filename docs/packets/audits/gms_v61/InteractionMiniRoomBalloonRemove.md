# InteractionMiniRoomBalloonRemove (← `CUser::OnMiniRoomBalloon#Remove`)

- **IDA:** 0x7920b9
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserCommonPacket GetUser)` | ✅ |  |
| 1 | byte | byte `roomType (0 -> CChatBalloon::DestroyMiniRoomBalloon; no trailing fields)` | ✅ |  |

