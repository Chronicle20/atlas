# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x46dadd
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (a2==2)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `dwOption (a2)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `nCommSN (arg0 serialNumber)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `IsZeroGoods int (zero-goods flag). NOTE: v83 has NO trailing byte oneADay (v95-only)` | ❌ | atlas: short — missing trailing field |

