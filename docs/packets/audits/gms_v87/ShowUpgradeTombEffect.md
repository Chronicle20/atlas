# ShowUpgradeTombEffect (← `CUserRemote::OnShowUpgradeTombEffect`)

- **IDA:** 0xa098f2
- **Atlas file:** `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (read by CUserPool dispatcher before OnShowUpgradeTombEffect, case 0xD0)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |
| 2 | int32 | int32 `x` | ✅ |  |
| 3 | int32 | int32 `y` | ✅ |  |

