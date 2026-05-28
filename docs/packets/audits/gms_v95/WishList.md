# WishList (← `CCashShop::OnCashItemResult#WishList`)

- **IDA:** 0x494d60
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x62 SET_WISH_DONE; op-byte consumed by dispatcher)` | ✅ |  |
| 1 | int32 | bytes `40 bytes = 10 x int32 wishlist SNs` | ❌ | width mismatch |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


> ack: tool limitation — analyzer flattens the 10×WriteInt loop; atlas writes mode+40 bytes matching v95 Decode1+DecodeBuffer(40). Wire-correct. See _pending.md "Cash tool-limitation false positives".
