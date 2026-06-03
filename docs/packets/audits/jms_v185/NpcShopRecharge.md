# NpcShopRecharge (← `CShopDlg::SendRechargeRequest`)

- **IDA:** 0x7caecf
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_recharge.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `op = 2 (recharge @0x7caff8)` | ❌ | width mismatch |
| 1 | byte | int16 `nPos / slot (@0x7cb001)` | ❌ | atlas: short — missing trailing field |


## Manual verdict (JMS v185, `CShopDlg::SendRechargeRequest` @0x7caecf)

The ❌ rows are an op-byte-discriminator alignment artifact, NOT a wire bug. JMS185
`SendRechargeRequest` builds `COutPacket(0x35) + Encode1(2=op) + Encode2(slot)`. Atlas carries
the op byte in the `Shop` wrapper (`NpcShop` ✅) and `ShopRecharge` emits `Short(slot)`. The
audit aligns the single body short against the op-prefixed IDA `calls`. Body matches.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
