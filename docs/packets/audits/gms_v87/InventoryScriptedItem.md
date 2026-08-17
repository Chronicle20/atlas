# InventoryScriptedItem (← `CWvsContext::SendScriptRunItemRequest`)

- **IDA:** 0xa9f3d2
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time())` | ✅ |  |
| 1 | int16 | int16 `source (nPOS)` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID)` | ✅ |  |

