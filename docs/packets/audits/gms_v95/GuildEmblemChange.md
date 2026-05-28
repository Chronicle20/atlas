# GuildEmblemChange (← `CWvsContext::OnGuildResult#EmblemChange`)

- **IDA:** 0xa0e394
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (69)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int16 | int16 `nMarkBg (logoBackground)` | ✅ |  |
| 3 | byte | byte `nMarkBgColor (logoBackgroundColor)` | ✅ |  |
| 4 | int16 | int16 `nMark (logo)` | ✅ |  |
| 5 | byte | byte `nMarkColor (logoColor)` | ✅ |  |

