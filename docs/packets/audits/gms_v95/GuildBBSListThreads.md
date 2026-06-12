# GuildBBSListThreads (← `CUIGuildBBS::SendLoadListRequest`)

- **IDA:** 0x7c3680
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_list_threads.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (list sub-op)` | ✅ |  |
| 1 | int32 | int32 `startIndex` | ✅ |  |

