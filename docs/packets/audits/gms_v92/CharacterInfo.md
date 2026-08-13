# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0x9daa40
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v92
- **Branch depth:** 4
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |
| 5 | string | string `` | ✅ |  |
| 6 | string | string `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | byte | byte `` | ✅ |  |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | string | int32 `` | ❌ | width mismatch |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int16 | int32 `` | ❌ | width mismatch |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | byte | bytes `` | ✅ |  |
| 15 | byte | int32 `` | ❌ | width mismatch |
| 16 | int32 | bytes `` | ✅ |  |
| 17 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 24 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

