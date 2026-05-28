# FieldEffectBossHp (← `CField::OnFieldEffect#BossHp`)

- **IDA:** 0x5330f7
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 5; boss HP tag)` | ✅ |  |
| 1 | int32 | int32 `monsterId (v38)` | ✅ |  |
| 2 | int32 | int32 `currentHp (v44)` | ✅ |  |
| 3 | int32 | int32 `maxHp (v42)` | ✅ |  |
| 4 | byte | byte `tagColor (v17)` | ✅ |  |
| 5 | byte | byte `tagBackgroundColor (v18)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
