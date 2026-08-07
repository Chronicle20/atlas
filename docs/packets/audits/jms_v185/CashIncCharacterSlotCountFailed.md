# CashIncCharacterSlotCountFailed (← `CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_FAILED`)

- **IDA:** 0x48d7ef
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x66 INC_CHARACTER_SLOT_COUNT_FAILED; op-byte consumed by dispatcher before OnCashItemResIncCharacterSlotCountFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

