# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0x9c3a1c
- **Atlas file:** `libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | byte | bytes `` | ✅ |  |
| 6 | int32 | bytes `` | ✅ |  |
| 7 | byte | int32 `` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | byte | bytes `` | ✅ |  |
| 10 | byte | bytes `` | ✅ |  |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | byte | byte `` | ✅ |  |
| 13 | int32 | int32 `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | int32 | int32 `` | ✅ |  |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

