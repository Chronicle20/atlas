# GuildBBSDeleteReply (← `CUIGuildBBS::OnCommentDelete`)

- **IDA:** ABSENT
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_delete_reply.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

