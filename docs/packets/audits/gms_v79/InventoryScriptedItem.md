# InventoryScriptedItem (← `CWvsContext::SendScriptRunItemRequest`)

- **IDA:** 0x955840
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time() result, v6)` | ✅ |  |
| 1 | int16 | int16 `source slot (nPOS, a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID, a3); gated a3/10000==243` | ✅ |  |

