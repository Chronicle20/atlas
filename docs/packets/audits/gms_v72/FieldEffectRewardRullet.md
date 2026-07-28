# FieldEffectRewardRullet (← `CField::OnFieldEffect#RewardRullet`)

- **IDA:** 0x5174bb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `nRewardJobIdx (case 7) @0x5177d4` | ❌ | width mismatch |
| 1 | int32 | int32 `nRewardPartIdx @0x5177dd` | ✅ |  |
| 2 | int32 | int32 `nRewardLevIdx @0x5177df` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

