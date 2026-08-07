# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0x7cbd86
- **Atlas file:** `libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | byte | int32 `` | ❌ | width mismatch |
| 6 | int32 | byte `` | ❌ | width mismatch |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | int32 | bytes `` | ✅ |  |
| 12 | byte | byte `` | ✅ |  |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | int32 | byte `` | ❌ | width mismatch |
| 15 | int32 | bytes `` | ✅ |  |
| 16 | byte | bytes `` | ✅ |  |
| 17 | byte | int32 `` | ❌ | width mismatch |
| 18 | byte | byte `` | ✅ |  |
| 19 | int32 | bytes `` | ✅ |  |
| 20 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

