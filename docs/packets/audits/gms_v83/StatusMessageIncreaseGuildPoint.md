# StatusMessageIncreaseGuildPoint (← `CWvsContext::OnMessage#IncreaseGuildPoint`)

- **IDA:** 0xa222c9
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (INCREASE_GUILD_POINT)` | ✅ |  |
| 1 | int32 | int32 `amount (signed)` | ✅ |  |

