# GuildInfo (← `CWvsContext::OnGuildResult#Info`)

- **IDA:** 0x4fb760
- **Atlas file:** `libs/atlas-packet/guild/clientbound/info.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (sub-op 0x1A=26 in GUILD_OPERATION)` | ✅ |  |
| 1 | byte | byte `inGuild bool` | ✅ |  |
| 2 | int32 | int32 `guildId` | ✅ |  |
| 3 | string | string `guildName` | ✅ |  |
| 4 | string | string `title[0]` | ✅ |  |
| 5 | byte | string `title[1]` | ❌ | width mismatch |
| 6 | int32 | string `title[2]` | ❌ | width mismatch |
| 7 | byte | string `title[3]` | 🔍 | sub-struct: gm — see _substruct/ |
| 8 | int32 | string `title[4]` | ❌ | width mismatch |
| 9 | int16 | byte `memberCount` | ❌ | width mismatch |
| 10 | byte | bytes `count × charId (4 bytes each, packed via DecodeBuffer)` | ❌ | width mismatch |
| 11 | int16 | bytes `count × GUILDMEMBER (37 bytes each, packed via DecodeBuffer)` | ❌ | width mismatch |
| 12 | byte | int32 `nMaxMemberNum (capacity)` | ❌ | width mismatch |
| 13 | string | int16 `nMarkBg (logoBackground)` | ❌ | width mismatch |
| 14 | int32 | byte `nMarkBgColor (logoBackgroundColor)` | ❌ | width mismatch |
| 15 | int32 | int16 `nMark (logo)` | ❌ | width mismatch |
| 16 | byte | byte `nMarkColor (logoColor)` | ❌ | atlas: short — missing trailing field |
| 17 | byte | string `notice` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `nPoint (points)` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `nAllianceID` | ❌ | atlas: short — missing trailing field |

