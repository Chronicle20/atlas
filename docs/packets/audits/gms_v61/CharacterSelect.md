# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x564f79
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId @0x565003` | ✅ |  |
| 1 | string | string `mac = GetLocalMacAddress @0x56503e` | ✅ |  |
| 2 | string | string `hwid = GetLocalMacAddressWithHDDSerialNo @0x565074` | ✅ |  |

