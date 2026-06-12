# GuildInfo (← `CWvsContext::OnGuildResult#Info`)

- **IDA:** 0x4fb760
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/info.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

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
| 7 | bytes | string `title[3]` | ❌ | width mismatch |
| 8 | int32 | string `title[4]` | ❌ | width mismatch |
| 9 | int32 | byte `memberCount` | ❌ | width mismatch |
| 10 | int32 | bytes `count × charId (4 bytes each, packed via DecodeBuffer)` | ✅ |  |
| 11 | int32 | bytes `count × GUILDMEMBER (37 bytes each, packed via DecodeBuffer)` | ✅ |  |
| 12 | int32 | int32 `nMaxMemberNum (capacity)` | ✅ |  |
| 13 | int32 | int16 `nMarkBg (logoBackground)` | ❌ | width mismatch |
| 14 | int32 | byte `nMarkBgColor (logoBackgroundColor)` | ❌ | width mismatch |
| 15 | int16 | int16 `nMark (logo)` | ✅ |  |
| 16 | byte | byte `nMarkColor (logoColor)` | ✅ |  |
| 17 | int16 | string `notice` | ❌ | width mismatch |
| 18 | byte | int32 `nPoint (points)` | ❌ | width mismatch |
| 19 | string | int32 `nAllianceID` | ❌ | width mismatch |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

