# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x48e530
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `dwOption (currency)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `nCommSN (serialNumber)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `m_bRequestBuyOneADay (NOT in atlas - atlas reads int zero spanning this+eventSN)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `nEventSN (NOT correctly modeled by atlas)` | ❌ | atlas: short — missing trailing field |

