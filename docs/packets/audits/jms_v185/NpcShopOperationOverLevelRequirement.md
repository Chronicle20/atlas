# NpcShopOperationOverLevelRequirement (← `CShopDlg::OnPacket#OverLevelRequirement`)

- **IDA:** 0x7cb04e
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xE over level requirement)` | ✅ |  |
| 1 | int32 | int32 `levelLimit / count (@0x7cb1c0 case 0xE)` | ✅ |  |
