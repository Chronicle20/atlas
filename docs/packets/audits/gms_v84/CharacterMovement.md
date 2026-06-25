# CharacterMovement (← `CUserRemote::OnMove`)

- **IDA:** 0x9b26cd
- **Atlas file:** `libs/atlas-packet/character/clientbound/movement.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (read by CUserPool::OnUserRemotePacket before dispatch)` | ✅ |  |
| 1 | bytes | bytes `CMovePath::OnMovePacket — opaque movement path block` | ✅ |  |

