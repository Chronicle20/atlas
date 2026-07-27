# CashIncCharacterSlotCountFailed (← `CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_FAILED`)

- **IDA:** 0x473bec
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5D INC_CHARACTER_SLOT_COUNT_FAILED; op-byte consumed by dispatcher before OnCashItemResIncCharacterSlotCountFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

