# InventoryChangeMove (← `CWvsContext::OnInventoryOperation#ChangeMove`)

- **IDA:** 0x96953e
- **Atlas file:** `libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest flag @0x969556 (if !=0 reset excl + get_update_time); Atlas WriteBool(!silent)` | ✅ |  |
| 1 | byte | byte `count (operation entries) @0x96959a; Atlas WriteByte(1)` | ✅ |  |
| 2 | byte | byte `action/mode @0x9695b8` | ✅ |  |
| 3 | byte | byte `invType @0x9695c3` | ✅ |  |
| 4 | int16 | int16 `oldSlot @0x9695cb (v8)` | ✅ |  |
| 5 | int16 | int16 `newSlot @0x96974f (v19, mode 2=Move); Atlas WriteInt16(newSlot)` | ✅ |  |
| 6 | byte | byte `trailing addMov byte @0x96997e — post-loop, ONLY if an entry set nCurItemPos (equip move/remove with a negative slot). For count=1 this coincides with the per-entry inline addMov Atlas writes.` | ✅ |  |

