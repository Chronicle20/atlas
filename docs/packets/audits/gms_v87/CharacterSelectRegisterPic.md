# CharacterSelectRegisterPic (← `CLogin::SendSelectCharPacket#CharacterSelectRegisterPic`)

- **IDA:** 0x62e9f6
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/character_select_register_pic.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `sSPW (PIC, v4==1 branch, opcode 0x1E)` | ❌ | width mismatch |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | string | string `sMacAddress` | ✅ |  |
| 3 | string | string `sMacAddressWithHDDSerial` | ✅ |  |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

