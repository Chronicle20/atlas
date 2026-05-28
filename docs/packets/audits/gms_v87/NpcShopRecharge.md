# NpcShopRecharge (← `CShopDlg::SendRechargeRequest`)

- **IDA:** 0x7a278f
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_recharge.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `inventory position (v7)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
