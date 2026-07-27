# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0x71caed
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v48
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | string `` | ❌ | width mismatch |
| 5 | string | byte `` | ❌ | width mismatch |
| 6 | string | int32 `` | ❌ | width mismatch |
| 7 | byte | string `` | ❌ | width mismatch |
| 8 | byte | byte `` | ✅ |  |
| 9 | int32 | int16 `` | ❌ | width mismatch |
| 10 | string | byte `` | ❌ | width mismatch |
| 11 | byte | int16 `` | ❌ | width mismatch |
| 12 | int16 | int32 `` | ❌ | width mismatch |
| 13 | byte | byte `` | ✅ |  |
| 14 | int16 | int32 `` | ❌ | width mismatch |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | byte | int32 `` | ❌ | width mismatch |
| 17 | byte | byte `` | ✅ |  |
| 18 | int32 | bytes `` | ✅ |  |
| 19 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 24 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 25 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 26 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 27 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 28 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 29 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

