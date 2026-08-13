# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x561000
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | bytes `` | ✅ |  |
| 2 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 3 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

