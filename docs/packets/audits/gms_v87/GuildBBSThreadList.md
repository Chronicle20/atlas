# GuildBBSThreadList (← `CUIGuildBBS::OnGuildBBSPacket#BBSThreadList`)

- **IDA:** 0x87a5df
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0)` | ✅ |  |
| 1 | byte | byte `isNotice flag` | ✅ |  |
| 2 | int32 | int32 `totalThreads` | ✅ |  |
| 3 | byte | int32 `noticeThreadId` | ❌ | width mismatch |
| 4 | int32 | byte `threadCount` | ❌ | width mismatch |
| 5 | int32 | int32 `thread.threadId (loop)` | ✅ |  |
| 6 | string | int32 `thread.characterId (loop)` | ❌ | width mismatch |
| 7 | int64 | string `thread.characterName (loop)` | ❌ | width mismatch |
| 8 | int32 | string `thread.title (loop)` | ❌ | width mismatch |
| 9 | int32 | byte `thread.icon (loop)` | ❌ | width mismatch |
| 10 | byte | int32 `thread.replyCount (loop)` | ❌ | width mismatch |
| 11 | int32 | int32 `thread.timestamp (loop)` | ✅ |  |
| 12 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

