# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x61b27c
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (6)` | ✅ |  |
| 1 | string | string `text` | ✅ |  |

