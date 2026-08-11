# ShowUpgradeTombEffect (← `CUserRemote::OnShowUpgradeTombEffect`)

- **IDA:** 0x983e40
- **Atlas file:** `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (read by CUserPool::OnUserRemotePacket dispatcher before the opcode switch)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |
| 2 | int32 | int32 `x` | ✅ |  |
| 3 | int32 | int32 `y` | ✅ |  |

