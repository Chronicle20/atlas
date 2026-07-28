# CashTransferWorldDone (← `CCashShop::OnCashItemResult#TRANSFER_WORLD_SUCCESS`)

- **IDA:** 0x463d96
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x7B TRANSFER_WORLD_SUCCESS; op-byte consumed by dispatcher before OnCashItemResTransferWorldDone)` | ✅ |  |
| 1 | bytes | bytes `GW_CashItemInfo blob (55B, single)` | ✅ |  |

