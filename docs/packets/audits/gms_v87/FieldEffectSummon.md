# FieldEffectSummon (← `CField::OnFieldEffect#Summon`)

- **IDA:** 0x55a948
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (effect-type discriminator = 0; summon)` | ✅ |  |
| 1 | byte | byte `effect (v4; summon animation index, @0x55a948)` | ✅ |  |
| 2 | int32 | int32 `x (a2, @0x55a952)` | ✅ |  |
| 3 | int32 | int32 `y (v5, @0x55a95c)` | ✅ |  |

