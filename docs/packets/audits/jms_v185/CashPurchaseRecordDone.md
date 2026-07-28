# CashPurchaseRecordDone (← `CCashShop::OnCashItemResult#PURCHASE_RECORD`)

- **IDA:** 0x48e72f
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x9d PURCHASE_RECORD; op-byte consumed by dispatcher before CCashShop::OnCashItemResPurchaseRecord)` | ✅ |  |
| 1 | int32 | int32 `goodsSN (int32, used as a ZMap<long,...> key when nonzero)` | ✅ |  |
| 2 | byte | byte `purchased (byte, compared != 0, recorded in m_mPurchaseRecord/m_nPurchaseRecord)` | ✅ |  |

