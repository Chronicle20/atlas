# FieldItcOperationRegisterSale (← `CITC::OnRegisterSaleEntry`)

- **IDA:** 0x604105
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (2 register-fixed-price) @0x604281` | ✅ |  |
| 1 | byte | bytes `item-slot blob GW_ItemSlotBase::Encode @0x60428c (Encode1 type + RawEncode; model.Asset recurse)` | 🔍 | sub-struct: itemCopy — see _substruct/ |
| 2 | int32 | int32 `quantity (nSlotNo) @0x604297` | ✅ |  |
| 3 | int32 | int32 `commodityId (v24) @0x6042a2` | ✅ |  |
| 4 | int32 | int32 `price (nTI) @0x6042ad` | ✅ |  |
| 5 | byte | byte `type (pItem) @0x6042b8` | ✅ |  |
| 6 | byte | byte `flag (n[0]) @0x6042c3` | ✅ |  |

