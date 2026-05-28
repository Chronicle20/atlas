# GuildEmblemChange (← `CWvsContext::OnGuildResult#EmblemChange`)

- **IDA:** 0xacf7d3
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (50)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int16 | int16 `logoBg` | ✅ |  |
| 3 | byte | byte `logoBgColor` | ✅ |  |
| 4 | int16 | int16 `logo` | ✅ |  |
| 5 | byte | byte `logoColor` | ✅ |  |

