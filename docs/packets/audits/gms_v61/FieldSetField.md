# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x659fd3
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `` | ❌ | width mismatch |
| 1 | int32 | byte `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int16 `` | ❌ | width mismatch |
| 4 | int16 | string `` | ❌ | width mismatch |
| 5 | int32 | string `` | ❌ | width mismatch |
| 6 | int64 | int32 `` | ❌ | width mismatch |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | bytes | bytes `` | ✅ |  |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | byte | bytes `` | ✅ |  |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | int32 | byte `` | ❌ | width mismatch |
| 15 | int64 | byte `` | ❌ | width mismatch |
| 16 | byte | byte `` | ✅ |  |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | int16 | bytes `` | ✅ |  |
| 22 | int16 | byte `` | ❌ | width mismatch |
| 23 | byte | int16 `` | ❌ | width mismatch |
| 24 | int16 | int16 `` | ✅ |  |
| 25 | int32 | int16 `` | ❌ | width mismatch |
| 26 | int16 | int16 `` | ✅ |  |
| 27 | int32 | int16 `` | ❌ | width mismatch |
| 28 | int32 | int16 `` | ❌ | width mismatch |
| 29 | byte | int16 `` | ❌ | width mismatch |
| 30 | int32 | int16 `` | ❌ | width mismatch |
| 31 | int16 | int16 `` | ✅ |  |
| 32 | int32 | int16 `` | ❌ | width mismatch |
| 33 | int16 | int16 `` | ✅ |  |
| 34 | byte | int32 `` | ❌ | width mismatch |
| 35 | int16 | int16 `` | ✅ |  |
| 36 | int64 | int32 `` | ❌ | width mismatch |
| 37 | byte | byte `` | 🔍 | sub-struct: RegularEquip — see _substruct/ |
| 38 | int16 | byte `` | ❌ | width mismatch |
| 39 | byte | int32 `` | 🔍 | sub-struct: CashEquip — see _substruct/ |
| 40 | int16 | byte `` | ❌ | width mismatch |
| 41 | byte | byte `` | 🔍 | sub-struct: EquipInv — see _substruct/ |
| 42 | int32 | byte `` | ❌ | width mismatch |
| 43 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 44 | byte | byte `` | ✅ |  |
| 45 | byte | byte `` | 🔍 | sub-struct: SetupInv — see _substruct/ |
| 46 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 47 | byte | byte `` | 🔍 | sub-struct: EtcInv — see _substruct/ |
| 48 | byte | byte `` | ✅ |  |
| 49 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 50 | byte | byte `` | ✅ |  |
| 51 | int16 | byte `` | ❌ | width mismatch |
| 52 | int32 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 53 | int32 | int16 `` | ❌ | width mismatch |
| 54 | int64 | int32 `` | ❌ | width mismatch |
| 55 | int32 | int32 `` | ✅ |  |
| 56 | int16 | int32 `` | ❌ | width mismatch |
| 57 | int32 | int16 `` | ❌ | width mismatch |
| 58 | int32 | int32 `` | ✅ |  |
| 59 | int16 | int16 `` | ✅ |  |
| 60 | string | int16 `` | ❌ | width mismatch |
| 61 | int16 | int16 `` | ✅ |  |
| 62 | int16 | string `` | ❌ | width mismatch |
| 63 | int64 | int16 `` | ❌ | width mismatch |
| 64 | int16 | int16 `` | ✅ |  |
| 65 | int16 | bytes `` | ✅ |  |
| 66 | int16 | int16 `` | ✅ |  |
| 67 | int16 | int32 `` | ❌ | width mismatch |
| 68 | int32 | int32 `` | ✅ |  |
| 69 | int32 | int32 `` | ✅ |  |
| 70 | int32 | int32 `` | ✅ |  |
| 71 | byte | int32 `` | ❌ | width mismatch |
| 72 | int16 | int16 `` | ✅ |  |
| 73 | int16 | int16 `` | ✅ |  |
| 74 | byte | int16 `` | ❌ | width mismatch |
| 75 | int32 | int32 `` | ✅ |  |
| 76 | int16 | int32 `` | ❌ | width mismatch |
| 77 | int32 | int32 `` | ✅ |  |
| 78 | int32 | int32 `` | ✅ |  |
| 79 | int32 | byte `` | ❌ | width mismatch |
| 80 | int32 | int16 `` | ❌ | width mismatch |
| 81 | int64 | byte `` | ❌ | width mismatch |
| 82 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

