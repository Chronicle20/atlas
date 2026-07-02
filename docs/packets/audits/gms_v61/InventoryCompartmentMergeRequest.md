# InventoryCompartmentMergeRequest (← `CWvsContext::SendGatherItemRequest`)

- **IDA:** 0x8314d0
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_merge.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

