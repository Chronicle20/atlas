# AutoDistributeAp (← `CWvsContext::SendAbilityUpRequest#AutoDistributeAp`)

- **IDA:** 0x96dd07
- **Atlas file:** `libs/atlas-packet/character/serverbound/auto_distribute_ap.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time()) @0x96de39` | ✅ |  |
| 1 | int32 | int32 `nValue (count of stat pairs = *(aStatUp-4)) @0x96de4b` | ✅ |  |
| 2 | int32 | int32 `dwStatFlag (stat flag for each pair) @0x96de63` | ✅ |  |
| 3 | int32 | int32 `nValue (amount for each stat pair) @0x96de71` | ✅ |  |

