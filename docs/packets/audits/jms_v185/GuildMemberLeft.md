# GuildMemberLeft (← `CWvsContext::OnGuildResult#MemberLeft`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (MemberLeft/',' or '/')` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `charId of departing member` | ✅ |  |
| 3 | string | string `departing member name` | ✅ |  |

