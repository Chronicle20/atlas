# CashRefreshLockerOrNotice (← `CCashShop::OnCashItemResult#REFRESH_LOCKER_OR_NOTICE`)

- **IDA:** 0x48c373
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa7=167 REFRESH_LOCKER_OR_NOTICE; op-byte consumed by dispatcher before CCashShop__OnCashItemResRefreshLockerOrNotice)` | ✅ |  |
| 1 | byte | byte `flag byte (v12; selects StringPool 5219 vs 5216 when the locker window is not open)` | ✅ |  |
| 2 | bytes | bytes `item: 55 bytes (0x37) GW_CashItemInfo blob (CashInventoryItem.EncodeBytes) decoded into a fresh locker slot` | ✅ |  |

