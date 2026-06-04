# NpcShopOperationLevelRequirement (← `CShopDlg::OnPacket#LevelRequirement`)

- **IDA:** 0x7a290d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (sub-op discriminator)` | ✅ |  |
| 1 | int32 | int32 `level/required amount (cases 0xE/0xF)` | ✅ |  |

