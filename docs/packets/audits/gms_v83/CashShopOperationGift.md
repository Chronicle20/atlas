# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x46f940
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `field A (*(this+1304)). NOTE: v83 leading is int; v95 sends EncodeStr sSPW here` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `serialNumber (*(this+1308))` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `recipient name (v33). NOTE: v83 has NO byte oneADay before name (v95-only)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `message (*(this+1312))` | ❌ | atlas: short — missing trailing field |

