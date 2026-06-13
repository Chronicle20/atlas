# DistributeAp (← `CWvsContext::SendAbilityUpRequest#DistributeAp`)

- **IDA:** 0xa23b3d
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_ap.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time())` | ✅ |  |
| 1 | int32 | int32 `nValue (count of stat pairs = aStatUp->a[-1].nValue)` | ✅ |  |
| 2 | byte | int32 `dwStatFlag (stat flag for each pair)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 3 | byte | int32 `nValue (amount for each stat pair)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |

