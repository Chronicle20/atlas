# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x6f07d9
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v79
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
| 10 | byte | byte `` | ✅ |  |
| 11 | byte | byte `` | ✅ |  |
| 12 | byte | int32 `` | ❌ | width mismatch |
| 13 | int32 | int32 `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int64 | int32 `` | ❌ | width mismatch |
| 16 | byte | int32 `` | ❌ | width mismatch |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int32 | int32 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int16 | bytes `` | ✅ |  |
| 21 | int16 | byte `` | ❌ | width mismatch |
| 22 | int16 | byte `` | ❌ | width mismatch |
| 23 | int16 | byte `` | ❌ | width mismatch |
| 24 | byte | byte `` | ✅ |  |
| 25 | int16 | int32 `` | ❌ | width mismatch |
| 26 | int32 | int32 `` | ✅ |  |
| 27 | int16 | int32 `` | ❌ | width mismatch |
| 28 | int32 | int32 `` | ✅ |  |
| 29 | int32 | bytes `` | ✅ |  |
| 30 | byte | byte `` | ✅ |  |
| 31 | int32 | int16 `` | ❌ | width mismatch |
| 32 | int16 | int16 `` | ✅ |  |
| 33 | int32 | int16 `` | ❌ | width mismatch |
| 34 | int16 | int16 `` | ✅ |  |
| 35 | int16 | int16 `` | ✅ |  |
| 36 | byte | int16 `` | ❌ | width mismatch |
| 37 | int64 | int16 `` | ❌ | width mismatch |
| 38 | byte | int16 `` | 🔍 | sub-struct: RegularEquip — see _substruct/ |
| 39 | int16 | int16 `` | ✅ |  |
| 40 | byte | int16 `` | 🔍 | sub-struct: CashEquip — see _substruct/ |
| 41 | int16 | int16 `` | ✅ |  |
| 42 | byte | int32 `` | 🔍 | sub-struct: EquipInv — see _substruct/ |
| 43 | int32 | int16 `` | ❌ | width mismatch |
| 44 | byte | int32 `` | 🔍 | sub-struct: UseInv — see _substruct/ |
| 45 | byte | int32 `` | ❌ | width mismatch |
| 46 | byte | byte `` | 🔍 | sub-struct: SetupInv — see _substruct/ |
| 47 | byte | int32 `` | ❌ | width mismatch |
| 48 | byte | byte `` | 🔍 | sub-struct: EtcInv — see _substruct/ |
| 49 | byte | byte `` | ✅ |  |
| 50 | byte | string `` | 🔍 | sub-struct: CashInv — see _substruct/ |
| 51 | byte | int32 `` | ❌ | width mismatch |
| 52 | int16 | byte `` | ❌ | width mismatch |
| 53 | int32 | int32 `` | ✅ |  |
| 54 | int32 | int32 `` | ✅ |  |
| 55 | int64 | byte `` | ❌ | width mismatch |
| 56 | int32 | byte `` | ❌ | width mismatch |
| 57 | int16 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 58 | int32 | byte `` | ❌ | width mismatch |
| 59 | int16 | byte `` | ❌ | width mismatch |
| 60 | int16 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 61 | int16 | byte `` | ❌ | width mismatch |
| 62 | string | byte `` | ❌ | width mismatch |
| 63 | int16 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 64 | int16 | byte `` | ❌ | width mismatch |
| 65 | int64 | byte `` | ❌ | width mismatch |
| 66 | int16 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 67 | int16 | int16 `` | ✅ |  |
| 68 | int32 | int32 `` | ✅ |  |
| 69 | int32 | int32 `` | ✅ |  |
| 70 | int32 | int32 `` | ✅ |  |
| 71 | int32 | int16 `` | ❌ | width mismatch |
| 72 | byte | int32 `` | ❌ | width mismatch |
| 73 | int16 | int16 `` | ✅ |  |
| 74 | int16 | int16 `` | ✅ |  |
| 75 | byte | int16 `` | ❌ | width mismatch |
| 76 | int16 | string `` | ❌ | width mismatch |
| 77 | int16 | int16 `` | ✅ |  |
| 78 | int16 | int16 `` | ✅ |  |
| 79 | int32 | bytes `` | ✅ |  |
| 80 | int32 | int16 `` | ❌ | width mismatch |
| 81 | int32 | int32 `` | ✅ |  |
| 82 | int32 | int32 `` | ✅ |  |
| 83 | int64 | int32 `` | ❌ | width mismatch |
| 84 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 85 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 86 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 100 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 101 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 102 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 103 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 104 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 105 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 106 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 107 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 108 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 109 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 110 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 111 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 112 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 113 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 114 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 115 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 116 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 117 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 118 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 119 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 120 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 121 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 122 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 123 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 124 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 125 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

