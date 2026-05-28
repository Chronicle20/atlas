# ShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x48085a
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `s (secondary password / SPW). op-byte 0x1E (NOT GMS 0x1F)` | ❌ | width mismatch |
| 1 | int32 | int32 `nCommSN (serialNumber). JMS has NO option int (atlas else-branch has option)` | ✅ |  |
| 2 | int32 | string `sGiveTo (recipient name)` | ❌ | width mismatch |
| 3 | string | string `sText (message)` | ✅ |  |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

