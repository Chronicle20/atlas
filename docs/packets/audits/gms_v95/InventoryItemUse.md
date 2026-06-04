# InventoryItemUse (← `CWvsContext::SendStatChangeItemUseRequest`)

- **IDA:** 0x9ddfe0
- **Atlas file:** `../../libs/atlas-packet/inventory/serverbound/item_use.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (line 142)` | ✅ |  |
| 1 | int16 | int16 `nPOS source (line 143)` | ✅ |  |
| 2 | int32 | int32 `nItemID (line 144)` | ✅ |  |

