# FieldItcOperationSaleCurrentItem (← `CITC::OnSaleCurrentItem`)

- **IDA:** 0x604330
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3) @0x60435c` | ✅ |  |
| 1 | byte | byte `type @0x604367` | ✅ |  |
| 2 | int32 | int32 `slotPos @0x604372` | ✅ |  |
| 3 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x60437e (Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 4 | int32 | int32 `commodityId @0x604389` | ✅ |  |

