# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x5ccae3
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId (v79 sub_5CCAE3@0x5ccae3)` | ✅ |  |
| 1 | string | string `sMacAddress (GetLocalMacAddress)` | ✅ |  |
| 2 | string | string `sMacAddressWithHDDSerial (GetLocalMacAddressWithHDDSerialNo)` | ✅ |  |

