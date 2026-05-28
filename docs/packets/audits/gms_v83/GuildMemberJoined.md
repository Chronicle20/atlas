# GuildMemberJoined (← `CWvsContext::OnGuildResult#MemberJoined`)

- **IDA:** 0xa37490
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x27)` | ✅ |  |
| 1 | int32 | int32 `guildId (v241)` | ✅ |  |
| 2 | int32 | int32 `characterId (v56)` | ✅ |  |
| 3 | byte | int32 `member.name (padded 13, via sub_4E4445 = 0x24 bytes struct)` | 🔍 | sub-struct: gm — see _substruct/ |
| 4 | byte | int32 `member.jobId` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `member.level` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `member.title` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `member.online flag (uint32)` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `member.signature` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `member.allianceTitle` | ❌ | atlas: short — missing trailing field |

