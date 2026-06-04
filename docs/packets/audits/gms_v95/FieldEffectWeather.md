# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x5468f0
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `m_nBlowType (atlas !active; 0=start admin-weather w/ message, 1=item/end)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `itemId (v4)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `message (only when itemId!=0 && m_nBlowType==0; start path)` | ❌ | atlas: short — missing trailing field |

