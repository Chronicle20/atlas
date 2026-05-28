# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x7f6140
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sText — formatted chat message ('name : msg'); op byte (=6) stripped by atlas Operation dispatcher` | ✅ |  |

