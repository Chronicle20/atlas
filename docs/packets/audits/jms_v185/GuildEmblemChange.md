# GuildEmblemChange (← `CWvsContext::OnGuildResult#EmblemChange`)

- **IDA:** 0xb22518
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (EmblemChange/'B')` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int16 | int16 `logoId` | ✅ |  |
| 3 | byte | byte `logoColor` | ✅ |  |
| 4 | int16 | int16 `backgroundId` | ✅ |  |
| 5 | byte | byte `backgroundColor` | ✅ |  |

