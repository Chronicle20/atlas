# NpcShopOperationOverLevelRequirement (← `CShopDlg::OnPacket#OverLevelRequirement`)

- **IDA:** 0x6eb7d0
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (over level-requirement sub-op)` | ✅ |  |
| 1 | int32 | int32 `levelLimit (case 0xE @0x6eba48)` | ✅ |  |
