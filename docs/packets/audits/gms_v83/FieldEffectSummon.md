# FieldEffectSummon (← `CField::OnFieldEffect#Summon`)

- **IDA:** 0x5330f7
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (effect-type discriminator = 0; summon)` | ✅ |  |
| 1 | byte | byte `effect (v4; summon animation index)` | ✅ |  |
| 2 | int32 | int32 `x (v42)` | ✅ |  |
| 3 | int32 | int32 `y (v5)` | ✅ |  |

