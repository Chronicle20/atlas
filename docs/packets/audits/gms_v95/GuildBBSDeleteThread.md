# GuildBBSDeleteThread (← `CUIGuildBBS::OnDelete`)

- **IDA:** 0x7c6520
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_delete_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (delete-thread sub-op)` | ✅ |  |
| 1 | int32 | int32 `threadId` | ✅ |  |

