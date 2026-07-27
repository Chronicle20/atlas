# FieldEffectSummon (← `CField::OnFieldEffect#Summon`)

- **IDA:** 0x5174bb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `effect (case 0) @0x5174f3` | ✅ |  |
| 1 | byte | int32 `x @0x5174fd` | ❌ | width mismatch |
| 2 | int32 | int32 `y @0x517507` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

