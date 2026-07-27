# InteractionInteractionMiniGameResult (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameResult`)

- **IDA:** 0x6e4463
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (62 RESULT; dispatch byte)` | ✅ |  |
| 1 | byte | byte `resultType (1 = tie -> no winner byte; else win/forfeit; §G5 RESULT)` | ✅ |  |
| 2 | byte | byte `winnerSlot (present for resultType != 1; win if == mySlot)` | ✅ |  |
| 3 | int32 | int32 `owner record: Unknown (sub_4E42FC DecodeBuffer(0x14) = 5 x int32)` | ✅ |  |
| 4 | int32 | int32 `owner record: Wins` | ✅ |  |
| 5 | int32 | int32 `owner record: Ties` | ✅ |  |
| 6 | int32 | int32 `owner record: Losses` | ✅ |  |
| 7 | int32 | int32 `owner record: Points` | ✅ |  |
| 8 | int32 | int32 `visitor record: Unknown (sub_4E42FC DecodeBuffer(0x14) = 5 x int32)` | ✅ |  |
| 9 | int32 | int32 `visitor record: Wins` | ✅ |  |
| 10 | int32 | int32 `visitor record: Ties` | ✅ |  |
| 11 | int32 | int32 `visitor record: Losses` | ✅ |  |
| 12 | int32 | int32 `visitor record: Points` | ✅ |  |

