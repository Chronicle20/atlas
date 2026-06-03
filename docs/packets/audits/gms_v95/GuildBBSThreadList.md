# GuildBBSThreadList (← `CUIGuildBBS::OnGuildBBSPacket#BBSThreadList`)

- **IDA:** 0x7c46c0
- **Atlas file:** `libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6 = list mode 0x06)` | ✅ |  |
| 1 | byte | byte `hasNotice (0 or 1)` | ✅ |  |
| 2 | int32 | int32 `notice.nEntryID (if hasNotice)` | ✅ |  |
| 3 | byte | int32 `notice.nCharacterID (if hasNotice)` | ❌ | width mismatch |
| 4 | int32 | string `notice.sTitle (if hasNotice)` | ❌ | width mismatch |
| 5 | int32 | bytes `notice.ftDate (8 bytes, if hasNotice)` | ❌ | width mismatch |
| 6 | string | int32 `notice.nEmoticon (if hasNotice)` | ❌ | width mismatch |
| 7 | int64 | int32 `notice.nComments (if hasNotice)` | ❌ | width mismatch |
| 8 | int32 | int32 `nEntryListTotalCount` | ✅ |  |
| 9 | int32 | int32 `pageEntryCount (if total > 0)` | ✅ |  |
| 10 | byte | int32 `entry.nEntryID (loop)` | ❌ | width mismatch |
| 11 | int32 | int32 `entry.nCharacterID (loop)` | ✅ |  |
| 12 | int32 | string `entry.sTitle (loop)` | ❌ | width mismatch |
| 13 | int32 | bytes `entry.ftDate 8 bytes (loop)` | ❌ | width mismatch |
| 14 | int32 | int32 `entry.nEmoticon (loop)` | ✅ |  |
| 15 | string | int32 `entry.nComments (loop)` | ❌ | width mismatch |
| 16 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

