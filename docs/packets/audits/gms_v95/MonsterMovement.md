# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x6521e0
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

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
| 7 | int32 | int32 `multiTargetForBall[i].x — per entry, loop nCount times` | ✅ |  |
| 8 | int32 | int32 `multiTargetForBall[i].y — per entry` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack count (nCount)` | ✅ |  |
| 10 | int32 | int32 `randTimeForAreaAttack[i] — per entry, loop nCount times` | ✅ |  |
| 11 | int32 | bytes `Movement body via CMovePath::OnMovePacket (variable-length elements)` | ❌ | width mismatch |
| 12 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

