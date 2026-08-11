# UseDeathItem (← `CUserLocal::RequestUpgradeTombEffect`)

- **IDA:** 0x95af8e
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_death_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId, hard-coded 5510000 (0x541370)` | ✅ |  |
| 1 | int32 | int32 `x = this[2025]` | ✅ |  |
| 2 | int32 | int32 `y = this[2026]` | ✅ |  |

