# CashTransferWorldDone (← `CCashShop::OnCashItemResult#TRANSFER_WORLD_SUCCESS`)

- **IDA:** 0x47f140
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa3 TRANSFER_WORLD_SUCCESS; op-byte consumed by dispatcher before CCashShop::OnCashItemResTransferWorldDone)` | ✅ |  |
| 1 | bytes | bytes `item: 55 bytes GW_CashItemInfo blob (CashInventoryItem.EncodeBytes); byte-identical body shape to NAME_CHANGE_BUY_DONE` | ✅ |  |

