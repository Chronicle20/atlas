# FieldItcOperationRegisterSale (← `CITC::OnRegisterSaleEntry`)

- **IDA:** 0x572e90
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (2 register-fixed-price) @0x5730c9` | ✅ |  |
| 1 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x5730d5 (model.Asset recurse, opaque)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x5730df` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x5730ed` | ✅ |  |
| 4 | int32 | int32 `price @0x5730fb` | ✅ |  |
| 5 | byte | byte `type @0x573109` | ✅ |  |
| 6 | byte | byte `flag @0x573117` | ✅ |  |

