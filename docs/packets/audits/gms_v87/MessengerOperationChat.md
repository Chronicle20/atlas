# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x8b978f
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `mode byte (6 = chat)` | ❌ | width mismatch |
| 1 | byte | string `message text` | ❌ | atlas: short — missing trailing field |

