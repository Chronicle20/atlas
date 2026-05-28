# CharacterSelectWithPic (← `CLogin::SendSelectCharPacket#CharacterSelectWithPic`)

- **IDA:** 0x62e9f6
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/character_select_with_pic.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `mode (literal 1u, m_bLoginOpt==0 else path, opcode 0x1D)` | ❌ | width mismatch |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |
| 4 | byte | string `sSPW (PIC) — appended last in v87 0x1D branch` | ❌ | atlas: short — missing trailing field |

