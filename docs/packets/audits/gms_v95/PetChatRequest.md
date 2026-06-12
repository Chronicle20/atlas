# PetChatRequest (← `CPet::DoAction`)

- **IDA:** 0x6a2340
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes _LARGE_INTEGER)` | ✅ |  |
| 1 | int32 | int32 `updateTime/remote tick (v95+)` | ✅ |  |
| 2 | byte | byte `nType (action type)` | ✅ |  |
| 3 | byte | byte `nAction (action no)` | ✅ |  |
| 4 | string | string `chat text (ZXString)` | ✅ |  |

