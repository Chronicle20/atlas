# GuildNoticeChange (← `CWvsContext::OnGuildResult#NoticeChange`)

- **IDA:** 0xa37490
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x44)` | ✅ |  |
| 1 | int32 | int32 `guildId (guard)` | ✅ |  |
| 2 | string | string `notice (v126)` | ✅ |  |

