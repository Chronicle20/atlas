# CashIncTrunkCountSuccess (← `CCashShop::OnCashItemResult#INC_TRUNK_COUNT_SUCCESS`)

- **IDA:** 0x494ed0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x6F INC_TRUNK_COUNT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResIncTrunkCountDone)` | ✅ |  |
| 1 | int16 | int16 `trunkCount (new absolute m_nTrunkCount); Decode2@0x494edf` | ✅ |  |

