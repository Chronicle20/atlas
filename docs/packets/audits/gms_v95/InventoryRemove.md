# InventoryRemove (← `CWvsContext::OnInventoryOperation#Remove`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | byte | byte `mode 3 = Remove (line 150)` | ✅ |  |
| 3 | byte | byte `inventoryType (line 151)` | ✅ |  |
| 4 | int16 | int16 `slot (line 152)` | ✅ |  |
| 5 | byte | byte `addMov SecondaryStatChangedPoint; read once after loop only if inventoryType==1 && slot<0 (line 374 sets flag; line 315 reads)` | ✅ |  |

