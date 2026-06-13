# MonsterMovementRequest (← `CMob::GenerateMovePath`)

- **IDA:** 0x66b6fc
- **Atlas file:** `libs/atlas-packet/monster/serverbound/movement.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | int16 | int16 `nMobCtrlSN (moveId)` | ✅ |  |
| 2 | byte | byte `flags byte (atlas dwFlag — multiBitFlag with nMoveType / bRiseByToss / etc.)` | ✅ |  |
| 3 | byte | byte `nAction (atlas nActionAndDir; packed `(2 * arg0) \| a2 & 1`)` | ✅ |  |
| 4 | int32 | int32 `ti (TARGETINFO — atlas skillData packed; v83 lacks the multiTargetForBall + randTimeForAreaAttack blocks that v87/v95/JMS-v185 carry between actionType and state byte)` | ✅ |  |
| 5 | int32 | byte `state byte (atlas moveFlags — packed `(v42 == 0) \| (16 * v43)`)` | ❌ | width mismatch |
| 6 | int32 | int32 `hackedCode (v12[288])` | ✅ |  |
| 7 | int32 | int32 `flyCtxTargetX (or 16768460 default when not in dev-mode)` | ✅ |  |
| 8 | int32 | int32 `flyCtxTargetY (or 16768460 default)` | ✅ |  |
| 9 | int32 | bytes `CMovePath::Flush body (Movement elements). v83 lacks the post-Flush bChasing/hasTarget/bChasing2/bChasingHack/tChaseDuration tail that v87+ carry.` | ✅ |  |
| 10 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

