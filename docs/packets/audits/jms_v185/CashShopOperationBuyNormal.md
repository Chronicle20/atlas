# CashShopOperationBuyNormal (← `CCashShop::OnBuyNormal`)

- **IDA:** 0x47f5ba
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_normal.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | byte | int32 `nCommSN (serialNumber). op-byte 0x21 consumed by dispatcher. JMS body = serialNumber only` | ❌ | atlas: short — missing trailing field |

