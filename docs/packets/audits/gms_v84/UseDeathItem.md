# UseDeathItem (← `CUserLocal::RequestUpgradeTombEffect`)

- **IDA:** 0x999277
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_death_item.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId — hard-coded 0x541370 (5510000)` | ✅ |  |
| 1 | int32 | int32 `x` | ✅ |  |
| 2 | int32 | int32 `y` | ✅ |  |

