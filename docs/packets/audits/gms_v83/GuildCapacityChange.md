# GuildCapacityChange (← `CWvsContext::OnGuildResult#CapacityChange`)

- **IDA:** 0xa37490
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x3A)` | ✅ |  |
| 1 | int32 | int32 `guildId (guard: must match own)` | ✅ |  |
| 2 | byte | byte `capacity byte (v97 = Decode1)` | ✅ |  |

