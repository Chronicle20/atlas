# NpcShopRecharge (← `CShopDlg::SendRechargeRequest`)

- **IDA:** 0x7caecf
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_recharge.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op = 2 (recharge @0x7caff8)` | ✅ |  |
| 1 | int16 | int16 `nPos / slot (@0x7cb001)` | ✅ |  |

