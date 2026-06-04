# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x47eaa7
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isPoints bool (v47 == 2). op-byte 3 consumed by dispatcher` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `nCommSN (serialNumber). JMS NX-system: NO dwOption/currency int, NO trailing zero/oneADay/eventSN — diverges from all atlas branches` | ❌ | atlas: short — missing trailing field |

