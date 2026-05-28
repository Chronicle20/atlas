# GuildBBSThread (← `CUIGuildBBS::OnGuildBBSPacket#BBSThread`)

- **IDA:** 0x87a5df
- **Atlas file:** `libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (1)` | ✅ |  |
| 1 | int32 | int32 `threadId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | int64 | string `characterName` | ❌ | width mismatch |
| 4 | string | string `title` | ✅ |  |
| 5 | string | string `body` | ✅ |  |
| 6 | int32 | byte `icon` | ❌ | width mismatch |
| 7 | int32 | int32 `timestamp` | ✅ |  |
| 8 | int32 | int32 `replyCount` | ✅ |  |
| 9 | int32 | int32 `reply.characterId (loop)` | ✅ |  |
| 10 | int64 | string `reply.characterName (loop)` | ❌ | width mismatch |
| 11 | string | int32 `reply.replyId (loop)` | ❌ | width mismatch |
| 12 | byte | string `reply.body (loop)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 13 | byte | int32 `reply.timestamp (loop)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |

