# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x47bced
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `commSN (serialNumber). op-byte 0x2E (NOT GMS 0x04). JMS gift = serialNumber ONLY — no SPW/birthday, no recipient name, no message, no oneADay. NX-system divergence` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |

