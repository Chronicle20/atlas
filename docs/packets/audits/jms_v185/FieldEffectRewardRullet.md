# FieldEffectRewardRullet (← `CField::OnFieldEffect#RewardRullet`)

- **IDA:** 0x570359
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 7; reward roulette)` | ✅ |  |
| 1 | int32 | int32 `nRewardJobIdx (v20 @line137)` | ✅ |  |
| 2 | int32 | int32 `nRewardPartIdx (v21 @line138)` | ✅ |  |
| 3 | int32 | int32 `nRewardLevIdx (v22 @line139)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
