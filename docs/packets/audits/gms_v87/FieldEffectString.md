# FieldEffectString (← `CField::OnFieldEffect#String`)

- **IDA:** 0x55a9fb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 2/3/4/6; string-payload effect)` | ✅ |  |
| 1 | string | string `name / path / bgm UOL string` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
