# DistributeAp (← `CWvsContext::SendAbilityUpRequest#DistributeAp`)

- **IDA:** 0xb0ad8c
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_ap.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time` | ✅ |  |
| 1 | int32 | int32 `dwFlag (stat flag to increment)` | ✅ |  |

