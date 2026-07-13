# GuildBBSDisplayThread (← `CUIGuildBBS::SendViewEntryRequest`)

- **IDA:** 0x609211
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_display_thread.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `threadId @0x609244 (after BBS_OPERATION mode 3=DISPLAY @0x609236)` | ✅ |  |

