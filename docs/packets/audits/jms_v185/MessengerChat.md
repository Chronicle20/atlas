# MessengerChat (← `CUIMessenger::OnPacket#Chat`)

- **IDA:** 0x8e4851
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 6 (Chat)` | ✅ |  |
| 1 | string | string `chat line (format: 'name : msg')` | ✅ |  |

