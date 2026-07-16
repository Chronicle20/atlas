# AllCharacterListSelect (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect`)

- **IDA:** 0x5650b6
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId @0x565201` | ✅ |  |
| 1 | int32 | int32 `worldId @0x565218` | ✅ |  |
| 2 | string | string `mac = GetLocalMacAddress @0x565253` | ✅ |  |
| 3 | string | string `hwid = GetLocalMacAddressWithHDDSerialNo @0x565289` | ✅ |  |

