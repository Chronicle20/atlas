# KeyMapChange (← `CFuncKeyMappedMan::SaveFuncKeyMap`)

- **IDA:** 0x568a60
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/key_map_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `mode (0 = key mapping change)` | ✅ |  |
| 1 | int32 | int32 `count (number of changed key slot indices); per-entry: Encode4(keyId) + Encode1(nType) + Encode4(nID) via FUNCKEY_MAPPED::Encode@0x4f6d80` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

