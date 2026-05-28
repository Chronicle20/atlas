# GuildMemberStatusUpdate (← `CWvsContext::OnGuildResult#MemberStatusUpdate`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (MemberStatusUpdate)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `charId` | ✅ |  |
| 3 | byte | byte `channelId` | ✅ |  |
| 4 | byte | int32 `job` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int16 `level` | ❌ | atlas: short — missing trailing field |

