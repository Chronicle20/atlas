# FieldItcOperationRegisterAuction (← `CITC::OnRegisterSaleEntry#RegisterAuction`)

- **IDA:** 0x5615e1
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x12 register-auction) @0x59ecc8` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_4E33D8 @0x59ecd4 (GW_ItemSlotBase Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x59ecdf` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x59ecea` | ✅ |  |
| 4 | int32 | int32 `arg0 selector (==1) @0x59ecf5` | ✅ |  |
| 5 | int32 | int32 `buyNowPrice @0x59ed00` | ✅ |  |
| 6 | byte | byte `type @0x59ed0b` | ✅ |  |
| 7 | byte | byte `flag @0x59ed16` | ✅ |  |
| 8 | int32 | int32 `durationHrs @0x59ed21` | ✅ |  |

