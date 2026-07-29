# CashGiftCouponDone (← `CCashShop::OnCashItemResult#GIFT_COUPON_SUCCESS`)

- **IDA:** 0x45487a
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x38 GIFT_COUPON_SUCCESS; op-byte consumed by dispatcher before OnCashItemResGiftCouponDone)` | ✅ |  |
| 1 | string | string `recipientName` | ✅ |  |
| 2 | byte | byte `itemCount (byte, count-prefix for the item list)` | ✅ |  |
| 3 | bytes | bytes `GW_CashItemInfo entry (55B CashInventoryItem blob)` | ✅ |  |
| 4 | int32 | int32 `maplePoint` | ✅ |  |

