# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x5723E6
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `itemId (v2 @line17) — JMS reads itemId FIRST; NO leading blow-type byte` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `extra int — ONLY if get_consume_cash_item_type(itemId)==51 (@line22; cash-weather variant); no atlas field` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `message — only if itemId!=0 (@line27)` | ❌ | atlas: short — missing trailing field |

