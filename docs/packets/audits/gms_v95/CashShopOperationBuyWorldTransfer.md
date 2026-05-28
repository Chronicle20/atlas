# CashShopOperationBuyWorldTransfer (← `CCashShop::SendBuyTransferWorldItemPacket`)

- **IDA:** 0x482f30
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_world_transfer.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 1 | int32 | int32 `nTargetWorldID (targetWorld)` | ✅ |  |

