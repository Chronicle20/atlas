# GuildMemberJoined (← `CWvsContext::OnGuildResult#MemberJoined`)

- **IDA:** 0xa0dbc0
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (41)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | byte | bytes `GUILDMEMBER::Decode (37 bytes raw = name[13] + job[4] + level[4] + title[4] + online[4] + signature[4] + allianceTitle[4])` | 🔍 | sub-struct: gm — see _substruct/ |

