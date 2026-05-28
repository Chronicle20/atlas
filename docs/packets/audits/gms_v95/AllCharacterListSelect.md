# AllCharacterListSelect (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect`)

- **IDA:** 0x5d7550
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID (m_bLoginOpt == 2/3 branch, opcode 0x0E)` | ✅ |  |
| 1 | int32 | int32 `m_anWorldID (int32)` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

