# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6a6cb3
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNotChangeAction (v87: present — gated GMS>83)` | ✅ |  |
| 3 | byte | byte `bNextAttackPossible` | ✅ |  |
| 4 | byte | byte `bLeft (action+flags)` | ✅ |  |
| 5 | int32 | int32 `sEffect.m_Data (skill effect id+level packed)` | ✅ |  |
| 6 | int32 | int32 `multiTargetForBall count` | ✅ |  |
| 7 | int32 | int32 `multiTargetForBall[i].x — loop` | ✅ |  |
| 8 | int32 | int32 `multiTargetForBall[i].y — loop` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 10 | int32 | int32 `randTimeForAreaAttack[i] — loop` | ✅ |  |
| 11 | int32 | bytes `Movement body via CMovePath::OnMovePacket` | ✅ |  |
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

