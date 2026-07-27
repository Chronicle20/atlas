# FieldItcOperationRegisterSale (← `CITC::OnRegisterSaleEntry`)

- **IDA:** 0x5ce967
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (2 register-fixed-price) @0x5ceb55` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_502670 @0x5ceb60 (model.Asset recurse, opaque)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x5ceb6b` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x5ceb76` | ✅ |  |
| 4 | int32 | int32 `price @0x5ceb81` | ✅ |  |
| 5 | byte | byte `type @0x5ceb8c` | ✅ |  |
| 6 | byte | byte `flag @0x5ceb97` | ✅ |  |

