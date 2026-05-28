# OperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x69973e
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time). PRESENT at v87 (line 16) — same as v95. NOT a v95-only field; gate tightened to GMS>=87.` | ✅ |  |
| 1 | string | string `message (chat text)` | ✅ |  |

