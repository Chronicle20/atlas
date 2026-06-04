# FieldEffectRewardRullet (← `CField::OnFieldEffect#RewardRullet`)

- **IDA:** 0x55abea
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 7; reward roulette)` | ✅ |  |
| 1 | int32 | int32 `nRewardJobIdx (v20, @0x55abea)` | ✅ |  |
| 2 | int32 | int32 `nRewardPartIdx (v21, @0x55abf3)` | ✅ |  |
| 3 | int32 | int32 `nRewardLevIdx (v22, @0x55abf5)` | ✅ |  |

