# GuildBBSThread (← `CUIGuildBBS::OnGuildBBSPacket#BBSThread`)

- **IDA:** 0x87a5df
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/bbs.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x07 BBSThread sub-op, read by OnGuildBBSPacket dispatcher)` | ✅ |  |
| 1 | int32 | int32 `threadId` | ✅ |  |
| 2 | int32 | int32 `posterCharId` | ✅ |  |
| 3 | int64 | bytes `createdAt FILETIME (8 bytes)` | ✅ |  |
| 4 | string | string `title` | ✅ |  |
| 5 | string | string `text` | ✅ |  |
| 6 | int32 | int32 `emoticonId` | ✅ |  |
| 7 | int32 | int32 `replyCount (loop count)` | ✅ |  |
| 8 | int32 | int32 `reply.id` | ✅ |  |
| 9 | int32 | int32 `reply.posterCharId` | ✅ |  |
| 10 | int64 | bytes `reply.createdAt FILETIME (8 bytes)` | ✅ |  |
| 11 | string | string `reply.message` | ✅ |  |

