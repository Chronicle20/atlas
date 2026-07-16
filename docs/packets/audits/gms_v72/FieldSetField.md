# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x6c0c9b
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v72
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
| 37 | byte | int32 `` | 🔍 | sub-struct: RegularEquip — see _substruct/ |
| 38 | int16 | byte `` | ❌ | width mismatch |
| 39 | byte | int32 `` | 🔍 | sub-struct: CashEquip — see _substruct/ |
| 40 | int16 | byte `` | ❌ | width mismatch |
| 41 | byte | int32 `` | 🔍 | sub-struct: EquipInv — see _substruct/ |
| 42 | int32 | byte `` | ❌ | width mismatch |
| 43 | byte | byte `` | 🔍 | sub-struct: UseInv — see _substruct/ |
| 44 | byte | byte `` | ✅ |  |
| 45 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 46 | byte | byte `` | ✅ |  |
| 47 | byte | byte `` | 🔍 | sub-struct: EtcInv — see _substruct/ |
| 48 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 49 | byte | byte `` | 🔍 | sub-struct: CashInv — see _substruct/ |
| 50 | byte | byte `` | ✅ |  |
| 51 | int16 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 52 | int32 | byte `` | ❌ | width mismatch |
| 53 | int32 | byte `` | ❌ | width mismatch |
| 54 | int64 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 55 | int32 | int16 `` | ❌ | width mismatch |
| 56 | int16 | int32 `` | ❌ | width mismatch |
| 57 | int32 | int32 `` | ✅ |  |
| 58 | int32 | int32 `` | ✅ |  |
| 59 | int16 | int16 `` | ✅ |  |
| 60 | string | int32 `` | ❌ | width mismatch |
| 61 | int16 | int16 `` | ✅ |  |
| 62 | int16 | int16 `` | ✅ |  |
| 63 | int64 | int16 `` | ❌ | width mismatch |
| 64 | int16 | string `` | ❌ | width mismatch |
| 65 | int16 | int16 `` | ✅ |  |
| 66 | int16 | int16 `` | ✅ |  |
| 67 | int16 | bytes `` | ✅ |  |
| 68 | int32 | int16 `` | ❌ | width mismatch |
| 69 | int32 | int32 `` | ✅ |  |
| 70 | int32 | int32 `` | ✅ |  |
| 71 | byte | int32 `` | ❌ | width mismatch |
| 72 | int32 | int32 `` | ✅ |  |
| 73 | byte | int32 `` | ❌ | width mismatch |
| 74 | int16 | int16 `` | ✅ |  |
| 75 | int16 | int16 `` | ✅ |  |
| 76 | int16 | int16 `` | ✅ |  |
| 77 | int32 | int32 `` | ✅ |  |
| 78 | int32 | int32 `` | ✅ |  |
| 79 | int32 | int32 `` | ✅ |  |
| 80 | int32 | byte `` | ❌ | width mismatch |
| 81 | int64 | int16 `` | ❌ | width mismatch |
| 82 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 83 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 84 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 100 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 101 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 102 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 103 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 104 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 105 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 106 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 107 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 108 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 109 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 110 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 111 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 112 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 113 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

