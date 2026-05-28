# MessengerOperationChat (← `CUIMessenger::ProcessChat`)

- **IDA:** 0x8511fc
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_chat.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `op (6)` | ❌ | width mismatch |
| 1 | byte | string `text` | ❌ | atlas: short — missing trailing field |

