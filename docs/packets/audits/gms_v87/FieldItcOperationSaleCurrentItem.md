# FieldItcOperationSaleCurrentItem (← `CITC::OnSaleCurrentItem`)

- **IDA:** 0x5cec03
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3) @0x5cec2f` | ✅ |  |
| 1 | byte | byte `type @0x5cec3a` | ✅ |  |
| 2 | int32 | int32 `slotPos @0x5cec45` | ✅ |  |
| 3 | byte | bytes `item-slot blob sub_502670 @0x5cec51 (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 4 | int32 | int32 `commodityId @0x5cec5c` | ✅ |  |

