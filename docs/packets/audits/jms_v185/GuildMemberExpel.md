# GuildMemberExpel (← `CWvsContext::OnGuildResult#MemberExpel`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode = '/' (47 = MemberExpel)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `charId of expelled member` | ✅ |  |
| 3 | string | string `expelled member name` | ✅ |  |

