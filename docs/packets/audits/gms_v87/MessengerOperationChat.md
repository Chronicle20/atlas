# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x8b978f
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6 = chat)` | ✅ |  |
| 1 | string | string `message text` | ✅ |  |

