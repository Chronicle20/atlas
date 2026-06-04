# FieldEffectTremble (← `CField::OnFieldEffect#Tremble`)

- **IDA:** 0x570359
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 1; tremble)` | ✅ |  |
| 1 | byte | byte `bHeavyNShortTremble (v18 @line73)` | ✅ |  |
| 2 | int32 | int32 `delay (v19 @line74)` | ✅ |  |

