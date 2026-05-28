# GuildEmblemChange (← `CWvsContext::OnGuildResult#EmblemChange`)

- **IDA:** 0xa37490
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x42)` | ✅ |  |
| 1 | int32 | int32 `guildId (guard)` | ✅ |  |
| 2 | int16 | int16 `logoBackground` | ✅ |  |
| 3 | byte | byte `logoBackgroundColor` | ✅ |  |
| 4 | int16 | int16 `logo` | ✅ |  |
| 5 | byte | byte `logoColor` | ✅ |  |

