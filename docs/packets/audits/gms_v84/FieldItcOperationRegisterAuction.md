# FieldItcOperationRegisterAuction (← `CITC::OnRegisterSaleEntry#RegisterAuction`)

- **IDA:** 0x5af045
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x12 register-auction) @0x5af064` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_4EA6F8 @0x5af070 (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x5af07b` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x5af086` | ✅ |  |
| 4 | int32 | int32 `selector (==1) @0x5af091` | ✅ |  |
| 5 | int32 | int32 `buyNowPrice @0x5af09c` | ✅ |  |
| 6 | byte | byte `type @0x5af0a7` | ✅ |  |
| 7 | byte | byte `flag @0x5af0b2` | ✅ |  |
| 8 | int32 | int32 `durationHrs @0x5af0bd` | ✅ |  |

