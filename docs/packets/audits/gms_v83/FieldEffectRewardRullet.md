# FieldEffectRewardRullet (← `CField::OnFieldEffect#RewardRullet`)

- **IDA:** 0x5330f7
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 7; reward roulette)` | ✅ |  |
| 1 | int32 | int32 `nRewardJobIdx (v24)` | ✅ |  |
| 2 | int32 | int32 `nRewardPartIdx (v25)` | ✅ |  |
| 3 | int32 | int32 `nRewardLevIdx (v26)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
