# FieldEffectTremble (← `CField::OnFieldEffect#Tremble`)

- **IDA:** 0x55abbb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 1; tremble)` | ✅ |  |
| 1 | byte | byte `bHeavyNShortTremble (v18; bool, @0x55abbb)` | ✅ |  |
| 2 | int32 | int32 `delay (v19, @0x55abbe)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
