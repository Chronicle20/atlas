# GuildMemberLeft (← `CWvsContext::OnGuildResult#MemberLeft`)

- **IDA:** 0xacf7d3
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (38)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |

