# StatusMessageSystemMessage (← `CWvsContext::OnMessage#SystemMessage`)

- **IDA:** 0xa21a78
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (SYSTEM_MESSAGE)` | ✅ |  |
| 1 | string | string `message` | ✅ |  |

