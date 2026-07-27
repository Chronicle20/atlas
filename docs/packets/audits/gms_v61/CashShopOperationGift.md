# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x45c607
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `mode (4 = regular gift)` | ❌ | width mismatch |
| 1 | int32 | int32 `birthday` | ✅ |  |
| 2 | string | int32 `serialNumber` | ❌ | width mismatch |
| 3 | string | string `name` | ✅ |  |
| 4 | byte | string `message` | ❌ | atlas: short — missing trailing field |

