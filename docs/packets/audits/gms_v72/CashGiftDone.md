# CashGiftDone (← `CCashShop::OnCashItemResult#GIFT_SUCCESS`)

- **IDA:** 0x4724e2
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x4a GIFT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResGiftDone)` | ✅ |  |
| 1 | string | string `recipientName` | ✅ |  |
| 2 | int32 | int32 `itemId (name-lookup key only; item never inserted into m_aCashItemInfo)` | ✅ |  |
| 3 | int16 | int16 `quantity` | ✅ |  |
| 4 | int32 | int32 `nxCashSpent (GMS-only; JMS build omits this trailing field — confirmed absent live)` | ✅ |  |

