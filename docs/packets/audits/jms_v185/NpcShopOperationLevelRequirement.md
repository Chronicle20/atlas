# NpcShopOperationLevelRequirement (← `CShopDlg::OnPacket#LevelRequirement`)

- **IDA:** 0x7cb04e
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xE over / 0xF under level requirement)` | ✅ |  |
| 1 | int32 | int32 `levelLimit / count (@0x7cb1c0 case 0xE; @0x7cb23d case 0xF)` | ✅ |  |

