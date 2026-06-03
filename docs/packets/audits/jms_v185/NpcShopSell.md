# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x7cacab
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `op = 1 (sell @0x7cae68)` | ❌ | width mismatch |
| 1 | int32 | int16 `nPOS / slot (@0x7cae73)` | ❌ | width mismatch |
| 2 | int16 | int32 `nItemID (@0x7cae7e)` | ❌ | width mismatch |
| 3 | byte | int16 `quantity (@0x7cae89)` | ❌ | atlas: short — missing trailing field |


## Manual verdict (JMS v185, `CShopDlg::SendSellRequest` @0x7cacab)

The ❌ rows are an op-byte-discriminator alignment artifact, NOT a wire bug. JMS185
`SendSellRequest` builds `COutPacket(0x35) + Encode1(1=op) + Encode2(slot) + Encode4(itemId)
+ Encode2(quantity)`. Atlas carries the op byte in the `Shop` wrapper (`NpcShop` ✅) and
`ShopSell` emits the body `Int16(slot)+Int(itemId)+Short(quantity)`. The audit aligns the body
against the op-prefixed IDA `calls`, shifting fields by one. Body matches field-for-field.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
