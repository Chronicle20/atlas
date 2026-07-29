# CashRefreshLocker (← `CCashShop::OnCashItemResult#REFRESH_LOCKER`)

- **IDA:** 0x48c321
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa2=162 REFRESH_LOCKER; op-byte consumed by dispatcher before CCashShop__OnCashItemResRefreshLocker)` | ✅ |  |
| 1 | bytes | bytes `item: 55 bytes (0x37) GW_CashItemInfo blob (CashInventoryItem.EncodeBytes) decoded into a fresh locker slot` | ✅ |  |

