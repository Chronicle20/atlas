# InventoryQuantityUpdate (← `CWvsContext::OnInventoryOperation#QuantityUpdate`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | byte | byte `action (1 = QuantityUpdate)` | ✅ |  |
| 3 | byte | byte `invType` | ✅ |  |
| 4 | int16 | int16 `slot` | ✅ |  |
| 5 | int16 | int16 `quantity` | ✅ |  |

