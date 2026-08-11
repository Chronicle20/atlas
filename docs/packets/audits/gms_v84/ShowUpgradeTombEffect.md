# ShowUpgradeTombEffect (← `CUserRemote::OnShowUpgradeTombEffect`)

- **IDA:** 0x9c4206
- **Atlas file:** `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (dispatcher-prefix pattern — consumed by CUserPool::OnUserRemotePacket Decode4 before OnShowUpgradeTombEffect)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |
| 2 | int32 | int32 `x` | ✅ |  |
| 3 | int32 | int32 `y` | ✅ |  |

