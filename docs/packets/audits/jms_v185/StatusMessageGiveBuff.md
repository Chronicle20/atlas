# StatusMessageGiveBuff (← `CWvsContext::OnMessage#GiveBuff`)

- **IDA:** 0xb0945d
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (GIVE_BUFF)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |

