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

