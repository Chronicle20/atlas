# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x48085a
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | string `s (secondary password / SPW). op-byte 0x1E (NOT GMS 0x1F)` | ❌ | width mismatch |
| 2 | string | int32 `nCommSN (serialNumber). JMS has NO option int (atlas else-branch has option)` | ❌ | width mismatch |
| 3 | string | string `sGiveTo (recipient name)` | ✅ |  |
| 4 | byte | string `sText (message)` | ❌ | atlas: short — missing trailing field |

