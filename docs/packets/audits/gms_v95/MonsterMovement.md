# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6521e0
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNotChangeAction` | ✅ |  |
| 3 | byte | byte `bNextAttackPossible` | ✅ |  |
| 4 | byte | byte `bLeft (action-byte: low bit = direction, high bits = action+flags)` | ✅ |  |
| 5 | int32 | int32 `sEffect.m_Data (skill effect id+level packed; LOBYTE=skillId, BYTE1=level)` | ✅ |  |
| 6 | int32 | int32 `multiTargetForBall count (nCount)` | ✅ |  |
| 7 | int32 | int32 `multiTargetForBall[i].x — per entry, loop nCount times` | ✅ |  |
| 8 | int32 | int32 `multiTargetForBall[i].y — per entry` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack count (nCount)` | ✅ |  |
| 10 | int32 | int32 `randTimeForAreaAttack[i] — per entry, loop nCount times` | ✅ |  |
| 11 | int32 | bytes `Movement body via CMovePath::OnMovePacket (variable-length elements)` | ✅ |  |
| 12 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

