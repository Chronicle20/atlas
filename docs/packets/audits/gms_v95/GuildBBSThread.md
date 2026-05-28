# GuildBBSThread (← `CUIGuildBBS::OnGuildBBSPacket#BBSThread`)

- **IDA:** 0x7c6630
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7 = view mode 0x07)` | ✅ |  |
| 1 | int32 | int32 `nCurEntryID` | ✅ |  |
| 2 | int32 | int32 `nCurCharacterID` | ✅ |  |
| 3 | int64 | bytes `ftCurDate (8 bytes)` | ❌ | width mismatch |
| 4 | string | string `sCurTitle` | ✅ |  |
| 5 | string | string `sCurText` | ✅ |  |
| 6 | int32 | int32 `nEmoticon` | ✅ |  |
| 7 | int32 | int32 `replyCount` | ✅ |  |
| 8 | int32 | int32 `reply.m_nSN (loop)` | ✅ |  |
| 9 | int32 | int32 `reply.m_nCharacterID (loop)` | ✅ |  |
| 10 | int64 | bytes `reply.m_ftDate (8 bytes, loop)` | ❌ | width mismatch |
| 11 | string | string `reply.m_sComment (loop)` | ✅ |  |

