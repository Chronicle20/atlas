# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x7ca2c9
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `op = 0 (buy @0x7caaf3)` | ❌ | width mismatch |
| 1 | int32 | int16 `slot / commodity index (@0x7cab10)` | ❌ | width mismatch |
| 2 | int16 | int32 `itemId (@0x7cab20)` | ❌ | width mismatch |
| 3 | byte | int16 `quantity (@0x7cab2b)` | ❌ | atlas: short — missing trailing field |


## Manual verdict (JMS v185, `CShopDlg::SendBuyRequest` @0x7ca2c9)

The ❌ rows are an op-byte-discriminator alignment artifact, NOT a wire bug. JMS185
`SendBuyRequest` builds `COutPacket(0x35) + Encode1(0=op) + Encode2(slot) + Encode4(itemId)
+ Encode2(quantity)`. Atlas splits the leading op byte into the separate `Shop` wrapper
(see `NpcShop` ✅) and `ShopBuy` carries only the body `Short(slot)+Int(itemId)+Short(quantity)`.
The audit aligns atlas's body against the op-prefixed IDA `calls`, shifting every field by one
(hence the width-mismatch rows). FIX APPLIED THIS TASK: the trailing GMS `discountPrice` int
(present in GMS v83/v87/v95 `SendBuyRequest`) is now gated `Region()=="GMS"`; JMS185 omits it
(`SendBuyRequest` ends at `Encode2(quantity)@0x7cab2b`), so atlas no longer over-writes/over-reads
4 bytes for JMS. Body fields match field-for-field after the op byte.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
