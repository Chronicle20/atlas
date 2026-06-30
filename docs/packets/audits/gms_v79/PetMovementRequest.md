# PetMovementRequest (← `CVecCtrlPet::EndUpdateActive`)

- **IDA:** 0x9150a1
- **Atlas file:** `libs/atlas-packet/pet/serverbound/movement.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `` | ✅ |  |
| 1 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 2 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 3 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

