# PetChat (← `CPet::OnAction`)

- **IDA:** 0x613543
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/chat.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | string `` | ❌ | width mismatch |
| 3 | byte | byte `` | ✅ |  |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

