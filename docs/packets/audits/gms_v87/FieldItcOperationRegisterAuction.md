# FieldItcOperationRegisterAuction (← `CITC::OnRegisterSaleEntry#RegisterAuction`)

- **IDA:** 0x5cea89
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x12 register-auction) @0x5cea89` | ✅ |  |
| 1 | byte | bytes `item-slot blob sub_502670 @0x5cea94 (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x5cea9f` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x5ceaaa` | ✅ |  |
| 4 | int32 | int32 `selector (==1) @0x5ceab5` | ✅ |  |
| 5 | int32 | int32 `buyNowPrice @0x5ceac0` | ✅ |  |
| 6 | byte | byte `type @0x5ceacb` | ✅ |  |
| 7 | byte | byte `flag @0x5cead6` | ✅ |  |
| 8 | int32 | int32 `durationHrs @0x5ceae1` | ✅ |  |

