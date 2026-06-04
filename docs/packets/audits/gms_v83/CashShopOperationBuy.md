# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x46dadd
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (a2==2)` | ✅ |  |
| 1 | int32 | int32 `dwOption (a2)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (arg0 serialNumber)` | ✅ |  |
| 3 | int32 | int32 `IsZeroGoods int (zero-goods flag). NOTE: v83 has NO trailing byte oneADay (v95-only)` | ✅ |  |

