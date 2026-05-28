# OperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x6382a0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time (get_update_time)` | ❌ | width mismatch |
| 1 | byte | string `message (chat text)` | ❌ | atlas: short — missing trailing field |


> defer: REAL ❌ — atlas missing leading uint32 update_time. Version-sensitive;
> no cross-version IDA. See `docs/packets/ida-exports/_pending.md` → "OperationChat".
