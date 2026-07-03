# GuildBBSReplyThread (← `CUIGuildBBS::OnComment`)

- **IDA:** 0x608fe6
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_reply_thread.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `threadId @0x60909a (after BBS_OPERATION mode 4=REPLY @0x609089)` | ✅ |  |
| 1 | string | string `reply message @0x6090b3` | ✅ |  |

