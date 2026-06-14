# CashShopOperationBuyNameChange (← `CCashShop::SendBuyNameChangeItemPacket`)

- **IDA:** 0x475f79
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | string | string `` | ✅ |  |
| 3 | byte | string `` | ❌ | atlas: short — missing trailing field |

