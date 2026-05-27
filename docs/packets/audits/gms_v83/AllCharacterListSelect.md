# AllCharacterListSelect (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect`)

- **IDA:** 0x5f76ae
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID (m_bLoginOpt == 2/3 branch)` | ✅ |  |
| 1 | int32 | int32 `m_anWorldID (int32)` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

