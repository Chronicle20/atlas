# InventoryQuantityUpdate (← `CWvsContext::OnInventoryOperation#QuantityUpdate`)

- **IDA:** 0x96953e
- **Atlas file:** `libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest flag @0x969556 (if !=0 reset excl + get_update_time); Atlas WriteBool(!silent)` | ✅ |  |
| 1 | byte | byte `count (operation entries) @0x96959a; Atlas WriteByte(1)` | ✅ |  |
| 2 | byte | byte `action/mode @0x9695b8` | ✅ |  |
| 3 | byte | byte `invType @0x9695c3` | ✅ |  |
| 4 | int16 | int16 `slot @0x9695cb` | ✅ |  |
| 5 | int16 | int16 `quantity @0x969893 (mode 1=QuantityUpdate); Atlas WriteShort(quantity)` | ✅ |  |

