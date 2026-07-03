# CashShopOperationBuyNameChange (← `CCashShop::SendBuyNameChangeItemPacket`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

