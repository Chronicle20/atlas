# NoteOperationDiscard (← `CMemoListDlg::SetRet`)

- **IDA:** 0x6c2d43
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_discard.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode=1 (DISCARD confirm) @0x6c2dd5` | ✅ |  |
| 1 | byte | byte `totalCount = length of this+204 memo array @0x6c2def` | ✅ |  |
| 2 | byte | byte `specialCount = count of entries with flag==3 in the array, pre-budget @0x6c2e1d` | ✅ |  |
| 3 | int32 | byte `emptySlots = sub_51F1E3(4) free ETC(type-4) inventory slot count @0x6c2e28` | ❌ | width mismatch |
| 4 | byte | int32 `entry SN @0x6c2e63 (normal path)` | ❌ | width mismatch |
| 5 | int32 | byte `entry flag @0x6c2e7b (normal path)` | ❌ | width mismatch |
| 6 | int32 | int32 `entry SN @0x6c2f81 (special path, only when emptySlots budget remains)` | ✅ |  |
| 7 | byte | byte `entry flag @0x6c2f99 (special path)` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `extra1 (v28, atoi() from a '.'-split formatted string via sub_485256) @0x6c2fa4` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `extra2 (a2.p, atoi(String) via sub_485239) @0x6c2faf` | ❌ | atlas: short — missing trailing field |

