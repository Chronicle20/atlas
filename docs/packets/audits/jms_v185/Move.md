# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0xaaa076
- **Atlas file:** `libs/atlas-packet/character/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `detectFlag (v26 = a1[151])` | ❌ | width mismatch |
| 1 | int32 | byte `fieldKey (CField+328, only if detectFlag)` | ❌ | width mismatch |
| 2 | byte | int32 `crc (CField+756, only if detectFlag)` | ❌ | width mismatch |
| 3 | int32 | bytes `CMovePath::Flush — movement data (only if detectFlag)` | ✅ |  |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

