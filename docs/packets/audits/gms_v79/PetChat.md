# PetChat (← `CPet::OnAction`)

- **IDA:** 0x690eec
- **Atlas file:** `libs/atlas-packet/pet/clientbound/chat.go`
- **Variant:** GMS/v79
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

