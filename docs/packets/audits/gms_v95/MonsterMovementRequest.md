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
| 5 | byte | int32 `multiTargetForBall count` | 🔍 | sub-struct: multiTargetForBall — see _substruct/ |
| 6 | byte | int32 `multiTargetForBall[i].x — loop` | 🔍 | sub-struct: randTimeForAreaAttack — see _substruct/ |
| 7 | byte | int32 `multiTargetForBall[i].y — loop` | ❌ | width mismatch |
| 8 | int32 | int32 `randTimeForAreaAttack count` | ✅ |  |
| 9 | int32 | int32 `randTimeForAreaAttack[i] — loop` | ✅ |  |
| 10 | int32 | byte `state byte (atlas moveFlags)` | ❌ | width mismatch |
| 11 | int32 | int32 `hackedCode` | ✅ |  |
| 12 | int32 | int32 `flyCtxTargetX (or 16768460 default)` | ✅ |  |
| 13 | byte | int32 `flyCtxTargetY (or 16768460 default)` | ❌ | width mismatch |
| 14 | byte | int32 `hackedCodeCRC (fall start CRC)` | 🔍 | sub-struct: Movement — see _substruct/ |
| 15 | byte | bytes `CMovePath::Flush body (Movement elements)` | ❌ | width mismatch |
| 16 | byte | byte `bChasing` | ✅ |  |
| 17 | byte | byte `hasTarget` | ✅ |  |
| 18 | byte | byte `bChasing2 (ladder)` | ✅ |  |
| 19 | int32 | byte `bChasingHack (zMass)` | ❌ | width mismatch |
| 20 | byte | int32 `tChaseDuration` | ❌ | atlas: short — missing trailing field |

