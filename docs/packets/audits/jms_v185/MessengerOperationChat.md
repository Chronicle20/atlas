# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x8e4f92
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 6 (CHAT)` | ✅ |  |
| 1 | string | string `chat line (format: 'name : msg')` | ✅ |  |

