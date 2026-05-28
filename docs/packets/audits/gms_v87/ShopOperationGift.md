# ShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x47a168
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `leading int (*(this+1308)). v87 is a 4-byte INT (line 147), NOT EncodeStr sSPW. The leading SPW string is v95-only; SPW gate >=95 CORRECT.` | ✅ |  |
| 1 | int32 | int32 `serialNumber (*(this+1312))` | ✅ |  |
| 2 | byte | byte `m_bRequestBuyOneADay byte (*(this+9928)). PRESENT at v87 (line 149) before name — NOT v95-only. oneADay gate tightened to GMS>=87 (split from the v95-only SPW gate).` | ✅ |  |
| 3 | string | string `recipient name (v33)` | ✅ |  |
| 4 | string | string `message (*(this+1316))` | ✅ |  |

