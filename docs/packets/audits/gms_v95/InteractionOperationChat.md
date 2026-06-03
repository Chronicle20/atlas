# InteractionOperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x6382a0
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time)` | ✅ |  |
| 1 | string | string `message (chat text)` | ✅ |  |

