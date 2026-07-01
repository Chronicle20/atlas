# AddCharacterEntry (← `CLogin::OnCreateNewCharacterResult`)

- **IDA:** 0x5b3c65
- **Atlas file:** `libs/atlas-packet/character/clientbound/add_entry.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code @0x5b3c80` | ✅ |  |
| 1 | byte | int32 `` | 🔍 | sub-struct:  — see _substruct/ |
| 2 | byte | int32 `` | 🔍 | sub-struct:  — see _substruct/ |
| 3 | bytes | bytes `` | ✅ |  |
| 4 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 36 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 38 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 40 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 41 | byte | bytes `` | ❌ | atlas: short — missing trailing field |

