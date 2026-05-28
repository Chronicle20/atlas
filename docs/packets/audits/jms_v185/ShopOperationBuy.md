# ShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x47eaa7
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isPoints bool (v47 == 2). op-byte 3 consumed by dispatcher` | ✅ |  |
| 1 | int32 | int32 `nCommSN (serialNumber). JMS NX-system: NO dwOption/currency int, NO trailing zero/oneADay/eventSN — diverges from all atlas branches` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

