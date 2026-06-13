# PetChatRequest (← `CPet::DoAction`)

- **IDA:** 0x7492a2
- **Atlas file:** `libs/atlas-packet/pet/serverbound/chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ✅ |  |
| 1 | int32 | byte `action type` | ❌ | width mismatch |
| 2 | byte | byte `action no` | ✅ |  |
| 3 | byte | string `chat text` | ❌ | width mismatch |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

