# AutoDistributeAp (← `CWvsContext::SendAbilityUpRequest#AutoDistributeAp`)

- **IDA:** 0xabb60b
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/auto_distribute_ap.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time())` | ✅ |  |
| 1 | int32 | int32 `nValue (count of stat pairs)` | ✅ |  |
| 2 | int32 | int32 `dwStatFlag (stat flag for each pair)` | ✅ |  |
| 3 | int32 | int32 `nValue (amount for each stat pair)` | ✅ |  |

