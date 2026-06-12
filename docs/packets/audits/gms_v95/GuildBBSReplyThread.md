# GuildBBSReplyThread (← `CUIGuildBBS::OnComment`)

- **IDA:** 0x7c4530
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_reply_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (reply sub-op)` | ✅ |  |
| 1 | int32 | int32 `threadId` | ✅ |  |
| 2 | string | string `message` | ✅ |  |

