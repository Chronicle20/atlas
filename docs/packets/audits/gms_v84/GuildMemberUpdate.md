# GuildMemberUpdate (← `CWvsContext::OnGuildResult#MemberUpdate`)

- **IDA:** 0xa82e2b
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (MEMBER_UPDATE)` | ✅ |  |
| 1 | int32 | int32 `guildId (match check)` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | int32 | int32 `level` | ✅ |  |
| 4 | int32 | int32 `job` | ✅ |  |

