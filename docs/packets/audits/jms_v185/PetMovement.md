# PetMovement (← `CPet::OnMove`)

- **IDA:** 0x76a534
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/movement.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `slot — read by dispatcher` | ✅ |  |
| 2 | bytes | bytes `Movement body` | ✅ |  |

