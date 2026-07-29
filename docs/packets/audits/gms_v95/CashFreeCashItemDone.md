# CashFreeCashItemDone (← `CCashShop::OnCashItemResult#FREE_CASH_ITEM_DONE`)

- **IDA:** 0x494880
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xaa FREE_CASH_ITEM_DONE; op-byte consumed by dispatcher before CCashShop::OnCashItemResFreeCashItemDone)` | ✅ |  |
| 1 | bytes | bytes `item: 55 bytes GW_CashItemInfo blob (CashInventoryItem.EncodeBytes); item-blob shape, not a small scalar` | ✅ |  |

