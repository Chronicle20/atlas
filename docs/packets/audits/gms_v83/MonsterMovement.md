# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x66be61
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNextAttackPossible (v83: no bNotChangeAction between these two)` | ✅ |  |
| 3 | byte | byte `bLeft (action+flags)` | ✅ |  |
| 4 | int32 | int32 `sEffect.m_Data (skill effect id+level packed)` | ✅ |  |
| 5 | int32 | bytes `Movement body via CMovePath::OnMovePacket — v83 lacks the multiTargetForBall / randTimeForAreaAttack loops present in v95+` | ✅ |  |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

