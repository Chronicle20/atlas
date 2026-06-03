# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x48e530
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 3 | byte | byte `m_bRequestBuyOneADay (NOT in atlas - atlas reads int zero spanning this+eventSN)` | ✅ |  |
| 4 | int32 | int32 `nEventSN (NOT correctly modeled by atlas)` | ✅ |  |

