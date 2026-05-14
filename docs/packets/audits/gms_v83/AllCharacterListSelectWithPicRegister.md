# AllCharacterListSelectWithPicRegister (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelectWithPicRegister`)

- **IDA:** 0x5f76ae
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/all_character_list_select_with_pic_register.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `opt (literal 1u, m_bLoginOpt == 0 branch)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | int32 | int32 `m_anWorldID (int32)` | ✅ |  |
| 3 | string | string `sMacAddress` | ✅ |  |
| 4 | string | string `sMacAddressWithHDDSerial` | ✅ |  |
| 5 | string | string `sSPW (PIC)` | ✅ |  |

