# FieldEffectString (← `CField::OnFieldEffect#String`)

- **IDA:** 0x5174bb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `name (case 2) @0x5175c0` | ❌ | width mismatch |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

