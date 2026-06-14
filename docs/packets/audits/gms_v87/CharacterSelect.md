# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x62e9f6
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID (m_bLoginOpt==0/2/3 branch, opcode 0x13)` | ✅ |  |
| 1 | string | string `sMacAddress` | ✅ |  |
| 2 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

