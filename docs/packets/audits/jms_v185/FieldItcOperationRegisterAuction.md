# FieldItcOperationRegisterAuction (← `CITC::OnRegisterSaleEntry#RegisterAuction`)

- **IDA:** 0x6041b9
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x12 register-auction) @0x6041b9` | ✅ |  |
| 1 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x6041c4 (Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity (nSlotNo) @0x6041cf` | ✅ |  |
| 3 | int32 | int32 `commodityId (v24) @0x6041da` | ✅ |  |
| 4 | int32 | int32 `selector (nRegType==1) @0x6041e5` | ✅ |  |
| 5 | int32 | int32 `buyNowPrice (v22) @0x6041f0` | ✅ |  |
| 6 | byte | byte `type (pItem) @0x6041fb` | ✅ |  |
| 7 | byte | byte `flag (n[0]) @0x604206` | ✅ |  |
| 8 | int32 | int32 `durationHrs (v21) @0x604211` | ✅ |  |

