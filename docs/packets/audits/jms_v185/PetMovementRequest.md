# PetMovementRequest (← `CVecCtrlPet::EndUpdateActive`)

- **IDA:** 0xaa25ab
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/movement.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ✅ |  |
| 1 | bytes | bytes `CMovePath::Flush body` | ✅ |  |

