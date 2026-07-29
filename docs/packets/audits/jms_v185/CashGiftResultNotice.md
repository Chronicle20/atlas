# CashGiftResultNotice (← `CCashShop::OnCashItemResult#GIFT_RESULT_NOTICE`)

- **IDA:** 0x48ba24
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4c=76 GIFT_RESULT_NOTICE; op-byte consumed by dispatcher before CCashShop__OnCashItemResShowGiftResultNotice)` | ✅ |  |
| 1 | byte | byte `reason byte (passed to gift-result notice sub_48F0F2@0x48f0f2: 214/215/216 -> StringPool 625/626/627, else 624)` | ✅ |  |

