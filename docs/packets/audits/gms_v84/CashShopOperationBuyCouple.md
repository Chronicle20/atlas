# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v84
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

