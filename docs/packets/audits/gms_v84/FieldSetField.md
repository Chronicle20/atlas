# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x798987
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v84
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
| 13 | int32 | bytes `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int64 | bytes `` | ✅ |  |
| 16 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 17 | int16 | byte `` | ❌ | width mismatch |
| 18 | int16 | byte `` | ❌ | width mismatch |
| 19 | int16 | string `` | ❌ | width mismatch |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | int16 | byte `` | ❌ | width mismatch |
| 22 | int32 | int32 `` | ✅ |  |
| 23 | int32 | int32 `` | ✅ |  |
| 24 | byte | int16 `` | ❌ | width mismatch |
| 25 | int16 | byte `` | ❌ | width mismatch |
| 26 | int32 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 27 | int16 | int16 `` | ✅ |  |
| 28 | int32 | byte `` | ❌ | width mismatch |
| 29 | int32 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 30 | byte | int16 `` | ❌ | width mismatch |
| 31 | int32 | byte `` | ❌ | width mismatch |
| 32 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 33 | byte | int16 `` | ❌ | width mismatch |
| 34 | int32 | byte `` | ❌ | width mismatch |
| 35 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 36 | byte | byte `` | ✅ |  |
| 37 | byte | byte `` | ✅ |  |
| 38 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 39 | byte | int16 `` | ❌ | width mismatch |
| 40 | int64 | int32 `` | ❌ | width mismatch |
| 41 | byte | int32 `` | 🔍 | sub-struct: RegularEquip — see _substruct/ |
| 42 | int16 | bytes `` | ✅ |  |
| 43 | byte | int32 `` | 🔍 | sub-struct: CashEquip — see _substruct/ |
| 44 | int16 | int16 `` | ✅ |  |
| 45 | byte | int32 `` | 🔍 | sub-struct: EquipInv — see _substruct/ |
| 46 | int32 | int16 `` | ❌ | width mismatch |
| 47 | byte | int16 `` | 🔍 | sub-struct: UseInv — see _substruct/ |
| 48 | byte | int16 `` | ❌ | width mismatch |
| 49 | byte | string `` | 🔍 | sub-struct: SetupInv — see _substruct/ |
| 50 | byte | int16 `` | ❌ | width mismatch |
| 51 | byte | int16 `` | 🔍 | sub-struct: EtcInv — see _substruct/ |
| 52 | byte | bytes `` | ✅ |  |
| 53 | byte | int16 `` | 🔍 | sub-struct: CashInv — see _substruct/ |
| 54 | byte | int32 `` | ❌ | width mismatch |
| 55 | int16 | int32 `` | ❌ | width mismatch |
| 56 | int32 | int32 `` | ✅ |  |
| 57 | int32 | int32 `` | ✅ |  |
| 58 | int64 | int32 `` | ❌ | width mismatch |
| 59 | int32 | int16 `` | ❌ | width mismatch |
| 60 | int16 | int16 `` | ✅ |  |
| 61 | int32 | int16 `` | ❌ | width mismatch |
| 62 | int32 | int32 `` | ✅ |  |
| 63 | int16 | int32 `` | ❌ | width mismatch |
| 64 | string | int32 `` | ❌ | width mismatch |
| 65 | int16 | byte `` | ❌ | width mismatch |
| 66 | int16 | int16 `` | ✅ |  |
| 67 | int64 | int16 `` | ❌ | width mismatch |
| 68 | int16 | byte `` | ❌ | width mismatch |
| 69 | int16 | int16 `` | ✅ |  |
| 70 | int16 | byte `` | ❌ | width mismatch |
| 71 | int16 | bytes `` | ✅ |  |
| 72 | int32 | byte `` | ❌ | width mismatch |
| 73 | int32 | bytes `` | ✅ |  |
| 74 | int32 | int16 `` | ❌ | width mismatch |
| 75 | byte | int32 `` | ❌ | width mismatch |
| 76 | int32 | int32 `` | ✅ |  |
| 77 | byte | string `` | ❌ | width mismatch |
| 78 | int16 | byte `` | ❌ | width mismatch |
| 79 | int16 | bytes `` | ✅ |  |
| 80 | int16 | int32 `` | ❌ | width mismatch |
| 81 | int32 | string `` | ❌ | width mismatch |
| 82 | int32 | byte `` | ❌ | width mismatch |
| 83 | int32 | byte `` | ❌ | width mismatch |
| 84 | int32 | bytes `` | ✅ |  |
| 85 | int64 | string `` | ❌ | width mismatch |
| 86 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 88 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 92 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 93 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 94 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 95 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 96 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 97 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 98 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 99 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

