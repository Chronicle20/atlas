# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x483f10
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v92
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | width mismatch |
| 4 | string | string `` | ✅ |  |
| 5 | byte | string `` | ❌ | atlas: short — missing trailing field |

