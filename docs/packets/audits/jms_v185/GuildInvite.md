# GuildInvite (← `CWvsContext::OnGuildResult#Invite`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode = 5 (Invite)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | string | string `inviter name` | ✅ |  |
| 3 | int32 | int32 `unknown (job or level-related)` | ✅ |  |
| 4 | int32 | int32 `skillId` | ✅ |  |

