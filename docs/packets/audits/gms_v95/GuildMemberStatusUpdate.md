# GuildMemberStatusUpdate (← `CWvsContext::OnGuildResult#MemberStatusUpdate`)

- **IDA:** 0xa0e563
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (63)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | byte | byte `online flag (0/1)` | ✅ |  |

