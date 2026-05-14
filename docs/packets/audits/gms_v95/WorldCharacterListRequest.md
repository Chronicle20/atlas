# WorldCharacterListRequest (← `CLogin::SendLoginPacket`)

- **IDA:** 0x5dbef0
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/world_character_list_request.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `gameStartMode (modified-client path: nGameStartMode != 1)` | ✅ |  |
| 1 | byte | byte `nWorldID` | ✅ |  |
| 2 | byte | byte `nChannelID` | ✅ |  |
| 3 | int32 | int32 `socket addr (S_addr from getsockname)` | ✅ |  |

