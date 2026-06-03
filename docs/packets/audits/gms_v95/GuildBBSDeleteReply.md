# GuildBBSDeleteReply (← `CUIGuildBBS::OnCommentDelete`)

- **IDA:** 0x7c3b70
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_delete_reply.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op byte (delete-reply sub-op)` | ❌ | width mismatch |
| 1 | int32 | int32 `threadId` | ✅ |  |
| 2 | byte | int32 `replyId` | ❌ | atlas: short — missing trailing field |

