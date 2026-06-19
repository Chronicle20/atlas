# StatusMessageJMSCounterNotice (← `CWvsContext::OnMessage#JMSCounterNotice`)

- **IDA:** 0xb0931c
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (15 / 0xF; jms-only)` | ✅ |  |
| 1 | int32 | int32 `amount (single int → StringPool 5603 chat-type-6 notice)` | ✅ |  |

