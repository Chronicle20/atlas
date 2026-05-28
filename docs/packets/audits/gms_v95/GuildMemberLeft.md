# GuildMemberLeft (← `CWvsContext::OnGuildResult#MemberLeft`)

- **IDA:** 0xa0dd06
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (46)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | string | string `character name` | ✅ |  |

