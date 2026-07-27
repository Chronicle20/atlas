# FieldItcOperationSaleCurrentItem (← `CITC::OnSaleCurrentItem`)

- **IDA:** 0x5731a0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (3) @0x5731ec` | ✅ |  |
| 1 | byte | byte `type @0x5731fa` | ✅ |  |
| 2 | int32 | int32 `slotPos @0x573208` | ✅ |  |
| 3 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x573216 (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 4 | int32 | int32 `commodityId @0x573224` | ✅ |  |

