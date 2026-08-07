# CashLoadReceivedGiftDone (← `CCashShop::OnCashItemResult#LOAD_RECEIVED_GIFT_SUCCESS`)

- **IDA:** 0x48ba3f
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4d=77 LOAD_RECEIVED_GIFT_SUCCESS; op-byte consumed by dispatcher before CCashShop__OnCashItemResLoadReceivedGiftDone)` | ✅ |  |
| 1 | byte | byte `flag byte (v26; when 0 the client shows StringPool notice 626 after the loop)` | ✅ |  |
| 2 | int32 | int32 `count (uint32, count-prefix for the received-gift list)` | ✅ |  |
| 3 | bytes | bytes `ReceivedGiftEntry (176B/0xB0 GW_GiftList-shaped: itemId i32@12, data1 i32@16, data2 i32@20, giftType i32@24, text char[101]@28, sender char[33]@129, itemName char[14]@162)` | ✅ |  |

