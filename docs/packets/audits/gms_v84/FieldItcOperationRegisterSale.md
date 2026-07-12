# FieldItcOperationRegisterSale (← `CITC::OnRegisterSaleEntry`)

- **IDA:** 0x5aefd2
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (2 register-fixed-price) @0x5af12e` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_4EA6F8 @0x5af13a (model.Asset recurse, opaque)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x5af145` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x5af150` | ✅ |  |
| 4 | int32 | int32 `price @0x5af15b` | ✅ |  |
| 5 | byte | byte `type @0x5af166` | ✅ |  |
| 6 | byte | byte `flag @0x5af171` | ✅ |  |

