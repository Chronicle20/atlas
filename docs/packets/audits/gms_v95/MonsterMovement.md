# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6521e0
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNotChangeAction` | ✅ |  |
| 3 | byte | byte `bNextAttackPossible` | ✅ |  |
| 4 | byte | byte `bLeft (action-byte: low bit = direction, high bits = action+flags)` | ✅ |  |
| 5 | int16 | int32 `sEffect.m_Data (skill effect id+level packed; LOBYTE=skillId, BYTE1=level)` | ❌ | width mismatch |
| 6 | int16 | int32 `multiTargetForBall count (nCount)` | ❌ | width mismatch |
| 7 | byte | int32 `multiTargetForBall[i].x — per entry, loop nCount times` | 🔍 | sub-struct: multiTargets — see _substruct/ |
| 8 | byte | int32 `multiTargetForBall[i].y — per entry` | 🔍 | sub-struct: randTimeForAreaAttack — see _substruct/ |
| 9 | int32 | int32 `randTimeForAreaAttack count (nCount)` | ✅ |  |
| 10 | byte | int32 `randTimeForAreaAttack[i] — per entry, loop nCount times` | ❌ | width mismatch |
| 11 | byte | bytes `Movement body via CMovePath::OnMovePacket (variable-length elements)` | 🔍 | sub-struct: Movement — see _substruct/ |

