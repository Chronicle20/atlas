# PetChatRequest (← `CPet::DoAction`)

- **IDA:** 0x76b3a0
- **Atlas file:** `libs/atlas-packet/pet/serverbound/chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ❌ | width mismatch |
| 1 | byte | byte `action type` | ✅ |  |
| 2 | byte | byte `action no` | ✅ |  |
| 3 | string | string `chat text` | ✅ |  |

