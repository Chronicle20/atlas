# PetActivated (← `CUserRemote::OnPetActivated`)

- **IDA:** 0x9c3e9d
- **Atlas file:** `libs/atlas-packet/pet/clientbound/activated.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | string `` | ❌ | width mismatch |
| 5 | string | bytes `` | ❌ | width mismatch |
| 6 | int64 | int16 `` | ❌ | width mismatch |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | int16 | byte `` | ❌ | width mismatch |
| 9 | byte | int16 `` | ❌ | width mismatch |
| 10 | int16 | byte `` | ❌ | width mismatch |
| 11 | byte | byte `` | ✅ |  |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

