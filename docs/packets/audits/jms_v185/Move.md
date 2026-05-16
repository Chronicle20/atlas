# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0xaaa076
- **Atlas file:** `libs/atlas-packet/character/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `detectFlag (v26 = a1[151])` | ✅ |  |
| 1 | byte | byte `fieldKey (CField+328, only if detectFlag)` | 🔍 | sub-struct: movement — see _substruct/ |
| 2 | byte | int32 `crc (CField+756, only if detectFlag)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | bytes `CMovePath::Flush — movement data (only if detectFlag)` | ❌ | atlas: short — missing trailing field |

