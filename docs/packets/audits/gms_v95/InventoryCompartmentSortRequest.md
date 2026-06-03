# InventoryCompartmentSortRequest (← `CWvsContext::SendSortItemRequest`)

- **IDA:** 0x9d5c60
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_sort.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (line 17)` | ✅ |  |
| 1 | byte | byte `nType compartmentType (line 18)` | ✅ |  |

