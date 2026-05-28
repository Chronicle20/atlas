# NpcShopRecharge (← `CShopDlg::SendRechargeRequest`)

- **IDA:** 0x6e4e90
- **Atlas file:** `libs/atlas-packet/npc/serverbound/shop_recharge.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (inventory position nPos)` | ✅ |  |


Ack: world-audit Phase 2g on 2026-05-28
