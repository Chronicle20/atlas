# GuildBBSListThreads (← `CUIGuildBBS::SendLoadListRequest`)

- **IDA:** 0x6091b1
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_list_threads.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `startIndex @0x6091e4 (after BBS_OPERATION mode 2=LIST @0x6091d6)` | ✅ |  |

