# FieldItcOperationRegisterAuction (← `CITC::OnRegisterSaleEntry#RegisterAuction`)

- **IDA:** 0x572fd0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x12 register-auction) @0x572fd0` | ✅ |  |
| 1 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x572fdc (model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity @0x572fe6` | ✅ |  |
| 3 | int32 | int32 `commodityId @0x572ff4` | ✅ |  |
| 4 | int32 | int32 `selector (==1) @0x573002` | ✅ |  |
| 5 | int32 | int32 `buyNowPrice @0x573010` | ✅ |  |
| 6 | byte | byte `type @0x57301e` | ✅ |  |
| 7 | byte | byte `flag @0x57302c` | ✅ |  |
| 8 | int32 | int32 `durationHrs @0x57303a` | ✅ |  |

