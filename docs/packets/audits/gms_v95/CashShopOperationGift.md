# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x487b60
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `m_sg.nCommSN (serialNumber)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | byte `m_bRequestBuyOneADay (NOT in atlas - missing byte between serialNumber and name)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `recipient name (sDone)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `m_sg.sText (message)` | ❌ | atlas: short — missing trailing field |

