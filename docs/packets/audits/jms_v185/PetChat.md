# PetChat (← `CPet::OnAction`)

- **IDA:** 0x76a557
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/chat.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `slot — read by dispatcher` | ✅ |  |
| 2 | byte | byte `action type` | ✅ |  |
| 3 | byte | byte `action no` | ✅ |  |
| 4 | string | string `chat text` | ✅ |  |
| 5 | byte | byte `trailing byte flag` | ✅ |  |

