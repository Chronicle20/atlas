# CashShowNotice1464 (← `CCashShop::OnCashItemResult#SHOW_NOTICE_1464`)

- **IDA:** 0x48c413
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa8=168 SHOW_NOTICE_1464; op-byte consumed by dispatcher; bodyless - handler reads nothing, shows StringPool notice 0x1464)` | ✅ |  |

