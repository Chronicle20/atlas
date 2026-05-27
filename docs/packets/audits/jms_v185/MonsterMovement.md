# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6e955a
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNotChangeAction (JMS: present per atlas `\|\| JMS` gate)` | ✅ |  |
| 3 | byte | byte `bNextAttackPossible` | ✅ |  |
| 4 | byte | byte `bLeft` | ✅ |  |
| 5 | int16 | int32 `sEffect.m_Data` | ❌ | width mismatch |
| 6 | int16 | int32 `multiTargetForBall count` | ❌ | width mismatch |
| 7 | byte | int32 `multiTargetForBall[i].x` | 🔍 | sub-struct: multiTargets — see _substruct/ |
| 8 | byte | int32 `multiTargetForBall[i].y` | 🔍 | sub-struct: randTimeForAreaAttack — see _substruct/ |
| 9 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 10 | byte | int32 `randTimeForAreaAttack[i]` | ❌ | width mismatch |
| 11 | byte | bytes `Movement body` | 🔍 | sub-struct: Movement — see _substruct/ |

