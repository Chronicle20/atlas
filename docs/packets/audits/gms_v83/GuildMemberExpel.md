# GuildMemberExpel (← `CWvsContext::OnGuildResult#MemberExpel`)

- **IDA:** 0xa37490
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x2F)` | ✅ |  |
| 1 | int32 | int32 `guildId (v239)` | ✅ |  |
| 2 | int32 | int32 `characterId` | ✅ |  |
| 3 | string | string `name (v240)` | ✅ |  |

