# InventoryCompartmentSortRequest (← `CWvsContext::SendSortItemRequest`)

- **IDA:** 0x831564
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_sort.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

