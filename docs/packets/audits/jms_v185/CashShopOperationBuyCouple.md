# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x48085a
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `s (secondary password / SPW). op-byte 0x1E (NOT GMS 0x1F)` | ✅ |  |
| 1 | int32 | int32 `nCommSN (serialNumber). JMS has NO option int (atlas else-branch has option)` | ✅ |  |
| 2 | string | string `sGiveTo (recipient name)` | ✅ |  |
| 3 | string | string `sText (message)` | ✅ |  |

