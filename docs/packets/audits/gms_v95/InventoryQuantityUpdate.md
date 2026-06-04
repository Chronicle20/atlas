# InventoryQuantityUpdate (← `CWvsContext::OnInventoryOperation#QuantityUpdate`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | byte | byte `mode 1 = QuantityUpdate (line 150)` | ✅ |  |
| 3 | byte | byte `inventoryType (line 151)` | ✅ |  |
| 4 | int16 | int16 `slot (line 152)` | ✅ |  |
| 5 | int16 | int16 `quantity (nNumber, case 1 line 193)` | ✅ |  |

