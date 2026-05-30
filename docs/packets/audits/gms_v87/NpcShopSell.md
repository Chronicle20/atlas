# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x7a256b
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `inventory position (v28)` | ✅ |  |
| 1 | int32 | int32 `itemId (v27)` | ✅ |  |
| 2 | int16 | int16 `quantity (v29)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
