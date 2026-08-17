# InventoryNpcItemUse (← `CWvsContext::SendSelectNpcItemUseRequest`)

- **IDA:** 0xaf43ee
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nPOS` | ✅ |  |
| 1 | int32 | int32 `nItemID` | ✅ |  |

