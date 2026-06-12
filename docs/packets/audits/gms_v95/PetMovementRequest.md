# PetMovementRequest (← `CVecCtrlPet::EndUpdateActive`)

- **IDA:** 0x99f5a0
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes — _LARGE_INTEGER) from owner ZRef` | ✅ |  |
| 1 | bytes | bytes `CMovePath::Flush body (variable-length movement elements)` | ✅ |  |

