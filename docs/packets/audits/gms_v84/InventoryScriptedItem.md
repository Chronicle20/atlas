# InventoryScriptedItem (← `CWvsContext::SendScriptRunItemRequest`)

- **IDA:** 0xa53f08
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time() via sub_9C7771; Encode4 at 0xa53f67, sourced 0xa53f5e)` | ✅ |  |
| 1 | int16 | int16 `source (nPOS; Encode2 at 0xa53f72)` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID; Encode4 at 0xa53f7d)` | ✅ |  |

