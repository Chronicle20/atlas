# InventoryAdd (← `CWvsContext::OnInventoryOperation#Add`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | byte | byte `mode 0 = Add (line 150)` | ✅ |  |
| 3 | byte | byte `inventoryType (line 151)` | ✅ |  |
| 4 | int16 | int16 `slot (line 152)` | ✅ |  |
| 5 | byte | bytes `asset GW_ItemSlotBase::Decode (case 0 line 158) — sub-struct, tool-opaque` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

