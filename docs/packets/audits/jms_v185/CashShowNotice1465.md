# CashShowNotice1465 (← `CCashShop::OnCashItemResult#SHOW_NOTICE_1465`)

- **IDA:** 0x48c26e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa6=166 SHOW_NOTICE_1465; op-byte consumed by dispatcher; bodyless - handler reads nothing, shows StringPool notice 0x1465)` | ✅ |  |

