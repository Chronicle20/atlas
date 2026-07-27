# DistributeAp (← `CWvsContext::SendAbilityUpRequest#DistributeAp`)

- **IDA:** 0x96db81
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_ap.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time()) @0x96dcb3` | ✅ |  |
| 1 | int32 | int32 `dwStatFlag (single stat flag) @0x96dcbe` | ✅ |  |

