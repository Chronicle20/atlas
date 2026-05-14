# CharacterMovement (← `CUserRemote::OnMove`)

- **IDA:** 0x948a80
- **Atlas file:** `libs/atlas-packet/character/clientbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

---

ack: descent gap + dispatcher-layer offset — CUserRemote::OnMove@0x948a80 immediately delegates to CMovePath::OnMovePacket (no direct decode calls visible). The IDA entry has 0 calls so all atlas fields appear as extra. Additionally the dispatcher reads characterId before calling OnMove, creating a +1 offset. Atlas CharacterMovement.Encode delegates to model.Movement which round-trips correctly; no wire bug detected.
