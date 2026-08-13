# FieldSetField (← `CStage::OnSetField`)

- **IDA:** 0x6fdc70
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_field.go`
- **Variant:** GMS/v92
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int32 `` | ❌ | width mismatch |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | int32 | string `` | ❌ | width mismatch |
| 6 | int64 | string `` | ❌ | width mismatch |
| 7 | int16 | int32 `` | ❌ | width mismatch |
| 8 | byte | int32 `` | ❌ | width mismatch |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | bytes | byte `` | ✅ |  |
| 11 | byte | int32 `` | ❌ | width mismatch |
| 12 | byte | byte `` | ✅ |  |
| 13 | byte | int16 `` | ❌ | width mismatch |
| 14 | int32 | byte `` | ❌ | width mismatch |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | int64 | int32 `` | ❌ | width mismatch |
| 17 | int64 | bytes `` | ✅ |  |
| 18 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 24 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 25 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 26 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 27 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 28 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 29 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 30 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 31 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 32 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 33 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 34 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 35 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 36 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 37 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 38 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 39 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 40 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 41 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 42 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 43 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 44 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 45 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 46 | int64 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 47 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 48 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 49 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 50 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 51 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 52 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 53 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 54 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 55 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 56 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 57 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 58 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 59 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 60 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 61 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 62 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 63 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 64 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 65 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 66 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 67 | int64 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 68 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 69 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 70 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 71 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 72 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 73 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 74 | string | byte `` | ✅ | absorbed by trailing opaque buffer |
| 75 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 76 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 77 | int64 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 78 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 79 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 80 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 81 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 82 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 83 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 84 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 85 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 86 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 87 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 88 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 89 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 90 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 91 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 92 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 93 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 94 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 95 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 96 | int64 | byte `` | ✅ | absorbed by trailing opaque buffer |

