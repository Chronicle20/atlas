# CashChangeMaplePointDone (← `CCashShop::OnCashItemResult#CHANGE_MAPLE_POINT_SUCCESS`)

- **IDA:** 0x498520
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xbb CHANGE_MAPLE_POINT_SUCCESS; op-byte consumed by dispatcher before CCashShop::OnCashItemResChangeMaplePointDone)` | ✅ |  |
| 1 | int64 | bytes `sn: 8 bytes LARGE_INTEGER (int64), matched against m_aCashItemInfo[i].liSN to decrement/remove the entry's nNumber` | ✅ |  |
| 2 | int32 | int32 `count (int32, formatted directly into the notice string)` | ✅ |  |

