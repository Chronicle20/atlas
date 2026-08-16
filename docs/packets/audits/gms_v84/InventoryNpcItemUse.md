# InventoryNpcItemUse (← `CWvsContext::SendSelectNpcItemUseRequest`)

- **IDA:** 0xa5a4b2
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `source (nPOS; Encode2 at 0xa5a58a)` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID; Encode4 at 0xa5a595)` | ✅ |  |

