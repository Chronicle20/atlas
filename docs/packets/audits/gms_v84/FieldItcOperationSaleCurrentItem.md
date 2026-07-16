# FieldItcOperationSaleCurrentItem (← `CITC::OnSaleCurrentItem`)

- **IDA:** 0x5af1db
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3) @0x5af207` | ✅ |  |
| 1 | byte | byte `type @0x5af212` | ✅ |  |
| 2 | int32 | int32 `slotPos @0x5af21d` | ✅ |  |
| 3 | byte | bytes `item-slot blob sub_4EA6F8 @0x5af229 (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 4 | int32 | int32 `commodityId @0x5af234` | ✅ |  |

