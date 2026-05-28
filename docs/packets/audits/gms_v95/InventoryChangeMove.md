# InventoryChangeMove (← `CWvsContext::OnInventoryOperation#ChangeMove`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | byte | byte `mode 2 = Move (line 150)` | ✅ |  |
| 3 | byte | byte `inventoryType (line 151)` | ✅ |  |
| 4 | int16 | int16 `oldSlot (line 152)` | ✅ |  |
| 5 | int16 | int16 `newSlot (case 2 line 223)` | ✅ |  |
| 6 | byte | byte `addMov SecondaryStatChangedPoint; read once after loop only if inventoryType==1 && (oldSlot<0\|\|newSlot<0) (line 225 sets flag; line 315 reads)` | ✅ |  |

