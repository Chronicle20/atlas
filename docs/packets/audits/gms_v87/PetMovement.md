# PetMovement (← `CPet::OnMove`)

- **IDA:** 0x74842a
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/movement.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | bytes | bytes `Movement body via CMovePath::OnMovePacket` | ✅ |  |

