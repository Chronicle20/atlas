# ShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x48e530
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 3 | int32 | byte `m_bRequestBuyOneADay (NOT in atlas - atlas reads int zero spanning this+eventSN)` | ❌ | width mismatch |
| 4 | byte | int32 `nEventSN (NOT correctly modeled by atlas)` | ❌ | atlas: short — missing trailing field |


> defer: version-gated — trailing byte(oneADay)+int(eventSN) is a later-GMS addition; atlas models them as a single int zero. v83 likely omits. See _pending.md "ShopOperationBuy — trailing oneADay byte + eventSN int".
