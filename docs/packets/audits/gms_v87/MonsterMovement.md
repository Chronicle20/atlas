# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6a6cb3
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNotChangeAction (v87: present — gated GMS>83)` | ✅ |  |
| 3 | byte | byte `bNextAttackPossible` | ✅ |  |
| 4 | byte | byte `bLeft (action+flags)` | ✅ |  |
| 5 | int16 | int32 `sEffect.m_Data (skill effect id+level packed)` | ❌ | width mismatch |
| 6 | int16 | int32 `multiTargetForBall count` | ❌ | width mismatch |
| 7 | byte | int32 `multiTargetForBall[i].x — loop` | 🔍 | sub-struct: multiTargets — see _substruct/ |
| 8 | byte | int32 `multiTargetForBall[i].y — loop` | 🔍 | sub-struct: randTimeForAreaAttack — see _substruct/ |
| 9 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 10 | byte | int32 `randTimeForAreaAttack[i] — loop` | ❌ | width mismatch |
| 11 | byte | bytes `Movement body via CMovePath::OnMovePacket` | 🔍 | sub-struct: Movement — see _substruct/ |

