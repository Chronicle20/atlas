# InventoryCompartmentSort (← `CWvsContext::OnSortItemResult`)

- **IDA:** 0x9f12b0
- **Atlas file:** `libs/atlas-packet/inventory/clientbound/compartment_sort.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `always 0 (eResult/padding) — CompartmentSort body` | ✅ |  |
| 1 | byte | byte `inventoryType (nType)` | ✅ |  |

