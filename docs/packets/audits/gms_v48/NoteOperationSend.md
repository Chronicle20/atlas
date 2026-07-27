# NoteOperationSend (← `CCashShop::OnCashItemResLoadLockerDone`)

- **IDA:** 0x453a91
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int16 `` | ❌ | width mismatch |
| 1 | string | bytes `` | ❌ | width mismatch |
| 2 | byte | int16 `` | ❌ | width mismatch |
| 3 | int32 | bytes `` | ✅ |  |
| 4 | int64 | int16 `` | ❌ | width mismatch |
| 5 | byte | byte `NOTE_ACTION(101) SEND sub-op: mode byte, per-entry gift-forward loop` | ❌ | atlas: short — missing trailing field |
| 6 | byte | string `toName` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `message` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `giftFlag — v48 has NO trailing giftIndex/giftSN (predates those fields)` | ❌ | atlas: short — missing trailing field |

