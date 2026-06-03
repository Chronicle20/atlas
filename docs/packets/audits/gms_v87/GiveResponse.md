# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0xab9c24
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 0 = GIVE)` | ✅ |  |
| 1 | string | string `toName (recipient of the fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int16 | int32 `nPOP (new total fame as int32)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

## Manual analysis

**v87 vs v95/v83:** Same pre-existing divergence seen in v83/v95. `OnGivePopularityResult` @ 0xab9c24 confirmed: mode 0 reads DecodeStr(toName) + Decode1(bInc) + Decode4(nPOP). The ❌ reflects atlas writing int16 (fame difference) + an extra int16 where the client reads int32 + nothing. This is a pre-existing issue tracked in the v83 and v95 passes — not a v87-specific regression. No new gate needed.

Ack: misc-audit Phase 3 v87 on 2026-06-03

