# FieldEffectTremble (← `CField::OnFieldEffect#Tremble`)

- **IDA:** 0x5330f7
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 1; tremble)` | ✅ |  |
| 1 | byte | byte `bHeavyNShortTremble (v22; bool)` | ✅ |  |
| 2 | int32 | int32 `delay (v23)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
