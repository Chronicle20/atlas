# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x48085a
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `s (secondary password / SPW). op-byte 0x1E (NOT GMS 0x1F)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `nCommSN (serialNumber). JMS has NO option int (atlas else-branch has option)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `sGiveTo (recipient name)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `sText (message)` | ❌ | atlas: short — missing trailing field |

