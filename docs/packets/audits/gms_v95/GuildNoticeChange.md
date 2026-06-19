# GuildNoticeChange (← `CWvsContext::OnGuildResult#NoticeChange`)

- **IDA:** 0xa0e44b
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (71)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `notice text` | ✅ |  |

