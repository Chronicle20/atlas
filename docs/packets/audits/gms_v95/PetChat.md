# PetChat (← `CPet::OnAction`)

- **IDA:** 0x6a3860
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | byte | byte `action type (v5)` | ✅ |  |
| 3 | byte | byte `action no (v6)` | ✅ |  |
| 4 | string | string `chat text` | ✅ |  |
| 5 | byte | byte `v10 (trailing byte flag)` | ✅ |  |

