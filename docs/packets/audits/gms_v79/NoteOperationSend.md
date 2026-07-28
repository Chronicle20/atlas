# NoteOperationSend (← `CCashShop::OnCashItemResLoadGiftDone`)

- **IDA:** 0x47252c
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `` | ❌ | width mismatch |
| 1 | string | string `` | ✅ |  |
| 2 | byte | string `` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | int64 | int32 `` | ❌ | width mismatch |
| 5 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

