# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x8e4f92
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `sub-op = 6 (CHAT)` | ❌ | width mismatch |
| 1 | byte | string `chat line (format: 'name : msg')` | ❌ | atlas: short — missing trailing field |

