# MonsterMovementRequest (← `CMob::GenerateMovePath`)

- **IDA:** 0x651100
- **Atlas file:** `libs/atlas-packet/monster/serverbound/movement.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | int16 | int16 `nMobCtrlSN (moveId)` | ✅ |  |
| 2 | byte | byte `flags byte (atlas dwFlag — multiBitFlag with nMoveType / bRiseByToss / etc.)` | ✅ |  |
| 3 | byte | byte `nAction (atlas nActionAndDir)` | ✅ |  |
| 4 | int32 | int32 `ti (TARGETINFO — atlas skillData packed)` | ✅ |  |
| 5 | int32 | int32 `multiTargetForBall count` | ✅ |  |
| 6 | int32 | int32 `multiTargetForBall[i].x — loop` | ✅ |  |
| 7 | int32 | int32 `multiTargetForBall[i].y — loop` | ✅ |  |
| 8 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack[i] — loop` | ✅ |  |
| 10 | byte | byte `state byte (atlas moveFlags)` | ✅ |  |
| 11 | int32 | int32 `hackedCode` | ✅ |  |
| 12 | int32 | int32 `flyCtxTargetX (or 16768460 default)` | ✅ |  |
| 13 | int32 | int32 `flyCtxTargetY (or 16768460 default)` | ✅ |  |
| 14 | int32 | int32 `hackedCodeCRC (fall start CRC)` | ✅ |  |
| 15 | int16 | bytes `CMovePath::Flush body (Movement elements)` | ❌ | width mismatch |
| 16 | int16 | byte `bChasing` | ❌ | width mismatch |
| 17 | byte | byte `hasTarget` | ✅ |  |
| 18 | byte | byte `bChasing2 (ladder)` | 🔍 | sub-struct: MovementCodec — see _substruct/ |
| 19 | byte | byte `bChasingHack (zMass)` | ✅ |  |
| 20 | byte | int32 `tChaseDuration` | ❌ | width mismatch |
| 21 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

