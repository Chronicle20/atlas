# InventoryAdd (← `CWvsContext::OnInventoryOperation#Add`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag (if !=0 reset excl + get_update_time)` | ✅ |  |
| 1 | byte | byte `count (number of operation entries)` | ✅ |  |
| 2 | byte | byte `action (0 = Add)` | ✅ |  |
| 3 | byte | byte `invType` | ✅ |  |
| 4 | int16 | int16 `slot` | ✅ |  |
| 5 | byte | bytes `asset GW_ItemSlotBase::Decode (case 0) — sub-struct, tool-opaque` | 🔍 | sub-struct: model.Asset — see _substruct/ |
| 6 | byte | byte `trailing addMov byte ONLY if any entry set nCurItemPos (equip move/remove)` | ❌ | atlas: short — missing trailing field |

