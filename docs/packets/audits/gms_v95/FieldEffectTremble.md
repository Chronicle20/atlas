# FieldEffectTremble (← `CField::OnFieldEffect#Tremble`)

- **IDA:** 0x53bb74
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 1; tremble)` | ✅ |  |
| 1 | byte | byte `bHeavyNShortTremble (v21; bool)` | ✅ |  |
| 2 | int32 | int32 `delay (v22)` | ✅ |  |

