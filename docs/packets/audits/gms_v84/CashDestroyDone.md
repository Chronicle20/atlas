# CashDestroyDone (← `CCashShop::OnCashItemResult#DESTROY_SUCCESS`)

- **IDA:** 0x47e5be
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6f DESTROY_SUCCESS; op-byte consumed by dispatcher before CCashShop::OnCashItemResDestroyDone)` | ✅ |  |
| 1 | int64 | bytes `sn: 8 bytes LARGE_INTEGER (int64), matched against m_aCashItemInfo[i].liSN to find-and-remove the locker entry` | ✅ |  |

