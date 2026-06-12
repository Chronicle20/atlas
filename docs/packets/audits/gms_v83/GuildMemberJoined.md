# GuildMemberJoined (← `CWvsContext::OnGuildResult#MemberJoined`)

- **IDA:** 0xa37490
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | int32 | int32 `charId` | ✅ |  |
| 3 | bytes | bytes `GUILDMEMBER::Decode (37-byte block)` | ✅ |  |

