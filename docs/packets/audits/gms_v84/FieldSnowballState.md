# FieldSnowballState (← `CField_SnowBall::OnSnowBallState`)

- **IDA:** 0x584a1c
- **Atlas file:** `libs/atlas-packet/field/clientbound/snowball_state.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `state` | ✅ |  |
| 1 | int32 | int32 `leftSnowmanHp m_aSnowMan[0].m_nHP` | ✅ |  |
| 2 | int32 | int32 `rightSnowmanHp m_aSnowMan[1].m_nHP` | ✅ |  |
| 3 | int16 | int16 `snowball0 x` | ✅ |  |
| 4 | byte | byte `snowball0 y` | ✅ |  |
| 5 | int16 | int16 `snowball1 x` | ✅ |  |
| 6 | byte | byte `snowball1 y` | ✅ |  |
| 7 | int16 | int16 `damageSnowBall (first-gated)` | ✅ |  |
| 8 | int16 | int16 `damageSnowMan0 (first-gated)` | ✅ |  |
| 9 | int16 | int16 `damageSnowMan1 (first-gated)` | ✅ |  |

