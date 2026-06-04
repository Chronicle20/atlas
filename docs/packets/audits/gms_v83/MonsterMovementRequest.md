# MonsterMovementRequest (← `CMob::GenerateMovePath`)

- **IDA:** 0x66b6fc
- **Atlas file:** `../../libs/atlas-packet/monster/serverbound/movement.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | int16 | int16 `nMobCtrlSN (moveId)` | ✅ |  |
| 2 | byte | byte `flags byte (atlas dwFlag — multiBitFlag with nMoveType / bRiseByToss / etc.)` | ✅ |  |
| 3 | byte | byte `nAction (atlas nActionAndDir; packed `(2 * arg0) \| a2 & 1`)` | ✅ |  |
| 4 | int32 | int32 `ti (TARGETINFO — atlas skillData packed; v83 lacks the multiTargetForBall + randTimeForAreaAttack blocks that v87/v95/JMS-v185 carry between actionType and state byte)` | ✅ |  |
| 5 | byte | byte `state byte (atlas moveFlags — packed `(v42 == 0) \| (16 * v43)`)` | ✅ |  |
| 6 | int32 | int32 `hackedCode (v12[288])` | ✅ |  |
| 7 | int32 | int32 `flyCtxTargetX (or 16768460 default when not in dev-mode)` | ✅ |  |
| 8 | int32 | int32 `flyCtxTargetY (or 16768460 default)` | ✅ |  |
| 9 | int16 | bytes `CMovePath::Flush body (Movement elements). v83 lacks the post-Flush bChasing/hasTarget/bChasing2/bChasingHack/tChaseDuration tail that v87+ carry.` | ✅ |  |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

