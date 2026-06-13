# CashShopOperationBuyWorldTransfer (← `CCashShop::SendBuyTransferWorldItemPacket`)

- **IDA:** 0x485038
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_world_transfer.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

