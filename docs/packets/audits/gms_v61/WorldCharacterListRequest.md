# WorldCharacterListRequest (← `CLogin::SendLoginPacket`)

- **IDA:** 0x564dc9
- **Atlas file:** `libs/atlas-packet/login/serverbound/world_character_list_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `worldId @0x564efc (*(BYTE*)v9)` | ✅ |  |
| 1 | byte | byte `channelId @0x564f07 (a3)` | ✅ |  |

