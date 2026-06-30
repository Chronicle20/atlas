# Attack (← `CUserRemote::OnAttack`)

- **IDA:** 0x8d66a1
- **Atlas file:** `libs/atlas-packet/character/clientbound/attack.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | width mismatch |
| 3 | byte | byte `` | ✅ |  |
| 4 | int32 | int16 `` | ❌ | width mismatch |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | byte | byte `` | ✅ |  |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int16 | int16 `` | ✅ |  |
| 14 | int16 | int16 `` | ✅ |  |
| 15 | int32 | int32 `` | ✅ |  |

