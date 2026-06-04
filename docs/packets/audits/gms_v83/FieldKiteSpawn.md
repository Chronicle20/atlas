# FieldKiteSpawn (← `CMessageBoxPool::OnMessageBoxEnterField`)

- **IDA:** 0x65acdf
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/kite_spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMessageBoxID (kite object id, v59)` | ✅ |  |
| 1 | int32 | int32 `nItemID (kite template/item id, +56)` | ✅ |  |
| 2 | string | string `sMsg (kite message)` | ✅ |  |
| 3 | string | string `sCharacterName (owner name)` | ✅ |  |
| 4 | int16 | int16 `ptMessageBox.x (spawn x, +28)` | ✅ |  |
| 5 | int16 | int16 `nType/y (spawn y or kite type, +32)` | ✅ |  |

