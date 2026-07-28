# InventoryCompartmentMergeRequest (← `CWvsContext::SendGatherItemRequest`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_merge.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

