# NpcShopOperationOverLevelRequirement (← `CShopDlg::OnPacket#OverLevelRequirement`)

- **IDA:** 0x7a290d
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (over level-requirement sub-op)` | ✅ |  |
| 1 | int32 | int32 `level/required amount (case 0xE @0x7a2af3)` | ✅ |  |
