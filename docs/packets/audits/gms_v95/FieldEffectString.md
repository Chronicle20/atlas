# FieldEffectString (← `CField::OnFieldEffect#String`)

- **IDA:** 0x53b8b3
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 2/3/4/6; string-payload effect)` | ✅ |  |
| 1 | string | string `name / path / bgm UOL string` | ✅ |  |

