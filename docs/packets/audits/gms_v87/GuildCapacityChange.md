# GuildCapacityChange (← `CWvsContext::OnGuildResult#CapacityChange`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (53)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | byte | int32 `maxMembers` | ❌ | width mismatch |

