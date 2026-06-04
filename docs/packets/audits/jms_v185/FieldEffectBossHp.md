# FieldEffectBossHp (← `CField::OnFieldEffect#BossHp`)

- **IDA:** 0x570359
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 5; boss HP tag)` | ✅ |  |
| 1 | int32 | int32 `monsterId (v32 @line110)` | ✅ |  |
| 2 | int32 | int32 `currentHp (sName @line111)` | ✅ |  |
| 3 | int32 | int32 `maxHp (result @line112)` | ✅ |  |
| 4 | byte | byte `tagColor (v14 @line113)` | ✅ |  |
| 5 | byte | byte `tagBackgroundColor (v15 @line114)` | ✅ |  |

