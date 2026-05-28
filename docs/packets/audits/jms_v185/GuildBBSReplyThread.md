# GuildBBSReplyThread (← `CUIGuildBBS::OnComment`)

- **IDA:** ABSENT
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_reply_thread.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

