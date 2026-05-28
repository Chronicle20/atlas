# OperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x65f438
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (chat text). NOTE: v83 has NO leading update_time (v95-only addition)` | ✅ |  |

