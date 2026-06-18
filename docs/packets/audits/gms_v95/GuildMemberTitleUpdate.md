# GuildMemberTitleUpdate (← `CWvsContext::OnGuildResult#MemberTitleUpdate`)

- **IDA:** 0xa0e0b5
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (66)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | byte | byte `newGrade/title (1-based)` | ✅ |  |

