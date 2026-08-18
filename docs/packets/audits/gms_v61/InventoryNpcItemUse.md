# InventoryNpcItemUse (← `CWvsContext::SendSelectNpcItemUseRequest`)

- **IDA:** 0x83778d
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `source (nPOS / a2), int16 slot` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID / a3)` | ✅ |  |

