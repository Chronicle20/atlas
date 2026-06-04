# GuildBBSDisplayThread (← `CUIGuildBBS::SendViewEntryRequest`)

- **IDA:** 0x7c3710
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_display_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op byte (view sub-op)` | ❌ | width mismatch |
| 1 | byte | int32 `threadId (entryID)` | ❌ | atlas: short — missing trailing field |

