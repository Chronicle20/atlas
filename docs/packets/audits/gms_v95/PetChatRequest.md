# PetChatRequest (← `CPet::DoAction`)

- **IDA:** 0x6a2340
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes — _LARGE_INTEGER) from this->m_liPetLockerSN` | ❌ | width mismatch |
| 1 | int32 | byte `action type` | ❌ | width mismatch |
| 2 | byte | byte `action no` | ✅ |  |
| 3 | byte | string `chat text (ZXString)` | ❌ | width mismatch |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

