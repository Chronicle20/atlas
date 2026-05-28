# CharacterSelectWithPic (← `CLogin::SendSelectCharPacket#CharacterSelectWithPic`)

- **IDA:** 0x5da2a0
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select_with_pic.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (PIC, m_bLoginOpt == 1 branch, opcode 0x1D)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

