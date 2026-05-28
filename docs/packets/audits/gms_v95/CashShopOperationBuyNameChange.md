# CashShopOperationBuyNameChange (← `CCashShop::SendBuyNameChangeItemPacket`)

- **IDA:** 0x488250
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 1 | string | string `sOldCharName (oldName)` | ✅ |  |
| 2 | string | string `sNewCharName (newName)` | ✅ |  |

