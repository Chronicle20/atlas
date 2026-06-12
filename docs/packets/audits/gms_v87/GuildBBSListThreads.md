# GuildBBSListThreads (← `CUIGuildBBS::SendLoadListRequest`)

- **IDA:** 0x87a5df
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_list_threads.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isNotice flag` | ✅ |  |
| 1 | int32 | int32 `startIndex (page offset)` | ✅ |  |

