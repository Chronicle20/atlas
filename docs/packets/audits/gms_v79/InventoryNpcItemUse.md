# InventoryNpcItemUse (← `CWvsContext::SendSelectNpcItemUseRequest`)

- **IDA:** 0x95b96c
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `source slot (nPOS, a2)` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID, a3); gated (a3/10000==545\|\|a3/10000==239)` | ✅ |  |

