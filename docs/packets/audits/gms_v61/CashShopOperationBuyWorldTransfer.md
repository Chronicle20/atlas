# CashShopOperationBuyWorldTransfer (← `CCashShop::SendBuyTransferWorldItemPacket`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_world_transfer.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

