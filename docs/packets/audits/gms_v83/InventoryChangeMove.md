# InventoryChangeMove (← `CWvsContext::OnInventoryOperation#ChangeMove`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | byte | byte `action (2 = Move)` | ✅ |  |
| 3 | byte | byte `invType` | ✅ |  |
| 4 | int16 | int16 `oldSlot` | ✅ |  |
| 5 | int16 | int16 `newSlot` | ✅ |  |
| 6 | byte | byte `trailing addMov byte ONLY if invType==1 && (oldSlot<0\|\|newSlot<0)` | ✅ |  |

