# AllCharacterListSelectWithPic (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelectWithPic`)

- **IDA:** 0x5d7550
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select_with_pic.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (PIC, m_bLoginOpt == 1 branch, opcode 0x1F)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | int32 | int32 `m_anWorldID (int32)` | ✅ |  |
| 3 | string | string `sMacAddress` | ✅ |  |
| 4 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

