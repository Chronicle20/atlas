# WorldCharacterListRequest (← `CLogin::SendLoginPacket`)

- **IDA:** 0x5cc905
- **Atlas file:** `libs/atlas-packet/login/serverbound/world_character_list_request.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nWorldID (v79 sub_5CC905@0x5cc905 *v9 byte; no gameStartMode below v83)` | ✅ |  |
| 1 | byte | byte `nChannelID (a3)` | ✅ |  |
| 2 | int32 | int32 `socket addr (S_addr from getsockname)` | ✅ |  |

