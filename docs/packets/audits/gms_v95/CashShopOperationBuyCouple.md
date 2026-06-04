# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x490d80
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `dwOption (option)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `nCommSN (serialNumber)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `sGiveTo (name)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `sText (message)` | ❌ | atlas: short — missing trailing field |

