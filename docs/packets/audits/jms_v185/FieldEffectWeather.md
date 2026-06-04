# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x5723E6
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId (v2 @line17) — JMS reads itemId FIRST; NO leading blow-type byte` | ✅ |  |
| 1 | int32 | int32 `extra int — ONLY if get_consume_cash_item_type(itemId)==51 (@line22; cash-weather variant); no atlas field` | ✅ |  |
| 2 | string | string `message — only if itemId!=0 (@line27)` | ✅ |  |

