# NpcShopOperationInventoryFull (← `CShopDlg::OnPacket#InventoryFull`)

- **IDA:** 0x6d6eb9
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (INVENTORY_FULL mode-only notice arm: v79 mode 3)` | ✅ |  |

