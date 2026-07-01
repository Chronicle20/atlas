# AutoDistributeAp (← `CWvsContext::SendAbilityUpRequest#AutoDistributeAp`)

- **IDA:** 0x91bce8
- **Atlas file:** `libs/atlas-packet/character/serverbound/auto_distribute_ap.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time @0x91be29` | ✅ |  |
| 1 | int32 | int32 `count @0x91be3b` | ✅ |  |
| 2 | int32 | int32 `flag @0x91be53` | ✅ |  |
| 3 | int32 | int32 `value @0x91be61` | ✅ |  |

