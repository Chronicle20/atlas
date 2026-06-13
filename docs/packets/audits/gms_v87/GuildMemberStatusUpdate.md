# GuildMemberStatusUpdate (← `CWvsContext::OnGuildResult#MemberStatusUpdate`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `charId` | ✅ |  |
| 3 | byte | byte `online (WriteBool)` | ✅ |  |

