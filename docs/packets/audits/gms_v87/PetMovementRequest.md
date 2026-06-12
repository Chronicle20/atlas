# PetMovementRequest (← `CVecCtrlPet::EndUpdateActive`)

- **IDA:** 0xa558b6
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/movement.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ✅ |  |
| 1 | bytes | bytes `CMovePath::Flush body` | ✅ |  |

