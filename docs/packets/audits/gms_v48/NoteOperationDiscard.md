# NoteOperationDiscard (← `CMemoListDlg::SetRet`)

- **IDA:** 0x534dc4
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_discard.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | byte `` | ❌ | width mismatch |
| 3 | byte | byte `` | ✅ |  |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |

