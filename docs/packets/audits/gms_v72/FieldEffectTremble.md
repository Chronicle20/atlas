# FieldEffectTremble (← `CField::OnFieldEffect#Tremble`)

- **IDA:** 0x5174bb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bHeavyNShortTremble (case 1) @0x5177a5` | ✅ |  |
| 1 | byte | int32 `delay @0x5177a8` | ❌ | width mismatch |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

