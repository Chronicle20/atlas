# AllCharacterListRequest (← `CLogin::SendViewAllCharPacket`)

- **IDA:** 0x6324e3
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_request.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `gameStartMode (m_nGameStartMode)` | ✅ |  |
| 1 | string | string `nexonPassport — only when gameStartMode==1` | ✅ |  |
| 2 | bytes | bytes `machineId (16 bytes) — only when gameStartMode==1` | ✅ |  |
| 3 | int32 | int32 `gameRoomClient — only when gameStartMode==1` | ✅ |  |
| 4 | byte | byte `gameStartMode (echoed) — only when gameStartMode==1` | ✅ |  |

