# NpcShop (← `CShopDlg::OnPacket#ShopDispatch`)

- **IDA:** 0x7ca2c9
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op (0=buy,1=sell,2=recharge)` | ✅ |  |

