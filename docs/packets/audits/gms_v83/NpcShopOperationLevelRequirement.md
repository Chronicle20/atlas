# NpcShopOperationLevelRequirement (← `CShopDlg::OnPacket#LevelRequirement`)

- **IDA:** 0x756da7
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (over/under level-requirement sub-op: v83 cases 14/15)` | ✅ |  |
| 1 | int32 | int32 `levelLimit` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
