# InteractionMiniRoomBalloon (← `CUser::OnMiniRoomBalloon`)

- **IDA:** 0x8d1920
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | int32 `` | ❌ | width mismatch |
| 2 | int32 | string `` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

