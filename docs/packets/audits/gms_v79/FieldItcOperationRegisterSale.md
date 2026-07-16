# FieldItcOperationRegisterSale (← `CITC::OnRegisterSaleEntry`)

- **IDA:** 0x57a20c
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (2 register-fixed-price) @0x59ed92` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_4E33D8 @0x59ed9e (GW_ItemSlotBase Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x59eda9` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x59edb4` | ✅ |  |
| 4 | int32 | int32 `price @0x59edbf` | ✅ |  |
| 5 | byte | byte `type @0x59edca` | ✅ |  |
| 6 | byte | byte `flag @0x59edd5` | ✅ |  |

