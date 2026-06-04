# MonsterMovementRequest (← `CMob::GenerateMovePath`)

- **IDA:** 0x6e8892
- **Atlas file:** `../../libs/atlas-packet/monster/serverbound/movement.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | int16 | int16 `nMobCtrlSN (moveId)` | ✅ |  |
| 2 | byte | byte `flags byte: nMoveEndingX \| (v109 << 1) \| (bRiseByToss << 2) \| (nMoveTypeb << 3) — atlas dwFlag` | ✅ |  |
| 3 | byte | byte `nAction (atlas nActionAndDir)` | ✅ |  |
| 4 | int32 | int32 `ti (TARGETINFO — atlas skillData; LOBYTE=skillId, BYTE1=skillLevel)` | ✅ |  |
| 5 | int32 | int32 `multiTargetForBall count` | ✅ |  |
| 6 | int32 | int32 `multiTargetForBall[i].x — loop` | ✅ |  |
| 7 | int32 | int32 `multiTargetForBall[i].y — loop` | ✅ |  |
| 8 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack[i] — loop` | ✅ |  |
| 10 | byte | byte `state byte: (nTyped == 0) \| (16 * (v76 != 0)) — atlas moveFlags` | ✅ |  |
| 11 | int32 | int32 `TSecType<long>.GetData(sub_6E9537) — atlas hackedCode` | ✅ |  |
| 12 | int32 | int32 `v107 ? TSecType<long>.GetData(v14[2]...+12) : 16768460 — atlas flyCtxTargetX` | ✅ |  |
| 13 | int32 | int32 `v107 ? TSecType<long>.GetData(v14[2]...) : 16768460 — atlas flyCtxTargetY` | ✅ |  |
| 14 | int32 | int32 `sub_515C39(v14[2].m_pfhFallStart) — atlas hackedCodeCRC` | ✅ |  |
| 15 | int16 | bytes `CMovePath::Flush body (Movement elements, variable length)` | ✅ |  |
| 16 | int16 | byte `bChasing (sub_68664B of m_bChasing)` | ❌ | width mismatch |
| 17 | byte | byte `hasTarget (LODWORD(v14[1].m_ap._ZtlSecureTear_vx[1]) != 0)` | ✅ |  |
| 18 | byte | byte `bChasing2 (m_pLadderOrRope flag)` | 🔍 | sub-struct: MovementCodec — see _substruct/ |
| 19 | byte | byte `bChasingHack (m_lZMass flag)` | ✅ |  |
| 20 | byte | int32 `tChaseDuration (vy / fall velocity)` | ❌ | width mismatch |
| 21 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

