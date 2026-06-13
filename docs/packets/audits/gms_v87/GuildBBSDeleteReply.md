# GuildBBSDeleteReply (← `CUIGuildBBS::OnCommentDelete`)

- **IDA:** 0x87a5df
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_delete_reply.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `threadId` | ✅ |  |
| 1 | int32 | int32 `replyId` | ✅ |  |

