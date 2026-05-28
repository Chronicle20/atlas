# PetMovement (← `CPet::OnMove`)

- **IDA:** 0x69fb60
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | int32 | bytes `Movement body via CMovePath::OnMovePacket (variable-length elements)` | ❌ | width mismatch |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

