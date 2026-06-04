# GuildCapacityChange (← `CWvsContext::OnGuildResult#CapacityChange`)

- **IDA:** 0xa0dfe2
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (60)` | ✅ |  |
| 1 | int32 | int32 `guildId` | ✅ |  |
| 2 | byte | byte `nMaxMemberNum (capacity) — atlas writes WriteInt (4 bytes), IDA reads Decode1 (1 byte); REAL BUG` | ✅ |  |

