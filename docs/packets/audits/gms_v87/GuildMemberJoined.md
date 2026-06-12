# GuildMemberJoined (← `CWvsContext::OnGuildResult#MemberJoined`)

- **IDA:** 0xacf7d3
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (39)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | bytes `GUILDMEMBER::Decode (member data)` | ✅ |  |
| 3 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

