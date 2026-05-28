# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x5723E6
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `itemId (v2 @line17) — JMS reads itemId FIRST; NO leading blow-type byte` | ❌ | width mismatch |
| 1 | int32 | int32 `extra int — ONLY if get_consume_cash_item_type(itemId)==51 (@line22; cash-weather variant); no atlas field` | ✅ |  |
| 2 | string | string `message — only if itemId!=0 (@line27)` | ✅ |  |


## Triage: ❌ — JMS185 wire divergence (DEFERRED, not a blind fix)

JMS185 dispatches BLOW_WEATHER (op 0x08B/139) via `CField::OnPacket` case 0x8B →
`sub_5723E6` @0x5723E6 (inlined; no `OnBlowWeather` symbol). It reads:
`Decode4 itemId` (@line17, FIRST — **no leading blow-type byte**) +
`[Decode4 extra if get_consume_cash_item_type(itemId)==51]` (@line22, cash-weather
variant) + `[DecodeStr message if itemId != 0]` (@line27).

Atlas `effect_weather.go` (version-agnostic, BAD-FORM, no region branch) writes
`WriteBool(!active)` (1 leading byte) + `WriteInt(itemId)` + `[string if active]`. Two
JMS divergences: (1) the leading `!active` byte is a 1-byte over-write JMS never reads
(GMS `OnBlowWeather` DOES read `Decode1 m_nBlowType` first, so this byte is correct for
GMS); (2) atlas has no field for the JMS cash-type-51 conditional int.

This is a **structural per-version divergence on a BAD-FORM struct** (design §8 keeps
EffectWeather un-refactored), not a single-field width tweak. A JMS fix would need a new
region branch that drops the leading byte AND models the conditional cash-int — out of
scope for an audit-cover fix. **DEFERRED** — see `_pending.md` WEATHER-jms-shape.

Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
