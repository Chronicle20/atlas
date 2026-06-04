# FieldEffectRewardRullet (← `CField::OnFieldEffect#RewardRullet`)

- **IDA:** 0x53bba4
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 7; reward roulette)` | ✅ |  |
| 1 | int32 | int32 `nRewardJobIdx (v23)` | ✅ |  |
| 2 | int32 | int32 `nRewardPartIdx (v24)` | ✅ |  |
| 3 | int32 | int32 `nRewardLevIdx (v25)` | ✅ |  |

