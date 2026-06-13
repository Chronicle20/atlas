# MonsterMovement (← `CMob::OnMove`)

- **IDA:** 0x66be61
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `bNotForceLandingWhenDiscard` | ✅ |  |
| 2 | byte | byte `bNextAttackPossible (v83: no bNotChangeAction between these two)` | ✅ |  |
| 3 | byte | byte `bLeft (action+flags)` | ✅ |  |
| 4 | byte | int32 `sEffect.m_Data (skill effect id+level packed)` | ❌ | width mismatch |
| 5 | int16 | bytes `Movement body via CMovePath::OnMovePacket — v83 lacks the multiTargetForBall / randTimeForAreaAttack loops present in v95+` | ✅ |  |
| 6 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 22 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 23 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 24 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

