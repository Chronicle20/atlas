# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x5bd279
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resetToDefault flag` | ✅ |  |
| 1 | byte | byte `FUNCKEY_MAPPED::nType (key type byte; loop 89 entries — v87=89, same as v83; v95=90)` | ✅ |  |
| 2 | byte | int32 `FUNCKEY_MAPPED::nID (key action int32; loop 89 entries)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

