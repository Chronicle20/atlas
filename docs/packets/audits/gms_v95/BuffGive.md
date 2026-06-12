# BuffGive (← `CWvsContext::OnTemporaryStatSet`)

- **IDA:** 0xa02fc0
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | bytes | bytes `SecondaryStat::DecodeForLocal — mask-driven opaque stat block (per-stat values+delays)` | ✅ |  |
| 1 | int16 | int16 `tDelay` | ✅ |  |
| 2 | byte | byte `MovementAffectingStat — conditional: only if SecondaryStat::IsMovementAffectingStat` | ✅ |  |

