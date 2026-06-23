# StatusMessageIncreaseFame (← `CWvsContext::OnMessage#IncreaseFame`)

- **IDA:** 0xb09180
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (INCREASE_FAME)` | ✅ |  |
| 1 | int32 | int32 `amount (signed)` | ✅ |  |

