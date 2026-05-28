# PetMovementRequest (← `CVecCtrlPet::EndUpdateActive`)

- **IDA:** 0xa558b6
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/movement.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ❌ | width mismatch |
| 1 | int16 | bytes `CMovePath::Flush body` | ❌ | width mismatch |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

