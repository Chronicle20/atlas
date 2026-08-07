# CashBuyNormalDone (← `CCashShop::OnCashItemResult#BUY_NORMAL_SUCCESS`)

- **IDA:** 0x486de1
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x92 BUY_NORMAL_SUCCESS; op-byte consumed by dispatcher before OnCashItemResBuyNormalDone)` | ✅ |  |
| 1 | int32 | int32 `count (int32, count-prefix for the packed-ref list)` | ✅ |  |
| 2 | bytes | bytes `packed _ULARGE_INTEGER-shaped record (8B: quantity u16@0, slotPos u16@2, itemId i32@4) — same shape as USE_COUPON_SUCCESS's second list` | ✅ |  |

