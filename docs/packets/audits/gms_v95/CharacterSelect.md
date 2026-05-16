# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x5da2a0
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId` | ✅ |  |
| 1 | string | string `sMacAddress` | ✅ |  |
| 2 | string | string `sMacAddressWithHDDSerial` | ✅ |  |

