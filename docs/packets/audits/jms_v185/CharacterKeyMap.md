# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x5e79aa
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resetToDefault flag` | ✅ |  |
| 1 | byte | byte `FUNCKEY_MAPPED::nType (loop body)` | ✅ |  |
| 2 | byte | int32 `FUNCKEY_MAPPED::nID (loop body)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

