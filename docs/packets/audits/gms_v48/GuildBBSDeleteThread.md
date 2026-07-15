# GuildBBSDeleteThread (← `CUIGuildBBS::OnDelete`)

- **IDA:** 0x608f31
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_delete_thread.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `threadId @0x608f90 (after BBS_OPERATION mode 1=DELETE @0x608f7f)` | ✅ |  |

