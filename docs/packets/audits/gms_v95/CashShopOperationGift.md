# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x487b60
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ✅ |  |
| 1 | int32 | int32 `m_sg.nCommSN (serialNumber)` | ✅ |  |
| 2 | byte | byte `m_bRequestBuyOneADay (NOT in atlas - missing byte between serialNumber and name)` | ✅ |  |
| 3 | string | string `recipient name (sDone)` | ✅ |  |
| 4 | string | string `m_sg.sText (message)` | ✅ |  |

