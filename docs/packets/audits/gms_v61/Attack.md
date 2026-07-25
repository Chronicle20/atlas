# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0x7c9403
- **Atlas file:** `libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | byte | int32 `` | ❌ | width mismatch |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | int32 | byte `` | ❌ | width mismatch |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | byte | int32 `` | ❌ | width mismatch |
| 13 | int32 | int32 `` | ✅ |  |
| 14 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

