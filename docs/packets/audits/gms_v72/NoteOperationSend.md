# NoteOperationSend (← `CCashShop::OnCashItemResLoadGiftDone`)

- **IDA:** 0x47122e
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int16 `` | ❌ | width mismatch |
| 1 | string | bytes `` | ❌ | width mismatch |
| 2 | byte | byte `mode byte (task-137 notesend-verify2, disasm-confirmed @0x47136b)` | ✅ |  |
| 3 | int32 | string `toName` | ❌ | width mismatch |
| 4 | int64 | string `message` | ❌ | width mismatch |
| 5 | byte | byte `giftFlag` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `giftIndex` | ❌ | atlas: short — missing trailing field |
| 7 | byte | bytes `giftSN (8 bytes)` | ❌ | atlas: short — missing trailing field |

