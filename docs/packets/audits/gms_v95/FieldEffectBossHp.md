# FieldEffectBossHp (← `CField::OnFieldEffect#BossHp`)

- **IDA:** 0x53b9c1
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 5; boss HP tag)` | ✅ |  |
| 1 | int32 | int32 `monsterId (v12)` | ✅ |  |
| 2 | int32 | int32 `currentHp (v13)` | ✅ |  |
| 3 | int32 | int32 `maxHp (nMaxHP)` | ✅ |  |
| 4 | byte | byte `tagColor (nColor)` | ✅ |  |
| 5 | byte | byte `tagBackgroundColor` | ✅ |  |

