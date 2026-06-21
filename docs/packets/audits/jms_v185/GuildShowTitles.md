# GuildShowTitles (← `CWvsContext::OnGuildResult#ShowTitles`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (SHOW_TITLES)` | ✅ |  |
| 1 | int32 | int32 `guildId (discarded)` | ✅ |  |
| 2 | int32 | int32 `count (loop bound)` | ✅ |  |
| 3 | string | string `entry name (per count)` | ✅ |  |
| 4 | int32 | int32 `value0 (per count)` | ✅ |  |
| 5 | byte | int32 `value1 (per count)` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `value2 (per count)` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `value3 (per count)` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `value4 (per count)` | ❌ | atlas: short — missing trailing field |

