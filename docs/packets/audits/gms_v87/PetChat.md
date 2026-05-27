# PetChat (← `CPet::OnAction`)

- **IDA:** 0x74844b
- **Atlas file:** `libs/atlas-packet/pet/clientbound/chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | byte | byte `action type` | ✅ |  |
| 3 | byte | byte `action no` | ✅ |  |
| 4 | string | string `chat text` | ✅ |  |
| 5 | byte | byte `trailing byte flag` | ✅ |  |

