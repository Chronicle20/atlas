# GuildBBSReplyThread (← `CUIGuildBBS::OnComment`)

- **IDA:** 0x7c4530
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_reply_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op byte (reply sub-op)` | ❌ | width mismatch |
| 1 | string | int32 `threadId` | ❌ | width mismatch |
| 2 | byte | string `message` | ❌ | atlas: short — missing trailing field |

