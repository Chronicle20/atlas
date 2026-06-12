# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0xaaa076
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `detectFlag (v26 = a1[151])` | ✅ |  |
| 1 | int16 | byte `fieldKey (CField+328, only if detectFlag)` | ❌ | width mismatch |
| 2 | int16 | int32 `crc (CField+756, only if detectFlag)` | ❌ | width mismatch |
| 3 | byte | bytes `CMovePath::Flush — movement data (only if detectFlag)` | ✅ |  |
| 4 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

