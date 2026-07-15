# GuildBBSDeleteReply (← `CUIGuildBBS::OnCommentDelete`)

- **IDA:** 0x6090f4
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_delete_reply.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `threadId @0x609158 (after BBS_OPERATION mode 5=DELETE_REPLY @0x609141)` | ✅ |  |
| 1 | int32 | int32 `replyId @0x60917a` | ✅ |  |

