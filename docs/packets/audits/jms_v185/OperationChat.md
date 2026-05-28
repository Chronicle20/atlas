# OperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x6db3ce
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time). JMS v185 PRESENT — same as GMS v95; atlas else-branch (no updateTime) is WRONG for JMS` | ✅ |  |
| 1 | string | string `message (s)` | ✅ |  |

