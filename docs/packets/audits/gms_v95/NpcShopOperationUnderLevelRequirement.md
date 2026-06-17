# NpcShopOperationUnderLevelRequirement (← `CShopDlg::OnPacket#UnderLevelRequirement`)

- **IDA:** 0x6eb7d0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (under level-requirement sub-op)` | ✅ |  |
| 1 | int32 | int32 `levelLimit (case 0xF @0x6ebaf7)` | ✅ |  |
