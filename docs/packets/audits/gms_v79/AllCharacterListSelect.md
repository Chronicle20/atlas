# AllCharacterListSelect (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect`)

- **IDA:** 0x5ccc1f
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId (v79 sub_5CCC1F@0x5ccc1f)` | ✅ |  |
| 1 | int32 | int32 `worldId (dwWorldID)` | ✅ |  |
| 2 | string | string `sMacAddress (GetLocalMacAddress)` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial (GetLocalMacAddressWithHDDSerialNo)` | ✅ |  |

