# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

