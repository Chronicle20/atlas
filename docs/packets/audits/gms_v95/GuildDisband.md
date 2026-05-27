# GuildDisband (← `CWvsContext::OnGuildResult#Disband`)

- **IDA:** 0xa0dfcb
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (52)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |

