# CashLoadGiftDone (← `CCashShop::OnCashItemResult#LOAD_GIFT_SUCCESS`)

- **IDA:** 0x47122e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x39 LOAD_GIFT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResLoadGiftDone)` | ✅ |  |
| 1 | int16 | int16 `count (uint16, count-prefix for the gift list)` | ✅ |  |
| 2 | bytes | bytes `GW_GiftList entry (98B: liSN i64@0, nItemID i32@8, sBuyCharacterName char[13]@12, sText char[73]@25)` | ✅ |  |

