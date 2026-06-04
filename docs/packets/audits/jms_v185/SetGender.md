# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

