# FieldItcOperationSaleCurrentItem (← `CITC::OnSaleCurrentItem`)

- **IDA:** 0x59ee3f
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3) @0x59ee6b` | ✅ |  |
| 1 | byte | byte `type @0x59ee76` | ✅ |  |
| 2 | int32 | int32 `slotPos @0x59ee81` | ✅ |  |
| 3 | byte | bytes `item-slot blob sub_4E33D8 @0x59ee8d (GW_ItemSlotBase Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 4 | int32 | int32 `commodityId @0x59ee98` | ✅ |  |

