# FieldEffectWeather (← `CField::OnBlowWeather`)

- **IDA:** 0x5468f0
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect_weather.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `m_nBlowType (atlas !active; 0=start admin-weather w/ message, 1=item/end)` | ✅ |  |
| 1 | int32 | int32 `itemId (v4)` | ✅ |  |
| 2 | string | string `message (only when itemId!=0 && m_nBlowType==0; start path)` | ✅ |  |


## Manual per-mode verdict (tool limitation)

`EffectWeather` is a BAD-FORM single struct whose first wire byte (`!active`) and
the presence of the trailing `message` string are decided at construction
(`NewFieldEffectWeatherStart` vs `NewFieldEffectWeatherEnd`). The flat verdict
above is ✅, but the conditional message-string branch is worth documenting
explicitly. Per design §8 this file is NOT refactored into per-mode structs.

IDA reference: `CField::OnBlowWeather`@0x5468f0 —
`Decode1(m_nBlowType)` + `Decode4(itemId)`; the message string is read only in
the `else` branch, i.e. when `itemId != 0 && m_nBlowType == 0`.

| mode | atlas constructor | first byte (`!active` / `m_nBlowType`) | IDA payload | atlas Encode payload | verdict |
|---|---|---|---|---|---|
| start (admin weather) | `NewFieldEffectWeatherStart(itemId, message)` | `0` | `Decode1(0) + Decode4(itemId) + DecodeStr(message)` | `WriteBool(false) + WriteInt(itemId) + WriteAsciiString(message)` | ✅ |
| end / item weather | `NewFieldEffectWeatherEnd(itemId)` | `1` | `Decode1(1) + Decode4(itemId)` (no string) | `WriteBool(true) + WriteInt(itemId)` (no string) | ✅ |

**Per-mode verdict: ✅ (both modes verified against
`CField::OnBlowWeather`@0x5468f0).** The atlas `active` flag inverts to the IDA
`m_nBlowType` byte (start→0, end→1) and gates the message string exactly as the
IDA `else` branch does.

Edge case (not a bug): IDA also suppresses the message string when `itemId == 0`
(`!v4 || m_nBlowType`). Atlas's start path would still write a string when
`itemId == 0`, but `itemId == 0` is never a valid weather item, so the divergence
is unreachable in practice. Documented under "## Tool limitations — world domain"
in `_pending.md`.

Ack: world-audit Phase 2d on 2026-05-28
