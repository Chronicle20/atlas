# GuildBBSDeleteReply (← `CUIGuildBBS::OnCommentDelete`)

- **IDA:** 0x7c3b70
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_delete_reply.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (delete-reply sub-op)` | ✅ |  |
| 1 | int32 | int32 `threadId` | ✅ |  |
| 2 | int32 | int32 `replyId` | ✅ |  |

