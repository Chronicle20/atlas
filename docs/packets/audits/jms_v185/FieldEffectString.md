# FieldEffectString (← `CField::OnFieldEffect#String`)

- **IDA:** 0x570359
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 2/3/4/6; string-payload effect)` | ✅ |  |
| 1 | string | string `name / path / bgm UOL string` | ✅ |  |


Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
