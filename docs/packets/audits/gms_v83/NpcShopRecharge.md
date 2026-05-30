# NpcShopRecharge (← `CShopDlg::SendRechargeRequest`)

- **IDA:** 0x756c28
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_recharge.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot / nPos (v7)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
