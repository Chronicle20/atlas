# NpcShop (← `CShopDlg::OnPacket#ShopDispatch`)

- **IDA:** 0x6e4b80
- **Atlas file:** `libs/atlas-packet/npc/serverbound/shop.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (shop request discriminator: 0=BUY,1=SELL,2=RECHARGE; runtime-config values)` | ✅ |  |


Ack: world-audit Phase 2g on 2026-05-28
