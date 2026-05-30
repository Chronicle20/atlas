# FieldKiteSpawn (← `CMessageBoxPool::OnMessageBoxEnterField`)

- **IDA:** 0x694e48
- **Atlas file:** `libs/atlas-packet/field/clientbound/kite_spawn.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMessageBoxID (kite object id, v58 @0x694e6f)` | ✅ |  |
| 1 | int32 | int32 `nItemID (kite template/item id, +56 @0x694eb2)` | ✅ |  |
| 2 | string | string `sMsg (kite message, +8 @0x694ebb)` | ✅ |  |
| 3 | string | string `sCharacterName (owner name, +12 @0x694eea)` | ✅ |  |
| 4 | int16 | int16 `ptMessageBox.x (spawn x, +28 @0x694f20)` | ✅ |  |
| 5 | int16 | int16 `nType (kite type, +32 @0x694f30)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
