# InventoryScriptedItem (← `CWvsContext::SendScriptRunItemRequest`)

- **IDA:** 0xaee7ce
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time()` | ✅ |  |
| 1 | int16 | int16 `nPOS` | ✅ |  |
| 2 | int32 | int32 `nItemID` | ✅ |  |

