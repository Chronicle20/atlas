# CharacterSelectRegisterPic (← `CLogin::SendSelectCharPacket#CharacterSelectRegisterPic`)

- **IDA:** 0x5f726d
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/character_select_register_pic.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (literal 1u, m_bLoginOpt == 0 branch)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |
| 4 | string | string `sSPW (PIC)` | ✅ |  |

