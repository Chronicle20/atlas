# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x535179
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `m_nBlowType (m_aSwimRect; 0=start admin-weather w/ message, 1=item/end, v3)` | ✅ |  |
| 1 | int32 | int32 `itemId (v5)` | ✅ |  |
| 2 | string | string `message (only when itemId!=0 && blowType==0; start path)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
