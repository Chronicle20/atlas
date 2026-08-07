# CashDestroyDone (← `CCashShop::OnCashItemResult#DESTROY_SUCCESS`)

- **IDA:** 0x463150
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4E DESTROY_SUCCESS; op-byte consumed by dispatcher before OnCashItemResDestroyDone)` | ✅ |  |
| 1 | int64 | bytes `sn (8B _LARGE_INTEGER SN; matched against existing m_aCashItemInfo[i].liSN, find-and-remove pattern)` | ✅ |  |

