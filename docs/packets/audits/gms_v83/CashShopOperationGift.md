# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x46f940
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `field A (*(this+1304)). NOTE: v83 leading is int; v95 sends EncodeStr sSPW here` | ✅ |  |
| 1 | int32 | int32 `serialNumber (*(this+1308))` | ✅ |  |
| 2 | string | string `recipient name (v33). NOTE: v83 has NO byte oneADay before name (v95-only)` | ✅ |  |
| 3 | string | string `message (*(this+1312))` | ✅ |  |

