# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x47bced
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `commSN (serialNumber). op-byte 0x2E (NOT GMS 0x04). JMS gift = serialNumber ONLY — no SPW/birthday, no recipient name, no message, no oneADay. NX-system divergence` | ❌ | atlas: short — missing trailing field |

