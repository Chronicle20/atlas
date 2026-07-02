# DistributeAp (← `CWvsContext::SendAbilityUpRequest#DistributeAp`)

- **IDA:** 0x8457ee
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_ap.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time @0x84589a` | ✅ |  |
| 1 | int32 | int32 `dwFlag @0x8458a5` | ✅ |  |

