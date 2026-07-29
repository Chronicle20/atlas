# CashUseCouponDone (← `CCashShop::OnCashItemResult#USE_COUPON_SUCCESS`)

- **IDA:** 0x479d8a
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x59 USE_COUPON_SUCCESS; op-byte consumed by dispatcher before OnCashItemResUseCouponDone)` | ✅ |  |
| 1 | byte | byte `itemCount (byte, count-prefix for the item list)` | ✅ |  |
| 2 | bytes | bytes `GW_CashItemInfo entry (55B CashInventoryItem blob)` | ✅ |  |
| 3 | int32 | int32 `maplePoint` | ✅ |  |
| 4 | int32 | int32 `uliCount (count-prefix for the packed-ref list)` | ✅ |  |
| 5 | bytes | bytes `packed _ULARGE_INTEGER-shaped record (8B: quantity u16@0, slotPos u16@2, itemId i32@4)` | ✅ |  |
| 6 | int32 | int32 `meso` | ✅ |  |

