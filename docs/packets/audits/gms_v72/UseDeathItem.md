# UseDeathItem (← `CUserLocal::RequestUpgradeTombEffect`)

- **IDA:** 0x867654
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_death_item.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId (COutPacket::Encode4(0x541370) — hard-coded 5510000)` | ✅ |  |
| 1 | int32 | int32 `x (v2 — m_ptRevive.x)` | ✅ |  |
| 2 | int32 | int32 `y (v6 — m_ptRevive.y)` | ✅ |  |

