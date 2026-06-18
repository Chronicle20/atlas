# GuildJoinErrorMaxMembers (← `CWvsContext::OnGuildResult#JoinErrorMaxMembers`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (MAX_NUMBER_OF_USERS)` | ✅ |  |

