# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x55c953
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `m_nBlowType (v3 @0x55c967; atlas !active; 0=start admin-weather w/ message, 1=item/end)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `itemId (v5, @0x55c97f)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `message (only when itemId!=0 && m_nBlowType==0; start path, @0x55c99a)` | ❌ | atlas: short — missing trailing field |

