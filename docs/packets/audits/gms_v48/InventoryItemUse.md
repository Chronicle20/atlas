# InventoryItemUse (← `CWvsContext::SendStatChangeItemUseRequest`)

- **IDA:** 0x70db3c
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/item_use.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime` | ✅ |  |
| 1 | int16 | int16 `slot` | ✅ |  |
| 2 | int32 | int32 `itemId` | ✅ |  |

