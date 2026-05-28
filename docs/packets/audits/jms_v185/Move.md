# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0xaaa076
- **Atlas file:** `libs/atlas-packet/character/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `detectFlag (v26 = a1[151])` | ✅ |  |
| 1 | int16 | byte `fieldKey (CField+328, only if detectFlag)` | ❌ | width mismatch |
| 2 | int16 | int32 `crc (CField+756, only if detectFlag)` | ❌ | width mismatch |
| 3 | byte | bytes `CMovePath::Flush — movement data (only if detectFlag)` | ❌ | width mismatch |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

