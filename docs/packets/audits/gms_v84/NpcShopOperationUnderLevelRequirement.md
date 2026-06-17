# NpcShopOperationUnderLevelRequirement (← `CShopDlg::OnPacket#UnderLevelRequirement`)

- **IDA:** 0x77905b
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (under level-requirement sub-op: case 15)` | ✅ |  |
| 1 | int32 | int32 `levelLimit (case 15)` | ✅ |  |
