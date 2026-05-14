# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x58ddb4
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resetToDefault flag (non-zero = reset to defaults; zero = read key bindings from packet)` | ✅ |  |
| 1 | byte | byte `FUNCKEY_MAPPED::nType (key type byte; loop 89 entries in v83 vs 90 in v95, only if resetToDefault==0)` | ✅ |  |
| 2 | byte | int32 `FUNCKEY_MAPPED::nID (key action int32; loop 89 entries in v83, only if resetToDefault==0)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

