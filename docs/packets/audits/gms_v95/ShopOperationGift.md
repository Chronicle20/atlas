# ShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x487b60
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | width mismatch |
| 1 | int32 | int32 `m_sg.nCommSN (serialNumber)` | ✅ |  |
| 2 | string | byte `m_bRequestBuyOneADay (NOT in atlas - missing byte between serialNumber and name)` | ❌ | width mismatch |
| 3 | string | string `recipient name (sDone)` | ✅ |  |
| 4 | byte | string `m_sg.sText (message)` | ❌ | atlas: short — missing trailing field |


> defer: version-gated — leading SPW string (atlas: int birthday) + missing byte(oneADay). See _pending.md SPW + ShopOperationBuy sections.
