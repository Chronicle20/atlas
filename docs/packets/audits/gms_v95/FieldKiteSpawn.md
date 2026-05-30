# FieldKiteSpawn (← `CMessageBoxPool::OnMessageBoxEnterField`)

- **IDA:** 0x6369c0
- **Atlas file:** `libs/atlas-packet/field/clientbound/kite_spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMessageBoxID (kite object id)` | ✅ |  |
| 1 | int32 | int32 `nItemID (kite template/item id)` | ✅ |  |
| 2 | string | string `sMsg (kite message)` | ✅ |  |
| 3 | string | string `sCharacterName (owner name)` | ✅ |  |
| 4 | int16 | int16 `ptMessageBox.x (spawn x)` | ✅ |  |
| 5 | int16 | int16 `nType (kite type)` | ✅ |  |


Ack: world-audit Phase 2c on 2026-05-28
