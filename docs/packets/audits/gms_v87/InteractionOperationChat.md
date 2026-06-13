# InteractionOperationChat (← `CMiniRoomBaseDlg::CheckAndSendChat`)

- **IDA:** 0x69973e
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | string | int32 `update_time (get_update_time). PRESENT at v87 (line 16) — same as v95. NOT a v95-only field; gate tightened to GMS>=87.` | ❌ | width mismatch |
| 2 | byte | string `message (chat text)` | ❌ | atlas: short — missing trailing field |

