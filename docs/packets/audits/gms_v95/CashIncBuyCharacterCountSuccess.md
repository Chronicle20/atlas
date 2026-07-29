# CashIncBuyCharacterCountSuccess (← `CCashShop::OnCashItemResult#INC_BUY_CHARACTER_COUNT_SUCCESS`)

- **IDA:** 0x495000
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x73 INC_BUY_CHARACTER_COUNT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResIncBuyCharacterCountDone)` | ✅ |  |
| 1 | int16 | int16 `buyCharacterCount (new absolute m_nBuyCharacterCount); Decode2@0x495007` | ✅ |  |

