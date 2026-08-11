# ShowUpgradeTombEffect (← `CUserRemote::OnShowUpgradeTombEffect`)

- **IDA:** 0x88d0e4
- **Atlas file:** `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — auto-prepended via dispatcher: per-user-remote (CUserPool::OnUserRemotePacket)` | ✅ |  |
| 1 | int32 | int32 `itemId (v3)` | ✅ |  |
| 2 | int32 | int32 `x (v4)` | ✅ |  |
| 3 | int32 | int32 `y (v5)` | ✅ |  |

